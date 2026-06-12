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

func TestInternalWorktreePaths_InternalWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	main := t.TempDir()
	run(t, main, "git", "init")
	run(t, main, "git", "commit", "--allow-empty", "-m", "init")

	// Create a worktree INSIDE the main repo directory.
	internalWt := filepath.Join(main, ".worktrees", "feature")
	run(t, main, "git", "worktree", "add", internalWt)

	paths := InternalWorktreePaths(main)
	if len(paths) != 1 {
		t.Fatalf("expected 1 internal worktree path, got %d: %v", len(paths), paths)
	}
	want := filepath.Join(".worktrees", "feature")
	if paths[0] != want {
		t.Errorf("expected %q, got %q", want, paths[0])
	}
}

func TestInternalWorktreePaths_ExternalWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	main := t.TempDir()
	run(t, main, "git", "init")
	run(t, main, "git", "commit", "--allow-empty", "-m", "init")

	// Create a worktree OUTSIDE the main repo directory.
	externalWt := filepath.Join(t.TempDir(), "feature")
	run(t, main, "git", "worktree", "add", externalWt)

	paths := InternalWorktreePaths(main)
	if len(paths) != 0 {
		t.Errorf("expected 0 internal worktree paths for external worktree, got %v", paths)
	}
}

func TestInternalWorktreePaths_NotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	paths := InternalWorktreePaths(dir)
	if len(paths) != 0 {
		t.Errorf("expected nil for non-repo, got %v", paths)
	}
}

func TestListWorktrees_NotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	dir := t.TempDir()

	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))

	paths, err := ListWorktrees(dir)
	if err == nil {
		t.Fatalf("expected error, got paths: %v", paths)
	}
}

func TestIsGitRoot_WithGitDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsGitRoot(dir) {
		t.Fatal("expected true for directory with .git dir")
	}
}

func TestIsGitRoot_WithGitFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /some/path\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsGitRoot(dir) {
		t.Fatal("expected true for directory with .git file (worktree)")
	}
}

func TestIsGitRoot_NoGit(t *testing.T) {
	dir := t.TempDir()
	if IsGitRoot(dir) {
		t.Fatal("expected false when no .git exists")
	}
}

func TestDiscoverNestedGitRepos_FindsNestedRepos(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// Create a non-git parent with two git sub-repos.
	parent := t.TempDir()
	repoA := filepath.Join(parent, "sub-a")
	repoB := filepath.Join(parent, "sub-b")
	if err := os.Mkdir(repoA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(repoB, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, repoA, "git", "init")
	run(t, repoB, "git", "init")

	repos := DiscoverNestedGitRepos(parent)
	if len(repos) != 2 {
		t.Fatalf("expected 2 nested repos, got %d: %v", len(repos), repos)
	}

	// Resolve symlinks for comparison.
	resolvedA, _ := filepath.EvalSymlinks(repoA)
	resolvedB, _ := filepath.EvalSymlinks(repoB)
	want := map[string]bool{resolvedA: true, resolvedB: true}
	for _, r := range repos {
		resolved, _ := filepath.EvalSymlinks(r)
		if !want[resolved] {
			t.Errorf("unexpected repo path %q", r)
		}
	}
}

func TestDiscoverNestedGitRepos_SkipsWhenRootIsGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// Parent is itself a git repo — should return nil.
	parent := t.TempDir()
	run(t, parent, "git", "init")

	sub := filepath.Join(parent, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, sub, "git", "init")

	repos := DiscoverNestedGitRepos(parent)
	if len(repos) != 0 {
		t.Fatalf("expected nil when root is a git repo, got %v", repos)
	}
}

func TestDiscoverNestedGitRepos_DeeplyNested(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// Non-git parent, with a git repo nested two levels deep.
	parent := t.TempDir()
	deep := filepath.Join(parent, "a", "b", "repo")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, deep, "git", "init")

	repos := DiscoverNestedGitRepos(parent)
	if len(repos) != 1 {
		t.Fatalf("expected 1 nested repo, got %d: %v", len(repos), repos)
	}

	resolved, _ := filepath.EvalSymlinks(deep)
	got, _ := filepath.EvalSymlinks(repos[0])
	if got != resolved {
		t.Errorf("expected %q, got %q", resolved, got)
	}
}

func TestDiscoverNestedGitRepos_NoNestedRepos(t *testing.T) {
	parent := t.TempDir()
	repos := DiscoverNestedGitRepos(parent)
	if len(repos) != 0 {
		t.Fatalf("expected nil for directory with no nested repos, got %v", repos)
	}
}

func TestDiscoverNestedGitRepos_DoesNotDescendIntoGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// Non-git parent with a git repo that itself contains another git repo.
	// Only the outer one should be discovered.
	parent := t.TempDir()
	outer := filepath.Join(parent, "outer")
	inner := filepath.Join(outer, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, outer, "git", "init")
	run(t, inner, "git", "init")

	repos := DiscoverNestedGitRepos(parent)
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo (outer only), got %d: %v", len(repos), repos)
	}

	resolved, _ := filepath.EvalSymlinks(outer)
	got, _ := filepath.EvalSymlinks(repos[0])
	if got != resolved {
		t.Errorf("expected %q, got %q", resolved, got)
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
