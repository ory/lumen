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
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/store"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanNow is the fixed "current time" used by the clean tests so age
// comparisons never depend on the wall clock.
var cleanNow = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

// daysAgo formats a timestamp n days before cleanNow the way the indexer does.
func daysAgo(n int) string {
	return cleanNow.AddDate(0, 0, -n).Format(time.RFC3339)
}

// seedIndex creates a real SQLite index DB in the hash directory for
// (projectPath, model) and applies meta to project_meta, so tests exercise the
// same metadata-scan code path as production. Note that store.New stamps
// last_accessed_at on open; pass an explicit value in meta to override it, or
// use deleteMeta to simulate an index written by an older binary.
func seedIndex(t *testing.T, projectPath, model string, meta map[string]string) string {
	t.Helper()
	dbPath := config.DBPathForProject(projectPath, model)
	require.NoError(t, os.MkdirAll(filepath.Dir(dbPath), 0o755))
	s, err := store.New(dbPath, 4)
	require.NoError(t, err)
	require.NoError(t, s.SetMeta("project_path", projectPath))
	for k, v := range meta {
		require.NoError(t, s.SetMeta(k, v))
	}
	require.NoError(t, s.Close())
	return filepath.Dir(dbPath)
}

// deleteMeta removes keys from the index's project_meta table, simulating
// indexes written by binaries that never recorded them.
func deleteMeta(t *testing.T, hashDir string, keys ...string) {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(hashDir, "index.db"))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	for _, k := range keys {
		_, err := db.Exec("DELETE FROM project_meta WHERE key = ?", k)
		require.NoError(t, err)
	}
}

// projectDir creates a stand-in project directory that exists on disk.
func projectDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(resolvedTempDir(t), name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	return dir
}

// runCleanIndexes invokes the cleanup sweep against the data dir under tmp.
func runCleanIndexes(t *testing.T, tmp string, days int) (stdout, stderr string, err error) {
	t.Helper()
	reporter := newBufferCleanReporter()
	summary, err := cleanIndexes(reporter, filepath.Join(tmp, "lumen"), days, cleanNow)
	return formatCleanSummary(summary), reporter.output.String(), err
}

type bufferCleanReporter struct {
	output bytes.Buffer
}

func newBufferCleanReporter() *bufferCleanReporter {
	return &bufferCleanReporter{}
}

func (r *bufferCleanReporter) Info(message string) {
	_, _ = fmt.Fprintln(&r.output, message)
}

func (r *bufferCleanReporter) Error(message string) {
	_, _ = fmt.Fprintln(&r.output, message)
}

// runCleanCmd invokes runClean through a command carrying the real clean flags.
func runCleanCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd := &cobra.Command{Use: "clean"}
	addCleanFlags(cmd)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	if err := cmd.Flags().Parse(args); err != nil {
		return "", "", err
	}
	err = runClean(cmd, cmd.Flags().Args())
	return outBuf.String(), errBuf.String(), err
}

func TestClean_KeepsRecentlyAccessedIndex(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "fresh")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": daysAgo(3),
	})

	stdoutOut, _, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.DirExists(t, hashDir, "recently accessed index must survive")
	assert.Contains(t, stdoutOut, "Removed 0 index")
	assert.Contains(t, stdoutOut, "skipped 1")
}

func TestClean_RemovesStaleIndex(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "stale")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": daysAgo(45),
	})

	stdoutOut, stderrOut, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.NoDirExists(t, hashDir, "index unused for 45 days must be removed")
	assert.Contains(t, stderrOut, "not accessed since")
	assert.Contains(t, stdoutOut, "Removed 1 index")
}

// TestClean_ExactCutoffIsStale pins the boundary: an index last accessed
// exactly on the cutoff counts as stale.
func TestClean_ExactCutoffIsStale(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "boundary")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": cleanNow.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
	})

	_, _, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.NoDirExists(t, hashDir, "index exactly on the cutoff must be removed")
}

