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
	"runtime"
	"strings"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/git"
	"github.com/ory/lumen/internal/merkle"
	"github.com/ory/lumen/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(purgeCmd)
}

var purgeCmd = &cobra.Command{
	Use:   "purge [path...]",
	Short: "Remove lumen index data",
	Long: `Deletes lumen index databases under ~/.local/share/lumen/.

With no arguments, removes every index (irreversible — all indexes will be
rebuilt on the next search).

With one or more paths, removes only the index directories associated with
those projects. Each path is normalized to its git root first, then matched
against the project_path recorded inside each index database, so switching
embedding models or using custom models never leaves orphan indexes.

Indexes created by older binaries that did not record project_path cannot be
matched by path; run "lumen purge" with no arguments to wipe those.

Note: a concurrently running indexer for a purged project may log a write
error and exit; re-run "lumen index" afterwards to rebuild.`,
	Args: cobra.ArbitraryArgs,
	RunE: runPurge,
}

func runPurge(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return purgeAll(cmd.ErrOrStderr())
	}
	return purgeProjects(cmd.ErrOrStderr(), cmd.OutOrStdout(), args)
}

func purgeAll(stderr io.Writer) error {
	dataDir := filepath.Join(config.XDGDataDir(), "lumen")

	info, err := os.Stat(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(stderr, "No index data found — nothing to purge.")
			return nil
		}
		return fmt.Errorf("stat data directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dataDir)
	}

	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("remove index data: %w", err)
	}
	_, _ = fmt.Fprintf(stderr, "Removed all index data (%s)\n", dataDir)
	return nil
}

func purgeProjects(stderr, stdout io.Writer, args []string) error {
	dataDir := filepath.Join(config.XDGDataDir(), "lumen")
	indexMap, err := scanIndexes(dataDir)
	if err != nil {
		return err
	}

	seen := make(map[string]bool)
	totalRemoved := 0
	for _, arg := range args {
		removed, err := purgeOneTarget(stderr, indexMap, seen, arg)
		if err != nil {
			return err
		}
		totalRemoved += removed
	}
	_, _ = fmt.Fprintf(stdout, "Removed %d index director%s.\n", totalRemoved, pluralY(totalRemoved))
	return nil
}

// scanIndexes walks dataDir (one level deep) and returns a map of stored
// project_path → list of hash directories for that project. Hash directories
// that can't be read or lack project_path metadata are silently skipped so a
// single broken index never blocks purging of others.
func scanIndexes(dataDir string) (map[string][]string, error) {
	result := make(map[string][]string)
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("read data dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		hashDir := filepath.Join(dataDir, entry.Name())
		dbPath := filepath.Join(hashDir, "index.db")
		stored, err := store.ReadMetaAt(dbPath, "project_path")
		if err != nil || stored == "" {
			continue
		}
		result[stored] = append(result[stored], hashDir)
	}
	return result, nil
}

// purgeOneTarget resolves arg to a project root and removes every hash
// directory whose stored project_path matches. seen tracks hash directories
// already deleted during this invocation so two args resolving to the same
// project are not double-counted.
func purgeOneTarget(stderr io.Writer, indexMap map[string][]string, seen map[string]bool, arg string) (int, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return 0, fmt.Errorf("resolve %q: %w", arg, err)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}

	target := abs
	inGitRepo := false
	if root, err := git.RepoRoot(abs); err == nil {
		if unindexable, _ := merkle.IsRootUnindexable(root); !unindexable {
			target = root
			inGitRepo = true
		}
	}

	match := lookupProjectIndex(indexMap, target)
	if match == "" && !inGitRepo {
		// Non-git fallback: match the deepest stored path that contains the
		// target. Mirrors `findAncestorIndex` semantics used by index/search.
		match = longestAncestor(indexMap, target)
	}

	if match == "" {
		_, _ = fmt.Fprintf(stderr, "No index found for %s.\n", abs)
		return 0, nil
	}

	removed := 0
	for _, hashDir := range indexMap[match] {
		if seen[hashDir] {
			continue
		}
		seen[hashDir] = true
		if err := os.RemoveAll(hashDir); err != nil {
			return removed, fmt.Errorf("remove %s: %w", hashDir, err)
		}
		removed++
	}
	_, _ = fmt.Fprintf(stderr, "Removed %d index director%s for %s.\n",
		removed, pluralY(removed), match)
	return removed, nil
}

func lookupProjectIndex(indexMap map[string][]string, target string) string {
	for stored := range indexMap {
		if sameProjectPath(stored, target) {
			return stored
		}
	}
	return ""
}

// longestAncestor returns the longest key in indexMap that is either equal to
// target or an ancestor directory of target, or "" if no such key exists.
func longestAncestor(indexMap map[string][]string, target string) string {
	best := ""
	for stored := range indexMap {
		if sameProjectPath(stored, target) || pathIsUnder(target, stored) {
			if len(stored) > len(best) {
				best = stored
			}
		}
	}
	return best
}

func sameProjectPath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func pathIsUnder(path, root string) bool {
	root = strings.TrimRight(root, `\/`)
	candidate := root + string(filepath.Separator)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
		candidate = strings.ToLower(candidate)
	}
	return strings.HasPrefix(path, candidate)
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
