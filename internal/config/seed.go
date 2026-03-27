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

package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ory/lumen/internal/git"
)

// FindDonorIndex searches sibling git worktrees for an existing lumen index
// DB that uses the same model and IndexVersion. Returns the DB path of the
// most recently modified candidate, or "" if no suitable donor exists.
func FindDonorIndex(projectPath, model string) string {
	return FindDonorIndexBase(XDGDataDir(), projectPath, model)
}

// FindDonorIndexBase is like FindDonorIndex but accepts an explicit data
// directory, making it safe for testing without side effects.
func FindDonorIndexBase(dataDir, projectPath, model string) string {
	worktrees, err := git.ListWorktrees(projectPath)
	if err != nil || len(worktrees) < 2 {
		return ""
	}

	// Resolve symlinks so comparisons work on macOS (/var → /private/var).
	resolvedProject := projectPath
	if r, err := filepath.EvalSymlinks(projectPath); err == nil {
		resolvedProject = r
	}

	// Find which worktree contains projectPath and compute the relative suffix.
	// This handles subdirectory effective roots (e.g., hydra-flake/backoffice)
	// by searching for the same subdirectory in sibling worktrees.
	var myWorktree string
	for _, wt := range worktrees {
		resolved := wt
		if r, err := filepath.EvalSymlinks(wt); err == nil {
			resolved = r
		}
		rel, err := filepath.Rel(resolved, resolvedProject)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		myWorktree = resolved
		break
	}
	if myWorktree == "" {
		return ""
	}

	relSuffix, err := filepath.Rel(myWorktree, resolvedProject)
	if err != nil {
		return ""
	}

	type candidate struct {
		path    string
		modTime int64
	}
	var candidates []candidate

	for _, wt := range worktrees {
		resolved := wt
		if r, err := filepath.EvalSymlinks(wt); err == nil {
			resolved = r
		}
		if resolved == myWorktree {
			continue // skip self
		}
		// Look for a DB at the same relative subdirectory in the sibling worktree.
		siblingProject := filepath.Join(resolved, relSuffix)
		dbPath := DBPathForProjectBase(dataDir, siblingProject, model)
		info, err := os.Stat(dbPath)
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate{path: dbPath, modTime: info.ModTime().UnixNano()})
	}

	if len(candidates) == 0 {
		return ""
	}

	// Pick the most recently modified index.
	slices.SortFunc(candidates, func(a, b candidate) int {
		if a.modTime > b.modTime {
			return -1
		}
		if a.modTime < b.modTime {
			return 1
		}
		return 0
	})

	return candidates[0].path
}
