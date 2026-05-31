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
	"strings"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/git"
	"github.com/ory/lumen/internal/store"
	"github.com/spf13/cobra"
)

const (
	flagAll     = "all"
	flagMissing = "missing"
	flagLegacy  = "legacy"
	flagDryRun  = "dry-run"
)

func init() {
	registerPurgeFlags(purgeCmd)
	rootCmd.AddCommand(purgeCmd)
}

// registerPurgeFlags declares the purge command's flags. Shared by init and
// the test helper so tests exercise the real flag set.
func registerPurgeFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(flagAll, false, "Remove every index under the data directory")
	cmd.Flags().Bool(flagMissing, false, "Remove indexes whose project folder no longer exists")
	cmd.Flags().Bool(flagLegacy, false, "Remove only legacy/unreadable indexes lacking project_path metadata")
	cmd.Flags().Bool(flagDryRun, false, "With --missing, list what would be removed without deleting")
}

// lumenDataDir returns the directory holding all lumen index databases.
func lumenDataDir() string {
	return filepath.Join(config.XDGDataDir(), "lumen")
}

var purgeCmd = &cobra.Command{
	Use:   "purge [path...]",
	Short: "Remove lumen index data",
	Long: `Deletes lumen index databases under ~/.local/share/lumen/.

With no arguments, removes only the index for the current working directory's
project (the path is normalized to its git root first).

With one or more paths, removes the index directories associated with those
projects. Each path is normalized to its git root, then matched against the
project_path recorded inside each index database, so switching embedding models
or using custom models never leaves orphan indexes.

  --all       Remove every index (irreversible — all indexes will be rebuilt on
              the next search). Also clears legacy indexes created by older
              binaries that did not record project_path.
  --missing   Remove every index whose recorded project folder no longer exists
              on disk (only deletes a project index when its folder is confirmed
              missing), plus any unreadable/corrupt index directories.
  --legacy    Remove only legacy indexes created by older binaries that did not
              record project_path, plus any unreadable/corrupt index directories.
              Legacy indexes are still usable by the system (located by path
              hash) but invisible to path- and --missing-based purge. Cannot be
              combined with other flags or paths.
  --dry-run   With --missing, list what would be removed without deleting.

Note: a concurrently running indexer for a purged project may log a write
error and exit; re-run "lumen index" afterwards to rebuild.`,
	Args: cobra.ArbitraryArgs,
	RunE: runPurge,
}

func runPurge(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool(flagAll)
	missing, _ := cmd.Flags().GetBool(flagMissing)
	legacy, _ := cmd.Flags().GetBool(flagLegacy)
	dryRun, _ := cmd.Flags().GetBool(flagDryRun)

	if err := validatePurgeFlags(all, missing, legacy, dryRun, len(args)); err != nil {
		return err
	}

	stderr := cmd.ErrOrStderr()
	stdout := cmd.OutOrStdout()

	switch {
	case all:
		return purgeAll(stderr)
	case legacy:
		return purgeLegacy(stderr, stdout)
	case missing:
		return purgeMissing(stderr, stdout, dryRun)
	default:
		if len(args) == 0 {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determine working directory: %w", err)
			}
			args = []string{cwd}
		}
		return purgeProjects(stderr, stdout, args)
	}
}

// validatePurgeFlags enforces mutual exclusivity of the purge modes. --all,
// --missing, and --legacy are the three exclusive whole-dataset modes; explicit
// paths select the default per-project mode and combine with none of them.
func validatePurgeFlags(all, missing, legacy, dryRun bool, nArgs int) error {
	modes := 0
	for _, set := range []bool{all, missing, legacy} {
		if set {
			modes++
		}
	}
	if modes > 1 {
		return fmt.Errorf("--all, --missing, and --legacy are mutually exclusive")
	}
	if modes > 0 && nArgs > 0 {
		return fmt.Errorf("--all, --missing, and --legacy cannot be combined with explicit paths")
	}
	if dryRun && !missing {
		return fmt.Errorf("--dry-run is only valid with --missing")
	}
	return nil
}

// purgeAll removes the entire lumen data directory. This is unconditional by
// design: --all must wipe everything regardless of whether individual indexes
// can be scanned or read. The pre-delete scan is best-effort and used only for
// per-index logging — its error is deliberately ignored, since a corrupt or
// unreadable index is exactly the kind of state --all exists to clear.
func purgeAll(stderr io.Writer) error {
	dataDir := lumenDataDir()

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

	// Log each index directory before wiping, matching the per-index logging
	// used by the other purge modes. Dirs without project_path metadata and
	// unreadable dirs are logged by path alone.
	indexMap, noMeta, unreadable, _ := scanIndexes(dataDir)
	for projectPath, hashDirs := range indexMap {
		for _, hashDir := range hashDirs {
			_, _ = fmt.Fprintf(stderr, "Removed %s (%s)\n", hashDir, projectPath)
		}
	}
	for _, dirs := range [][]string{noMeta, unreadable} {
		for _, hashDir := range dirs {
			_, _ = fmt.Fprintf(stderr, "Removed %s\n", hashDir)
		}
	}

	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("remove index data: %w", err)
	}
	_, _ = fmt.Fprintf(stderr, "Removed all index data (%s)\n", dataDir)
	return nil
}

