#!/usr/bin/env bash
set -euo pipefail

# Determine plugin root: prefer an agent-set env var, then fall back to the
# repository layout so the same launcher works for Claude, Codex, Cursor,
# OpenCode, and direct local invocation.
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-${CURSOR_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}}"

# Platform detection
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
esac

# Find binary: check bin/ first, then goreleaser dist/ output, then download
BINARY=""
for candidate in \
  "${PLUGIN_ROOT}/bin/lumen" \
  "${PLUGIN_ROOT}/bin/lumen-${OS}-${ARCH}"; do
  if [ -x "$candidate" ]; then
    BINARY="$candidate"
    break
  fi
done

# Download on first run if no binary found
if [ -z "$BINARY" ]; then
  BINARY="${PLUGIN_ROOT}/bin/lumen-${OS}-${ARCH}"

  REPO="ory/lumen"

  # Always use the version pinned in the manifest — keeps plugin and binary in sync
  MANIFEST="${PLUGIN_ROOT}/.release-please-manifest.json"
  if [ ! -f "$MANIFEST" ]; then
    echo "Error: .release-please-manifest.json not found in ${PLUGIN_ROOT}" >&2
    exit 1
  fi
  VERSION="v$(grep '"[.]"' "$MANIFEST" | sed 's/.*"[^"]*"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
  if [ -z "$VERSION" ] || [ "$VERSION" = "v" ]; then
    echo "Error: could not read version from ${MANIFEST}" >&2
    exit 1
  fi

  ASSET="lumen-${VERSION#v}-${OS}-${ARCH}"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

  echo "Downloading lumen ${VERSION} for ${OS}/${ARCH}..." >&2
  mkdir -p "$(dirname "$BINARY")"

  if ! curl -fL --progress-bar --max-time 300 --retry 3 --retry-delay 2 "$URL" -o "$BINARY"; then
    # Fallback: manifest version not released yet — resolve latest from GitHub API
    echo "Version ${VERSION} not found, resolving latest release..." >&2

    AUTH_ARGS=()
    if [ -n "${GITHUB_TOKEN:-}" ]; then
      AUTH_ARGS=(-H "Authorization: token $GITHUB_TOKEN")
    fi

    LATEST_TAG=$(curl -sfL "${AUTH_ARGS[@]}" \
      --max-time 30 --retry 2 --retry-delay 2 \
      "https://api.github.com/repos/${REPO}/releases/latest" \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

    if [ -z "$LATEST_TAG" ] || ! echo "$LATEST_TAG" | grep -qE '^v[0-9]'; then
      echo "Error: could not resolve latest release from GitHub API" >&2
      exit 1
    fi

    echo "Falling back to ${LATEST_TAG}..." >&2
    VERSION="$LATEST_TAG"
    ASSET="lumen-${VERSION#v}-${OS}-${ARCH}"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

    curl -fL --progress-bar --max-time 300 --retry 3 --retry-delay 2 "$URL" -o "$BINARY"
  fi

  chmod +x "$BINARY"
  echo "Installed lumen to ${BINARY}" >&2
fi

# Always ensure generic binary exists for PATH usage (backfills old installs)
GENERIC_BINARY="${PLUGIN_ROOT}/bin/lumen"
PLATFORM_BINARY="${PLUGIN_ROOT}/bin/lumen-${OS}-${ARCH}"
if [ ! -e "$GENERIC_BINARY" ] || [ ! -x "$GENERIC_BINARY" ]; then
  # If we have a platform-specific binary, copy it to the generic name
  if [ -x "$PLATFORM_BINARY" ]; then
    if ! cp "$PLATFORM_BINARY" "$GENERIC_BINARY"; then
      echo "Error: failed to create generic binary at ${GENERIC_BINARY}" >&2
      exit 1
    fi
    if ! chmod +x "$GENERIC_BINARY"; then
      echo "Error: failed to make ${GENERIC_BINARY} executable" >&2
      exit 1
    fi
    echo "Created ${GENERIC_BINARY} for direct CLI usage" >&2
  fi
fi

# Auto-install symlink to $HOME/.local/bin for CLI usage (no sudo required).
# Only attempt this if we have a working generic binary.
if [ -x "$GENERIC_BINARY" ]; then
  LOCAL_BIN="${HOME}/.local/bin"
  SYMLINK_TARGET="${LOCAL_BIN}/lumen"

  # Create ~/.local/bin if it doesn't exist
  if [ ! -d "$LOCAL_BIN" ]; then
    if mkdir -p "$LOCAL_BIN" 2>/dev/null; then
      echo "Created ${LOCAL_BIN}" >&2
    fi
  fi

  # Create symlink if directory exists and symlink doesn't point to our binary
  if [ -d "$LOCAL_BIN" ] && [ ! -L "$SYMLINK_TARGET" ] || [ "$(readlink "$SYMLINK_TARGET" 2>/dev/null)" != "$GENERIC_BINARY" ]; then
    # Remove old symlink/file if it exists
    rm -f "$SYMLINK_TARGET" 2>/dev/null || true

    if ln -sf "$GENERIC_BINARY" "$SYMLINK_TARGET" 2>/dev/null; then
      echo "Installed lumen CLI to ${SYMLINK_TARGET}" >&2

      # Check if $LOCAL_BIN is in PATH
      if ! echo "$PATH" | tr ':' '\n' | grep -qx "$LOCAL_BIN"; then
        echo "" >&2
        echo "✨ To use 'lumen' from anywhere, add to your PATH:" >&2
        echo "" >&2
        echo "  # Add to ~/.bashrc or ~/.zshrc:" >&2
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\"" >&2
        echo "" >&2
      fi
    fi
  fi
fi

exec "$BINARY" "$@"
