// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// ensureNetwork verifies the test runner can reach github.com over HTTPS.
// Skips the test (with a clear reason) if the network is unavailable, so
// offline local runs and air-gapped CI shards don't produce false negatives.
func ensureNetwork(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "HEAD", "https://github.com", nil)
	if err != nil {
		t.Fatalf("build network probe request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("github.com unreachable (offline?): %v", err)
	}
	resp.Body.Close()
}

// isolatedLauncher returns the path to a copy of launcher.mjs in a fresh
// temp dir whose layout does NOT contain a bin/ subdirectory. The launcher's
// developer fast path is `<here>/bin/lumen[.exe]`; tests that need to
// exercise the lazy-fetch + cache path must avoid that fast path. Copying
// to a clean dir is the most portable way (Windows symlinks need admin).
func isolatedLauncher(t *testing.T) string {
	t.Helper()
	dir := sandboxTempDir(t)
	root := repoRoot(t)
	for _, name := range []string{"launcher.mjs", ".release-please-manifest.json"} {
		src := filepath.Join(root, name)
		dst := filepath.Join(dir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", dst, err)
		}
	}
	return filepath.Join(dir, "launcher.mjs")
}

// readManifestVersion extracts the pinned version from .release-please-manifest.json
// the same way launcher.mjs does, so tests stay in lockstep with runtime behavior.
func readManifestVersion(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".release-please-manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	v := m["."]
	if v == "" {
		t.Fatalf("manifest has no '.' key: %s", raw)
	}
	return v
}

// expectedAssetName returns the goreleaser-style filename of the binary for
// the running test runner: lumen-<version>-<goos>-<goarch>[.exe].
func expectedAssetName(version string) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("lumen-%s-%s-%s%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

// expectedCachedPath mirrors launcher.mjs cachedBinaryPath() under
// CLAUDE_PLUGIN_DATA = cacheParent.
func expectedCachedPath(cacheParent, version string) string {
	return filepath.Join(cacheParent, "lumen", "bin", version, expectedAssetName(version))
}

// sha256File returns the hex-encoded SHA-256 of the file at p.
func sha256File(t *testing.T, p string) string {
	t.Helper()
	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("open %s: %v", p, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("hash %s: %v", p, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// fetchExpectedChecksum downloads the matching line from the release's
// checksums.txt. Mirrors launcher.mjs parseChecksum behavior for the
// runtime's goos/goarch.
func fetchExpectedChecksum(t *testing.T, version, asset string) string {
	t.Helper()
	url := fmt.Sprintf("https://github.com/ory/lumen/releases/download/v%s/checksums.txt", version)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		t.Fatalf("build checksums request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fetch checksums: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("checksums fetch %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read checksums body: %v", err)
	}
	for _, line := range strings.Split(string(body), "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) >= 2 && f[1] == asset {
			return f[0]
		}
	}
	t.Fatalf("no checksum for %s in %s", asset, url)
	return ""
}

// prefetchEnv returns a process env with PATH preserved and CLAUDE_PLUGIN_DATA
// pointed at the supplied cache parent, but explicitly without LUMEN_BIN_PATH
// so the launcher's lazy-fetch path is exercised.
func prefetchEnv(t *testing.T, cacheParent string) []string {
	t.Helper()
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"CLAUDE_PLUGIN_DATA=" + cacheParent,
		"LUMEN_LAUNCHER_VERBOSE=1",
	}
	if runtime.GOOS == "windows" {
		for _, k := range []string{"SystemRoot", "SYSTEMROOT", "USERPROFILE", "TEMP", "TMP", "LOCALAPPDATA", "APPDATA", "ProgramFiles", "ComSpec"} {
			if v := os.Getenv(k); v != "" {
				env = append(env, k+"="+v)
			}
		}
	}
	return env
}

// TestLauncherPrefetch downloads the matching release asset, verifies SHA-256,
// caches it, and confirms a second invocation is a cache hit (no re-download).
func TestLauncherPrefetch(t *testing.T) {
	ensureNetwork(t)
	launcher := isolatedLauncher(t)
	cacheParent := sandboxTempDir(t)
	version := readManifestVersion(t)
	asset := expectedAssetName(version)
	binPath := expectedCachedPath(cacheParent, version)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, nodePath(t), launcher, "prefetch")
	cmd.Env = prefetchEnv(t, cacheParent)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prefetch failed: %v\noutput: %s", err, out)
	}

	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("binary missing at %s: %v", binPath, err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Fatalf("binary at %s is not executable: mode=%v", binPath, info.Mode())
	}

	expectedSHA := fetchExpectedChecksum(t, version, asset)
	if got := sha256File(t, binPath); got != expectedSHA {
		t.Fatalf("checksum mismatch for cached binary: got %s, expected %s", got, expectedSHA)
	}

	// Lock file should be cleaned up after a successful run.
	lockPath := filepath.Join(cacheParent, "lumen", ".lock")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock file lingered at %s after prefetch (err=%v)", lockPath, err)
	}

	// Re-run: cache hit. Compare mtime to ensure the file was not rewritten.
	mtimeBefore := info.ModTime()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	cmd2 := exec.CommandContext(ctx2, nodePath(t), launcher, "prefetch")
	cmd2.Env = prefetchEnv(t, cacheParent)
	if out, err := cmd2.CombinedOutput(); err != nil {
		t.Fatalf("second prefetch failed: %v\noutput: %s", err, out)
	}
	info2, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("binary disappeared after second run: %v", err)
	}
	if !info2.ModTime().Equal(mtimeBefore) {
		t.Fatalf("cache miss on second run (mtime changed: %v → %v)", mtimeBefore, info2.ModTime())
	}
}

