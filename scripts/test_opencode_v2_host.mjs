import { createHash } from "node:crypto";
import { spawn, spawnSync } from "node:child_process";
import { createServer } from "node:http";
import {
  accessSync,
  constants,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const API_REQUEST_TIMEOUT_MS = 10_000;
const PLUGIN_READY_TIMEOUT_MS = 30_000;
const SERVER_START_TIMEOUT_MS = 30_000;
const MAX_LOG_CHARS = 64 * 1024;
const POLL_INTERVAL_MS = 250;
const PACKAGE_NAME = "@ory/lumen-opencode";

/** Resolves the installed Lumen package and rejects source or unrelated packages. */
function requirePackageRoot(input) {
  if (!input) {
    throw new Error(
      "Usage: node scripts/test_opencode_v2_host.mjs <package-root> <packed-tarball>",
    );
  }

  const packageRoot = realpathSync(path.resolve(input));
  const packageJson = JSON.parse(
    readFileSync(path.join(packageRoot, "package.json"), "utf8"),
  );
  if (packageJson.name !== PACKAGE_NAME) {
    throw new Error(`Unexpected OpenCode package at ${packageRoot}`);
  }
  return packageRoot;
}

/** Resolves the exact packed tarball used to install the package under test. */
function requirePackedTarball(input) {
  if (!input) {
    throw new Error(
      "Usage: node scripts/test_opencode_v2_host.mjs <package-root> <packed-tarball>",
    );
  }

  return realpathSync(path.resolve(input));
}

/** Hashes regular package files while excluding dependencies installed beside the artifact. */
function packageFileHashes(packageRoot) {
  const root = realpathSync(packageRoot);
  const hashes = new Map();

  function visit(directory) {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      if (entry.name === "node_modules") continue;

      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) {
        visit(entryPath);
      } else if (entry.isFile()) {
        const relativePath = path.relative(root, entryPath);
        const hash = createHash("sha256")
          .update(readFileSync(entryPath))
          .digest("hex");
        hashes.set(relativePath, hash);
      } else {
        throw new Error(`Unexpected package entry type: ${entryPath}`);
      }
    }
  }

  visit(root);
  if (hashes.size === 0) {
    throw new Error(`Package contains no regular files: ${root}`);
  }
  return hashes;
}

/** Starts a loopback npm registry that serves only the package artifact under test. */
async function startPackageRegistry(packageRoot, tarballPath) {
  const manifest = JSON.parse(
    readFileSync(path.join(packageRoot, "package.json"), "utf8"),
  );
  if (
    manifest.name !== PACKAGE_NAME ||
    typeof manifest.version !== "string" ||
    !manifest.version
  ) {
    throw new Error("Packed artifact has an unexpected package identity");
  }

  const tarball = readFileSync(tarballPath);
  const requests = [];
  const server = createServer((request, response) => {
    let requestPath;
    try {
      requestPath = decodeURIComponent(
        new URL(request.url ?? "/", "http://127.0.0.1").pathname,
      );
    } catch {
      response.writeHead(400);
      response.end();
      return;
    }
    requests.push(requestPath);

    if (request.method !== "GET" && request.method !== "HEAD") {
      response.writeHead(405, { allow: "GET, HEAD" });
      response.end();
      return;
    }

    let status = 404;
    let contentType = "application/json";
    let body = Buffer.from("{}");
    if (requestPath === `/${PACKAGE_NAME}`) {
      const address = server.address();
      if (!address || typeof address === "string") {
        response.writeHead(500);
        response.end();
        return;
      }
      const version = manifest.version;
      const packageMetadata = {
        ...manifest,
        dist: {
          tarball: `http://127.0.0.1:${address.port}/package.tgz`,
          shasum: createHash("sha1").update(tarball).digest("hex"),
          integrity: `sha512-${createHash("sha512").update(tarball).digest("base64")}`,
        },
      };
      body = Buffer.from(
        JSON.stringify({
          _id: `${PACKAGE_NAME}@${version}`,
          name: PACKAGE_NAME,
          "dist-tags": { latest: version },
          versions: { [version]: packageMetadata },
        }),
      );
      status = 200;
      contentType = "application/json";
    } else if (requestPath === "/package.tgz") {
      status = 200;
      contentType = "application/octet-stream";
      body = tarball;
    }

    response.writeHead(status, {
      "content-length": body.length,
      "content-type": contentType,
    });
    response.end(request.method === "HEAD" ? undefined : body);
  });
  server.headersTimeout = API_REQUEST_TIMEOUT_MS;
  server.requestTimeout = API_REQUEST_TIMEOUT_MS;

  await new Promise((resolve, reject) => {
    const onError = (error) => reject(error);
    server.once("error", onError);
    // ASVS 15.4.2: let the OS bind an ephemeral loopback port atomically.
    server.listen(0, "127.0.0.1", () => {
      server.removeListener("error", onError);
      resolve();
    });
  });

  const address = server.address();
  if (!address || typeof address === "string") {
    await new Promise((resolve) => server.close(resolve));
    throw new Error("Local package registry did not publish its loopback port");
  }
  return {
    baseUrl: `http://127.0.0.1:${address.port}`,
    requests,
    server,
  };
}

