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

// Package integration hosts native-spawn tests that mirror the consumer
// invocation path used by Claude Code, Cursor, OpenCode, and Codex.
// exec.Command on Unix calls posix_spawn(2); on Windows it calls
// CreateProcessW. No shell is in the loop — any byte-0 / shebang / extension
// regression in the launcher chain fails these tests deterministically.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// repoRoot locates the repository root by walking up from this test file
// until it finds go.mod. Tests live at internal/integration/, so the answer
// is two levels up — but we walk dynamically to be robust to relocation.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(here)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test file")
		}
		dir = parent
	}
}

// launcherPath returns the absolute path to launcher.mjs at the repo root.
// Skips the test if launcher.mjs is missing — signals an incomplete checkout
// rather than a real regression.
func launcherPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "launcher.mjs")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("launcher.mjs not found at %s: %v", p, err)
	}
	return p
}

// lookupBuiltBinary returns the path to a pre-built lumen binary for the
// current OS/arch. Resolution order:
//  1. $LUMEN_BIN_PATH (CI integration job sets this to the artifact downloaded
//     from build / build-windows-cross).
//  2. <repoRoot>/bin/lumen[.exe] (matches `make build-local` output).
//
// Skips the test (rather than failing) when no binary is present, so a fresh
// checkout without a build doesn't produce confusing test failures.
func lookupBuiltBinary(t *testing.T) string {
	t.Helper()
	if env := os.Getenv("LUMEN_BIN_PATH"); env != "" {
		if _, err := os.Stat(env); err == nil {
			return env
		}
		t.Fatalf("$LUMEN_BIN_PATH set to %q but file is missing", env)
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	p := filepath.Join(repoRoot(t), "bin", "lumen"+ext)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("no built binary at %s and $LUMEN_BIN_PATH unset; build via `make build-local` or set the env var", p)
	}
	return p
}

// nodePath looks up the Node binary the launcher will be invoked through.
// Skips the test if Node is not in PATH, since the launcher is JS.
func nodePath(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node not in PATH: %v", err)
	}
	return p
}

// sandboxTempDir returns an isolated temp dir whose cleanup tolerates
// Windows file-locking errors. The SessionStart hook spawns a detached
// background indexer (cmd/hook_spawn_windows.go) that opens debug.log and
// index.db; on Windows those files cannot be unlinked while handles are
// open, which makes t.TempDir() cleanup fail and mark the test FAIL even
// though the test logic succeeded. Best-effort RemoveAll + log-on-error
// preserves test correctness; CI runner resets sweep any residue.
func sandboxTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "lumen-integration-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("sandbox cleanup non-fatal error (likely Windows handle held by background indexer): %v", err)
		}
	})
	return dir
}

// sandboxEnv returns an env slice that points lumen at temp directories for
// HOME / XDG_DATA_HOME / CLAUDE_PLUGIN_DATA so the runner's real lumen state
// is untouched. PATH is preserved so node (and its children) can resolve.
// LUMEN_BIN_PATH is forwarded so launcher.mjs uses the pre-built binary.
func sandboxEnv(t *testing.T) []string {
	t.Helper()
	tmp := sandboxTempDir(t)
	env := []string{
		"HOME=" + tmp,
		"XDG_DATA_HOME=" + tmp,
		"XDG_CONFIG_HOME=" + tmp,
		"XDG_CACHE_HOME=" + tmp,
		"CLAUDE_PLUGIN_DATA=" + tmp,
		"PATH=" + os.Getenv("PATH"),
		"LUMEN_BIN_PATH=" + lookupBuiltBinary(t),
	}
	// Windows needs SystemRoot, USERPROFILE, etc. for CreateProcess /
	// runtime initialization. Forward what the parent has — without these,
	// node and the spawned binary may fail to start.
	if runtime.GOOS == "windows" {
		for _, k := range []string{"SystemRoot", "SYSTEMROOT", "USERPROFILE", "TEMP", "TMP", "LOCALAPPDATA", "APPDATA", "PROGRAMFILES", "ProgramFiles", "ComSpec"} {
			if v := os.Getenv(k); v != "" {
				env = append(env, k+"="+v)
			}
		}
	}
	return env
}
