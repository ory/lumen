# Installing Lumen for OpenCode

<!-- Updated 2026-11-10T17:42:18-03:00: align setup and verification with OpenCode V2. -->

Install Lumen as an OpenCode plugin from npm. The plugin registers a local MCP
server automatically.

## Prerequisites

- [OpenCode.ai](https://opencode.ai) V2, version 2.2.7 or newer

## Installation

Add Lumen to the `plugins` array in your `opencode.json`:

```json
{
  "plugins": ["@ory/lumen-opencode"]
}
```

The plugin registers a local `mcp.lumen` server that runs `scripts/run stdio`
on all platforms (cross-spawn resolves `scripts/run.cmd` via PATHEXT on
Windows). OpenCode reloads configuration changes automatically; use `/reload`
if the server does not appear after installation.

## Verify

In an active OpenCode TUI, ask it to call the Lumen `semantic_search`,
`health_check`, or `index_status` MCP tools. If the server does not appear,
use `/reload` to reload the plugin configuration.

## Updating

Update the package version in `opencode.json` to select a specific release:

```json
{
  "plugins": ["@ory/lumen-opencode@<version>"]
}
```

## Uninstalling

Remove the `@ory/lumen-opencode` entry from `opencode.json`.
