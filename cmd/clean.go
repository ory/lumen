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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/store"
	"github.com/spf13/cobra"
)

// defaultCleanDays is how long an index may go unused before `lumen clean`
// removes it.
const defaultCleanDays = 30

func init() {
	addCleanFlags(cleanCmd)
	rootCmd.AddCommand(cleanCmd)
}

// addCleanFlags registers the clean flags. Shared with the tests so the flag
// definition never drifts from what runClean reads.
func addCleanFlags(cmd *cobra.Command) {
	cmd.Flags().Int("days", defaultCleanDays,
		"remove indexes not used in the last N days (0 removes every index that is not currently being written)")
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove unused or orphaned lumen indexes",
	Long: fmt.Sprintf(`Deletes unused lumen index databases under ~/.local/share/lumen/.

An index is removed when it has not been opened for --days days (default %d),
or when the project it was built for no longer exists — indexes are keyed by
project path, embedding model, and index version, so renamed projects, deleted
checkouts, and abandoned models leave behind data that is never read again.

Indexes written by older binaries that never recorded an access time fall back
to their last indexing time; those without any usable timestamp are removed.

Use "lumen clean --days 0" to drop every cached index on this host, and
"lumen index --force <project-path>" to rebuild a single project from scratch.

Indexes with an indexer currently running are always kept.`, defaultCleanDays),
	Args: cobra.NoArgs,
	RunE: runClean,
}

func runClean(cmd *cobra.Command, _ []string) error {
	days, err := cmd.Flags().GetInt("days")
	if err != nil {
		return err
	}
	if days < 0 {
		return fmt.Errorf("--days must not be negative, got %d", days)
	}
	dataDir := filepath.Join(config.XDGDataDir(), "lumen")
	return cleanIndexes(cmd.ErrOrStderr(), cmd.OutOrStdout(), dataDir, days, time.Now())
}

// cleanIndexes removes every stale index directory directly under dataDir,
// reporting each decision on stderr and a summary on stdout. now is injected so
// the age cutoff is testable. Failures to remove a single directory are
// reported and the sweep continues; the first such failure is returned once
// every directory has been considered.
func cleanIndexes(stderr, stdout io.Writer, dataDir string, days int, now time.Time) error {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(stderr, "No index data found — nothing to clean.")
			return nil
		}
		return fmt.Errorf("read data dir: %w", err)
	}

	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	removed, skipped := 0, 0
	var firstErr error

	for _, entry := range entries {
		// Only hash-named index directories are candidates; the shared
		// debug.log lives in the same data dir.
		if !entry.IsDir() {
			continue
		}
		hashDir := filepath.Join(dataDir, entry.Name())
		dbPath := filepath.Join(hashDir, "index.db")

		if indexlock.IsHeld(indexlock.LockPathForDB(dbPath)) {
			_, _ = fmt.Fprintf(stderr, "Keeping %s: an indexer is currently running.\n", entry.Name())
			skipped++
			continue
		}

		stale, reason := isIndexStale(dbPath, days, cutoff)
		if !stale {
			skipped++
			continue
		}
		if err := os.RemoveAll(hashDir); err != nil {
			_, _ = fmt.Fprintf(stderr, "Failed to remove %s: %v\n", hashDir, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("remove %s: %w", hashDir, err)
			}
			skipped++
			continue
		}
		_, _ = fmt.Fprintf(stderr, "Removed %s (%s).\n", entry.Name(), reason)
		removed++
	}

	_, _ = fmt.Fprintf(stdout, "Removed %d index director%s, skipped %d.\n",
		removed, pluralY(removed), skipped)
	return firstErr
}

// isIndexStale reports whether the index at dbPath is no longer worth keeping,
// along with a human-readable reason. The metadata read is read-only so it does
// not itself count as an access.
func isIndexStale(dbPath string, days int, cutoff time.Time) (bool, string) {
	if days == 0 {
		return true, "--days 0"
	}

	meta, err := store.ReadMetaAt(dbPath, "project_path", store.MetaLastAccessedAt, "last_indexed_at")
	if err != nil {
		// Missing, truncated, or non-lumen database: nothing here can be read
		// again, so it is pure waste.
		return true, "no readable index metadata"
	}

	projectPath := meta["project_path"]
	if projectPath == "" {
		return true, "no project path recorded"
	}
	info, statErr := os.Stat(projectPath)
	switch {
	case statErr == nil && !info.IsDir():
		return true, fmt.Sprintf("project path %s is not a directory", projectPath)
	case os.IsNotExist(statErr):
		return true, fmt.Sprintf("project %s no longer exists", projectPath)
	}
	// Any other stat error (e.g. an unreadable parent directory) is
	// inconclusive — the project may well still be there, so fall through to
	// the age check rather than deleting a live index.

	if ts, ok := parseIndexTime(meta[store.MetaLastAccessedAt]); ok {
		if ts.After(cutoff) {
			return false, ""
		}
		return true, fmt.Sprintf("not accessed since %s", ts.Format(time.RFC3339))
	}
	if ts, ok := parseIndexTime(meta["last_indexed_at"]); ok {
		if ts.After(cutoff) {
			return false, ""
		}
		return true, fmt.Sprintf("not indexed since %s", ts.Format(time.RFC3339))
	}
	return true, "no usable access timestamp"
}

// parseIndexTime parses an RFC3339 metadata timestamp, reporting whether the
// value was present and well-formed.
func parseIndexTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
