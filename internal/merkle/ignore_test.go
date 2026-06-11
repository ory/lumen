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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMakeSkip_GitignorePatterns(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "*.log\nbuild/\n")

	skip := MakeSkip(dir, []string{".go", ".log"})

	// .log files should be skipped by gitignore even though extension is allowed
	if !skip("app.log", false) {
		t.Error("expected app.log to be skipped via .gitignore")
	}
	// build/ directory should be skipped by gitignore
	if !skip("build", true) {
		t.Error("expected build/ to be skipped via .gitignore")
	}
	// .go files should pass
	if skip("main.go", false) {
		t.Error("expected main.go to pass")
	}
	// .txt files should be skipped by extension filter (not in exts)
	if !skip("readme.txt", false) {
		t.Error("expected readme.txt to be skipped by extension filter")
	}
}

func TestMakeSkip_NoGitignore(t *testing.T) {
	dir := t.TempDir()
	// No .gitignore created

	skip := MakeSkip(dir, []string{".go", ".py"})

	if skip("main.go", false) {
		t.Error("expected main.go to pass without .gitignore")
	}
	if skip("script.py", false) {
		t.Error("expected script.py to pass without .gitignore")
	}
	if !skip("readme.md", false) {
		t.Error("expected readme.md to be skipped by extension filter")
	}
	// Hardcoded dirs still skipped
	if !skip("node_modules", true) {
		t.Error("expected node_modules to be skipped")
	}
}

func TestMakeSkip_NegationPattern(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "*.gen.go\n!important.gen.go\n")

	skip := MakeSkip(dir, []string{".go"})

	if !skip("foo.gen.go", false) {
		t.Error("expected foo.gen.go to be skipped via .gitignore")
	}
	if skip("important.gen.go", false) {
		t.Error("expected important.gen.go to pass via negation pattern")
	}
	if skip("main.go", false) {
		t.Error("expected main.go to pass")
	}
}

