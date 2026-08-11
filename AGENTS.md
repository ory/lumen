# AGENTS.md

## Repository Expectations

- The repo root is the shared agent integration surface. Keep `.claude-plugin/`, `.codex/`, `.cursor-plugin/`, `.cursor/`, `.opencode/`, `.agents/plugins/marketplace.json`, `hooks/`, `skills/`, `scripts/`, `plugins/lumen/`, and `package.json` aligned where they represent the same Lumen distribution.
- Do not add a repo-root `mcp.json` or repo-root `.codex-plugin/`. Claude Code reads repo-root `mcp.json` as project-scoped MCP config, which would change behavior for this repository. The Cursor MCP config lives in `.cursor/mcp.json`.
- Keep the launchers and skills under `plugins/lumen/` byte-identical to their repo-root counterparts; release tests enforce this parity.
- When changing plugin metadata for releases, keep `.claude-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `plugins/lumen/.codex-plugin/plugin.json`, and `package.json` aligned unless a difference is intentionally product-specific.

## Commands

- `make build-local` builds the local binary.
- `make test` runs the Go test suite.
- `make lint` runs `golangci-lint`.
