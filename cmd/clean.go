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
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/store"
	"github.com/ory/lumen/internal/tui"
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
	reporter := interactiveCleanReporter{progress: tui.NewProgress(os.Stderr)}
	summary, err := cleanIndexes(reporter, dataDir, days, time.Now())
	if output := formatCleanSummary(summary); output != "" {
		fmt.Printf("%s", output)
	}
	return err
}

type cleanReporter interface {
	Info(string)
	Error(string)
}

type interactiveCleanReporter struct {
	progress *tui.Progress
}

func (r interactiveCleanReporter) Info(message string) {
	r.progress.Info(message)
}

func (interactiveCleanReporter) Error(message string) {
	fmt.Fprintf(os.Stderr, "%s\n", message)
}

type slogCleanReporter struct {
	logger *slog.Logger
}

func (r slogCleanReporter) Info(message string) {
	r.logger.Info("daily cleanup detail", "message", message)
}

func (r slogCleanReporter) Error(message string) {
	r.logger.Warn("daily cleanup issue", "message", message)
}

type cleanSummary struct {
	noData          bool
	removed         int
	skipped         int
	projectsRemoved int
	vectorsRemoved  int
	bytesReclaimed  int64
}

func formatCleanSummary(summary cleanSummary) string {
	if summary.noData {
		return ""
	}
	output := fmt.Sprintf("Removed %d index director%s, skipped %d.\n",
		summary.removed, pluralY(summary.removed), summary.skipped)
	if summary.projectsRemoved > 0 || summary.vectorsRemoved > 0 || summary.bytesReclaimed > 0 {
		output += fmt.Sprintf("Shared cleanup: %d projects, %d vectors, %d bytes reclaimed.\n",
			summary.projectsRemoved, summary.vectorsRemoved, summary.bytesReclaimed)
	}
	return output
}

// cleanIndexes removes every stale index directory directly under dataDir and
// reports individual decisions through reporter. The returned summary is
// rendered by the caller using the output strategy for its execution context.
// Failures to remove a single directory are reported and the sweep continues;
// the first such failure is returned after every directory has been considered.
func cleanIndexes(reporter cleanReporter, dataDir string, days int, now time.Time) (cleanSummary, error) {
	var summary cleanSummary
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			reporter.Info("No index data found — nothing to clean.")
			summary.noData = true
			return summary, nil
		}
		return summary, fmt.Errorf("read data dir: %w", err)
	}

	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	var firstErr error

	for _, entry := range entries {
		// Only hash-named index directories are candidates; the shared
		// debug.log lives in the same data dir.
		if !entry.IsDir() {
			continue
		}
		hashDir := filepath.Join(dataDir, entry.Name())
		wasRemoved, sharedStats, cleanErr := cleanIndex(reporter, entry.Name(), hashDir, days, cutoff)
		summary.projectsRemoved += sharedStats.ProjectsRemoved
		summary.vectorsRemoved += sharedStats.VectorsRemoved
		summary.bytesReclaimed += sharedStats.BytesReclaimed
		if wasRemoved {
			summary.removed++
		} else {
			summary.skipped++
		}
		if cleanErr != nil && firstErr == nil {
			firstErr = cleanErr
		}
	}
	return summary, firstErr
}

// cleanIndex cleans one legacy index or shared collection while retaining the
// exclusive collection lock for the entire database cleanup and removal.
func cleanIndex(reporter cleanReporter, name, hashDir string, days int, cutoff time.Time) (bool, store.CleanupStats, error) {
	dbPath := filepath.Join(hashDir, "index.db")
	lock, lockErr := tryAcquireExclusive(indexlock.LockPathForDB(dbPath))
	if lockErr != nil {
		reporter.Error(fmt.Sprintf("Failed to acquire index lock for %s: %v", name, lockErr))
		return false, store.CleanupStats{}, fmt.Errorf("acquire index lock for %s: %w", name, lockErr)
	}
	if lock == nil {
		reporter.Info(fmt.Sprintf("Keeping %s: an indexer is currently running.", name))
		return false, store.CleanupStats{}, nil
	}
	defer lock.Release()

	sharedStats, shared, sharedErr := cleanupCollectionAt(dbPath, cutoff)
	if shared {
		if sharedErr != nil {
			reporter.Error(fmt.Sprintf("Failed to clean shared collection %s: %v", name, sharedErr))
			return false, store.CleanupStats{}, fmt.Errorf("clean shared collection %s: %w", name, sharedErr)
		}
		if sharedStats.ProjectsLeft > 0 {
			reporter.Info(fmt.Sprintf("Cleaned %s: removed %d projects and %d vectors.", name, sharedStats.ProjectsRemoved, sharedStats.VectorsRemoved))
			return false, sharedStats, nil
		}
		// Empty collections have no future owner and can be removed as a
		// directory, reclaiming sidecars and metadata in one operation.
		if err := removeIndexDir(hashDir); err != nil {
			reporter.Error(fmt.Sprintf("Failed to remove %s: %v", hashDir, err))
			return false, sharedStats, fmt.Errorf("remove empty collection %s: %w", hashDir, err)
		}
		return true, sharedStats, nil
	}

	stale, reason := isIndexStale(dbPath, days, cutoff)
	if !stale {
		return false, store.CleanupStats{}, nil
	}
	if err := removeIndexDir(hashDir); err != nil {
		reporter.Error(fmt.Sprintf("Failed to remove %s: %v", hashDir, err))
		return false, store.CleanupStats{}, fmt.Errorf("remove %s: %w", hashDir, err)
	}
	reporter.Info(fmt.Sprintf("Removed %s (%s).", name, reason))
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
	summary, err := cleanIndexes(slogCleanReporter{logger: logger}, dataDir, defaultCleanDays, now)
	if err != nil {
		logger.Warn("daily cleanup failed", "error", err)
		return
	}
	logger.Info("daily cleanup complete",
		"indexes_removed", summary.removed,
		"indexes_skipped", summary.skipped,
		"projects_removed", summary.projectsRemoved,
		"vectors_removed", summary.vectorsRemoved,
		"bytes_reclaimed", summary.bytesReclaimed,
	)
	if err := os.WriteFile(stampPath, []byte(now.UTC().Format(time.RFC3339)), 0o600); err != nil {
		logger.Warn("daily cleanup: write stamp", "path", stampPath, "error", err)
	}
}
