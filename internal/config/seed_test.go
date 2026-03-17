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
	"testing"
)

func TestFindDonorIndex_NoSiblings(t *testing.T) {
	dir := t.TempDir()
	result := FindDonorIndexBase(dir, "/some/path", "model")
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestFindDonorIndex_WithSibling(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// Create a git repo with a worktree.
	main := t.TempDir()
	gitRun(t, main, "git", "init")
	gitRun(t, main, "git", "commit", "--allow-empty", "-m", "init")

	wt := filepath.Join(t.TempDir(), "wt")
	gitRun(t, main, "git", "worktree", "add", wt)

	// Resolve symlinks so DB path matches what ListWorktrees returns.
	mainResolved, err := filepath.EvalSymlinks(main)
	if err != nil {
		t.Fatal(err)
	}

	// Create a fake donor index for the main worktree using resolved path.
	dataDir := t.TempDir()
	donorDB := DBPathForProjectBase(dataDir, mainResolved, "test-model")
	if err := os.MkdirAll(filepath.Dir(donorDB), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(donorDB, []byte("fake-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	// From the worktree, we should find the main's index.
	result := FindDonorIndexBase(dataDir, wt, "test-model")
	if result != donorDB {
		t.Fatalf("expected %q, got %q", donorDB, result)
	}
}

func TestFindDonorIndex_WrongModel(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	main := t.TempDir()
	gitRun(t, main, "git", "init")
	gitRun(t, main, "git", "commit", "--allow-empty", "-m", "init")

	wt := filepath.Join(t.TempDir(), "wt")
	gitRun(t, main, "git", "worktree", "add", wt)

	// Resolve symlinks for DB path consistency.
	mainResolved, err := filepath.EvalSymlinks(main)
	if err != nil {
		t.Fatal(err)
	}

	// Create index for different model.
	dataDir := t.TempDir()
	donorDB := DBPathForProjectBase(dataDir, mainResolved, "model-a")
	if err := os.MkdirAll(filepath.Dir(donorDB), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(donorDB, []byte("fake-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Looking for model-b should not find model-a's index.
	result := FindDonorIndexBase(dataDir, wt, "model-b")
	if result != "" {
		t.Fatalf("expected empty string for wrong model, got %q", result)
	}
}

func gitRun(t *testing.T, dir string, name string, args ...string) {
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
