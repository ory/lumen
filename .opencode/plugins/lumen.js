import { Plugin } from "@opencode/plugin";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const pluginRoot = path.resolve(__dirname, "../..");
const launcher = process.platform === "win32" ? "run.cmd" : "run";
const runCommand = path.join(pluginRoot, "scripts", launcher);

export default Plugin.define({
  id: "ory.lumen",
  /** Preserve an existing Lumen MCP configuration while supplying a local default. */
  async setup(ctx) {
    await ctx.mcp.transform((editor) => {
      if (!editor.get("lumen")) {
        editor.set("lumen", {
          type: "local",
          command: [runCommand, "stdio"],
          disabled: false,
        });
      }
    });
  },
});
