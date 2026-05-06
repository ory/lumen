# Installing Lumen for OpenCode

Install Lumen as an OpenCode plugin from npm. The plugin registers a local MCP
server automatically.

## Prerequisites

- [OpenCode.ai](https://opencode.ai) installed
- Node.js 22+ (Active LTS; used by the plugin's launcher — OpenCode itself runs on Bun/Node, so this is normally already present)

## Installation

Add Lumen to the `plugin` array in your `opencode.json`:

```json
{
  "plugin": ["@ory/lumen-opencode"]
}
```

Restart OpenCode. The plugin registers a local `mcp.lumen` server that runs
`node launcher.mjs stdio` on all platforms; the launcher resolves the
matching native lumen binary per OS/arch and lazily fetches it from GitHub
Releases on first use (subsequent runs are cache hits).

## Verify

```bash
opencode mcp list
```

Then ask OpenCode to call the Lumen `semantic_search`, `health_check`, or
`index_status` MCP tools directly.

## Updating

Restart OpenCode after updating the version pin in `opencode.json`, or pin a
specific version:

```json
{
  "plugin": ["@ory/lumen-opencode@0.0.29"]
}
```

## Uninstalling

Remove the `@ory/lumen-opencode` entry from `opencode.json` and restart
OpenCode.
