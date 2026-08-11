# Installing Lumen for Codex

Install Lumen as a native Codex plugin from Ory's marketplace. This bundles
the MCP server configuration and the `doctor` and `reindex` skills in one
package.

## Prerequisites

- Codex CLI 0.147.0 or newer
- Git
- [Ollama](https://ollama.com/) or [LM Studio](https://lmstudio.ai/)

## Fresh installation

Add the Ory marketplace the first time you use it:

```bash
codex plugin marketplace add ory/claude-plugins
codex plugin add lumen@ory
```

If the `ory` marketplace is already configured, refresh it before installing:

```bash
codex plugin marketplace upgrade ory
codex plugin add lumen@ory
```

Restart Codex after installation so the new skills and MCP tools are loaded.

## Repairing a legacy or broken installation

Older marketplace snapshots could install Lumen's Claude manifest, leaving an
unexpanded `${CLAUDE_PLUGIN_ROOT}` in the effective MCP command. Refresh the
marketplace and reinstall the plugin from scratch:

```bash
codex plugin marketplace upgrade ory
codex plugin remove lumen@ory
codex plugin add lumen@ory
```

Then restart Codex.

## Migrating from the manual clone installation

If you previously cloned Lumen and registered it directly, remove both the
standalone MCP registration and skill link before installing the native plugin.
This prevents duplicate `lumen` MCP servers or skill names.

```bash
codex mcp remove lumen
rm "$HOME/.agents/skills/lumen"
codex plugin marketplace add ory/claude-plugins
codex plugin add lumen@ory
```

If `ory` is already configured, use `codex plugin marketplace upgrade ory`
instead of adding it again. The old clone can be deleted separately after you
confirm the plugin works.

On Windows, remove the old junction and MCP registration before installing:

```powershell
codex mcp remove lumen
Remove-Item "$env:USERPROFILE\.agents\skills\lumen"
codex plugin marketplace add ory/claude-plugins
codex plugin add lumen@ory
```

## Verify

```bash
codex mcp get lumen --json
```

The reported `command` must be an absolute cached path ending in `scripts/run`
(or the Windows-resolved launcher) and must not contain
`${CLAUDE_PLUGIN_ROOT}`. The plugin downloads its platform binary into Codex's
writable plugin data directory on first use.

## Updating

```bash
codex plugin marketplace upgrade ory
codex plugin add lumen@ory
```

Restart Codex to pick up updated skills and MCP configuration.

## Uninstalling

```bash
codex plugin remove lumen@ory
```
