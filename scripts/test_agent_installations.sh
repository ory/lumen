#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
INSTALL_ROOT=$(mktemp -d)
trap 'rm -rf "$INSTALL_ROOT"' EXIT

SMOKE_BIN="$INSTALL_ROOT/lumen-mcp-smoke"
go build -o "$SMOKE_BIN" "$REPO_ROOT/scripts/testdata/mcp_smoke_client"

export OLLAMA_HOST="${OLLAMA_HOST:-http://localhost:11434}"
export LUMEN_BACKEND=ollama
export LUMEN_EMBED_MODEL="${LUMEN_EMBED_MODEL:-all-minilm}"
export LUMEN_EMBED_DIMS="${LUMEN_EMBED_DIMS:-384}"
export LUMEN_MAX_CHUNK_TOKENS=100
FIXTURE="$REPO_ROOT/testdata/sample-project"

echo "[install-smoke] Codex: fresh native plugin install"
CODEX_HOME="$INSTALL_ROOT/codex-home"
mkdir -p "$CODEX_HOME"
export CODEX_HOME
codex plugin marketplace add "$REPO_ROOT"
codex plugin add lumen@lumen-local
codex plugin list | tee "$INSTALL_ROOT/codex-plugin-list.txt"
grep -q "lumen" "$INSTALL_ROOT/codex-plugin-list.txt"
codex mcp list | tee "$INSTALL_ROOT/codex-mcp-list.txt"
grep -q "lumen" "$INSTALL_ROOT/codex-mcp-list.txt"
codex mcp get lumen --json > "$INSTALL_ROOT/codex-mcp.json"
XDG_DATA_HOME="$INSTALL_ROOT/codex-data" XDG_CONFIG_HOME="$INSTALL_ROOT/codex-config" \
  "$SMOKE_BIN" --codex-config "$INSTALL_ROOT/codex-mcp.json" --project "$FIXTURE"

echo "[install-smoke] Claude Code: fresh local plugin discovery"
CLAUDE_CONFIG_DIR="$INSTALL_ROOT/claude-config" claude --plugin-dir "$REPO_ROOT" mcp list \
  | tee "$INSTALL_ROOT/claude-mcp-list.txt"
grep -q "lumen" "$INSTALL_ROOT/claude-mcp-list.txt"
grep -qi "connected" "$INSTALL_ROOT/claude-mcp-list.txt"
XDG_DATA_HOME="$INSTALL_ROOT/claude-data" XDG_CONFIG_HOME="$INSTALL_ROOT/claude-xdg-config" \
  "$SMOKE_BIN" --command "$REPO_ROOT/scripts/run" --arg stdio --project "$FIXTURE" \
  --env "CLAUDE_PLUGIN_ROOT=$REPO_ROOT" --env "PLUGIN_DATA=$INSTALL_ROOT/claude-plugin-data"

echo "[install-smoke] OpenCode V2: installed package and MCP host canary"
PACK_DIR="$INSTALL_ROOT/pack"
NPM_PREFIX="$INSTALL_ROOT/npm-install"
mkdir -p "$PACK_DIR" "$NPM_PREFIX"
PACKAGE_NAME=$(cd "$REPO_ROOT" && npm pack --pack-destination "$PACK_DIR")
npm install --no-audit --no-fund --prefix "$NPM_PREFIX" "$PACK_DIR/$PACKAGE_NAME"
PACKAGE_ROOT="$NPM_PREFIX/node_modules/@ory/lumen-opencode"
node "$REPO_ROOT/scripts/verify_opencode_package.mjs" "$PACKAGE_ROOT"
node "$REPO_ROOT/scripts/test_opencode_v2_host.mjs" \
  "$PACKAGE_ROOT" "$PACK_DIR/$PACKAGE_NAME"
XDG_DATA_HOME="$INSTALL_ROOT/opencode-smoke-data" XDG_CONFIG_HOME="$INSTALL_ROOT/opencode-smoke-config" \
  "$SMOKE_BIN" --command "$PACKAGE_ROOT/scripts/run" --arg stdio --project "$FIXTURE"

echo "[install-smoke] all fresh-install canaries passed"