// TestClean_FallsBackToLastIndexedAt covers indexes written before
// last_accessed_at existed: their indexing timestamp decides staleness.
func TestClean_FallsBackToLastIndexedAt(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	freshProject := projectDir(t, "legacy-fresh")
	freshDir := seedIndex(t, freshProject, embedder.DefaultModel, map[string]string{
		"last_indexed_at": daysAgo(2),
	})
	deleteMeta(t, freshDir, "last_accessed_at")

	staleProject := projectDir(t, "legacy-stale")
	staleDir := seedIndex(t, staleProject, embedder.DefaultModel, map[string]string{
		"last_indexed_at": daysAgo(90),
	})
	deleteMeta(t, staleDir, "last_accessed_at")

	_, stderrOut, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.DirExists(t, freshDir, "recently indexed legacy index must survive")
	assert.NoDirExists(t, staleDir, "long-unindexed legacy index must be removed")
	assert.Contains(t, stderrOut, "not indexed since")
}

func TestClean_RemovesIndexWithNoTimestamps(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "no-timestamps")
	hashDir := seedIndex(t, project, embedder.DefaultModel, nil)
	deleteMeta(t, hashDir, "last_accessed_at", "last_indexed_at")

	_, stderrOut, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.NoDirExists(t, hashDir, "index without any timestamp must be removed")
	assert.Contains(t, stderrOut, "no usable access timestamp")
}

func TestClean_RemovesIndexWithInvalidTimestamps(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "invalid-timestamps")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": "not-a-timestamp",
		"last_indexed_at":  "also-not-a-timestamp",
	})

	_, _, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.NoDirExists(t, hashDir, "index with unparseable timestamps must be removed")
}

// TestClean_InvalidAccessTimestampFallsBackToIndexedAt verifies a corrupt
// last_accessed_at does not discard a perfectly good last_indexed_at.
func TestClean_InvalidAccessTimestampFallsBackToIndexedAt(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "invalid-access")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": "garbage",
		"last_indexed_at":  daysAgo(1),
	})

	_, _, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.DirExists(t, hashDir, "recent last_indexed_at must keep the index alive")
}

func TestClean_RemovesIndexWhenProjectIsGone(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "deleted-project")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": daysAgo(1),
	})
	require.NoError(t, os.RemoveAll(project))

	_, stderrOut, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.NoDirExists(t, hashDir, "index of a deleted project must be removed regardless of age")
	assert.Contains(t, stderrOut, "no longer exists")
}

func TestClean_RemovesIndexWhenProjectPathIsNotADirectory(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := filepath.Join(resolvedTempDir(t), "a-file")
	require.NoError(t, os.WriteFile(project, []byte("not a project"), 0o600))
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": daysAgo(1),
	})

	_, stderrOut, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.NoDirExists(t, hashDir, "index whose project path is a file must be removed")
	assert.Contains(t, stderrOut, "not a directory")
}

func TestClean_RemovesIndexWithoutProjectPath(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "unrecorded")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": daysAgo(1),
	})
	deleteMeta(t, hashDir, "project_path")

	_, stderrOut, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.NoDirExists(t, hashDir, "index without a recorded project path must be removed")
	assert.Contains(t, stderrOut, "no project path recorded")
}

func TestClean_RemovesMalformedIndexDirectory(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	dataDir := filepath.Join(tmp, "lumen")
	empty := filepath.Join(dataDir, "0000000000000000")
	require.NoError(t, os.MkdirAll(empty, 0o755))
	garbage := filepath.Join(dataDir, "1111111111111111")
	require.NoError(t, os.MkdirAll(garbage, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(garbage, "index.db"), []byte("not sqlite"), 0o600))

	stdoutOut, _, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.NoDirExists(t, empty, "index directory without a database must be removed")
	assert.NoDirExists(t, garbage, "unreadable index database must be removed")
	assert.Contains(t, stdoutOut, "Removed 2 index")
}

// TestClean_LeavesNonIndexFilesAlone verifies the sweep only touches index
// directories — the shared debug log lives in the same data dir.
func TestClean_LeavesNonIndexFilesAlone(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	dataDir := filepath.Join(tmp, "lumen")
	require.NoError(t, os.MkdirAll(dataDir, 0o755))
	logPath := filepath.Join(dataDir, "debug.log")
	require.NoError(t, os.WriteFile(logPath, []byte("log line\n"), 0o600))

	_, _, err := runCleanIndexes(t, tmp, 0)
	require.NoError(t, err)
	assert.FileExists(t, logPath, "debug.log must not be removed")
}

