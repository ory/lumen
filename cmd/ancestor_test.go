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
	"path/filepath"
	"testing"

	"github.com/ory/lumen/internal/config"
)

func TestFindAncestorIndex(t *testing.T) {
	const model = "test-model"

	t.Run("returns empty when no ancestor has an index", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmpDir)

		got := findAncestorIndex(filepath.Join(t.TempDir(), "some", "deep", "path"), model)
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("finds parent with existing DB on disk", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmpDir)

		project := filepath.Join(t.TempDir(), "project")
		child := filepath.Join(project, "scripts", "util")
		if err := os.MkdirAll(child, 0o755); err != nil {
			t.Fatal(err)
		}

		parentDBPath := config.DBPathForProject(project, model)
		if err := os.MkdirAll(filepath.Dir(parentDBPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(parentDBPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		got := findAncestorIndex(child, model)
		if got != project {
			t.Fatalf("expected %q, got %q", project, got)
		}
	})

	t.Run("skips ancestor when path crosses a SkipDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmpDir)

		project := filepath.Join(t.TempDir(), "project")
		child := filepath.Join(project, "testdata", "fixtures", "go")
		if err := os.MkdirAll(child, 0o755); err != nil {
			t.Fatal(err)
		}

		parentDBPath := config.DBPathForProject(project, model)
		if err := os.MkdirAll(filepath.Dir(parentDBPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(parentDBPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		// "testdata" is in merkle.SkipDirs; the parent index would never contain
		// these files, so findAncestorIndex must return "".
		got := findAncestorIndex(child, model)
		if got != "" {
			t.Fatalf("expected empty string (skip dir in route), got %q", got)
		}
	})

	t.Run("stops at filesystem root without panicking", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmpDir)

		// No DBs exist anywhere; should return "" without panic.
		got := findAncestorIndex(filepath.Join(t.TempDir(), "a", "b", "c", "d", "e"), model)
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("skips ancestor whose .lumenignore declares it un-indexable", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmpDir)

		// Create a real on-disk ancestor with an existing DB AND a catch-all
		// .lumenignore. This mirrors the user's $HOME case: walking up from
		// a non-git subdirectory must NOT resolve to a path that the user has
		// declared un-indexable.
		ancestor := filepath.Join(tmpDir, "home")
		if err := os.MkdirAll(ancestor, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ancestor, ".lumenignore"), []byte("**\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		dbPath := config.DBPathForProject(ancestor, model)
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		got := findAncestorIndex(filepath.Join(ancestor, "sub", "deep"), model)
		if got != "" {
			t.Fatalf("expected empty (ancestor declared un-indexable), got %q", got)
		}
	})

	t.Run("returns nearest ancestor when multiple exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmpDir)

		project := filepath.Join(t.TempDir(), "project")
		src := filepath.Join(project, "src")
		pkg := filepath.Join(src, "pkg")
		if err := os.MkdirAll(pkg, 0o755); err != nil {
			t.Fatal(err)
		}

		for _, dir := range []string{project, src} {
			dbPath := config.DBPathForProject(dir, model)
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
				t.Fatal(err)
			}
		}

		got := findAncestorIndex(pkg, model)
		if got != src {
			t.Fatalf("expected nearest ancestor %q, got %q", src, got)
		}
	})
}
