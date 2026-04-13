#!/usr/bin/env bash
# Tests for run.sh URL construction and OS/arch detection logic.
# Runs entirely offline — no real HTTP calls are made.
set -euo pipefail

PASS=0
FAIL=0

ok() {
  echo "  PASS: $1"
  PASS=$((PASS + 1))
}

fail() {
  echo "  FAIL: $1"
  echo "        expected: $2"
  echo "        got:      $3"
  FAIL=$((FAIL + 1))
}

assert_eq() {
  local desc="$1" expected="$2" got="$3"
  if [ "$expected" = "$got" ]; then
    ok "$desc"
  else
    fail "$desc" "$expected" "$got"
  fi
}

# ---------------------------------------------------------------------------
# asset_name <version_tag> <os> <arch>
# Mirrors the logic in run.sh: strip leading 'v', build asset filename.
# ---------------------------------------------------------------------------
asset_name() {
  local version="$1" os="$2" arch="$3"
  local ver_no_v="${version#v}"
  case "$os" in
    windows) echo "lumen-${ver_no_v}-${os}-${arch}.exe" ;;
    *)       echo "lumen-${ver_no_v}-${os}-${arch}" ;;
  esac
}

# download_url <repo> <version_tag> <os> <arch>
download_url() {
  local repo="$1" version="$2" os="$3" arch="$4"
  local asset
  asset="$(asset_name "$version" "$os" "$arch")"
  echo "https://github.com/${repo}/releases/download/${version}/${asset}"
}

# ---------------------------------------------------------------------------
# arch normalisation (mirrors run.sh case statement)
# ---------------------------------------------------------------------------
normalise_arch() {
  case "$1" in
    x86_64)  echo "amd64" ;;
    aarch64) echo "arm64" ;;
    *)       echo "$1" ;;
  esac
}

echo "=== asset name tests ==="
assert_eq "macOS arm64 asset" \
  "lumen-0.0.1-alpha.4-darwin-arm64" \
  "$(asset_name "v0.0.1-alpha.4" "darwin" "arm64")"

assert_eq "macOS amd64 asset" \
  "lumen-0.0.1-alpha.4-darwin-amd64" \
  "$(asset_name "v0.0.1-alpha.4" "darwin" "amd64")"

assert_eq "Linux amd64 asset" \
  "lumen-0.0.1-alpha.4-linux-amd64" \
  "$(asset_name "v0.0.1-alpha.4" "linux" "amd64")"

assert_eq "Linux arm64 asset" \
  "lumen-0.0.1-alpha.4-linux-arm64" \
  "$(asset_name "v0.0.1-alpha.4" "linux" "arm64")"

assert_eq "Windows amd64 asset (.exe)" \
  "lumen-0.0.1-alpha.4-windows-amd64.exe" \
  "$(asset_name "v0.0.1-alpha.4" "windows" "amd64")"

echo ""
echo "=== download URL tests ==="
REPO="ory/lumen"
VERSION="v0.0.1-alpha.4"

assert_eq "macOS arm64 URL" \
  "https://github.com/ory/lumen/releases/download/v0.0.1-alpha.4/lumen-0.0.1-alpha.4-darwin-arm64" \
  "$(download_url "$REPO" "$VERSION" "darwin" "arm64")"

assert_eq "Linux amd64 URL" \
  "https://github.com/ory/lumen/releases/download/v0.0.1-alpha.4/lumen-0.0.1-alpha.4-linux-amd64" \
  "$(download_url "$REPO" "$VERSION" "linux" "amd64")"

assert_eq "Windows amd64 URL" \
  "https://github.com/ory/lumen/releases/download/v0.0.1-alpha.4/lumen-0.0.1-alpha.4-windows-amd64.exe" \
  "$(download_url "$REPO" "$VERSION" "windows" "amd64")"

echo ""
echo "=== arch normalisation tests ==="
assert_eq "x86_64 → amd64"  "amd64" "$(normalise_arch "x86_64")"
assert_eq "aarch64 → arm64" "arm64" "$(normalise_arch "aarch64")"
assert_eq "arm64 passthrough" "arm64" "$(normalise_arch "arm64")"
assert_eq "amd64 passthrough" "amd64" "$(normalise_arch "amd64")"

