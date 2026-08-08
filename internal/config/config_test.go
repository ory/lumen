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

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDBPathForProject(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project", "model-a")
		p2 := DBPathForProject("/home/user/project", "model-a")
		if p1 != p2 {
			t.Fatalf("expected same path, got %q and %q", p1, p2)
		}
	})

	t.Run("different project paths produce different hashes", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project-a", "model-a")
		p2 := DBPathForProject("/home/user/project-b", "model-a")
		if p1 == p2 {
			t.Fatalf("expected different paths, got same: %q", p1)
		}
	})

	t.Run("different models produce different hashes", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project", "model-a")
		p2 := DBPathForProject("/home/user/project", "model-b")
		if p1 == p2 {
			t.Fatalf("expected different paths, got same: %q", p1)
		}
	})

	t.Run("uses IndexVersion not runtime state", func(t *testing.T) {
		// The path must be stable regardless of build-time variables.
		// We verify this by computing the path twice and confirming stability,
		// and by checking that IndexVersion is a non-empty hardcoded constant.
		if IndexVersion == "" {
			t.Fatal("IndexVersion must not be empty")
		}
		p1 := DBPathForProject("/some/path", "some-model")
		p2 := DBPathForProject("/some/path", "some-model")
		if p1 != p2 {
			t.Fatalf("path not stable: %q vs %q", p1, p2)
		}
	})

	t.Run("ends with index.db", func(t *testing.T) {
		p := DBPathForProject("/some/path", "model")
		if !strings.HasSuffix(p, "index.db") {
			t.Fatalf("expected path to end with index.db, got %q", p)
		}
	})
}

func TestDBPathForProjectProfileSharesGitWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	root := filepath.Join(t.TempDir(), "repo")
	worktree := filepath.Join(t.TempDir(), "worktree")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Lumen", "GIT_AUTHOR_EMAIL=lumen@example.test", "GIT_COMMITTER_NAME=Lumen", "GIT_COMMITTER_EMAIL=lumen@example.test")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run(root, "init")
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(root, "add", "README")
	run(root, "commit", "-m", "initial")
	run(root, "worktree", "add", worktree, "-b", "worktree-test")

	dataDir := t.TempDir()
	mainPath := DBPathForProjectProfileBase(dataDir, root, "model", 768, "int8", 512)
	worktreePath := DBPathForProjectProfileBase(dataDir, worktree, "model", 768, "int8", 512)
	if mainPath != worktreePath {
		t.Fatalf("worktrees should share a collection: %q != %q", mainPath, worktreePath)
	}
	floatPath := DBPathForProjectProfileBase(dataDir, root, "model", 768, "float32", 512)
	if floatPath == mainPath {
		t.Fatal("vector storage must be part of the collection profile")
	}
}

func TestDBPathForProjectProfileResolvesNonGitSymlinks(t *testing.T) {
	realProject := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(realProject, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "project-link")
	if err := os.Symlink(realProject, link); err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	realPath := DBPathForProjectProfileBase(dataDir, realProject, "model", 768, "int8", 512)
	linkPath := DBPathForProjectProfileBase(dataDir, link, "model", 768, "int8", 512)
	if realPath != linkPath {
		t.Fatalf("symlinked non-Git project should share identity: %q != %q", realPath, linkPath)
	}
}

func TestXDGConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		if got := XDGConfigDir(); got != "/custom/config" {
			t.Errorf("XDGConfigDir() = %q, want %q", got, "/custom/config")
		}
	})
	t.Run("falls back to ~/.config", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".config")
		if got := XDGConfigDir(); got != want {
			t.Errorf("XDGConfigDir() = %q, want %q", got, want)
		}
	})
}

func TestVectorStorageConfiguration(t *testing.T) {
	t.Run("defaults to int8", func(t *testing.T) {
		t.Setenv("LUMEN_VECTOR_STORAGE", "")
		cfg, err := NewConfigService("")
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.VectorStorage(); got != "int8" {
			t.Fatalf("VectorStorage() = %q, want int8", got)
		}
	})
	t.Run("normalizes float32 override", func(t *testing.T) {
		t.Setenv("LUMEN_VECTOR_STORAGE", "  FLOAT32  ")
		cfg, err := NewConfigService("")
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.VectorStorage(); got != "float32" {
			t.Fatalf("VectorStorage() = %q, want float32", got)
		}
	})
	t.Run("rejects unknown storage", func(t *testing.T) {
		t.Setenv("LUMEN_VECTOR_STORAGE", "float16")
		if _, err := NewConfigService(""); err == nil {
			t.Fatal("expected invalid vector storage to fail validation")
		}
	})
}
