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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ory/lumen/internal/indexlock"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
)

// SeedFromDonor snapshots the donor SQLite database to dstPath if dstPath does
// not already exist. It stamps projectPath as the new database owner, then
// atomically publishes the snapshot.
func SeedFromDonor(donorPath, dstPath, projectPath string) (bool, error) {
	return SeedFromDonorContext(context.Background(), donorPath, dstPath, projectPath)
}

// SeedFromDonorContext is SeedFromDonor with cancellation support.
//
// Seeding is safe to run concurrently from multiple processes (e.g. the
// SessionStart background indexer and the first MCP search racing to warm the
// same fresh worktree): an advisory seed lock is claimed before reading the
// donor, so only the winner copies while other callers wait and then observe
// the published destination. Advisory locks are released by the OS on process
// exit, avoiding stale claims after a crash.
//
// Returns (true, nil) if seeded successfully, (false, nil) if dstPath already
// exists, or (false, error) on failure.
func SeedFromDonorContext(ctx context.Context, donorPath, dstPath, projectPath string) (bool, error) {
	if _, err := os.Stat(dstPath); err == nil {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return false, fmt.Errorf("create dst directory: %w", err)
	}

	seedLock, err := indexlock.Acquire(ctx, dstPath+".seed.lock")
	if err != nil {
		return false, fmt.Errorf("acquire seed lock: %w", err)
	}
	defer seedLock.Release()

	// Another seeder may have published the destination while this caller
	// waited for the seed lock. Exit before opening or copying the donor.
	if _, err := os.Stat(dstPath); err == nil {
		return false, nil
	}

	// Verify donor has completed at least one full indexing pass.
	// A missing or empty root_hash means the donor is still being built
	// (or was interrupted), so its data is incomplete and potentially
	// inconsistent — skip seeding to avoid inheriting corrupt state.
	// mode=ro prevents sqlite from creating an empty donor if it disappears
	// after discovery. VACUUM INTO reads a transactionally consistent snapshot,
	// including committed WAL content, without modifying the live donor.
	db, err := sql.Open("sqlite3", sqliteFileDSN(donorPath, "ro"))
	if err != nil {
		return false, fmt.Errorf("open donor: %w", err)
	}
	var rootHash sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT value FROM project_meta WHERE key = 'root_hash'").Scan(&rootHash); err != nil && !errors.Is(err, sql.ErrNoRows) {
		_ = db.Close()
		return false, fmt.Errorf("read donor metadata: %w", err)
	}

	if !rootHash.Valid || rootHash.String == "" {
		if err := db.Close(); err != nil {
			return false, fmt.Errorf("close donor: %w", err)
		}
		return false, nil
	}

	// The seed lock makes a fixed temp name safe. A crash can leave at most this
	// one file behind, and the next attempt removes and replaces it instead of
	// accumulating full-size index.db.seed-* orphans.
	tmp := dstPath + ".seed-tmp"
	removeSQLiteFiles(tmp)
	defer func() {
		removeSQLiteFiles(tmp)
	}()

	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", tmp); err != nil {
		_ = db.Close()
		return false, fmt.Errorf("snapshot donor: %w", err)
	}
	if err := db.Close(); err != nil {
		return false, fmt.Errorf("close donor: %w", err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return false, fmt.Errorf("set seed temp permissions: %w", err)
	}

	// Root hashes use relative paths, so an unchanged sibling worktree can
	// return early from EnsureFresh without rewriting metadata. Stamp the new
	// owner before publication so project-scoped purge targets the right index.
	if err := setSeedProjectPath(ctx, tmp, projectPath); err != nil {
		return false, fmt.Errorf("set seed project path: %w", err)
	}

	seeded, err := publishSeed(tmp, dstPath, os.Link, os.Rename)
	if err != nil {
		return false, err
	}
	return seeded, nil
}

func sqliteFileDSN(path, mode string) string {
	return (&url.URL{
		Scheme:   "file",
		Path:     filepath.ToSlash(path),
		RawQuery: "mode=" + mode,
	}).String()
}

func setSeedProjectPath(ctx context.Context, dbPath, projectPath string) error {
	db, err := sql.Open("sqlite3", sqliteFileDSN(dbPath, "rw"))
	if err != nil {
		return err
	}

	var shared bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'collection_meta')`,
	).Scan(&shared); err != nil {
		_ = db.Close()
		return err
	}
	if shared {
		if err := setSharedSeedProjectPath(ctx, db, projectPath); err != nil {
			_ = db.Close()
			return err
		}
	} else if _, err := db.ExecContext(ctx,
		`INSERT INTO project_meta (key, value) VALUES ('project_path', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		projectPath,
	); err != nil {
		_ = db.Close()
		return err
	}
	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		_ = db.Close()
		return err
	}
	return db.Close()
}

// setSharedSeedProjectPath translates the legacy single-owner metadata update
// to the repository-scoped schema. The completed donor project retains all of
// its project_meta and project_files rows; only its canonical path changes.
func setSharedSeedProjectPath(ctx context.Context, db *sql.DB, projectPath string) error {
	if abs, err := filepath.Abs(projectPath); err == nil {
		projectPath = filepath.Clean(abs)
	}

	var donorID int64
	err := db.QueryRowContext(ctx, `
		SELECT pm.project_id
		FROM project_meta pm
		JOIN projects p ON p.id = pm.project_id
		WHERE pm.key = 'root_hash' AND pm.value <> ''
		ORDER BY p.last_accessed_at DESC, p.id DESC
		LIMIT 1`,
	).Scan(&donorID)
	if err != nil {
		return err
	}

	var existingID int64
	err = db.QueryRowContext(ctx, `SELECT id FROM projects WHERE path = ?`, projectPath).Scan(&existingID)
	switch {
	case err == nil && existingID == donorID:
		return nil
	case err == nil:
		return fmt.Errorf("seed project path %q already belongs to project %d", projectPath, existingID)
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}

	_, err = db.ExecContext(ctx, `UPDATE projects SET path = ? WHERE id = ?`, projectPath, donorID)
	return err
}

func removeSQLiteFiles(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}
}

// publishSeed prefers create-if-absent hard-link publication. Filesystems that
// do not support hard links (for example exFAT and some network/overlay mounts)
// fall back to atomic rename while the caller holds the seed lock.
func publishSeed(tmp, dst string, link, rename func(string, string) error) (bool, error) {
	if err := link(tmp, dst); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrExist) {
		return false, nil
	}

	// A non-ErrExist link failure may mean the filesystem has no hard-link
	// support. Do not overwrite a destination created by a non-cooperating
	// process before attempting the portable rename fallback.
	if _, err := os.Stat(dst); err == nil {
		return false, nil
	}
	if err := rename(tmp, dst); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, fmt.Errorf("publish seed: link and rename fallback failed: %w", err)
	}
	return true, nil
}