/** Stops the private registry after the OpenCode process group has exited. */
async function stopPackageRegistry(registry) {
  if (!registry) return;
  await new Promise((resolve, reject) => {
    registry.server.close((error) => {
      if (error) reject(error);
      else resolve();
    });
  });
}

/** Proves package resolution and loaded file contents match the npm-installed artifact. */
function verifyPackageResolution(
  plugin,
  registry,
  packageRoot,
  cacheDirectory,
  logs,
) {
  if (
    plugin.source?.type !== "package" ||
    plugin.source.target !== PACKAGE_NAME
  ) {
    throw new Error(
      `OpenCode loaded an unexpected plugin source: ${JSON.stringify(plugin.source ?? null)}`,
    );
  }
  if (!registry.requests.includes(`/${PACKAGE_NAME}`)) {
    throw new Error(
      `OpenCode did not resolve ${PACKAGE_NAME} through the isolated registry (requests=${registry.requests.join(", ") || "none"})`,
    );
  }

  const entrypoint = logs.match(
    /entrypoint=(file:\/\/\S+?\.opencode\/plugins\/lumen\.js)/,
  );
  if (!entrypoint) {
    throw new Error("OpenCode did not report the loaded package entrypoint");
  }
  const loadedPackageRoot = realpathSync(
    path.resolve(path.dirname(fileURLToPath(entrypoint[1])), "..", ".."),
  );
  const isolatedCache = realpathSync(cacheDirectory);
  const relativeToCache = path.relative(isolatedCache, loadedPackageRoot);
  if (
    relativeToCache === ".." ||
    relativeToCache.startsWith(`..${path.sep}`) ||
    path.isAbsolute(relativeToCache)
  ) {
    throw new Error("OpenCode loaded the package outside the isolated cache");
  }
  const packedFiles = packageFileHashes(packageRoot);
  const loadedFiles = packageFileHashes(loadedPackageRoot);
  for (const [relativePath, expectedHash] of packedFiles) {
    if (loadedFiles.get(relativePath) !== expectedHash) {
      throw new Error(
        `OpenCode loaded a package file that differs from the packed artifact: ${relativePath}`,
      );
    }
  }
}

/** Finds the OpenCode CLI without connecting to the user's shared server. */
function resolveOpenCode() {
  for (const directory of (process.env.PATH ?? "").split(path.delimiter)) {
    if (!directory) continue;

    const candidate = path.resolve(directory, "opencode");
    try {
      accessSync(candidate, constants.X_OK);
      return candidate;
    } catch {}
  }

  throw new Error("OpenCode CLI is not available on PATH");
}

