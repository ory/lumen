import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const packageRoot = path.resolve(process.argv[2] || ".");
const pluginPath = path.join(packageRoot, ".opencode", "plugins", "lumen.js");
const launcherPath = path.join(packageRoot, "scripts", process.platform === "win32" ? "run.cmd" : "run");

for (const required of [pluginPath, launcherPath, path.join(packageRoot, ".release-please-manifest.json")]) {
  if (!fs.existsSync(required)) throw new Error(`packed OpenCode plugin is missing ${required}`);
}

const pluginModule = await import(pathToFileURL(pluginPath));
if (typeof pluginModule.default !== "function") throw new Error("OpenCode plugin has no default factory");
const hooks = await pluginModule.default({});
if (typeof hooks.config !== "function") throw new Error("OpenCode plugin has no config hook");

const config = {};
await hooks.config(config);
const command = config.mcp?.lumen?.command;
if (!Array.isArray(command) || !path.isAbsolute(command[0])) {
  throw new Error(`OpenCode did not resolve an absolute MCP command: ${JSON.stringify(command)}`);
}
if (fs.realpathSync(command[0]) !== fs.realpathSync(launcherPath) || command[1] !== "stdio") {
  throw new Error(`OpenCode resolved the wrong MCP command: ${JSON.stringify(command)}`);
}

console.log(`packed OpenCode plugin resolves ${command.join(" ")}`);
