# Installing Lumen for Codex

Enable Lumen in Codex with native skill discovery plus a registered MCP
server and Codex startup hooks.

## Prerequisites

- [Codex CLI](https://developers.openai.com/codex/cli)
- Git

## Installation

1. Clone the repository:
   ```bash
   CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
   git clone https://github.com/ory/lumen.git "$CODEX_HOME/lumen"
   ```

2. Run the Codex installer:
   ```bash
   "$CODEX_HOME/lumen/scripts/run" codex install
   ```

3. Restart Codex.

The installer enables Codex hooks in `config.toml`, registers the `lumen` MCP
server, links or copies the shared Lumen skills into Codex's user skill
directory, and installs a user-level Codex `SessionStart` hook. That hook runs
on startup, resume, and clear events and reuses Lumen's background indexer to
proactively warm the project index.

## Windows (PowerShell)

```powershell
$codexHome = if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $env:USERPROFILE ".codex" }
git clone https://github.com/ory/lumen.git "$codexHome\lumen"
cmd /c "$codexHome\lumen\scripts\run.cmd" codex install
```

Restart Codex after the installer completes.

## Migrating from the old repo-local plugin

If you previously used the repo-local Codex marketplace package:

1. Remove the old plugin from Codex's plugin UI.
2. Run `"$CODEX_HOME/lumen/scripts/run" codex install`.
3. Restart Codex.

If you previously hand-edited Codex MCP or hook configuration, remove stale
manual entries after confirming the installer-managed configuration works. The
installer owns the user-level Codex MCP registration, skill link or copy, and
`hooks.json` `SessionStart` entry for Lumen.

## Verify

```bash
"${CODEX_HOME:-$HOME/.codex}/lumen/scripts/run" codex doctor
codex mcp get lumen
```

## Updating

```bash
cd "${CODEX_HOME:-$HOME/.codex}/lumen" && git pull
"${CODEX_HOME:-$HOME/.codex}/lumen/scripts/run" codex install
```

## Uninstalling

```bash
codex mcp remove lumen
skills="$HOME/.agents/skills/lumen"
if [ -L "$skills" ]; then
  target="$(readlink "$skills")"
  case "$target" in
    */lumen/skills) rm "$skills" ;;
    *) echo "Refusing to remove non-Lumen skills link: $skills -> $target" >&2 ;;
  esac
elif [ -f "$skills/.lumen-skills-source" ]; then
  rm -rf "$skills"
else
  echo "No installer-managed Lumen skills path found at $skills" >&2
fi
```

Remove the Lumen `SessionStart` group from
`${CODEX_HOME:-$HOME/.codex}/hooks.json` if you want to remove the startup hook
as well. If you no longer use any Codex hooks, you can also remove
`hooks = true` from the `[features]` table in
`${CODEX_HOME:-$HOME/.codex}/config.toml`. Optionally delete the clone:
`rm -rf "${CODEX_HOME:-$HOME/.codex}/lumen"`.
