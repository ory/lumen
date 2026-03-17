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
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver for WAL checkpoint
)

// SeedFromDonor copies the donor SQLite database to dstPath if dstPath does
// not already exist. It checkpoints the WAL first to ensure a self-contained
// copy, then performs an atomic copy (write to temp file + rename).
//
// Returns (true, nil) if seeded successfully, (false, nil) if dstPath already
// exists, or (false, error) on failure.
func SeedFromDonor(donorPath, dstPath string) (bool, error) {
	if _, err := os.Stat(dstPath); err == nil {
		return false, nil
	}

	// Checkpoint the WAL so the main DB file is self-contained.
	db, err := sql.Open("sqlite3", donorPath+"?mode=ro")
	if err != nil {
		return false, fmt.Errorf("open donor: %w", err)
	}
	_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	_ = db.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return false, fmt.Errorf("create dst directory: %w", err)
	}

	// Atomic copy: write to temp file then rename.
	tmp := dstPath + ".seed-tmp"
	if err := copyFile(donorPath, tmp); err != nil {
		_ = os.Remove(tmp)
		return false, fmt.Errorf("copy donor: %w", err)
	}

	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return false, fmt.Errorf("rename seed: %w", err)
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
