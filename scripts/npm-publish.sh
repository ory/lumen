#!/usr/bin/env bash
# Publishes @ory/lumen and platform-specific npm packages.
# Downloads binaries from the GitHub release, verifies checksums, and wraps them in npm packages.
#
# Usage: npm-publish.sh <tag>  (e.g. v0.0.12)
# Requires: NODE_AUTH_TOKEN env var set to an npm automation token with publish access.
set -euo pipefail

TAG="${1:?Usage: npm-publish.sh <tag> [--dry-run]}"
DRY_RUN=""
[ "${2:-}" = "--dry-run" ] && DRY_RUN="--dry-run"
VERSION="${TAG#v}"
REPO="ory/lumen"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

# goreleaser_os goreleaser_arch npm_package npm_os npm_cpu binary_ext
PLATFORMS=(
  "darwin  amd64 darwin-x64   darwin x64  "
  "darwin  arm64 darwin-arm64 darwin arm64 "
  "linux   amd64 linux-x64    linux  x64  "
  "linux   arm64 linux-arm64  linux  arm64 "
  "windows amd64 win32-x64    win32  x64  .exe"
)

# NODE_AUTH_TOKEN is set by actions/setup-node and consumed by npm automatically.
# For local use outside CI, set it explicitly: NODE_AUTH_TOKEN=<token> ./scripts/npm-publish.sh <tag>
[ -z "${DRY_RUN}" ] && : "${NODE_AUTH_TOKEN:?NODE_AUTH_TOKEN must be set}"

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

# Download checksums file for verification
CHECKSUMS_FILE="${TMP_DIR}/checksums.txt"
echo "Downloading checksums..."
curl -sfL "${BASE_URL}/checksums.txt" -o "${CHECKSUMS_FILE}"

# Publish platform packages first
for row in "${PLATFORMS[@]}"; do
  read -r GO_OS GO_ARCH NPM_PKG NPM_OS NPM_CPU EXT <<< "$row"

  ASSET="lumen-${VERSION}-${GO_OS}-${GO_ARCH}${EXT}"
  PKG_DIR="${TMP_DIR}/@ory/lumen-${NPM_PKG}"
  mkdir -p "${PKG_DIR}/bin"

  echo "Downloading ${ASSET}..."
  curl -sfL "${BASE_URL}/${ASSET}" -o "${PKG_DIR}/bin/lumen${EXT}"

  # Verify checksum
  EXPECTED="$(grep " ${ASSET}$" "${CHECKSUMS_FILE}" | awk '{print $1}')"
  ACTUAL="$(sha256sum "${PKG_DIR}/bin/lumen${EXT}" | awk '{print $1}')"
  if [ "${EXPECTED}" != "${ACTUAL}" ]; then
    echo "Checksum mismatch for ${ASSET}: expected ${EXPECTED}, got ${ACTUAL}" >&2
    exit 1
  fi

  chmod +x "${PKG_DIR}/bin/lumen${EXT}"

  cat > "${PKG_DIR}/package.json" << PKGJSON
{
  "name": "@ory/lumen-${NPM_PKG}",
  "version": "${VERSION}",
  "description": "Platform binary for @ory/lumen (${NPM_PKG})",
  "repository": { "type": "git", "url": "git+https://github.com/ory/lumen.git" },
  "license": "Apache-2.0",
  "os": ["${NPM_OS}"],
  "cpu": ["${NPM_CPU}"],
  "files": ["bin/"]
}
PKGJSON

  (cd "${PKG_DIR}" && npm publish --access public ${DRY_RUN})
  echo "Published @ory/lumen-${NPM_PKG}@${VERSION}"
done

# Build and publish wrapper package
WRAPPER_DIR="${TMP_DIR}/@ory/lumen"
mkdir -p "${WRAPPER_DIR}/bin"

cp "${REPO_ROOT}/npm/lumen.js" "${WRAPPER_DIR}/bin/lumen.js"
chmod +x "${WRAPPER_DIR}/bin/lumen.js"
# Copy only the files needed at runtime (not publish tooling)
mkdir -p "${WRAPPER_DIR}/scripts"
cp "${REPO_ROOT}/scripts/run.sh" "${WRAPPER_DIR}/scripts/run.sh"
[ -f "${REPO_ROOT}/scripts/run.bat" ] && cp "${REPO_ROOT}/scripts/run.bat" "${WRAPPER_DIR}/scripts/run.bat"
cp -r "${REPO_ROOT}/.claude-plugin" "${WRAPPER_DIR}/"
cp -r "${REPO_ROOT}/hooks" "${WRAPPER_DIR}/"
cp -r "${REPO_ROOT}/skills" "${WRAPPER_DIR}/"
[ -f "${REPO_ROOT}/LICENSE" ] && cp "${REPO_ROOT}/LICENSE" "${WRAPPER_DIR}/"

cat > "${WRAPPER_DIR}/package.json" << WRAPPERJSON
{
  "name": "@ory/lumen",
  "version": "${VERSION}",
  "description": "Precise local semantic code search via MCP. Indexes your codebase with Go AST parsing, embeds with Ollama or LM Studio, and exposes vector search to Claude — no cloud, no npm.",
  "repository": { "type": "git", "url": "git+https://github.com/ory/lumen.git" },
  "license": "Apache-2.0",
  "keywords": ["semantic-search", "code-index", "mcp", "embeddings", "ollama"],
  "bin": {
    "lumen": "./bin/lumen.js"
  },
  "files": [
    "bin/",
    "scripts/",
    ".claude-plugin/",
    "hooks/",
    "skills/"
  ],
  "optionalDependencies": {
    "@ory/lumen-darwin-arm64": "${VERSION}",
    "@ory/lumen-darwin-x64": "${VERSION}",
    "@ory/lumen-linux-arm64": "${VERSION}",
    "@ory/lumen-linux-x64": "${VERSION}",
    "@ory/lumen-win32-x64": "${VERSION}"
  }
}
WRAPPERJSON

(cd "${WRAPPER_DIR}" && npm publish --access public ${DRY_RUN})
echo "Published @ory/lumen@${VERSION}"
