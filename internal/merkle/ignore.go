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
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	ignore "github.com/sabhiram/go-gitignore"
)

// SkipFiles is the canonical set of file basenames that are always skipped
// during tree building. These are typically large generated or binary lock files
// that add noise without indexing value.
var SkipFiles = map[string]bool{
	// JS/Node package managers
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"bun.lock":          true, "bun.lockb": true,
	// Go
	"go.sum": true,
	// PHP
	"composer.lock": true,
	// Python
	"poetry.lock": true, "Pipfile.lock": true,
	// Ruby
	"Gemfile.lock": true,
	// Rust
	"Cargo.lock": true,
	// Swift
	"Package.resolved": true,
	// Dart/Flutter
	"pubspec.lock": true,
	// Elixir
	"mix.lock": true,
	// Nix
	"flake.lock": true,
	// .NET/NuGet
	"packages.lock.json": true,
}

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

// dirIgnore holds compiled matchers for a single directory level.
type dirIgnore struct {
	gitignore     *ignore.GitIgnore // from .gitignore
	lumenIgnore   *ignore.GitIgnore // from .lumenignore
	gitattributes *ignore.GitIgnore // linguist-generated patterns from .gitattributes
}

// IgnoreTree manages hierarchical ignore rules, lazily loading them as
// filepath.WalkDir traverses directories. It is safe for concurrent use.
type IgnoreTree struct {
	rootDir       string
	extSet        map[string]bool
	extraSkipDirs map[string]bool // relative paths of directories to always skip

	globalIgnore *ignore.GitIgnore // from core.excludesFile

	mu   sync.Mutex
	dirs map[string]*dirIgnore // keyed by relative dir path ("" = root)
}

// NewIgnoreTree creates an IgnoreTree rooted at rootDir that filters by the
// given file extensions. Root-level ignore files are loaded eagerly.
func NewIgnoreTree(rootDir string, exts []string) *IgnoreTree {
	extSet := make(map[string]bool, len(exts))
	for _, ext := range exts {
		extSet[ext] = true
	}
	t := &IgnoreTree{
		rootDir: rootDir,
		extSet:  extSet,
		dirs:    make(map[string]*dirIgnore),
	}
	t.loadDir("") // eagerly load root
	if globalPath := globalGitignorePath(); globalPath != "" {
		if gi, err := ignore.CompileIgnoreFile(globalPath); err == nil {
			t.globalIgnore = gi
		}
	}
	return t
}

// loadDir loads ignore files for a directory (relative path). Must be called
// with t.mu held.
func (t *IgnoreTree) loadDir(dirRel string) *dirIgnore {
	if d, ok := t.dirs[dirRel]; ok {
		return d
	}

	absDir := filepath.Join(t.rootDir, dirRel)
	d := &dirIgnore{}

	if gi, err := ignore.CompileIgnoreFile(filepath.Join(absDir, ".gitignore")); err == nil {
		d.gitignore = gi
	}
	if ai, err := ignore.CompileIgnoreFile(filepath.Join(absDir, ".lumenignore")); err == nil {
		d.lumenIgnore = ai
	}
	if ga := parseLinguistExcluded(filepath.Join(absDir, ".gitattributes")); ga != nil {
		d.gitattributes = ga
	}

	t.dirs[dirRel] = d
	return d
}

// shouldSkip implements SkipFunc. It checks the six filtering layers:
// 1. SkipDirs, 2. SkipFiles, 3. .gitignore, 4. .lumenignore, 5. .gitattributes, 6. extension.
func (t *IgnoreTree) shouldSkip(relPath string, isDir bool) bool {
	relPath = filepath.Clean(relPath)
	slashRelPath := filepath.ToSlash(relPath)
	base := filepath.Base(relPath)
	if isDir && SkipDirs[base] {
		return true
	}
	if isDir && t.extraSkipDirs[relPath] {
		return true
	}
	if !isDir && SkipFiles[base] {
		return true
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.globalIgnore != nil && t.globalIgnore.MatchesPath(slashRelPath) {
		return true
	}

	parentDir := filepath.Dir(relPath)
	ancestors := ancestorDirs(parentDir)
	for _, anc := range ancestors {
		if t.checkIgnoreRules(relPath, anc, isDir) {
			return true
		}
	}

	return !isDir && !t.extSet[filepath.Ext(relPath)]
}

func (t *IgnoreTree) checkIgnoreRules(relPath, anc string, isDir bool) bool {
	d := t.loadDir(anc)
	pathFromAnc := getPathFromAncestor(relPath, anc)
	matchPath := pathFromAnc
	if isDir {
		matchPath = pathFromAnc + "/"
	}

	if d.gitignore != nil && d.gitignore.MatchesPath(matchPath) {
		return true
	}
	if d.lumenIgnore != nil && d.lumenIgnore.MatchesPath(matchPath) {
		return true
	}
	if !isDir && d.gitattributes != nil && d.gitattributes.MatchesPath(pathFromAnc) {
		return true
	}
	return false
}

func getPathFromAncestor(relPath, anc string) string {
	if anc == "" {
		return filepath.ToSlash(relPath)
	}
	pathFromAnc, _ := filepath.Rel(anc, relPath)
	return filepath.ToSlash(pathFromAnc)
}

// ancestorDirs returns the directory hierarchy from root ("") to dirRel.
// For "a/b/c" it returns ["", "a", "a/b", "a/b/c"].
// For "." or "" it returns [""].
func ancestorDirs(dirRel string) []string {
	if dirRel == "." || dirRel == "" {
		return []string{""}
	}
	parts := strings.Split(filepath.ToSlash(dirRel), "/")
	result := make([]string, 0, len(parts)+1)
	result = append(result, "")
	for i := range parts {
		result = append(result, filepath.Join(parts[:i+1]...))
	}
	return result
}

// globalGitignorePath returns the path to the global gitignore file,
// or "" if not configured or not found.
func globalGitignorePath() string {
	out, err := exec.Command("git", "config", "--global", "core.excludesFile").Output()
	if err == nil {
		path := strings.TrimSpace(string(out))
		if path != "" {
			if strings.HasPrefix(path, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					path = filepath.Join(home, path[2:])
				}
			}
			return path
		}
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "git", "ignore")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "git", "ignore")
	}
	return ""
}

