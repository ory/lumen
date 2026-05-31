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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/store"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedIndexWithMeta creates a real SQLite index DB at the hash-named directory
// for (projectPath, model). When recordPath is true it records project_path in
// project_meta (the modern layout exercised by the metadata-scan code path);
// when false it omits it, mimicking indexes written by older binaries.
func seedIndexWithMeta(t *testing.T, projectPath, model string, recordPath bool) string {
	t.Helper()
	dbPath := config.DBPathForProject(projectPath, model)
	require.NoError(t, os.MkdirAll(filepath.Dir(dbPath), 0o755))
	s, err := store.New(dbPath, 4)
	require.NoError(t, err)
	if recordPath {
		require.NoError(t, s.SetMeta("project_path", projectPath))
	}
	require.NoError(t, s.Close())
	return filepath.Dir(dbPath)
}

// seedIndex creates an index DB that records project_path.
func seedIndex(t *testing.T, projectPath, model string) string {
	t.Helper()
	return seedIndexWithMeta(t, projectPath, model, true)
}

// seedLegacyIndex creates an index DB with NO project_path metadata. Such DBs
// are fully usable by the system (located by path hash) but invisible to
// project_path-based purge, so scanIndexes classifies them as no-metadata legacy.
func seedLegacyIndex(t *testing.T, projectPath, model string) string {
	t.Helper()
	return seedIndexWithMeta(t, projectPath, model, false)
}

// seedCorruptDir creates a hash directory whose index.db is missing, mimicking
// an unreadable/corrupt index. scanIndexes classifies it as unreadable.
func seedCorruptDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(lumenDataDir(), name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	return dir
}

// runPurgeCmd invokes runPurge with the provided args and optional flag names
// (each set to "true") and returns captured stdout, stderr, and the error.
func runPurgeCmd(t *testing.T, args []string, flags ...string) (stdout, stderr string, err error) {
	t.Helper()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	registerPurgeFlags(cmd)
	for _, f := range flags {
		require.NoError(t, cmd.Flags().Set(f, "true"))
	}
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	err = runPurge(cmd, args)
	return outBuf.String(), errBuf.String(), err
}

func TestPurge_NoArgs_PurgesCwdOnly(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	projectA := filepath.Join(tmp, "projectA")
	projectB := filepath.Join(tmp, "projectB")
	require.NoError(t, os.MkdirAll(projectA, 0o755))
	require.NoError(t, os.MkdirAll(projectB, 0o755))
	runGit(t, projectA, "init")
	runGit(t, projectB, "init")

	hashDirA := seedIndex(t, projectA, embedder.DefaultModel)
	hashDirB := seedIndex(t, projectB, embedder.DefaultModel)

	// No-args default operates on the current working directory.
	t.Chdir(projectA)

	_, _, err := runPurgeCmd(t, nil)
	require.NoError(t, err)

	_, err = os.Stat(hashDirA)
	assert.True(t, os.IsNotExist(err), "cwd project A hash dir should be gone")
	_, err = os.Stat(hashDirB)
	assert.NoError(t, err, "project B hash dir should be untouched")
}

func TestPurge_NoArgs_CwdWithoutIndex_ReportsNone(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	empty := filepath.Join(tmp, "empty")
	require.NoError(t, os.MkdirAll(empty, 0o755))
	t.Chdir(empty)

	_, stderrOut, err := runPurgeCmd(t, nil)
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(stderrOut), "no index found")
}

func TestPurge_All_RemovesEverything(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	seedIndex(t, "/project/a", embedder.DefaultModel)
	hashDirB := seedIndex(t, "/project/b", embedder.DefaultModel)

	// A legacy index dir with no project_path metadata must also be wiped.
	lumenRoot := filepath.Join(tmp, "lumen")
	legacyDir := filepath.Join(lumenRoot, "legacyhash")
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))

	_, stderrOut, err := runPurgeCmd(t, nil, flagAll)
	require.NoError(t, err)
	assert.Contains(t, stderrOut, "Removed all index data")
	assert.Contains(t, stderrOut, hashDirB, "should log each removed index dir")

	_, err = os.Stat(lumenRoot)
	assert.True(t, os.IsNotExist(err), "lumen data dir should be gone, got err=%v", err)
}

