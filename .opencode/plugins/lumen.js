import { Plugin } from "@opencode/plugin";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const pluginRoot = path.resolve(__dirname, "../..");
const runCommand = path.join(pluginRoot, "scripts", "run");

export default Plugin.define({
  id: "ory.lumen",
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
