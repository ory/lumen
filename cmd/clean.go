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
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/store"
	"github.com/spf13/cobra"
)

// defaultCleanDays is how long an index may go unused before `lumen clean`
// removes it.
const (
	defaultCleanDays = 30
	maxCleanDays     = 106751
)

const dailyCleanupInterval = 24 * time.Hour

var (
	removeIndexDir      = os.RemoveAll
	cleanupCollectionAt = store.CleanupCollectionAt
	tryAcquireExclusive = indexlock.TryAcquire
)

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
	Long: fmt.Sprintf(`Garbage-collects lumen indexes under ~/.local/share/lumen/.

Shared collections lose project memberships that have not been opened for
--days days (default %d), or whose worktree no longer exists. Unreferenced file
revisions, chunks, and vectors are then deleted and free pages are reclaimed.
Legacy per-project index directories are removed using the same age policy.

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
	if days > maxCleanDays {
		return fmt.Errorf("--days must not exceed %d, got %d", maxCleanDays, days)
	}
	dataDir := filepath.Join(config.XDGDataDir(), "lumen")
	return cleanIndexes(cmd.ErrOrStderr(), cmd.OutOrStdout(), dataDir, days, time.Now())
}

// cleanIndexes removes every stale index directory directly under dataDir,
// reporting each decision on the injected stderr and a summary on the injected
// stdout. The injected writers deliberately keep this reusable by both the
// interactive CLI and the MCP background cleanup without mutating pterm's
// process-global state. now is injected so the age cutoff is testable. Failures
// to remove a single directory are
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
	projectsRemoved, vectorsRemoved := 0, 0
	var bytesReclaimed int64
	var firstErr error

	for _, entry := range entries {
		// Only hash-named index directories are candidates; the shared
		// debug.log lives in the same data dir.
		if !entry.IsDir() {
			continue
		}
		hashDir := filepath.Join(dataDir, entry.Name())
		wasRemoved, sharedStats, cleanErr := cleanIndex(stderr, entry.Name(), hashDir, days, cutoff)
		projectsRemoved += sharedStats.ProjectsRemoved
		vectorsRemoved += sharedStats.VectorsRemoved
		bytesReclaimed += sharedStats.BytesReclaimed
		if wasRemoved {
			removed++
		} else {
			skipped++
		}
		if cleanErr != nil && firstErr == nil {
			firstErr = cleanErr
		}
	}

	_, _ = fmt.Fprintf(stdout, "Removed %d index director%s, skipped %d.\n",
		removed, pluralY(removed), skipped)
	if projectsRemoved > 0 || vectorsRemoved > 0 || bytesReclaimed > 0 {
		_, _ = fmt.Fprintf(stdout, "Shared cleanup: %d projects, %d vectors, %d bytes reclaimed.\n",
			projectsRemoved, vectorsRemoved, bytesReclaimed)
	}
	return firstErr
}

// cleanIndex cleans one legacy index or shared collection while retaining the
// exclusive collection lock for the entire database cleanup and removal.
func cleanIndex(stderr io.Writer, name, hashDir string, days int, cutoff time.Time) (bool, store.CleanupStats, error) {
	dbPath := filepath.Join(hashDir, "index.db")
	lock, lockErr := tryAcquireExclusive(indexlock.LockPathForDB(dbPath))
	if lockErr != nil {
		_, _ = fmt.Fprintf(stderr, "Failed to acquire index lock for %s: %v\n", name, lockErr)
		return false, store.CleanupStats{}, fmt.Errorf("acquire index lock for %s: %w", name, lockErr)
	}
	if lock == nil {
		_, _ = fmt.Fprintf(stderr, "Keeping %s: an indexer is currently running.\n", name)
		return false, store.CleanupStats{}, nil
	}
	defer lock.Release()

	sharedStats, shared, sharedErr := cleanupCollectionAt(dbPath, cutoff)
	if shared {
		if sharedErr != nil {
			_, _ = fmt.Fprintf(stderr, "Failed to clean shared collection %s: %v\n", name, sharedErr)
			return false, store.CleanupStats{}, fmt.Errorf("clean shared collection %s: %w", name, sharedErr)
		}
		if sharedStats.ProjectsLeft > 0 {
			_, _ = fmt.Fprintf(stderr, "Cleaned %s: removed %d projects and %d vectors.\n", name, sharedStats.ProjectsRemoved, sharedStats.VectorsRemoved)
			return false, sharedStats, nil
		}
		// Empty collections have no future owner and can be removed as a
		// directory, reclaiming sidecars and metadata in one operation.
		if err := removeIndexDir(hashDir); err != nil {
			return false, sharedStats, fmt.Errorf("remove empty collection %s: %w", hashDir, err)
		}
		return true, sharedStats, nil
	}

	stale, reason := isIndexStale(dbPath, days, cutoff)
	if !stale {
		return false, store.CleanupStats{}, nil
	}
	if err := removeIndexDir(hashDir); err != nil {
		_, _ = fmt.Fprintf(stderr, "Failed to remove %s: %v\n", hashDir, err)
		return false, store.CleanupStats{}, fmt.Errorf("remove %s: %w", hashDir, err)
	}
	_, _ = fmt.Fprintf(stderr, "Removed %s (%s).\n", name, reason)
	return true, store.CleanupStats{}, nil
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

// runDailyCleanup performs the MCP-startup maintenance sweep at most once per
// day. The stamp is deliberately outside collection directories so it is not
// mistaken for an index by cleanIndexes.
func runDailyCleanup(dataDir string, now time.Time, logger *slog.Logger) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	stampPath := filepath.Join(dataDir, ".last-cleanup")
	if info, err := os.Stat(stampPath); err == nil && now.Sub(info.ModTime()) < dailyCleanupInterval {
		logger.Debug("daily cleanup skipped: stamp is fresh", "stamp_path", stampPath)
		return
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		logger.Warn("daily cleanup: create data directory", "path", dataDir, "error", err)
		return
	}
	var stderr, stdout bytes.Buffer
	if err := cleanIndexes(&stderr, &stdout, dataDir, defaultCleanDays, now); err != nil {
		logger.Warn("daily cleanup failed", "error", err, "details", strings.TrimSpace(stderr.String()))
		return
	}
	logger.Info("daily cleanup complete", "summary", strings.TrimSpace(stdout.String()), "details", strings.TrimSpace(stderr.String()))
	if err := os.WriteFile(stampPath, []byte(now.UTC().Format(time.RFC3339)), 0o600); err != nil {
		logger.Warn("daily cleanup: write stamp", "path", stampPath, "error", err)
	}
}