func TestPurge_SinglePath_RemovesOnlyThatProject(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	// Two independent projects with seeded indexes.
	projectA := filepath.Join(tmp, "projectA")
	projectB := filepath.Join(tmp, "projectB")
	require.NoError(t, os.MkdirAll(projectA, 0o755))
	require.NoError(t, os.MkdirAll(projectB, 0o755))
	runGit(t, projectA, "init")
	runGit(t, projectB, "init")

	hashDirA := seedIndex(t, projectA, embedder.DefaultModel)
	hashDirB := seedIndex(t, projectB, embedder.DefaultModel)

	_, stderrOut, err := runPurgeCmd(t, []string{projectA})
	require.NoError(t, err)
	assert.Contains(t, stderrOut, projectA, "should log the purged project path")

	_, err = os.Stat(hashDirA)
	assert.True(t, os.IsNotExist(err), "project A hash dir should be gone")
	_, err = os.Stat(hashDirB)
	assert.NoError(t, err, "project B hash dir should be untouched")
}

func TestPurge_PathInsideGitRepo_ResolvesToGitRoot(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	repoDir := filepath.Join(tmp, "repo")
	subDir := filepath.Join(repoDir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	runGit(t, repoDir, "init")

	hashDir := seedIndex(t, repoDir, embedder.DefaultModel)

	_, _, err := runPurgeCmd(t, []string{subDir})
	require.NoError(t, err)

	_, err = os.Stat(hashDir)
	assert.True(t, os.IsNotExist(err), "git-root hash dir should be removed when passing a subdirectory")
}

func TestPurge_PathWithAncestorIndex_ResolvesToAncestor(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	grandparent := filepath.Join(tmp, "workspace")
	child := filepath.Join(grandparent, "a", "b")
	require.NoError(t, os.MkdirAll(child, 0o755))

	hashDir := seedIndex(t, grandparent, embedder.DefaultModel)

	_, _, err := runPurgeCmd(t, []string{child})
	require.NoError(t, err)

	_, err = os.Stat(hashDir)
	assert.True(t, os.IsNotExist(err), "ancestor hash dir should be removed")
}

func TestPurge_PathWithoutIndex_ReportsNoneAndExitsZero(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	// Path exists but has no index anywhere up the tree.
	dir := filepath.Join(tmp, "empty")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	_, stderrOut, err := runPurgeCmd(t, []string{dir})
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(stderrOut), "no index found")
}

func TestPurge_MultiplePaths_RemovesEach(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	projectA := filepath.Join(tmp, "projectA")
	projectB := filepath.Join(tmp, "projectB")
	require.NoError(t, os.MkdirAll(projectA, 0o755))
	require.NoError(t, os.MkdirAll(projectB, 0o755))
	runGit(t, projectA, "init")
	runGit(t, projectB, "init")

	hashDirA := seedIndex(t, projectA, embedder.DefaultModel)
	hashDirB := seedIndex(t, projectB, embedder.DefaultModel)

	_, _, err := runPurgeCmd(t, []string{projectA, projectB})
	require.NoError(t, err)

	_, err = os.Stat(hashDirA)
	assert.True(t, os.IsNotExist(err), "project A hash dir should be gone")
	_, err = os.Stat(hashDirB)
	assert.True(t, os.IsNotExist(err), "project B hash dir should be gone")
}