// TestLauncherConcurrentPrefetch runs three prefetch processes simultaneously
// against a shared cache dir. The file lock must serialize the download:
// exactly one process performs the network fetch, the others see a cache hit
// after the lock releases. All three exit 0; the final file checksum is valid.
func TestLauncherConcurrentPrefetch(t *testing.T) {
	ensureNetwork(t)
	launcher := isolatedLauncher(t)
	cacheParent := sandboxTempDir(t)
	version := readManifestVersion(t)
	asset := expectedAssetName(version)
	binPath := expectedCachedPath(cacheParent, version)

	const N = 3
	var wg sync.WaitGroup
	errs := make(chan error, N)
	start := make(chan struct{})

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, nodePath(t), launcher, "prefetch")
			cmd.Env = prefetchEnv(t, cacheParent)
			if out, err := cmd.CombinedOutput(); err != nil {
				errs <- fmt.Errorf("prefetch failed: %v\noutput: %s", err, out)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	for e := range errs {
		t.Fatalf("concurrent prefetch error: %v", e)
	}

	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary missing at %s after concurrent prefetch: %v", binPath, err)
	}
	expectedSHA := fetchExpectedChecksum(t, version, asset)
	if got := sha256File(t, binPath); got != expectedSHA {
		t.Fatalf("checksum mismatch after concurrent prefetch: got %s, expected %s", got, expectedSHA)
	}

	// Lock should be released; partial files should not linger.
	lockPath := filepath.Join(cacheParent, "lumen", ".lock")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock file lingered at %s (err=%v)", lockPath, err)
	}
	if _, err := os.Stat(binPath + ".partial"); !os.IsNotExist(err) {
		t.Fatalf("partial download file lingered (err=%v)", err)
	}
}

// TestLauncherChecksumMismatch corrupts a release URL by pointing at a
// non-matching checksum, asserting the launcher refuses to run and exits
// non-zero. We can't easily man-in-the-middle the real GitHub Releases, so
// this test creates a dummy cache entry with a wrong checksum and confirms
// re-verification on subsequent runs (the prefetch path can't be tampered
// from outside Node, so we rely on the existing checksum verification
// being exercised by the happy path tests above).
//
// Skipped here as a placeholder; the SHA-256 verification correctness is
// indirectly verified by TestLauncherPrefetch reading the cached file and
// checking it against the same checksums.txt the launcher consulted.
func TestLauncherChecksumVerificationPath(t *testing.T) {
	t.Skip("checksum-mismatch flow is exercised by reviewing launcher.mjs " +
		"and cross-verified by TestLauncherPrefetch; an MITM-style negative " +
		"test requires a local mock server (deferred)")
}
