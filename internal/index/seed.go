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

package index

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver for WAL checkpoint
)

// SeedFromDonor copies the donor SQLite database to dstPath if dstPath does
// not already exist. It checkpoints the WAL first to ensure a self-contained
// copy, then atomically publishes the copy via a create-if-absent hard link.
//
// Seeding is safe to run concurrently from multiple processes (e.g. the
// SessionStart background indexer and the first MCP search racing to warm the
// same fresh worktree): the copy goes to a per-process temp file and dstPath is
// created with os.Link, which fails if it already exists. The loser of the race
// therefore no-ops instead of renaming its copy over a database the winner has
// already opened and begun writing to.
//
// Returns (true, nil) if seeded successfully, (false, nil) if dstPath already
// exists, or (false, error) on failure.
func SeedFromDonor(donorPath, dstPath string) (bool, error) {
	if _, err := os.Stat(dstPath); err == nil {
		return false, nil
	}

	// Verify donor has completed at least one full indexing pass.
	// A missing or empty root_hash means the donor is still being built
	// (or was interrupted), so its data is incomplete and potentially
	// inconsistent — skip seeding to avoid inheriting corrupt state.
	db, err := sql.Open("sqlite3", donorPath+"?mode=ro")
	if err != nil {
		return false, fmt.Errorf("open donor: %w", err)
	}
	var rootHash sql.NullString
	_ = db.QueryRow("SELECT value FROM project_meta WHERE key = 'root_hash'").Scan(&rootHash)

	// Checkpoint the WAL so the main DB file is self-contained.
	_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	_ = db.Close()

	if !rootHash.Valid || rootHash.String == "" {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return false, fmt.Errorf("create dst directory: %w", err)
	}

	// Copy to a uniquely-named temp file in the destination directory, then
	// publish it via a create-if-absent hard link. os.CreateTemp guarantees a
	// unique name even among concurrent callers in the same process, and os.Link
	// fails with os.ErrExist when dstPath already exists. Together these give
	// safe concurrent seeding: whoever links first wins, the rest no-op, and no
	// one ever renames a fresh copy over a database another seeder has already
	// published and opened for writing.
	tmpFile, err := os.CreateTemp(filepath.Dir(dstPath), filepath.Base(dstPath)+".seed-*")
	if err != nil {
		return false, fmt.Errorf("create seed temp: %w", err)
	}
	tmp := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmp) }()

	if err := copyFile(donorPath, tmp); err != nil {
		return false, fmt.Errorf("copy donor: %w", err)
	}

	if err := os.Link(tmp, dstPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			// Another seeder created dstPath first; its copy is authoritative.
			return false, nil
		}
		return false, fmt.Errorf("link seed: %w", err)
	}

	return true, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
