import { spawn, spawnSync } from "node:child_process";
import {
  accessSync,
  constants,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  realpathSync,
  rmSync,
  symlinkSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

const API_REQUEST_TIMEOUT_MS = 10_000;
const PLUGIN_READY_TIMEOUT_MS = 30_000;
const SERVER_START_TIMEOUT_MS = 30_000;
const MAX_LOG_CHARS = 64 * 1024;
const POLL_INTERVAL_MS = 250;

function requirePackageRoot(input) {
  if (!input) {
    throw new Error(
      "Usage: node scripts/test_opencode_v2_host.mjs <package-root>",
    );
  }

  const packageRoot = realpathSync(path.resolve(input));
  const packageJson = JSON.parse(
    readFileSync(path.join(packageRoot, "package.json"), "utf8"),
  );
  if (packageJson.name !== "@ory/lumen-opencode") {
    throw new Error(`Unexpected OpenCode package at ${packageRoot}`);
  }
  return packageRoot;
}

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

    function cleanup() {
      clearTimeout(timeout);
      server.stdout.removeListener("data", inspect);
      server.stderr.removeListener("data", inspect);
      server.removeListener("error", onError);
      server.removeListener("exit", onExit);
    }

    function finish(error, details) {
      if (settled) return;
      settled = true;
      cleanup();
      if (error) reject(error);
      else resolve(details);
    }

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

    function onError(error) {
      finish(new Error(`Could not start OpenCode V2: ${error.message}`));
    }

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

function redactLogs(logs, password) {
  const redactedPassword = password
    ? logs.replaceAll(password, "[redacted]")
    : logs;
  return redactedPassword
    .replace(/server password\s+\S+/gi, "server password [redacted]")
    .slice(-8_000);
}

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

function processGroupExists(pid) {
  try {
    process.kill(-pid, 0);
    return true;
  } catch (error) {
    if (error.code === "ESRCH") return false;
    throw error;
  }
}

async function waitForProcessGroupExit(server, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  do {
    const serverExited = server.exitCode !== null || server.signalCode !== null;
    if (serverExited && !processGroupExists(server.pid)) return true;
    await new Promise((resolve) => setTimeout(resolve, 50));
  } while (Date.now() < deadline);
  return false;
}

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

async function main() {
  const packageRoot = requirePackageRoot(process.argv[2]);
  const tempParent = path.join(tmpdir(), "opencode");
  mkdirSync(tempParent, { mode: 0o700, recursive: true });
  // ASVS 5.3.2: use an internally generated directory for all canary files.
  const root = mkdtempSync(path.join(tempParent, "lumen-opencode-v2-canary-"));
  let logOutput = "";
  let server;
  let password = "";
  try {
    const projectDirectory = path.join(root, "project");
    mkdirSync(projectDirectory, { mode: 0o700 });

    const env = createIsolatedEnvironment(root);
    const pluginDirectory = path.join(
      env.XDG_CONFIG_HOME,
      "opencode",
      "plugins",
    );
    mkdirSync(pluginDirectory, { mode: 0o700, recursive: true });
    symlinkSync(packageRoot, path.join(pluginDirectory, "lumen"), "dir");

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
    await waitForPlugin(baseUrl, authorization, projectDirectory);

    console.log(
      `OpenCode V2 host canary passed: ${version}; Lumen MCP connected`,
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
      rmSync(root, { force: true, recursive: true });
    }
  }
}

main().catch((error) => {
  console.error(error.message);
  process.exitCode = 1;
});
