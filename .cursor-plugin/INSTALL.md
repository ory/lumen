# Installing Lumen for Cursor

Lumen ships a native Cursor plugin bundle in this repository.

## Bundle contents

- `.cursor-plugin/plugin.json` - Cursor plugin manifest
- `.cursor/mcp.json` - local `lumen` MCP server wiring
- `hooks/hooks-cursor.json` - SessionStart hook
- `skills/` - shared `doctor` and `reindex` skills
- `scripts/run.cjs` - Node-based MCP launcher (manifest entry)
- `scripts/run.cmd` - polyglot launcher used by hooks (and as a backward-compat fallback)

## Installation

Use Cursor's plugin installation or distribution workflow with this repository's
bundle rooted at `.cursor-plugin/plugin.json`.

This repository contains the package surface and runtime assets needed by
Cursor, but public marketplace publication is separate from the repository
itself.

## Verify

Open a new Cursor agent session and ask it to use the `doctor` skill or the
Lumen `semantic_search` tool.

## Updating

Update or republish the Cursor plugin bundle after pulling the latest version
of this repository.
