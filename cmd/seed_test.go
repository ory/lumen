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
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/store"
)

// TestSeedFromDonorIfNew_SeedsWorktreeFromSibling reproduces the bug where
// `lumen index` (the background indexer spawned by the SessionStart hook) does
// not copy an existing sibling-worktree index before indexing a fresh
// worktree, causing a full re-index from scratch instead of an incremental
// merkle update. A worktree whose sibling is already indexed must be seeded.
func TestSeedFromDonorIfNew_SeedsWorktreeFromSibling(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	t.Setenv("XDG_DATA_HOME", t.TempDir())
	const model = "test-model"

	// Main repo with a worktree, mirroring `git worktree add`.
	main := t.TempDir()
	runGit(t, main, "init")
	runGit(t, main, "config", "user.email", "test@test.com")
	runGit(t, main, "config", "user.name", "test")
	runGit(t, main, "commit", "--allow-empty", "-m", "init")

	wt := filepath.Join(t.TempDir(), "wt")
	runGit(t, main, "worktree", "add", wt)

	mainResolved, err := filepath.EvalSymlinks(main)
	if err != nil {
		t.Fatal(err)
	}
	wtResolved, err := filepath.EvalSymlinks(wt)
	if err != nil {
		t.Fatal(err)
	}

	// Build a minimal but valid donor index for the main worktree: a real
	// SQLite store carrying a non-empty root_hash, which is what SeedFromDonor
	// requires to treat the donor as a completed index.
	donorDB := config.DBPathForProject(mainResolved, model)
	if err := os.MkdirAll(filepath.Dir(donorDB), 0o755); err != nil {
		t.Fatal(err)
	}
	donor, err := store.New(donorDB, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := donor.SetMeta("root_hash", "donor-root-hash"); err != nil {
		t.Fatal(err)
	}
	if err := donor.SetMeta("project_path", mainResolved); err != nil {
		t.Fatal(err)
	}
	if err := donor.Close(); err != nil {
		t.Fatal(err)
	}

	dstDB := config.DBPathForProject(wtResolved, model)
	if _, err := os.Stat(dstDB); !os.IsNotExist(err) {
		t.Fatalf("precondition failed: worktree DB should not exist yet (stat err=%v)", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	seedFromDonorIfNew(dstDB, wtResolved, model, logger)

	if _, err := os.Stat(dstDB); err != nil {
		t.Fatalf("worktree index was NOT seeded from sibling donor: %v", err)
	}

	seeded, err := store.New(dstDB, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = seeded.Close() }()
	got, err := seeded.GetMeta("root_hash")
	if err != nil {
		t.Fatal(err)
	}
	if got != "donor-root-hash" {
		t.Fatalf("expected seeded DB to carry donor root_hash, got %q", got)
	}
}