func TestPurge_UnknownModelName_StillPurgedByStoredMetadata(t *testing.T) {
	// Indexes created with custom or aliased model names (not in KnownModels)
	// must still be purged — the match is by stored project_path, not by
	// enumerating known models and recomputing the hash.
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	project := filepath.Join(tmp, "project")
	require.NoError(t, os.MkdirAll(project, 0o755))
	runGit(t, project, "init")

	hashDir := seedIndex(t, project, "some-custom-alias-model-not-in-registry")

	_, _, err := runPurgeCmd(t, []string{project})
	require.NoError(t, err)

	_, err = os.Stat(hashDir)
	assert.True(t, os.IsNotExist(err), "custom-model hash dir should be removed via stored project_path")
}

func TestPurge_MultiplePaths_MixedHitsAndMisses(t *testing.T) {
	tmp := resolvedTempDir(t)
	t.Setenv("XDG_DATA_HOME", tmp)

	projectA := filepath.Join(tmp, "projectA")
	empty := filepath.Join(tmp, "empty")
	require.NoError(t, os.MkdirAll(projectA, 0o755))
	require.NoError(t, os.MkdirAll(empty, 0o755))
	runGit(t, projectA, "init")

	hashDirA := seedIndex(t, projectA, embedder.DefaultModel)

	_, stderrOut, err := runPurgeCmd(t, []string{projectA, empty})
	require.NoError(t, err)

	_, err = os.Stat(hashDirA)
	assert.True(t, os.IsNotExist(err), "project A hash dir should be gone")
	assert.Contains(t, strings.ToLower(stderrOut), "no index found", "miss should be reported")
}

