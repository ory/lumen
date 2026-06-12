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
	"os"
	"path/filepath"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/merkle"
)

// findAncestorIndex walks up from path's parent directory, checking whether
// any ancestor has an existing lumen index DB on disk. Returns the ancestor
// path if found, or "" if none exists.
//
// This is the CLI equivalent of indexerCache.findEffectiveRoot for one-shot
// commands (index, search, hook) that don't maintain an in-memory cache.
// Only use for non-git directories; callers should try git.RepoRoot first.
//
// Unlike findEffectiveRoot (MCP), this does not cap the walk at the git root
// because callers have already confirmed path is not inside a git repo.
func findAncestorIndex(path, model string) string {
	candidate := filepath.Dir(path)
	for {
		unindexable, _ := merkle.IsRootUnindexable(candidate)
		if !pathCrossesSkipDir(candidate, path) && !unindexable {
			if _, err := os.Stat(config.DBPathForProject(candidate, model)); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
		candidate = parent
	}
	return ""
}

func hasLumenBoundaryFile(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".lumenignore"))
	return err == nil && !info.IsDir()
}