func TestCleanIndexReportsLockAcquisitionErrors(t *testing.T) {
	original := tryAcquireExclusive
	t.Cleanup(func() { tryAcquireExclusive = original })
	tryAcquireExclusive = func(string) (*indexlock.Lock, error) {
		return nil, errors.New("permission denied")
	}
	reporter := newBufferCleanReporter()
	removed, _, err := cleanIndex(reporter, "abc", t.TempDir(), 30, time.Now())
	if err == nil || removed {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	if !strings.Contains(reporter.output.String(), "Failed to acquire index lock") || strings.Contains(reporter.output.String(), "currently running") {
		t.Fatalf("unexpected stderr: %s", reporter.output.String())
	}
}

func TestRunDailyCleanupUsesProvidedLoggerAndStampsSuccess(t *testing.T) {
	dataDir := t.TempDir()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	runDailyCleanup(dataDir, now, logger)
	stamp, err := os.ReadFile(filepath.Join(dataDir, ".last-cleanup"))
	if err != nil {
		t.Fatal(err)
	}
	if string(stamp) != now.Format(time.RFC3339) {
		t.Fatalf("stamp = %q, want %q", stamp, now.Format(time.RFC3339))
	}
	if !strings.Contains(logs.String(), "daily cleanup complete") {
		t.Fatalf("provided logger did not receive cleanup summary: %s", logs.String())
	}
}

// TestClean_HandlesMultipleModelsPerProject verifies each model's index is aged
// independently, since switching models creates a separate index directory.
func TestClean_HandlesMultipleModelsPerProject(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "multi-model")
	current := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": daysAgo(1),
	})
	abandoned := seedIndex(t, project, "some-other-model", map[string]string{
		"last_accessed_at": daysAgo(200),
	})
	require.NotEqual(t, current, abandoned, "models must map to distinct index dirs")

	stdoutOut, _, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.DirExists(t, current, "index for the model in use must survive")
	assert.NoDirExists(t, abandoned, "index for the abandoned model must be removed")
	assert.Contains(t, stdoutOut, "Removed 1 index")
}

func TestClean_DaysZeroRemovesEveryIndex(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "in-use")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": cleanNow.Format(time.RFC3339),
	})

	stdoutOut, _, err := runCleanIndexes(t, tmp, 0)
	require.NoError(t, err)
	assert.NoDirExists(t, hashDir, "--days 0 must remove even a just-used index")
	assert.Contains(t, stdoutOut, "Removed 1 index")
}

func TestClean_SkipsLockedIndex(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "indexing-now")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": daysAgo(500),
	})

	lockPath := indexlock.LockPathForDB(filepath.Join(hashDir, "index.db"))
	lock, err := indexlock.TryAcquire(lockPath)
	require.NoError(t, err)
	require.NotNil(t, lock)
	defer lock.Release()
	require.True(t, indexlock.IsHeld(lockPath), "precondition: lock must read as held")

	stdoutOut, stderrOut, err := runCleanIndexes(t, tmp, 0)
	require.NoError(t, err)
	assert.DirExists(t, hashDir, "an index being written must not be removed")
	assert.Contains(t, stderrOut, "indexer is currently running")
	assert.Contains(t, stdoutOut, "Removed 0 index")
	assert.Contains(t, stdoutOut, "skipped 1")
}

func TestClean_NoIndexDataIsNoOp(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	_, stderrOut, err := runCleanIndexes(t, tmp, 30)
	require.NoError(t, err)
	assert.Contains(t, stderrOut, "No index data found")
}

func TestClean_NegativeDaysFails(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "kept")
	hashDir := seedIndex(t, project, embedder.DefaultModel, nil)

	_, _, err := runCleanCmd(t, "--days", "-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--days")
	assert.DirExists(t, hashDir, "a rejected invocation must not delete anything")
}

func TestClean_MaxDaysAccepted(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "max-days")
	hashDir := seedIndex(t, project, embedder.DefaultModel, nil)

	_, _, err := runCleanCmd(t, "--days", "106751")
	require.NoError(t, err)
	assert.DirExists(t, hashDir, "the largest safe whole-day duration must be accepted")
}

func TestClean_TooManyDaysFailsWithoutDeleting(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "overflow-days")
	hashDir := seedIndex(t, project, embedder.DefaultModel, nil)

	_, _, err := runCleanCmd(t, "--days", "106752")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "106751")
	assert.DirExists(t, hashDir, "an overflowing duration must be rejected before cleanup")
}

