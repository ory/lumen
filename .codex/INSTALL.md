# Installing Lumen for Codex

Enable Lumen in Codex with native skill discovery plus a registered MCP
server.

## Prerequisites

- [Codex CLI](https://developers.openai.com/codex/cli)
- Git
- **Node.js 22+** (current Active LTS) — Lumen's MCP entry point is
  `launcher.mjs`, a small Node shim that selects the matching native binary
  for your OS/arch and fetches it on first use (subsequent runs are cache
  hits). Check with `node --version`.

## Installation

1. Clone the repository:
   ```bash
   CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
   git clone https://github.com/ory/lumen.git "$CODEX_HOME/lumen"
   ```

2. Create the skills symlink:
   ```bash
   mkdir -p "$HOME/.agents/skills"
   ln -s "$CODEX_HOME/lumen/skills" "$HOME/.agents/skills/lumen"
   ```

3. Register the MCP server with a generous startup timeout (Codex's default
   is 10 s, which can be tight on a cold-cache first run over a slow link):
   ```bash
   codex mcp add lumen --startup-timeout-sec 60 -- \
     node "$CODEX_HOME/lumen/launcher.mjs" stdio
   ```

4. Restart Codex.

## Windows (PowerShell)

```powershell
$codexHome = if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $env:USERPROFILE ".codex" }
git clone https://github.com/ory/lumen.git "$codexHome\lumen"
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.agents\skills" | Out-Null
cmd /c mklink /J "$env:USERPROFILE\.agents\skills\lumen" "$codexHome\lumen\skills"
codex mcp add lumen --startup-timeout-sec 60 -- `
  node "$codexHome\lumen\launcher.mjs" stdio
```

## Troubleshooting

### `node` not found when Codex starts the MCP server

Codex's spawned subprocess inherits a PATH that may not match your interactive
shell's PATH ([codex#6243](https://github.com/openai/codex/issues/6243)). If
Codex reports `node not found` while `which node` works in your terminal,
re-register with the absolute path:

```bash
codex mcp add lumen --startup-timeout-sec 60 -- \
  "$(which node)" "$CODEX_HOME/lumen/launcher.mjs" stdio
```

### MCP startup times out on first run

The launcher fetches a ~30 MB binary from GitHub Releases on first invocation
(subsequent runs are cache hits in <100 ms). Increase the timeout further
on slow connections:

```bash
codex mcp remove lumen
codex mcp add lumen --startup-timeout-sec 120 -- \
  node "$CODEX_HOME/lumen/launcher.mjs" stdio
```

The flag persists into `~/.codex/config.toml` under
`mcp_servers.lumen.startup_timeout_sec` — preserve that key if you later edit
the file by hand.

### Pre-warm the cache before first session

The launcher exposes a `prefetch` subcommand that downloads + verifies the
matching binary and exits — useful in CI or as a manual warm-up:

```bash
node "$CODEX_HOME/lumen/launcher.mjs" prefetch
```

## Migrating from the old repo-local plugin

If you previously used the repo-local Codex marketplace package:

1. Remove the old plugin from Codex's plugin UI.
2. Register the MCP server with `codex mcp add` as above.
3. Create the `~/.agents/skills/lumen` symlink.
4. Restart Codex.

## Verify

```bash
codex mcp get lumen
ls -la "$HOME/.agents/skills/lumen"
```

## Updating

```bash
cd "${CODEX_HOME:-$HOME/.codex}/lumen" && git pull
```

The launcher reads the version from `.release-please-manifest.json` after the
pull and fetches the matching binary on next invocation.

## Uninstalling

```bash
codex mcp remove lumen
rm "$HOME/.agents/skills/lumen"
```

Optionally delete the clone: `rm -rf "${CODEX_HOME:-$HOME/.codex}/lumen"`.

The cached binaries live under `$XDG_DATA_HOME/lumen/` (Unix) or
`%LOCALAPPDATA%\lumen\` (Windows); remove that directory if you want a clean
slate.