// TestPurge_FlagBehavior consolidates the flag-validation and --missing/--legacy
// behaviors into one table-driven test. Each case sets up its own data dir state
// via setup (returning named paths for postCheck) and asserts error, stderr, and
// resulting filesystem state.
func TestPurge_FlagBehavior(t *testing.T) {
	cases := []struct {
		name               string
		flags              []string
		args               []string
		setup              func(t *testing.T, tmp string) map[string]string
		wantErr            bool
		wantErrContains    []string
		wantStderrContains string
		postCheck          func(t *testing.T, paths map[string]string, stderrOut string)
	}{
		{
			name:            "all with paths errors",
			flags:           []string{flagAll},
			args:            []string{"/some/path"},
			wantErr:         true,
			wantErrContains: []string{"--all"},
		},
		{
			name:            "all and missing errors",
			flags:           []string{flagAll, flagMissing},
			wantErr:         true,
			wantErrContains: []string{"--all", "--missing"},
		},
		{
			name:            "missing with paths errors",
			flags:           []string{flagMissing},
			args:            []string{"/some/path"},
			wantErr:         true,
			wantErrContains: []string{"--missing"},
		},
		{
			name:            "dry-run without missing errors",
			flags:           []string{flagDryRun},
			wantErr:         true,
			wantErrContains: []string{"--dry-run is only valid with --missing"},
		},
		{
			name:            "legacy with all errors",
			flags:           []string{flagLegacy, flagAll},
			wantErr:         true,
			wantErrContains: []string{"--legacy"},
		},
		{
			name:            "legacy with paths errors",
			flags:           []string{flagLegacy},
			args:            []string{"/some/path"},
			wantErr:         true,
			wantErrContains: []string{"--legacy"},
		},
		{
			name:  "missing removes deleted folders and unreadable dirs",
			flags: []string{flagMissing},
			setup: func(t *testing.T, tmp string) map[string]string {
				gone := filepath.Join(tmp, "gone")
				alive := filepath.Join(tmp, "alive")
				require.NoError(t, os.MkdirAll(gone, 0o755))
				require.NoError(t, os.MkdirAll(alive, 0o755))
				hashGone := seedIndex(t, gone, embedder.DefaultModel)
				hashAlive := seedIndex(t, alive, embedder.DefaultModel)
				hashCorrupt := seedCorruptDir(t, "corrupthash")
				require.NoError(t, os.RemoveAll(gone)) // delete one project's folder
				return map[string]string{"gone": hashGone, "alive": hashAlive, "corrupt": hashCorrupt}
			},
			postCheck: func(t *testing.T, paths map[string]string, stderrOut string) {
				assert.Contains(t, stderrOut, paths["gone"], "should log the removed index dir")
				_, err := os.Stat(paths["gone"])
				assert.True(t, os.IsNotExist(err), "index for deleted folder should be removed")
				_, err = os.Stat(paths["corrupt"])
				assert.True(t, os.IsNotExist(err), "unreadable index dir should be removed")
				_, err = os.Stat(paths["alive"])
				assert.NoError(t, err, "index for existing folder should be kept")
			},
		},
		{
			name:  "missing keeps existing folders",
			flags: []string{flagMissing},
			setup: func(t *testing.T, tmp string) map[string]string {
				alive := filepath.Join(tmp, "alive")
				require.NoError(t, os.MkdirAll(alive, 0o755))
				return map[string]string{"alive": seedIndex(t, alive, embedder.DefaultModel)}
			},
			postCheck: func(t *testing.T, paths map[string]string, stderrOut string) {
				_, err := os.Stat(paths["alive"])
				assert.NoError(t, err, "index for existing folder should be kept")
			},
		},
		{
			name:               "dry-run missing deletes nothing",
			flags:              []string{flagMissing, flagDryRun},
			wantStderrContains: "Would remove",
			setup: func(t *testing.T, tmp string) map[string]string {
				gone := filepath.Join(tmp, "gone")
				require.NoError(t, os.MkdirAll(gone, 0o755))
				hashGone := seedIndex(t, gone, embedder.DefaultModel)
				require.NoError(t, os.RemoveAll(gone))
				return map[string]string{"gone": hashGone}
			},
			postCheck: func(t *testing.T, paths map[string]string, stderrOut string) {
				assert.Contains(t, stderrOut, paths["gone"])
				_, err := os.Stat(paths["gone"])
				assert.NoError(t, err, "dry-run must not delete the index dir")
			},
		},
		{
			name:  "legacy removes metadata-less and unreadable indexes",
			flags: []string{flagLegacy},
			setup: func(t *testing.T, tmp string) map[string]string {
				alive := filepath.Join(tmp, "alive")
				require.NoError(t, os.MkdirAll(alive, 0o755))
				hashAlive := seedIndex(t, alive, embedder.DefaultModel)
				hashLegacy := seedLegacyIndex(t, filepath.Join(tmp, "legacyproj"), embedder.DefaultModel)
				hashCorrupt := seedCorruptDir(t, "corrupthash")
				return map[string]string{"alive": hashAlive, "legacy": hashLegacy, "corrupt": hashCorrupt}
			},
			postCheck: func(t *testing.T, paths map[string]string, stderrOut string) {
				assert.Contains(t, stderrOut, paths["legacy"], "should log the removed legacy dir")
				_, err := os.Stat(paths["legacy"])
				assert.True(t, os.IsNotExist(err), "legacy (no-metadata) index dir should be removed")
				_, err = os.Stat(paths["corrupt"])
				assert.True(t, os.IsNotExist(err), "unreadable index dir should be removed")
				_, err = os.Stat(paths["alive"])
				assert.NoError(t, err, "index with project_path metadata should be kept")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := resolvedTempDir(t)
			t.Setenv("XDG_DATA_HOME", tmp)

			var paths map[string]string
			if tc.setup != nil {
				paths = tc.setup(t, tmp)
			}

			_, stderrOut, err := runPurgeCmd(t, tc.args, tc.flags...)

			if tc.wantErr {
				require.Error(t, err)
				for _, want := range tc.wantErrContains {
					assert.Contains(t, err.Error(), want)
				}
				return
			}
			require.NoError(t, err)
			if tc.wantStderrContains != "" {
				assert.Contains(t, stderrOut, tc.wantStderrContains)
			}
			if tc.postCheck != nil {
				tc.postCheck(t, paths, stderrOut)
			}
		})
	}
}