/** Creates fresh HOME/XDG roots and forwards only canary-required settings. */
function createIsolatedEnvironment(root) {
  const home = path.join(root, "home");
  const config = path.join(root, "config");
  const data = path.join(root, "data");
  const cache = path.join(root, "cache");
  const state = path.join(root, "state");
  const temp = path.join(root, "tmp");
  for (const directory of [home, config, data, cache, state, temp]) {
    mkdirSync(directory, { mode: 0o700, recursive: true });
  }

  // ASVS 13.3.2: do not pass unrelated host credentials into the plugin process.
  const env = {
    CI: "1",
    HOME: home,
    NO_COLOR: "1",
    PATH: process.env.PATH ?? "/usr/bin:/bin",
    TMPDIR: temp,
    XDG_CACHE_HOME: cache,
    XDG_CONFIG_HOME: config,
    XDG_DATA_HOME: data,
    XDG_STATE_HOME: state,
  };
  for (const key of [
    "LANG",
    "LC_ALL",
    "LUMEN_BACKEND",
    "LUMEN_EMBED_DIMS",
    "LUMEN_EMBED_MODEL",
    "LUMEN_MAX_CHUNK_TOKENS",
    "OLLAMA_HOST",
  ]) {
    if (process.env[key] !== undefined) env[key] = process.env[key];
  }

  return env;
}

/** Waits for the private host's endpoint and ephemeral authentication password. */
function waitForServer(server, getLogs) {
  return new Promise((resolve, reject) => {
    let settled = false;
    const timeout = setTimeout(() => {
      finish(
        new Error(
          "OpenCode V2 server did not publish its endpoint and password",
        ),
      );
    }, SERVER_START_TIMEOUT_MS);

    /** Removes startup listeners after the promise reaches a terminal state. */
    function cleanup() {
      clearTimeout(timeout);
      server.stdout.removeListener("data", inspect);
      server.stderr.removeListener("data", inspect);
      server.removeListener("error", onError);
      server.removeListener("exit", onExit);
    }

    /** Settles startup once and releases the listeners used to observe it. */
    function finish(error, details) {
      if (settled) return;
      settled = true;
      cleanup();
      if (error) reject(error);
      else resolve(details);
    }

    /** Extracts and validates connection details from the server's startup logs. */
    function inspect() {
      const logs = getLogs();
      const endpoint = logs.match(
        /server listening on http:\/\/127\.0\.0\.1:(\d+)/i,
      );
      const password = logs.match(/server password\b\s*[:=]?\s*(\S+)/i);
      if (!endpoint || !password) return;

      const port = Number(endpoint[1]);
      if (!Number.isInteger(port) || port < 1 || port > 65_535) {
        finish(new Error("OpenCode V2 published an invalid loopback port"));
        return;
      }
      finish(undefined, { password: password[1], port });
    }

    /** Converts process-spawn failures into a bounded startup error. */
    function onError(error) {
      finish(new Error(`Could not start OpenCode V2: ${error.message}`));
    }

    /** Rejects startup if the host exits before publishing its endpoint. */
    function onExit(code, signal) {
      finish(
        new Error(
          `OpenCode V2 exited before startup (code=${code}, signal=${signal})`,
        ),
      );
    }

    server.stdout.on("data", inspect);
    server.stderr.on("data", inspect);
    server.once("error", onError);
    server.once("exit", onExit);
    inspect();
  });
}

/** Removes the ephemeral API credential from server diagnostics. */
function redactLogs(logs, password) {
  const redactedPassword = password
    ? logs.replaceAll(password, "[redacted]")
    : logs;
  return redactedPassword
    .replace(/server password\s+\S+/gi, "server password [redacted]")
    .slice(-8_000);
}

/** Sends a bounded authenticated request to the isolated host API. */
async function requestJson(baseUrl, route, authorization, body) {
  const hasBody = body !== undefined;
  // ASVS 13.2.6: bound local API requests and do not forward auth across redirects.
  const response = await fetch(new URL(route, baseUrl), {
    body: hasBody ? JSON.stringify(body) : undefined,
    headers: {
      authorization,
      ...(hasBody ? { "content-type": "application/json" } : {}),
    },
    method: hasBody ? "POST" : "GET",
    redirect: "error",
    signal: AbortSignal.timeout(API_REQUEST_TIMEOUT_MS),
  });
  const text = await response.text();
  if (!response.ok) {
    throw new Error(
      `OpenCode API ${route} returned ${response.status}: ${text.slice(0, 500)}`,
    );
  }
  return text ? JSON.parse(text) : null;
}