func purgeProjects(stderr, stdout io.Writer, args []string) error {
	indexMap, _, _, err := scanIndexes(lumenDataDir())
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

func purgeMissing(stderr, stdout io.Writer, dryRun bool) error {
	indexMap, _, unreadable, err := scanIndexes(lumenDataDir())
	if err != nil {
		return err
	}

	verb := "Removed"
	if dryRun {
		verb = "Would remove"
	}

	remove := func(hashDir, reason string) error {
		if !dryRun {
			if err := os.RemoveAll(hashDir); err != nil {
				return fmt.Errorf("remove %s: %w", hashDir, err)
			}
		}
		_, _ = fmt.Fprintf(stderr, "%s %s (%s)\n", verb, hashDir, reason)
		return nil
	}

	removed := 0
	for projectPath, hashDirs := range indexMap {
		if _, statErr := os.Stat(projectPath); statErr == nil {
			continue // folder still exists — keep the index
		} else if !os.IsNotExist(statErr) {
			// Conservative: any error other than "not exist" must never delete.
			_, _ = fmt.Fprintf(stderr, "Skipping %s: cannot stat (%v)\n", projectPath, statErr)
			continue
		}
		for _, hashDir := range hashDirs {
			if err := remove(hashDir, projectPath); err != nil {
				return err
			}
			removed++
		}
	}

	// Unreadable/corrupt index dirs have no folder mapping and can never be
	// served or rebuilt in place, so --missing clears them too.
	for _, hashDir := range unreadable {
		if err := remove(hashDir, "unreadable"); err != nil {
			return err
		}
		removed++
	}

	_, _ = fmt.Fprintf(stdout, "%s %d index director%s (missing folders and unreadable indexes).\n", verb, removed, pluralY(removed))
	return nil
}

// purgeLegacy removes legacy hash directories: readable DBs that do not record
// project_path (still usable by the system, located by path hash, but invisible
// to path- and --missing-based purge) plus unreadable/corrupt directories.
// --legacy is the explicit way to clear both.
func purgeLegacy(stderr, stdout io.Writer) error {
	_, noMeta, unreadable, err := scanIndexes(lumenDataDir())
	if err != nil {
		return err
	}

	removed := 0
	for _, dirs := range [][]string{noMeta, unreadable} {
		for _, hashDir := range dirs {
			if err := os.RemoveAll(hashDir); err != nil {
				return fmt.Errorf("remove %s: %w", hashDir, err)
			}
			_, _ = fmt.Fprintf(stderr, "Removed %s\n", hashDir)
			removed++
		}
	}

	_, _ = fmt.Fprintf(stdout, "Removed %d legacy index director%s.\n", removed, pluralY(removed))
	return nil
}

// scanIndexes walks dataDir (one level deep) and classifies each hash directory
// into three buckets:
//
//   - indexMap: stored project_path → hash directories (readable DB with
//     project_path metadata).
//   - noMeta: readable DBs that do not record project_path (created by older
//     binaries). These remain usable by the system — they are located by path
//     hash, not by metadata — so path- and --missing-based purge leave them
//     alone; only --all and --legacy remove them.
//   - unreadable: directories whose index.db is missing or corrupt. These can
//     never be served or rebuilt in place, so every purge mode that scans
//     (--missing and --legacy) clears them.
//
// Path-based purge ignores the non-indexMap buckets so a single broken index
// never blocks purging of others.
func scanIndexes(dataDir string) (indexMap map[string][]string, noMeta, unreadable []string, err error) {
	indexMap = make(map[string][]string)
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return indexMap, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("read data dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		hashDir := filepath.Join(dataDir, entry.Name())
		dbPath := filepath.Join(hashDir, "index.db")
		stored, readErr := store.ReadMetaAt(dbPath, "project_path")
		switch {
		case readErr != nil:
			unreadable = append(unreadable, hashDir)
		case stored == "":
			noMeta = append(noMeta, hashDir)
		default:
			indexMap[stored] = append(indexMap[stored], hashDir)
		}
	}
	return indexMap, noMeta, unreadable, nil
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
		target = root
		inGitRepo = true
	}

	match := ""
	if _, ok := indexMap[target]; ok {
		match = target
	} else if !inGitRepo {
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

// longestAncestor returns the longest key in indexMap that is either equal to
// target or an ancestor directory of target, or "" if no such key exists.
func longestAncestor(indexMap map[string][]string, target string) string {
	best := ""
	for stored := range indexMap {
		if stored == target || strings.HasPrefix(target, stored+string(filepath.Separator)) {
			if len(stored) > len(best) {
				best = stored
			}
		}
	}
	return best
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
