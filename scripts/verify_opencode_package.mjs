import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { pathToFileURL } from "node:url";

const packageRoot = path.resolve(process.argv[2] || ".");
const packageJsonPath = path.join(packageRoot, "package.json");
const pluginPath = path.join(packageRoot, ".opencode", "plugins", "lumen.js");
const launcherPath = path.join(
  packageRoot,
  "scripts",
  process.platform === "win32" ? "run.cmd" : "run",
);

for (const required of [
  packageJsonPath,
  pluginPath,
  launcherPath,
  path.join(packageRoot, ".release-please-manifest.json"),
]) {
  if (!fs.existsSync(required))
    throw new Error(`packed OpenCode plugin is missing ${required}`);
}

const packageName = JSON.parse(fs.readFileSync(packageJsonPath, "utf8")).name;
const packageEntry = createRequire(packageJsonPath).resolve(packageName);
if (fs.realpathSync(packageEntry) !== fs.realpathSync(pluginPath)) {
  throw new Error(
    `OpenCode package resolves to the wrong entrypoint: ${packageEntry}`,
  );
}

const pluginModule = await import(pathToFileURL(packageEntry));
const plugin = pluginModule.default;
if (!plugin || typeof plugin !== "object") {
  throw new Error("OpenCode V2 plugin must default-export a plugin definition");
}
if (plugin.id !== "ory.lumen" || typeof plugin.setup !== "function") {
  throw new Error(
    "OpenCode V2 plugin definition must include its stable id and setup function",
  );
}

const servers = new Map();
const pluginContext = {
  mcp: {
    transform: (transform) => {
      transform({
        get: (name) => servers.get(name),
        set: (name, config) => servers.set(name, config),
      });
    },
  },
};
await plugin.setup(pluginContext);

const server = servers.get("lumen");
if (server?.type !== "local" || server.disabled !== false) {
  throw new Error(
    `OpenCode did not register an enabled local Lumen MCP server: ${JSON.stringify(server)}`,
  );
}
const command = server.command;
if (!Array.isArray(command) || !path.isAbsolute(command[0])) {
  throw new Error(
    `OpenCode did not resolve an absolute MCP command: ${JSON.stringify(command)}`,
  );
}
if (
  fs.realpathSync(command[0]) !== fs.realpathSync(launcherPath) ||
  command[1] !== "stdio"
) {
  throw new Error(
    `OpenCode resolved the wrong MCP command: ${JSON.stringify(command)}`,
  );
}

const existingServer = { type: "remote", url: "https://example.invalid/mcp" };
servers.set("lumen", existingServer);
await plugin.setup(pluginContext);
if (servers.get("lumen") !== existingServer) {
  throw new Error("OpenCode plugin overwrote a preconfigured Lumen MCP server");
}

console.log(`packed OpenCode V2 plugin resolves ${command.join(" ")}`);