/** Polls one project location until the Lumen plugin and MCP server are ready. */
async function waitForPlugin(baseUrl, authorization, projectDirectory) {
  // ASVS 1.2.2: encode the project path as a query value rather than building a URL by hand.
  const location = new URLSearchParams({
    "location[directory]": projectDirectory,
  }).toString();
  const deadline = Date.now() + PLUGIN_READY_TIMEOUT_MS;
  let lumenPlugin;
  let lumenServer;

  do {
    // Resolve the location before inspecting plugin and MCP state; V2 loads these lazily.
    await requestJson(baseUrl, `/api/agent?${location}`, authorization);
    const plugins = await requestJson(
      baseUrl,
      `/api/plugin?${location}`,
      authorization,
    );
    const servers = await requestJson(
      baseUrl,
      `/api/mcp?${location}`,
      authorization,
    );
    if (!Array.isArray(plugins.data) || !Array.isArray(servers.data)) {
      throw new Error("OpenCode V2 returned malformed plugin or MCP status");
    }
    lumenPlugin = plugins.data.find((item) => item.id === "ory.lumen");
    lumenServer = servers.data.find((item) => item.name === "lumen");

    if (
      lumenPlugin?.state?.status === "active" &&
      lumenServer?.status?.status === "connected"
    ) {
      return { lumenPlugin, lumenServer };
    }
    if (
      lumenPlugin?.state?.status === "failed" ||
      lumenServer?.status?.status === "failed"
    ) {
      break;
    }
    await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
  } while (Date.now() < deadline);

  throw new Error(
    `Lumen did not activate and connect (plugin=${lumenPlugin?.state?.status ?? "missing"}, MCP=${lumenServer?.status?.status ?? "missing"})`,
  );
}

/** Checks whether the detached canary process group still exists. */
function processGroupExists(pid) {
  try {
    process.kill(-pid, 0);
    return true;
  } catch (error) {
    if (error.code === "ESRCH") return false;
    throw error;
  }
}

/** Waits for the host and its child processes to exit before a fixed deadline. */
async function waitForProcessGroupExit(server, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  do {
    const serverExited = server.exitCode !== null || server.signalCode !== null;
    if (serverExited && !processGroupExists(server.pid)) return true;
    await new Promise((resolve) => setTimeout(resolve, 50));
  } while (Date.now() < deadline);
  return false;
}

/** Stops the isolated process group, escalating only if graceful shutdown fails. */
async function stopServer(server) {
  if (!server?.pid) return;

  for (const signal of ["SIGTERM", "SIGKILL"]) {
    try {
      process.kill(-server.pid, signal);
    } catch (error) {
      if (error.code !== "ESRCH") throw error;
    }
    if (
      await waitForProcessGroupExit(
        server,
        signal === "SIGTERM" ? 10_000 : 5_000,
      )
    ) {
      return;
    }
  }

  throw new Error("OpenCode V2 canary process group did not exit");
}

