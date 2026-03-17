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

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsWorktree_GitDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if IsWorktree(dir) {
		t.Fatal("expected false for .git directory")
	}
}

func TestIsWorktree_GitFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /some/path\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsWorktree(dir) {
		t.Fatal("expected true for .git file")
	}
}

func TestIsWorktree_NoGit(t *testing.T) {
	dir := t.TempDir()
	if IsWorktree(dir) {
		t.Fatal("expected false when .git does not exist")
	}
}

func TestListWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// Create a real git repo.
	main := t.TempDir()
	run(t, main, "git", "init")
	run(t, main, "git", "commit", "--allow-empty", "-m", "init")

	// Add a worktree.
	wt := filepath.Join(t.TempDir(), "wt")
	run(t, main, "git", "worktree", "add", wt)

	// Resolve symlinks for comparison (macOS /var → /private/var).
	wtResolved, err := filepath.EvalSymlinks(wt)
	if err != nil {
		t.Fatal(err)
	}

	paths, err := ListWorktrees(main)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 2 {
		t.Fatalf("expected ≥2 worktrees, got %d: %v", len(paths), paths)
	}

	found := false
	for _, p := range paths {
		if p == wtResolved {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected worktree %q in list %v", wtResolved, paths)
	}
}

func TestCommonDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	main := t.TempDir()
	run(t, main, "git", "init")
	run(t, main, "git", "commit", "--allow-empty", "-m", "init")

	wt := filepath.Join(t.TempDir(), "wt")
	run(t, main, "git", "worktree", "add", wt)

	commonFromMain, err := CommonDir(main)
	if err != nil {
		t.Fatal(err)
	}
	commonFromWT, err := CommonDir(wt)
	if err != nil {
		t.Fatal(err)
	}

	if commonFromMain != commonFromWT {
		t.Fatalf("expected same common dir, got %q and %q", commonFromMain, commonFromWT)
	}
}

func TestListWorktrees_NotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	dir := t.TempDir()
	paths, err := ListWorktrees(dir)
	if err == nil {
		t.Fatalf("expected error, got paths: %v", paths)
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}
