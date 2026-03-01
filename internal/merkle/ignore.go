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

package merkle

import (
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

// SkipDirs is the canonical set of directory basenames that are always skipped
// during tree building, regardless of .gitignore rules.
var SkipDirs = map[string]bool{
	// VCS
	".git": true, ".hg": true, ".svn": true,
	// Go
	"vendor": true,
	// JS/Node
	"node_modules": true, "bower_components": true, ".next": true, ".nuxt": true,
	// Python
	"__pycache__": true, ".venv": true, "venv": true, ".tox": true, ".eggs": true,
	// Ruby
	".bundle": true,
	// Rust
	"target": true,
	// Java
	".gradle": true,
	// Elixir/Erlang
	"_build": true, "deps": true,
	// General build/cache
	"dist": true, ".cache": true, ".output": true, ".build": true,
	// IDE
	".idea": true, ".vscode": true,
	// Test fixtures (Go convention)
	"testdata": true,
}

// MakeSkip returns a SkipFunc that layers three filters:
//  1. SkipDirs — map lookup on directory basename (cheapest check)
//  2. .gitignore patterns from rootDir/.gitignore (if the file exists)
//  3. Extension filter — only index files whose extension is in exts
//
// If no .gitignore exists at rootDir, the gitignore layer is silently skipped.
func MakeSkip(rootDir string, exts []string) SkipFunc {
	extSet := make(map[string]bool, len(exts))
	for _, ext := range exts {
		extSet[ext] = true
	}

	gitignorePath := filepath.Join(rootDir, ".gitignore")
	gi, _ := ignore.CompileIgnoreFile(gitignorePath) // nil if file doesn't exist

	return func(relPath string, isDir bool) bool {
		base := filepath.Base(relPath)
		if isDir {
			if SkipDirs[base] {
				return true
			}
			if gi != nil && gi.MatchesPath(relPath+"/") {
				return true
			}
			return false
		}
		if gi != nil && gi.MatchesPath(relPath) {
			return true
		}
		return !extSet[filepath.Ext(relPath)]
	}
}
