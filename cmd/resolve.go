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
	"os"
	"path/filepath"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/git"
	"github.com/ory/lumen/internal/merkle"
)

// resolveIndexRoot determines the index root and search path for a given
// combination of path and cwd flags. This is the shared path resolution logic
// used by both the CLI search command and the MCP handler.
//
// The returned indexRoot is the directory whose DB should be opened (the
// project-level index root). searchPath is the directory to use for path-prefix
// filtering (may equal indexRoot when no subdirectory narrowing is needed).
//
// Resolution order:
//  1. Determine the search target (path flag, cwd flag, or working directory).
//  2. Resolve the index root via git root, then ancestor index, then cwd if an
//     index already exists there.
//  3. If cwd is provided and differs from the resolved index root, use it as a
//     preferred root only when an index already exists at that location —
//     matching the MCP guard that avoids creating a brand-new index at a large
//     ancestor tree.
func resolveIndexRoot(pathFlag, cwdFlag, model string) (indexRoot, searchPath string, err error) {
	// Determine the search target path.
	searchPath = pathFlag
	if searchPath == "" {
		if cwdFlag != "" {
			searchPath = cwdFlag
		} else {
			searchPath, err = os.Getwd()
			if err != nil {
				return "", "", fmt.Errorf("get working directory: %w", err)
			}
		}
	}
	searchPath, err = filepath.Abs(searchPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve search path: %w", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(searchPath); resolveErr == nil {
		searchPath = resolved
	}

	// Resolve the index root from the search path: try git root first, then
	// ancestor index walk.
	indexRoot = searchPath
	if root, gitErr := git.RepoRoot(searchPath); gitErr == nil {
		if unindexable, _ := merkle.IsRootUnindexable(root); !unindexable {
			indexRoot = root
		} else if !hasLumenBoundaryFile(searchPath) {
			if ancestor := findAncestorIndex(searchPath, model); ancestor != "" {
				indexRoot = ancestor
			}
		}
	} else if !hasLumenBoundaryFile(searchPath) {
		if ancestor := findAncestorIndex(searchPath, model); ancestor != "" {
			indexRoot = ancestor
		}
	}

	// When cwd is provided and the search path resolved to itself (no git root,
	// no ancestor), check if cwd has an existing index and adopt it.  This
	// matches the MCP guard: only reuse cwd as index root when an index already
	// exists there, to avoid triggering a full scan of a large ancestor tree.
	if cwdFlag != "" {
		absCwd, absErr := filepath.Abs(cwdFlag)
		if absErr != nil {
			return "", "", fmt.Errorf("resolve cwd: %w", absErr)
		}

		if indexRoot == searchPath {
			// searchPath had no git root and no ancestor — check if cwd has an index.
			if _, statErr := os.Stat(config.DBPathForProject(absCwd, model)); statErr == nil {
				indexRoot = absCwd
			}
		}
	}

	return indexRoot, searchPath, nil
}