func TestIsRootUnindexable(t *testing.T) {
	// .lumenignore-based scenarios: each case writes the given contents (or no
	// file when ignoreContents is the sentinel "<no file>") to a temp dir.
	const noFile = "<no file>"
	lumenIgnoreCases := []struct {
		name           string
		ignoreContents string
		want           bool
		wantReason     string
	}{
		{"no .lumenignore", noFile, false, ""},
		{"empty .lumenignore", "", false, ""},
		{"specific patterns", "node_modules/\n*.log\nbuild/\n", false, ""},
		{"doublestar matches everything", "**\n", true, ".lumenignore catch-all pattern"},
		{"doublestar-slash-star matches everything", "**/*\n", true, ".lumenignore catch-all pattern"},
		// The user's actual $HOME case — multiple broad patterns combined.
		{"combined catch-all patterns", "*\n**/**\n*/*\n**/*\n", true, ".lumenignore catch-all pattern"},
		// In gitignore semantics, a bare `*` (no `/`) matches any name at any
		// depth — so it ignores every file in the tree. That is a catch-all.
		{"single-star alone", "*\n", true, ".lumenignore catch-all pattern"},
		{"comments and blank lines do not trigger", "# comment\n\n  \n", false, ""},
	}
	for _, tc := range lumenIgnoreCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.ignoreContents != noFile {
				writeFile(t, dir, ".lumenignore", tc.ignoreContents)
			}
			got, reason := IsRootUnindexable(dir)
			if got != tc.want {
				t.Errorf("IsRootUnindexable() = %v, want %v", got, tc.want)
			}
			if reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", reason, tc.wantReason)
			}
		})
	}

	t.Run("hardcoded system paths are refused", func(t *testing.T) {
		// These paths should always be refused as index roots, regardless of
		// whether a .lumenignore exists. Indexing them is never the user's
		// intent and on macOS would trigger TCC prompts.
		//
		// The list is platform-aware: Unix paths only assert refusal when the
		// path exists on disk (so /Applications and /Library don't fail on
		// Linux), and Windows paths only run on Windows.
		unixPaths := []string{
			"/",
			"/Users",
			"/home",
			"/tmp",
			"/private/tmp",
			"/var",
			"/private/var",
			"/etc",
			"/usr",
			"/opt",
			"/Applications",
			"/Library",
			"/System",
		}
		windowsPaths := []string{
			`C:\`,
			`D:\`,
			`C:\Windows`,
			`C:\Program Files`,
			`C:\Program Files (x86)`,
			`C:\Users`,
			`C:\ProgramData`,
		}
		var paths []string
		if runtime.GOOS == "windows" {
			paths = windowsPaths
		} else {
			paths = unixPaths
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err != nil {
				// Skip paths that don't exist on this host — EvalSymlinks
				// will fail there and the fallback to filepath.Clean still
				// catches them, but we keep the assertion focused on real
				// roots that users could actually point lumen at.
				continue
			}
			got, reason := IsRootUnindexable(p)
			if !got {
				t.Errorf("expected %q to be refused as an index root", p)
			}
			if runtime.GOOS == "windows" && filepath.VolumeName(p) != "" && filepath.Clean(p) == filepath.VolumeName(p)+`\` {
				if reason != "windows drive root" {
					t.Errorf("reason for %q = %q, want %q", p, reason, "windows drive root")
				}
				continue
			}
			if reason != "hardcoded system root" {
				t.Errorf("reason for %q = %q, want %q", p, reason, "hardcoded system root")
			}
		}
	})

	t.Run("symlink to home is refused", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home dir available")
		}
		linkParent := t.TempDir()
		link := filepath.Join(linkParent, "home-symlink")
		if err := os.Symlink(home, link); err != nil {
			t.Skipf("symlinks unsupported on this platform: %v", err)
		}
		got, reason := IsRootUnindexable(link)
		if !got {
			t.Errorf("expected %q (symlink to home) to be refused", link)
		}
		if reason != "user home directory" {
			t.Errorf("reason = %q, want %q", reason, "user home directory")
		}
	})

	t.Run("home directory is refused", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home dir available")
		}
		got, reason := IsRootUnindexable(home)
		if !got {
			t.Errorf("expected %q (home) to be refused as an index root", home)
		}
		if reason != "user home directory" {
			t.Errorf("reason = %q, want %q", reason, "user home directory")
		}
	})

	t.Run("agent session stores are refused", func(t *testing.T) {
		root := t.TempDir()
		cases := []struct {
			name string
			path string
		}{
			{name: "claude projects", path: filepath.Join(root, ".claude", "projects")},
			{name: "codex sessions", path: filepath.Join(root, ".codex", "sessions")},
			{name: "codex history", path: filepath.Join(root, ".codex", "history")},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := os.MkdirAll(tc.path, 0o755); err != nil {
					t.Fatal(err)
				}
				got, reason := IsRootUnindexable(tc.path)
				if !got {
					t.Fatalf("expected %q to be refused as an agent session store", tc.path)
				}
				if reason != "agent session store" {
					t.Fatalf("reason = %q, want %q", reason, "agent session store")
				}
			})
		}
	})

	t.Run("agent config directories remain indexable inside projects", func(t *testing.T) {
		root := t.TempDir()
		for _, dir := range []string{filepath.Join(root, ".claude"), filepath.Join(root, ".codex")} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if got, reason := IsRootUnindexable(dir); got {
				t.Fatalf("expected config dir %q to remain indexable, got reason %q", dir, reason)
			}
		}
	})

	t.Run("hardcoded refusal does not falsely flag siblings of home", func(t *testing.T) {
		// A directory that is NOT in the refusal list and has no .lumenignore
		// must still be indexable. Use the test's tempdir which is outside any
		// hardcoded path.
		dir := t.TempDir()
		if got, _ := IsRootUnindexable(dir); got {
			t.Errorf("expected %q to be indexable (no .lumenignore, not hardcoded)", dir)
		}
	})
}

func TestMakeSkip_HardcodedFiles(t *testing.T) {
	dir := t.TempDir()
	skip := MakeSkip(dir, []string{".go", ".json", ".yaml"})

	for name := range SkipFiles {
		if !skip(name, false) {
			t.Errorf("expected hardcoded file %q to be skipped", name)
		}
	}

	// Regular files with same extensions should pass
	if skip("package.json", false) {
		t.Error("expected package.json to pass")
	}
	if skip("main.go", false) {
		t.Error("expected main.go to pass")
	}
}

func TestMakeSkip_HardcodedDirs(t *testing.T) {
	dir := t.TempDir()
	skip := MakeSkip(dir, []string{".go"})

	for name := range SkipDirs {
		if !skip(name, true) {
			t.Errorf("expected hardcoded dir %q to be skipped", name)
		}
	}

	// Non-skipped dirs should pass
	if skip("src", true) {
		t.Error("expected src/ to pass")
	}
	if skip("pkg", true) {
		t.Error("expected pkg/ to pass")
	}
}

func TestMakeSkip_NestedGitignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "*.log\n")
	writeFile(t, dir, "sub/.gitignore", "secret.go\n")
	writeFile(t, dir, "sub/main.go", "package sub\n")
	writeFile(t, dir, "sub/secret.go", "package sub\n")

	skip := MakeSkip(dir, []string{".go", ".log"})

	// Root .gitignore applies everywhere
	if !skip("app.log", false) {
		t.Error("expected app.log to be skipped via root .gitignore")
	}
	if !skip("sub/app.log", false) {
		t.Error("expected sub/app.log to be skipped via root .gitignore")
	}
	// Nested .gitignore applies in its directory
	if !skip("sub/secret.go", false) {
		t.Error("expected sub/secret.go to be skipped via nested .gitignore")
	}
	// Nested .gitignore does not affect root
	if skip("secret.go", false) {
		t.Error("expected root secret.go to pass (nested .gitignore doesn't apply at root)")
	}
	// Normal file in sub passes
	if skip("sub/main.go", false) {
		t.Error("expected sub/main.go to pass")
	}
}

func TestMakeSkip_LumenIgnore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".lumenignore", "generated/\n*.pb.go\n")

	skip := MakeSkip(dir, []string{".go"})

	if !skip("generated", true) {
		t.Error("expected generated/ to be skipped via .lumenignore")
	}
	if !skip("foo.pb.go", false) {
		t.Error("expected foo.pb.go to be skipped via .lumenignore")
	}
	if skip("main.go", false) {
		t.Error("expected main.go to pass")
	}
}

func TestMakeSkip_NestedLumenIgnore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/.lumenignore", "mock_*.go\n")
	writeFile(t, dir, "api/handler.go", "package api\n")
	writeFile(t, dir, "api/mock_handler.go", "package api\n")

	skip := MakeSkip(dir, []string{".go"})

	if !skip("api/mock_handler.go", false) {
		t.Error("expected api/mock_handler.go to be skipped via nested .lumenignore")
	}
	if skip("api/handler.go", false) {
		t.Error("expected api/handler.go to pass")
	}
	// Root level mock_ files are not affected
	if skip("mock_handler.go", false) {
		t.Error("expected root mock_handler.go to pass (nested ignore doesn't apply)")
	}
}

func TestMakeSkip_GitattributesGenerated(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitattributes", "bench-results linguist-generated\ndocs/plans linguist-generated=true\n")

	skip := MakeSkip(dir, []string{".go", ".txt"})

	if !skip("bench-results/data.txt", false) {
		t.Error("expected bench-results/data.txt to be skipped via linguist-generated")
	}
	if !skip("docs/plans/plan.txt", false) {
		t.Error("expected docs/plans/plan.txt to be skipped via linguist-generated=true")
	}
	if skip("src/main.go", false) {
		t.Error("expected src/main.go to pass")
	}
}

func TestMakeSkip_GitattributesNestedDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pkg/.gitattributes", "generated_*.go linguist-generated\n")

	skip := MakeSkip(dir, []string{".go"})

	if !skip("pkg/generated_types.go", false) {
		t.Error("expected pkg/generated_types.go to be skipped via nested .gitattributes")
	}
	if skip("pkg/types.go", false) {
		t.Error("expected pkg/types.go to pass")
	}
	// Root level is not affected
	if skip("generated_types.go", false) {
		t.Error("expected root generated_types.go to pass")
	}
}

func TestMakeSkip_GitattributesNonGenerated(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitattributes", "*.go linguist-language=Go\nkeep.go linguist-generated=false\n")

	skip := MakeSkip(dir, []string{".go"})

	// Non-generated attributes should not cause skipping
	if skip("main.go", false) {
		t.Error("expected main.go to pass (linguist-language is not linguist-generated)")
	}
	if skip("keep.go", false) {
		t.Error("expected keep.go to pass (linguist-generated=false)")
	}
}

func TestMakeSkip_AllLayersCombined(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "*.log\n")
	writeFile(t, dir, ".lumenignore", "scratch/\n")
	writeFile(t, dir, ".gitattributes", "generated.go linguist-generated\n")
	writeFile(t, dir, "sub/.gitignore", "tmp_*.go\n")

	skip := MakeSkip(dir, []string{".go", ".log"})

	// Layer 1: SkipDirs
	if !skip("node_modules", true) {
		t.Error("expected node_modules to be skipped (SkipDirs)")
	}
	// Layer 2: .gitignore (root)
	if !skip("debug.log", false) {
		t.Error("expected debug.log to be skipped (.gitignore)")
	}
	// Layer 2: .gitignore (nested)
	if !skip("sub/tmp_data.go", false) {
		t.Error("expected sub/tmp_data.go to be skipped (nested .gitignore)")
	}
	// Layer 3: .lumenignore
	if !skip("scratch", true) {
		t.Error("expected scratch/ to be skipped (.lumenignore)")
	}
	// Layer 4: .gitattributes
	if !skip("generated.go", false) {
		t.Error("expected generated.go to be skipped (.gitattributes)")
	}
	// Layer 5: extension filter
	if !skip("readme.md", false) {
		t.Error("expected readme.md to be skipped (extension filter)")
	}
	// Normal file passes all layers
	if skip("main.go", false) {
		t.Error("expected main.go to pass all layers")
	}
}

func TestParseLinguistExcluded(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
		match   []string // paths that should match
		noMatch []string // paths that should not match
	}{
		{
			name:    "bare linguist-generated",
			content: "bench-results linguist-generated\n",
			match:   []string{"bench-results"},
			noMatch: []string{"other"},
		},
		{
			name:    "linguist-generated=true",
			content: "docs/plans linguist-generated=true\n",
			match:   []string{"docs/plans"},
			noMatch: []string{"docs/other"},
		},
		{
			name:    "linguist-generated=false ignored",
			content: "keep.go linguist-generated=false\n",
			noMatch: []string{"keep.go"},
		},
		{
			name:    "negated -linguist-generated ignored",
			content: "keep.go -linguist-generated\n",
			noMatch: []string{"keep.go"},
		},
		{
			name:    "multiple attributes",
			content: "*.pb.go linguist-generated text\n",
			match:   []string{"foo.pb.go"},
			noMatch: []string{"foo.go"},
		},
		{
			name:    "comments and blanks",
			content: "# comment\n\n*.gen.go linguist-generated\n",
			match:   []string{"foo.gen.go"},
			noMatch: []string{"foo.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".gitattributes")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			gi := parseLinguistExcluded(path)

			for _, p := range tt.match {
				if gi == nil || !gi.MatchesPath(p) {
					t.Errorf("expected %q to match", p)
				}
			}
			for _, p := range tt.noMatch {
				if gi != nil && gi.MatchesPath(p) {
					t.Errorf("expected %q to NOT match", p)
				}
			}
		})
	}

	// Non-existent file
	if gi := parseLinguistExcluded(filepath.Join(dir, "nonexistent")); gi != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestParseLinguistAttributes_Vendored(t *testing.T) {
	dir := t.TempDir()
	attrs := "third_party/* linguist-vendored\ngenerated.go linguist-generated=true\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte(attrs), 0o644); err != nil {
		t.Fatal(err)
	}

	tree := NewIgnoreTree(dir, []string{".go"})
	if !tree.shouldSkip("third_party/lib.go", false) {
		t.Error("linguist-vendored file should be skipped")
	}
	if !tree.shouldSkip("generated.go", false) {
		t.Error("linguist-generated=true file should be skipped")
	}
	if tree.shouldSkip("normal.go", false) {
		t.Error("normal.go should not be skipped")
	}
}

func TestMakeSkipWithExtra_SkipsWorktreePaths(t *testing.T) {
	dir := t.TempDir()

	extraPaths := []string{".worktrees/feature", ".worktrees/bugfix"}
	skip := MakeSkipWithExtra(dir, []string{".go"}, extraPaths)

	// Extra paths should be skipped as directories
	if !skip(".worktrees/feature", true) {
		t.Error("expected .worktrees/feature dir to be skipped")
	}
	if !skip(".worktrees/bugfix", true) {
		t.Error("expected .worktrees/bugfix dir to be skipped")
	}
	// Files are not subject to the extra skip (directory pruning handles it)
	if skip("main.go", false) {
		t.Error("expected main.go to pass")
	}
	// A similarly named dir NOT in the skip list should pass
	if skip(".worktrees/other", true) {
		t.Error("expected .worktrees/other to pass (not in extra skip list)")
	}
}

func TestBuildTree_SkipsInternalWorktrees(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main\n")
	writeFile(t, dir, ".worktrees/feature/main.go", "package main\n")
	writeFile(t, dir, ".worktrees/feature/util.go", "package util\n")

	skip := MakeSkipWithExtra(dir, []string{".go"}, []string{".worktrees/feature"})
	tree, err := BuildTree(dir, skip)
	if err != nil {
		t.Fatal(err)
	}

	if len(tree.Files) != 1 {
		t.Fatalf("expected 1 file (main.go only), got %d: %v", len(tree.Files), tree.Files)
	}
	if _, ok := tree.Files["main.go"]; !ok {
		t.Error("expected main.go in tree")
	}
}

func TestAncestorDirs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", []string{""}},
		{".", []string{""}},
		{"a", []string{"", "a"}},
		{"a/b", []string{"", "a", filepath.Join("a", "b")}},
		{"a/b/c", []string{"", "a", filepath.Join("a", "b"), filepath.Join("a", "b", "c")}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ancestorDirs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("ancestorDirs(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ancestorDirs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildTree_WithGitignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "generated/\n*.tmp\n")
	writeFile(t, dir, "main.go", "package main\n")
	writeFile(t, dir, "util.go", "package main\n")

	if err := os.MkdirAll(filepath.Join(dir, "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "generated/code.go", "package generated\n")
	writeFile(t, dir, "data.tmp", "temp data")

	skip := MakeSkip(dir, []string{".go", ".tmp"})
	tree, err := BuildTree(dir, skip)
	if err != nil {
		t.Fatal(err)
	}

	// Should only have main.go and util.go
	if len(tree.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(tree.Files), tree.Files)
	}
	if _, ok := tree.Files["main.go"]; !ok {
		t.Error("expected main.go in tree")
	}
	if _, ok := tree.Files["util.go"]; !ok {
		t.Error("expected util.go in tree")
	}
	if _, ok := tree.Files["generated/code.go"]; ok {
		t.Error("expected generated/code.go to be excluded")
	}
	if _, ok := tree.Files["data.tmp"]; ok {
		t.Error("expected data.tmp to be excluded")
	}
}

func TestBuildTree_WithNestedGitignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "*.log\n")
	writeFile(t, dir, "main.go", "package main\n")
	writeFile(t, dir, "sub/sub.go", "package sub\n")
	writeFile(t, dir, "sub/.gitignore", "internal_*.go\n")
	writeFile(t, dir, "sub/internal_helper.go", "package sub\n")
	writeFile(t, dir, "sub/helper.go", "package sub\n")
	writeFile(t, dir, "app.log", "log data")

	skip := MakeSkip(dir, []string{".go", ".log"})
	tree, err := BuildTree(dir, skip)
	if err != nil {
		t.Fatal(err)
	}

	// main.go, sub/sub.go, sub/helper.go should be present
	// app.log (root .gitignore), sub/internal_helper.go (nested .gitignore) excluded
	expected := map[string]bool{
		"main.go":                         true,
		filepath.Join("sub", "sub.go"):    true,
		filepath.Join("sub", "helper.go"): true,
	}
	if len(tree.Files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(tree.Files), tree.Files)
	}
	for f := range expected {
		if _, ok := tree.Files[f]; !ok {
			t.Errorf("expected %s in tree", f)
		}
	}
	if _, ok := tree.Files["app.log"]; ok {
		t.Error("expected app.log to be excluded by root .gitignore")
	}
	if _, ok := tree.Files[filepath.Join("sub", "internal_helper.go")]; ok {
		t.Error("expected sub/internal_helper.go to be excluded by nested .gitignore")
	}
}

func TestIgnoreTree_GlobalGitignore(t *testing.T) {
	// Create global gitignore
	globalIgnoreDir := t.TempDir()
	globalIgnorePath := filepath.Join(globalIgnoreDir, "gitignore_global")
	if err := os.WriteFile(globalIgnorePath, []byte("*.scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create git config pointing to it
	configPath := filepath.Join(globalIgnoreDir, "gitconfig")
	configContent := fmt.Sprintf("[core]\n\texcludesFile = %s\n", filepath.ToSlash(globalIgnorePath))
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override git config location
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)

	dir := t.TempDir()
	tree := NewIgnoreTree(dir, []string{".go", ".scratch"})
	if !tree.shouldSkip("notes.scratch", false) {
		t.Error("*.scratch should be skipped by global gitignore")
	}
	if tree.shouldSkip("main.go", false) {
		t.Error("main.go should not be skipped")
	}
}
