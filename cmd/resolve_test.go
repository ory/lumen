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

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ory/lumen/internal/config"
)

// resolvedTempDir returns a t.TempDir() with symlinks resolved (macOS /var → /private/var).
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	resolved, err := filepath.EvalSymlinks(d)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestResolveIndexRoot(t *testing.T) {
	const model = "test-model"

	t.Run("path only with git repo uses git root", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		t.Setenv("XDG_DATA_HOME", tmp)

		// Create a git repo at tmp/repo.
		repoDir := filepath.Join(tmp, "repo")
		if err := os.MkdirAll(filepath.Join(repoDir, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		runGit(t, repoDir, "init")

		indexRoot, searchPath, err := resolveIndexRoot(filepath.Join(repoDir, "sub"), "", model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if indexRoot != repoDir {
			t.Fatalf("expected indexRoot=%q (git root), got %q", repoDir, indexRoot)
		}
		if searchPath != filepath.Join(repoDir, "sub") {
			t.Fatalf("expected searchPath=%q, got %q", filepath.Join(repoDir, "sub"), searchPath)
		}
	})

	t.Run("path only without git repo and no ancestor index uses path itself", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		t.Setenv("XDG_DATA_HOME", tmp)

		dir := filepath.Join(tmp, "nongit", "sub")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		indexRoot, searchPath, err := resolveIndexRoot(dir, "", model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if indexRoot != dir {
			t.Fatalf("expected indexRoot=%q, got %q", dir, indexRoot)
		}
		if searchPath != dir {
			t.Fatalf("expected searchPath=%q, got %q", dir, searchPath)
		}
	})

	t.Run("cwd with path where subproject is git repo uses git root of subproject", func(t *testing.T) {
		// This is the exact scenario from issue #97:
		// lumen search "..." --cwd /project -p /project/sub-project-a
		// where sub-project-a is a git repo.
		tmp := resolvedTempDir(t)
		t.Setenv("XDG_DATA_HOME", tmp)

		parentDir := filepath.Join(tmp, "project")
		subDir := filepath.Join(parentDir, "sub-project-a")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatal(err)
		}
		runGit(t, subDir, "init")

		indexRoot, searchPath, err := resolveIndexRoot(subDir, parentDir, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should resolve to the git root of sub-project-a, NOT parentDir.
		if indexRoot != subDir {
			t.Fatalf("expected indexRoot=%q (git root of subproject), got %q", subDir, indexRoot)
		}
		if searchPath != subDir {
			t.Fatalf("expected searchPath=%q, got %q", subDir, searchPath)
		}
	})

	t.Run("cwd with existing index at cwd is adopted when path has no git root", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		t.Setenv("XDG_DATA_HOME", tmp)

		parentDir := filepath.Join(tmp, "project")
		subDir := filepath.Join(parentDir, "sub")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a fake index at parentDir.
		dbPath := config.DBPathForProject(parentDir, model)
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		indexRoot, searchPath, err := resolveIndexRoot(subDir, parentDir, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// parentDir has an existing index and path has no git root, so cwd is adopted.
		if indexRoot != parentDir {
			t.Fatalf("expected indexRoot=%q (cwd with existing index), got %q", parentDir, indexRoot)
		}
		if searchPath != subDir {
			t.Fatalf("expected searchPath=%q, got %q", subDir, searchPath)
		}
	})

	t.Run("cwd without existing index is not adopted", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		t.Setenv("XDG_DATA_HOME", tmp)

		parentDir := filepath.Join(tmp, "project")
		subDir := filepath.Join(parentDir, "sub")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatal(err)
		}
		// No index at parentDir.

		indexRoot, searchPath, err := resolveIndexRoot(subDir, parentDir, model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// No git root, no ancestor index, no index at cwd → falls back to searchPath.
		if indexRoot != subDir {
			t.Fatalf("expected indexRoot=%q, got %q", subDir, indexRoot)
		}
		if searchPath != subDir {
			t.Fatalf("expected searchPath=%q, got %q", subDir, searchPath)
		}
	})

	t.Run("ancestor index found via walk", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		t.Setenv("XDG_DATA_HOME", tmp)

		grandparentDir := filepath.Join(tmp, "workspace")
		subDir := filepath.Join(grandparentDir, "a", "b")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a fake index at grandparentDir.
		dbPath := config.DBPathForProject(grandparentDir, model)
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		indexRoot, searchPath, err := resolveIndexRoot(subDir, "", model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if indexRoot != grandparentDir {
			t.Fatalf("expected indexRoot=%q (ancestor index), got %q", grandparentDir, indexRoot)
		}
		if searchPath != subDir {
			t.Fatalf("expected searchPath=%q, got %q", subDir, searchPath)
		}
	})

	t.Run("lumenignore boundary does not adopt ancestor index", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		t.Setenv("XDG_DATA_HOME", tmp)

		parentDir := filepath.Join(tmp, "workspace")
		subDir := filepath.Join(parentDir, "dashboard")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(subDir, ".lumenignore"), []byte("vendor/\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		dbPath := config.DBPathForProject(parentDir, model)
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		indexRoot, searchPath, err := resolveIndexRoot(subDir, "", model)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if indexRoot != subDir {
			t.Fatalf("expected indexRoot=%q (.lumenignore boundary), got %q", subDir, indexRoot)
		}
		if searchPath != subDir {
			t.Fatalf("expected searchPath=%q, got %q", subDir, searchPath)
		}
	})
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}