func TestClean_HoldsLockDuringRemovalAndReleasesItAfterFailure(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "remove-failure")
	hashDir := seedIndex(t, project, embedder.DefaultModel, map[string]string{
		"last_accessed_at": daysAgo(45),
	})
	lockPath := indexlock.LockPathForDB(filepath.Join(hashDir, "index.db"))
	removeErr := errors.New("injected removal failure")
	lockHeldDuringRemoval := false
	originalRemoveIndexDir := removeIndexDir
	removeIndexDir = func(path string) error {
		assert.Equal(t, hashDir, path)
		lockHeldDuringRemoval = indexlock.IsHeld(lockPath)
		return removeErr
	}
	t.Cleanup(func() { removeIndexDir = originalRemoveIndexDir })

	_, _, err := runCleanIndexes(t, tmp, 30)
	require.ErrorIs(t, err, removeErr)
	assert.True(t, lockHeldDuringRemoval, "cleanup must hold the index lock while removing the directory")
	assert.DirExists(t, hashDir, "a failed removal must leave the index directory in place")

	lock, lockErr := indexlock.TryAcquire(lockPath)
	require.NoError(t, lockErr)
	require.NotNil(t, lock, "cleanup must release the index lock after a removal failure")
	lock.Release()
}

func TestClean_HoldsCollectionLockThroughSharedCleanupAndRemoval(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "stale-shared")
	dbPath := config.DBPathForProjectProfile(project, embedder.DefaultModel, 4, "int8", 512)
	require.NoError(t, os.MkdirAll(filepath.Dir(dbPath), 0o755))
	s, err := store.NewCollection(dbPath, 4, "int8", project)
	require.NoError(t, err)
	require.NoError(t, s.Close())
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE projects SET last_accessed_at = ? WHERE path = ?`, daysAgo(45), project)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	hashDir := filepath.Dir(dbPath)
	lockPath := indexlock.LockPathForDB(dbPath)
	removeErr := errors.New("injected removal failure")
	cleanupLockHeld := false
	removeLockHeld := false
	originalCleanup := cleanupCollectionAt
	originalRemove := removeIndexDir
	cleanupCollectionAt = func(path string, cutoff time.Time) (store.CleanupStats, bool, error) {
		assert.Equal(t, dbPath, path)
		cleanupLockHeld = indexlock.IsAnyHeld(lockPath)
		return originalCleanup(path, cutoff)
	}
	removeIndexDir = func(path string) error {
		assert.Equal(t, hashDir, path)
		removeLockHeld = indexlock.IsAnyHeld(lockPath)
		return removeErr
	}
	t.Cleanup(func() {
		cleanupCollectionAt = originalCleanup
		removeIndexDir = originalRemove
	})

	_, _, err = runCleanIndexes(t, tmp, 30)
	require.ErrorIs(t, err, removeErr)
	assert.True(t, cleanupLockHeld, "cleanup must hold the collection lock while mutating the database")
	assert.True(t, removeLockHeld, "cleanup must retain the collection lock while removing the directory")
	assert.DirExists(t, hashDir, "the injected removal failure must leave the collection directory")

	lock, lockErr := indexlock.TryAcquire(lockPath)
	require.NoError(t, lockErr)
	require.NotNil(t, lock, "cleanup must release the collection lock after a failure")
	lock.Release()
}

func TestClean_RejectsPositionalArgs(t *testing.T) {
	require.Error(t, cleanCmd.Args(cleanCmd, []string{"/some/project"}),
		"clean takes no positional arguments")
	require.NoError(t, cleanCmd.Args(cleanCmd, nil))
}

// TestClean_DefaultDays pins the documented 30-day default.
func TestClean_DefaultDays(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := projectDir(t, "default-days")
	fresh := seedIndex(t, project, embedder.DefaultModel, nil)
	stale := seedIndex(t, project, "stale-model", map[string]string{
		"last_accessed_at": time.Now().UTC().AddDate(0, 0, -31).Format(time.RFC3339),
	})

	_, _, err := runCleanCmd(t)
	require.NoError(t, err)
	assert.DirExists(t, fresh, "index accessed just now must survive the default cutoff")
	assert.NoDirExists(t, stale, "index unused for 31 days must be removed by default")
}