echo ""
echo "=== binary candidate priority tests ==="
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

BIN_DIR="${TMP_DIR}/bin"
mkdir -p "$BIN_DIR"

# Simulate: only downloaded binary present → should pick lumen-os-arch
touch "${BIN_DIR}/lumen-linux-amd64"
chmod +x "${BIN_DIR}/lumen-linux-amd64"

FOUND=""
for candidate in "${BIN_DIR}/lumen" "${BIN_DIR}/lumen-linux-amd64"; do
  if [ -x "$candidate" ]; then FOUND="$candidate"; break; fi
done
assert_eq "picks lumen-linux-amd64 when lumen absent" \
  "${BIN_DIR}/lumen-linux-amd64" "$FOUND"

# Simulate: both present → local dev build wins
touch "${BIN_DIR}/lumen"
chmod +x "${BIN_DIR}/lumen"

FOUND=""
for candidate in "${BIN_DIR}/lumen" "${BIN_DIR}/lumen-linux-amd64"; do
  if [ -x "$candidate" ]; then FOUND="$candidate"; break; fi
done
assert_eq "prefers bin/lumen (dev build) over downloaded binary" \
  "${BIN_DIR}/lumen" "$FOUND"

echo ""
echo "=== version resolution tests ==="
TMP_MANIFEST_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_MANIFEST_DIR" "$TMP_DIR"' EXIT

MANIFEST="${TMP_MANIFEST_DIR}/.release-please-manifest.json"
printf '{\n  ".": "1.2.3"\n}\n' > "$MANIFEST"

resolved_version_from_manifest() {
  local manifest="$1"
  local ver="v$(grep '"[.]"' "$manifest" | sed 's/.*"[^"]*"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
  echo "$ver"
}

assert_eq "manifest version resolution" \
  "v1.2.3" \
  "$(resolved_version_from_manifest "$MANIFEST")"

assert_eq "pre-release version preserved" \
  "v0.0.1-alpha.4" \
  "$(printf '{\n  ".": "0.0.1-alpha.4"\n}\n' | grep '"[.]"' | sed 's/.*"[^"]*"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/v\1/')"

echo ""
echo "=== stdio download-on-first-run integration test ==="

# Verifies that run.sh stdio downloads the binary and execs it when no binary
# is present (first install). curl is stubbed in PATH — no real network calls.
(
  _SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
  _TMPROOT="$(mktemp -d)"
  _FAKE_CURL_DIR="$(mktemp -d)"
  trap 'rm -rf "$_TMPROOT" "$_FAKE_CURL_DIR"' EXIT

  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH_RAW="$(uname -m)"
  case "$ARCH_RAW" in x86_64) ARCH="amd64" ;; aarch64) ARCH="arm64" ;; *) ARCH="$ARCH_RAW" ;; esac
  _EXPECTED_BINARY="${_TMPROOT}/bin/lumen-${OS}-${ARCH}"

  printf '{\n  ".": "0.0.1"\n}\n' > "${_TMPROOT}/.release-please-manifest.json"
  mkdir -p "${_TMPROOT}/bin"

  # Stub curl: write a minimal executable to the -o destination and succeed
  cat > "${_FAKE_CURL_DIR}/curl" <<'FAKECURL'
#!/usr/bin/env bash
while [ $# -gt 0 ]; do
  case "$1" in
    -o) mkdir -p "$(dirname "$2")"; printf '#!/usr/bin/env bash\nexit 0\n' > "$2"; chmod +x "$2"; shift 2 ;;
    *)  shift ;;
  esac
done
exit 0
FAKECURL
  chmod +x "${_FAKE_CURL_DIR}/curl"

  EXIT_CODE=0
  CLAUDE_PLUGIN_ROOT="${_TMPROOT}" PATH="${_FAKE_CURL_DIR}:${PATH}" \
    bash "${_SCRIPT_DIR}/run.sh" stdio >/dev/null 2>&1 || EXIT_CODE=$?

  if [ "$EXIT_CODE" -ne 0 ]; then
    echo "  FAIL: run.sh stdio with missing binary should download and exec (exit $EXIT_CODE)"
    exit 1
  fi
  if [ ! -x "$_EXPECTED_BINARY" ]; then
    echo "  FAIL: run.sh stdio should have downloaded binary to ${_EXPECTED_BINARY}"
    exit 1
  fi
  echo "  PASS: run.sh stdio downloads binary and execs it on first install"
) && PASS=$((PASS + 1)) || FAIL=$((FAIL + 1))