/** Runs the installed-package canary and removes all generated host state. */
async function main() {
  const packageRoot = requirePackageRoot(process.argv[2]);
  const tarballPath = requirePackedTarball(process.argv[3]);
  const tempParent = path.join(tmpdir(), "opencode");
  mkdirSync(tempParent, { mode: 0o700, recursive: true });
  // ASVS 5.3.2: use an internally generated directory for all canary files.
  const root = mkdtempSync(path.join(tempParent, "lumen-opencode-v2-canary-"));
  let logOutput = "";
  let server;
  let registry;
  let password = "";
  try {
    const projectDirectory = path.join(root, "project");
    mkdirSync(projectDirectory, { mode: 0o700 });

    const env = createIsolatedEnvironment(root);
    registry = await startPackageRegistry(packageRoot, tarballPath);
    const npmrcPath = path.join(env.HOME, ".npmrc");
    // ASVS 5.3.2: keep temporary registry configuration inside the private canary home.
    writeFileSync(npmrcPath, `@ory:registry=${registry.baseUrl}/\n`, {
      mode: 0o600,
    });
    // ASVS 13.3.2: forward only the scoped registry setting, not host npm credentials.
    env.npm_config_userconfig = npmrcPath;
    env.NPM_CONFIG_USERCONFIG = npmrcPath;
    writeFileSync(
      path.join(projectDirectory, "opencode.json"),
      JSON.stringify({
        plugins: [PACKAGE_NAME],
      }),
      { mode: 0o600 },
    );

    const opencode = resolveOpenCode();
    // ASVS 1.2.5: pass fixed arguments directly, without shell interpolation.
    const versionResult = spawnSync(opencode, ["--version"], {
      encoding: "utf8",
      env,
      shell: false,
      timeout: API_REQUEST_TIMEOUT_MS,
    });
    const version = versionResult.stdout?.trim() ?? "";
    if (
      versionResult.error ||
      versionResult.status !== 0 ||
      !/\bv?2\.\d+\.\d+\b/i.test(version)
    ) {
      throw new Error(
        `Expected OpenCode V2 on PATH; got ${version || versionResult.stderr?.trim() || "no version"}`,
      );
    }

    // Let OpenCode generate its one-run password so MCP children never inherit it.
    // ASVS 15.4.2: let the OS assign the port atomically instead of probing then binding.
    server = spawn(
      opencode,
      [
        "serve",
        "--hostname",
        "127.0.0.1",
        "--port",
        "0",
        "--print-logs",
        "--log-level",
        "debug",
      ],
      {
        cwd: projectDirectory,
        detached: true,
        env,
        shell: false,
        stdio: ["ignore", "pipe", "pipe"],
      },
    );
    /** Keeps only bounded host output for failure diagnostics. */
    const capture = (chunk) => {
      logOutput = (logOutput + chunk.toString("utf8")).slice(-MAX_LOG_CHARS);
    };
    server.stdout.on("data", capture);
    server.stderr.on("data", capture);

    const { port, password: generatedPassword } = await waitForServer(
      server,
      () => logOutput,
    );
    password = generatedPassword;
    // ASVS 13.2.1/14.2.1: authenticate with the ephemeral credential, never in a URL.
    const authorization = `Basic ${Buffer.from(`opencode:${password}`).toString("base64")}`;
    const baseUrl = `http://127.0.0.1:${port}`;
    await requestJson(baseUrl, "/api/session", authorization, {
      location: { directory: projectDirectory },
      title: "Lumen V2 canary",
    });
    const { lumenPlugin } = await waitForPlugin(
      baseUrl,
      authorization,
      projectDirectory,
    );
    verifyPackageResolution(
      lumenPlugin,
      registry,
      packageRoot,
      env.XDG_CACHE_HOME,
      logOutput,
    );

    console.log(
      `OpenCode V2 host canary passed: ${version}; ${PACKAGE_NAME} matched the packed artifact; Lumen MCP connected`,
    );
  } catch (error) {
    // ASVS 16.5.1: prevent the generated API password from reaching CI diagnostics.
    const logs = redactLogs(logOutput, password);
    const originalMessage =
      error instanceof Error ? error.message : String(error);
    const message = password
      ? originalMessage.replaceAll(password, "[redacted]")
      : originalMessage;
    throw new Error(`${message}\nOpenCode logs:\n${logs}`, { cause: error });
  } finally {
    try {
      await stopServer(server);
    } finally {
      try {
        await stopPackageRegistry(registry);
      } finally {
        rmSync(root, { force: true, recursive: true });
      }
    }
  }
}

main().catch((error) => {
  console.error(error.message);
  process.exitCode = 1;
});
