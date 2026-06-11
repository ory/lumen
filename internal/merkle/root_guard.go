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
	"os"
	"path/filepath"
	"runtime"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// refusedRoots are filesystem roots that are never appropriate as a Lumen
// index root regardless of configuration. Indexing them would walk huge,
// machine-managed trees and on macOS trigger TCC prompts for protected
// folders. The user's $HOME is added at lookup time via os.UserHomeDir.
//
// Entries cover Unix/macOS and Windows. Keys are compared against
// filepath.Clean(dir), which is platform-dependent (forward slashes on
// Unix, backslashes on Windows), so Windows entries only match on Windows
// and vice versa -- harmless to include both.
var refusedRoots = map[string]bool{
	// Unix / macOS
	"/":             true,
	"/Users":        true,
	"/home":         true,
	"/tmp":          true,
	"/private/tmp":  true,
	"/var":          true,
	"/private/var":  true,
	"/etc":          true,
	"/usr":          true,
	"/opt":          true,
	"/Applications": true,
	"/Library":      true,
	"/System":       true,
	// Windows
	`C:\`:                    true,
	`C:\Windows`:             true,
	`C:\Program Files`:       true,
	`C:\Program Files (x86)`: true,
	`C:\Users`:               true,
	`C:\ProgramData`:         true,
}

// refusedRootSubtrees are protected Windows roots whose descendants are also
// machine-managed. Keep broad user containers such as C:\Users exact-only.
var refusedRootSubtrees = []string{
	`C:\Windows`,
	`C:\Program Files`,
	`C:\Program Files (x86)`,
	`C:\ProgramData`,
}

// resolvePath returns filepath.EvalSymlinks(dir) when it succeeds, falling
// back to filepath.Clean(dir) when it does not (e.g. dir does not exist on
// disk, or a symlink in the chain cannot be resolved). This ensures that a
// symlink to $HOME or a refused root is caught by IsRootUnindexable rather
// than slipping through on the symlink path alone.
func resolvePath(dir string) string {
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		return resolved
	}
	return filepath.Clean(dir)
}

// IsRootUnindexable reports whether dir is unsuitable as a Lumen index root.
// When true, the returned string is a short human-readable reason suitable for
// inclusion in an error message. When false, the reason is empty.
//
// Two checks combine:
//
//  1. Hardcoded refusal list -- filesystem roots ($HOME, /, /Users, /tmp,
//     /var, /etc, /usr, /Applications, /Library and macOS /private/* twins)
//     that should never be project roots regardless of user config.
//  2. .lumenignore probe -- if dir/.lumenignore contains patterns broad
//     enough to match every file (e.g. "**", "**/*", "*"), the user has
//     declared the directory un-indexable at its boundary.
//
// Without these guards the indexer walks the entire tree, ignores every
// file, produces an empty index, and on macOS triggers TCC prompts along
// the way. Callers (findAncestorIndex, `lumen index`) use this to refuse
// such roots upfront.
func IsRootUnindexable(dir string) (bool, string) {
	// Check both the cleaned input and the symlink-resolved form against the
	// refusal map. On macOS, /etc -> /private/etc and /tmp -> /private/tmp via
	// symlinks: resolving lets a user-supplied "/private/etc" still match,
	// while the cleaned-input check keeps "/etc" itself matching.
	clean := filepath.Clean(dir)
	resolved := resolvePath(dir)
	if runtime.GOOS == "windows" && (isWindowsDriveRoot(clean) || isWindowsDriveRoot(resolved)) {
		return true, "windows drive root"
	}
	if refusedRoots[clean] || refusedRoots[resolved] {
		return true, "hardcoded system root"
	}
	if isRefusedRootSubtree(clean) || isRefusedRootSubtree(resolved) {
		return true, "hardcoded system root"
	}
	if home, err := os.UserHomeDir(); err == nil {
		homeClean := filepath.Clean(home)
		homeResolved := resolvePath(home)
		if homeClean == clean || homeClean == resolved || homeResolved == clean || homeResolved == resolved {
			return true, "user home directory"
		}
	}
	if isAgentSessionStoreRoot(clean) || isAgentSessionStoreRoot(resolved) {
		return true, "agent session store"
	}

	gi, err := ignore.CompileIgnoreFile(filepath.Join(dir, ".lumenignore"))
	if err != nil || gi == nil {
		return false, ""
	}
	// Probe with both a root-level entry and a nested entry using long random
	// sentinels that no realistic specific pattern would match. Patterns like
	// "*" alone match only the root probe (gitignore `*` doesn't cross `/`);
	// patterns like "**", "**/*", "*/*" match both probes -- which is what we
	// take as "ignores everything".
	const probeRoot = "lumen-root-probe-X9F2K7M3"
	const probeNested = "lumen-root-probe-X9F2K7M3/L8B4Q1P5R6N2"
	if gi.MatchesPath(probeRoot) && gi.MatchesPath(probeNested) {
		return true, ".lumenignore catch-all pattern"
	}
	return false, ""
}

func isWindowsDriveRoot(path string) bool {
	volume := filepath.VolumeName(path)
	if volume == "" {
		return false
	}
	rest := strings.TrimPrefix(path, volume)
	return rest == `\` || rest == `/` || rest == ""
}

func isRefusedRootSubtree(path string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	for _, root := range refusedRootSubtrees {
		if sameOrUnderRoot(path, root) {
			return true
		}
	}
	return false
}

func sameOrUnderRoot(path, root string) bool {
	path = strings.ToLower(filepath.Clean(path))
	root = strings.ToLower(filepath.Clean(root))
	if path == root {
		return true
	}
	root = strings.TrimRight(root, `\/`)
	return strings.HasPrefix(path, root+string(filepath.Separator))
}

func isAgentSessionStoreRoot(path string) bool {
	parts := strings.FieldsFunc(filepath.Clean(path), func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for i := 0; i+1 < len(parts); i++ {
		parent := strings.ToLower(parts[i])
		child := strings.ToLower(parts[i+1])
		if parent == ".claude" && child == "projects" {
			return true
		}
		if parent == ".codex" && (child == "sessions" || child == "history") {
			return true
		}
	}
	return false
}