// parseLinguistExcluded reads a .gitattributes file and returns a compiled
// matcher for patterns marked with linguist-generated, linguist-generated=true,
// linguist-vendored, or linguist-vendored=true.
// Returns nil if the file doesn't exist or contains no such patterns.
func parseLinguistExcluded(path string) *ignore.GitIgnore {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}

		// Split into pattern and attributes
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pattern := fields[0]
		for _, attr := range fields[1:] {
			if attr == "linguist-generated" || attr == "linguist-generated=true" ||
				attr == "linguist-vendored" || attr == "linguist-vendored=true" {
				patterns = append(patterns, pattern)
				break
			}
		}
	}

	if len(patterns) == 0 {
		return nil
	}
	return ignore.CompileIgnoreLines(patterns...)
}

// refusedRoots are filesystem roots that are never appropriate as a Lumen
// index root regardless of configuration. Indexing them would walk huge,
// machine-managed trees and on macOS trigger TCC prompts for protected
// folders. The user's $HOME is added at lookup time via os.UserHomeDir.
//
// Entries cover Unix/macOS and Windows. Keys are compared against
// filepath.Clean(dir), which is platform-dependent (forward slashes on
// Unix, backslashes on Windows), so Windows entries only match on Windows
// and vice versa — harmless to include both.
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
//  1. Hardcoded refusal list — filesystem roots ($HOME, /, /Users, /tmp,
//     /var, /etc, /usr, /Applications, /Library and macOS /private/* twins)
//     that should never be project roots regardless of user config.
//  2. .lumenignore probe — if dir/.lumenignore contains patterns broad
//     enough to match every file (e.g. "**", "**/*", "*"), the user has
//     declared the directory un-indexable at its boundary.
//
// Without these guards the indexer walks the entire tree, ignores every
// file, produces an empty index, and on macOS triggers TCC prompts along
// the way. Callers (findAncestorIndex, `lumen index`) use this to refuse
// such roots upfront.
func IsRootUnindexable(dir string) (bool, string) {
	// Check both the cleaned input and the symlink-resolved form against the
	// refusal map. On macOS, /etc → /private/etc and /tmp → /private/tmp via
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
	if home, err := os.UserHomeDir(); err == nil {
		homeClean := filepath.Clean(home)
		homeResolved := resolvePath(home)
		if homeClean == clean || homeClean == resolved || homeResolved == clean || homeResolved == resolved {
			return true, "user home directory"
		}
	}

	gi, err := ignore.CompileIgnoreFile(filepath.Join(dir, ".lumenignore"))
	if err != nil || gi == nil {
		return false, ""
	}
	// Probe with both a root-level entry and a nested entry using long random
	// sentinels that no realistic specific pattern would match. Patterns like
	// "*" alone match only the root probe (gitignore `*` doesn't cross `/`);
	// patterns like "**", "**/*", "*/*" match both probes — which is what we
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

// MakeSkipWithExtra is like MakeSkip but also skips directories whose relative
// paths are listed in extraSkipDirs. This is used to exclude git worktrees
// that are checked out inside the project root directory.
func MakeSkipWithExtra(rootDir string, exts []string, extraSkipDirs []string) SkipFunc {
	tree := NewIgnoreTree(rootDir, exts)
	if len(extraSkipDirs) > 0 {
		tree.extraSkipDirs = make(map[string]bool, len(extraSkipDirs))
		for _, p := range extraSkipDirs {
			tree.extraSkipDirs[filepath.Clean(p)] = true
		}
	}
	return tree.shouldSkip
}

// MakeSkip returns a SkipFunc that layers six filters:
//  1. SkipDirs — map lookup on directory basename (cheapest check)
//  2. SkipFiles — map lookup on file basename (lock files and other noise)
//  3. .gitignore — root + nested, hierarchical matching
//  4. .lumenignore — root + nested, hierarchical matching
//  5. .gitattributes — linguist-generated patterns, root + nested
//  6. Extension filter — only index files whose extension is in exts
//
// Ignore files are discovered lazily as the walk proceeds.
func MakeSkip(rootDir string, exts []string) SkipFunc {
	tree := NewIgnoreTree(rootDir, exts)
	return tree.shouldSkip
}