echo ""
echo "=== GitHub API tag parsing tests ==="

# Simulates the sed command used in run.sh to extract tag_name from JSON
parse_tag_from_json() {
  echo "$1" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
}

assert_eq "parse tag from typical API response" \
  "v1.2.3" \
  "$(parse_tag_from_json '  "tag_name": "v1.2.3",')"

assert_eq "parse tag with pre-release suffix" \
  "v0.0.1-alpha.4" \
  "$(parse_tag_from_json '  "tag_name": "v0.0.1-alpha.4",')"

assert_eq "parse tag with surrounding fields" \
  "v2.0.0" \
  "$(parse_tag_from_json '{"url":"https://...","tag_name": "v2.0.0","name":"v2.0.0"}')"

assert_eq "empty on missing tag_name" \
  "" \
  "$(parse_tag_from_json '{"error": "Not Found"}')"

assert_eq "empty on HTML error page" \
  "" \
  "$(parse_tag_from_json '<html><body>404</body></html>')"

echo ""
echo "=== fallback URL construction tests ==="

# Given a resolved latest tag, verify the fallback URL is correct
fallback_url() {
  local repo="$1" tag="$2" os="$3" arch="$4"
  local ver_no_v="${tag#v}"
  local asset
  case "$os" in
    windows) asset="lumen-${ver_no_v}-${os}-${arch}.exe" ;;
    *)       asset="lumen-${ver_no_v}-${os}-${arch}" ;;
  esac
  echo "https://github.com/${repo}/releases/download/${tag}/${asset}"
}

assert_eq "fallback URL for darwin arm64" \
  "https://github.com/ory/lumen/releases/download/v0.0.31/lumen-0.0.31-darwin-arm64" \
  "$(fallback_url "ory/lumen" "v0.0.31" "darwin" "arm64")"

assert_eq "fallback URL for linux amd64" \
  "https://github.com/ory/lumen/releases/download/v0.0.31/lumen-0.0.31-linux-amd64" \
  "$(fallback_url "ory/lumen" "v0.0.31" "linux" "amd64")"

assert_eq "fallback URL for windows amd64" \
  "https://github.com/ory/lumen/releases/download/v0.0.31/lumen-0.0.31-windows-amd64.exe" \
  "$(fallback_url "ory/lumen" "v0.0.31" "windows" "amd64")"

echo ""
echo "=== tag validation tests ==="

# Mirrors the validation in run.sh: tag must be non-empty and match ^v[0-9]
validate_tag() {
  local tag="$1"
  if [ -z "$tag" ] || ! echo "$tag" | grep -qE '^v[0-9]'; then
    echo "invalid"
  else
    echo "valid"
  fi
}

assert_eq "valid semver tag"    "valid"   "$(validate_tag "v1.2.3")"
assert_eq "valid pre-release"   "valid"   "$(validate_tag "v0.0.1-alpha.4")"
assert_eq "empty tag"           "invalid" "$(validate_tag "")"
assert_eq "garbage tag"         "invalid" "$(validate_tag "not-a-version")"
assert_eq "html fragment"       "invalid" "$(validate_tag "<html>")"

echo ""
echo "=== stdio first-install MCP handshake test ==="

# End-to-end guard against the bug in #125: when the binary is missing and
# run.sh is invoked as `run.sh stdio` (how Claude Code starts the MCP server
# on first install), the launcher must download the binary AND the resulting
# process must speak MCP over stdio. If the stdio path fast-exits before
# reaching the download, or the downloaded artefact is never actually exec'd
# with stdin/stdout intact, the MCP server is dead for the entire session —
# Claude Code does not retry failed MCP servers.
#
# A cross-compiled mock MCP server stands in for the real binary. A stubbed
# `curl` (shadowing real curl via PATH) copies the mock into place. A real
# JSON-RPC initialize request is sent on stdin; the stdout response must be
# a well-formed MCP initialize result from the mock. Passing transitively
# proves the launcher did not fast-exit, reached the download code path,
# wrote the artefact where it would exec it, made it executable, and
# exec'd it with stdin/stdout inherited correctly.
(
  _SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
  _REPO_ROOT="$(cd "${_SCRIPT_DIR}/.." && pwd)"
  _TMPROOT="$(mktemp -d)"
  _FAKE_CURL_DIR="$(mktemp -d)"
  _MOCK_BIN_DIR="$(mktemp -d)"
  trap 'rm -rf "$_TMPROOT" "$_FAKE_CURL_DIR" "$_MOCK_BIN_DIR"' EXIT

  _OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  _ARCH_RAW="$(uname -m)"
  case "$_ARCH_RAW" in
    x86_64)  _ARCH="amd64" ;;
    aarch64) _ARCH="arm64" ;;
    *)       _ARCH="$_ARCH_RAW" ;;
  esac
  _EXPECTED_BINARY="${_TMPROOT}/bin/lumen-${_OS}-${_ARCH}"

  _MOCK_BIN="${_MOCK_BIN_DIR}/mock_lumen"
  if ! (cd "${_REPO_ROOT}" && CGO_ENABLED=0 go build -o "${_MOCK_BIN}" ./scripts/testdata/mock_mcp_server) >"${_TMPROOT}/mock_build.log" 2>&1; then
    echo "  FAIL: could not build mock MCP server"
    sed 's/^/          /' "${_TMPROOT}/mock_build.log"
    exit 1
  fi

  printf '{\n  ".": "0.0.1"\n}\n' > "${_TMPROOT}/.release-please-manifest.json"
  mkdir -p "${_TMPROOT}/bin"

  # Stub curl: parse -o <target> and copy the prebuilt mock into place.
  cat > "${_FAKE_CURL_DIR}/curl" <<'FAKECURL'
#!/usr/bin/env bash
while [ $# -gt 0 ]; do
  case "$1" in
    -o) mkdir -p "$(dirname "$2")"; cp "$LUMEN_MOCK_BINARY" "$2"; chmod +x "$2"; shift 2 ;;
    *)  shift ;;
  esac
done
exit 0
FAKECURL
  chmod +x "${_FAKE_CURL_DIR}/curl"

  _INIT_REQ='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"launcher-e2e","version":"1.0"}}}'

  _STDOUT="${_TMPROOT}/stdout.txt"
  _STDERR="${_TMPROOT}/stderr.txt"
  EXIT_CODE=0
  printf '%s\n' "$_INIT_REQ" | \
    CLAUDE_PLUGIN_ROOT="${_TMPROOT}" \
    PATH="${_FAKE_CURL_DIR}:${PATH}" \
    LUMEN_MOCK_BINARY="${_MOCK_BIN}" \
    bash "${_SCRIPT_DIR}/run.sh" stdio >"${_STDOUT}" 2>"${_STDERR}" \
    || EXIT_CODE=$?

  if [ "$EXIT_CODE" -ne 0 ]; then
    echo "  FAIL: run.sh stdio exited $EXIT_CODE — MCP server would be dead for the session"
    echo "        stderr:"
    sed 's/^/          /' "${_STDERR}"
    exit 1
  fi
  if [ ! -x "$_EXPECTED_BINARY" ]; then
    echo "  FAIL: run.sh stdio did not place artefact at ${_EXPECTED_BINARY}"
    exit 1
  fi
  if ! grep -q '"jsonrpc":"2.0"' "${_STDOUT}"; then
    echo "  FAIL: MCP initialize produced no JSON-RPC 2.0 response on stdout"
    echo "        stdout:"
    sed 's/^/          /' "${_STDOUT}"
    exit 1
  fi
  if ! grep -q '"name":"mock-lumen"' "${_STDOUT}"; then
    echo "  FAIL: MCP response did not come from the exec'd mock — launcher may be swallowing stdout"
    echo "        stdout:"
    sed 's/^/          /' "${_STDOUT}"
    exit 1
  fi
  echo "  PASS: run.sh stdio downloads, execs, and brokers MCP initialize on first install"
) && PASS=$((PASS + 1)) || FAIL=$((FAIL + 1))

echo ""
echo "=== summary ==="
echo "  passed: $PASS"
echo "  failed: $FAIL"
[ "$FAIL" -eq 0 ] || exit 1
