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

//go:build e2e

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type pathTopologyCase struct {
	name          string
	setup         func(t *testing.T) topologySetup
	query         string
	wantReindexed bool
	wantMinFiles  int      // 0 = unchecked
	wantSymbols   []string // all must appear in results
	wantNoSymbols []string // none must appear (verifies pathPrefix scoping)
	second        *secondCall
}

type topologySetup struct {
	searchPath string
	cwd        string // empty = omit from MCP request
}

// secondCall describes an optional second search call on the same repo.
type secondCall struct {
	query         string
	searchPath    string
	wantReindexed bool
	wantSymbols   []string
	wantNoSymbols []string // verifies pathPrefix on second call too
}

// makeCanonicalDir creates a plain directory (no git) with pkg/server.go and api/handler.go.
func makeCanonicalDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	apiDir := filepath.Join(dir, "api")
	for _, d := range []string{pkgDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", d, err)
		}
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "server.go"), []byte(`package pkg

// StartServer starts the main server loop.
func StartServer() {}
`), 0o644); err != nil {
		t.Fatalf("failed to write server.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "handler.go"), []byte(`package api

// HandleLogin processes login requests.
func HandleLogin() {}
`), 0o644); err != nil {
		t.Fatalf("failed to write handler.go: %v", err)
	}
	return dir
}

// makeCanonicalRepo creates a git repo with the canonical dir layout and an empty initial commit.
// The files written by makeCanonicalDir are intentionally left untracked — only the .git
// directory matters for root detection and worktree operations in these tests.
func makeCanonicalRepo(t *testing.T) string {
	t.Helper()
	dir := makeCanonicalDir(t)
	gitE2ERun(t, dir, "init")
	gitE2ERun(t, dir, "commit", "--allow-empty", "-m", "init")
	return dir
}

// makeCanonicalWorktree adds a detached worktree at wtDir and writes the canonical
// pkg/server.go and api/handler.go fixtures into it, mirroring makeCanonicalDir's layout.
func makeCanonicalWorktree(t *testing.T, repo, wtDir string) {
	t.Helper()
	gitE2ERun(t, repo, "worktree", "add", wtDir)
	pkgDir := filepath.Join(wtDir, "pkg")
	apiDir := filepath.Join(wtDir, "api")
	for _, d := range []string{pkgDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", d, err)
		}
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "server.go"), []byte(`package pkg

// StartServer starts the main server loop.
func StartServer() {}
`), 0o644); err != nil {
		t.Fatalf("failed to write server.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "handler.go"), []byte(`package api

// HandleLogin processes login requests.
func HandleLogin() {}
`), 0o644); err != nil {
		t.Fatalf("failed to write handler.go: %v", err)
	}
}

// TestE2E_PathTopologies exercises 10 different path/topology configurations
// to verify that search scoping, worktree isolation, and symlink resolution
// all work correctly across varied filesystem layouts.
func TestE2E_PathTopologies(t *testing.T) {
	t.Run("semantic", func(t *testing.T) { runPathTopologies(t, startServer(t)) })
}

func TestE2E_BM25_PathTopologies(t *testing.T) {
	t.Run("bm25", func(t *testing.T) {
		session, trap, _ := startBM25Server(t)
		runPathTopologies(t, session)
		trap.assertUnused(t)
	})
}

func runPathTopologies(t *testing.T, session *mcp.ClientSession) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Parallel()

	cases := []pathTopologyCase{
		{
			name: "plain-dir",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				dir := makeCanonicalDir(t)
				return topologySetup{searchPath: dir}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
		},
		{
			name: "git-root",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				return topologySetup{searchPath: repo}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
		},
		{
			name: "git-subdir",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				return topologySetup{searchPath: filepath.Join(repo, "pkg")}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
			wantNoSymbols: []string{"HandleLogin"},
		},
		{
			name: "git-subdir-sibling",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				return topologySetup{searchPath: filepath.Join(repo, "pkg")}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
			wantNoSymbols: []string{"HandleLogin"},
			second: &secondCall{
				query:         "login handler",
				searchPath:    "", // filled in dynamically via setup closure capture
				wantReindexed: false,
				wantSymbols:   []string{"HandleLogin"},
				wantNoSymbols: []string{"StartServer"},
			},
		},
		{
			name: "git-subdir-cwd",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				return topologySetup{
					searchPath: filepath.Join(repo, "pkg"),
					cwd:        repo,
				}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
		},
		{
			name: "worktree-root",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				wtDir := t.TempDir()
				makeCanonicalWorktree(t, repo, wtDir)
				return topologySetup{searchPath: wtDir}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
		},
		{
			name: "worktree-subdir",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				wtDir := t.TempDir()
				makeCanonicalWorktree(t, repo, wtDir)
				// Whole worktree repo indexed; results scoped to pkg/ by pathPrefix.
				return topologySetup{searchPath: filepath.Join(wtDir, "pkg")}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
			wantNoSymbols: []string{"HandleLogin"},
		},
		{
			name: "internal-worktree-subdir",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				wtDir := filepath.Join(repo, ".worktrees", "feat")
				gitE2ERun(t, repo, "worktree", "add", wtDir, "-b", "feat")
				// Write different files into the internal worktree.
				pkgDir := filepath.Join(wtDir, "pkg")
				apiDir := filepath.Join(wtDir, "api")
				for _, d := range []string{pkgDir, apiDir} {
					if err := os.MkdirAll(d, 0o755); err != nil {
						t.Fatalf("failed to create dir %s: %v", d, err)
					}
				}
				if err := os.WriteFile(filepath.Join(pkgDir, "worker.go"), []byte(`package pkg

// RunWorker executes the background worker.
func RunWorker() {}
`), 0o644); err != nil {
					t.Fatalf("failed to write worker.go: %v", err)
				}
				if err := os.WriteFile(filepath.Join(apiDir, "auth.go"), []byte(`package api

// AuthenticateUser checks user credentials.
func AuthenticateUser() {}
`), 0o644); err != nil {
					t.Fatalf("failed to write auth.go: %v", err)
				}
				return topologySetup{searchPath: filepath.Join(wtDir, "pkg")}
			},
			query:         "run worker",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"RunWorker"},
			wantNoSymbols: []string{"AuthenticateUser"},
		},
		{
			name: "symlink-root",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				symlinkBase := t.TempDir()
				linkPath := filepath.Join(symlinkBase, "repo-link")
				if err := os.Symlink(repo, linkPath); err != nil {
					t.Skipf("symlinks not available: %v", err)
				}
				return topologySetup{searchPath: linkPath}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
		},
		{
			// wantMinFiles=2: whole repo indexed via resolved symlink root.
			// wantNoSymbols: results scoped to pkg/ by pathPrefix despite full-repo index.
			name: "symlink-subdir",
			setup: func(t *testing.T) topologySetup {
				t.Helper()
				repo := makeCanonicalRepo(t)
				symlinkBase := t.TempDir()
				linkPath := filepath.Join(symlinkBase, "repo-link")
				if err := os.Symlink(repo, linkPath); err != nil {
					t.Skipf("symlinks not available: %v", err)
				}
				return topologySetup{searchPath: filepath.Join(linkPath, "pkg")}
			},
			query:         "start server",
			wantReindexed: true,
			wantMinFiles:  2,
			wantSymbols:   []string{"StartServer"},
			wantNoSymbols: []string{"HandleLogin"},
		},
	}

	// "ancestor-reuse" is tested separately below (TestE2E_AncestorReuse_MCP)
	// because the two-call pattern with dynamic paths does not fit the
	// pathTopologyCase struct cleanly.

	// Subtests run sequentially: they share session and the server's per-path index state.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setup := tc.setup(t)

			args := map[string]any{
				"query":     tc.query,
				"path":      setup.searchPath,
				"limit":     10,
				"min_score": -1,
			}
			if setup.cwd != "" {
				args["cwd"] = setup.cwd
			}

			out := callSearch(t, session, args)

			// Assert wantReindexed.
			if out.Reindexed != tc.wantReindexed {
				t.Errorf("Reindexed: got %v, want %v", out.Reindexed, tc.wantReindexed)
			}

			// Assert wantMinFiles (only when reindexed and non-zero).
			if tc.wantReindexed && tc.wantMinFiles > 0 {
				if out.IndexedFiles < tc.wantMinFiles {
					t.Errorf("IndexedFiles: got %d, want >= %d", out.IndexedFiles, tc.wantMinFiles)
				}
			}

			// Assert wantSymbols — all must appear.
			for _, sym := range tc.wantSymbols {
				if findResult(out.Results, sym) == nil {
					t.Errorf("expected symbol %q in results, got: %v", sym, resultSymbols(out.Results))
				}
			}

			// Assert wantNoSymbols — none must appear.
			for _, sym := range tc.wantNoSymbols {
				if findResult(out.Results, sym) != nil {
					t.Errorf("unexpected symbol %q in results (pathPrefix scoping failed): %v", sym, resultSymbols(out.Results))
				}
			}

			if tc.second != nil {
				// Determine second call searchPath.
				// For the "git-subdir-sibling" case the second path is the api/ sibling.
				// We derive it from setup.searchPath by replacing the last component.
				secondPath := tc.second.searchPath
				if secondPath == "" {
					// Default: derive api/ sibling from pkg/ path.
					secondPath = filepath.Join(filepath.Dir(setup.searchPath), "api")
				}

				secondArgs := map[string]any{
					"query":     tc.second.query,
					"path":      secondPath,
					"limit":     10,
					"min_score": -1,
				}

				out2 := callSearch(t, session, secondArgs)

				if out2.Reindexed != tc.second.wantReindexed {
					t.Errorf("second call Reindexed: got %v, want %v", out2.Reindexed, tc.second.wantReindexed)
				}

				for _, sym := range tc.second.wantSymbols {
					if findResult(out2.Results, sym) == nil {
						t.Errorf("second call: expected symbol %q in results, got: %v", sym, resultSymbols(out2.Results))
					}
				}

				for _, sym := range tc.second.wantNoSymbols {
					if findResult(out2.Results, sym) != nil {
						t.Errorf("second call: unexpected symbol %q in results (pathPrefix scoping failed): %v", sym, resultSymbols(out2.Results))
					}
				}
			}
		})
	}
}

// TestE2E_AncestorReuse_MCP verifies that an MCP search from a non-git
// subdirectory reuses the parent directory's cached index instead of creating
// a new one.
func TestE2E_AncestorReuse_MCP(t *testing.T) {
	t.Run("semantic", func(t *testing.T) { runAncestorReuse_MCP(t, startServer(t)) })
}

func TestE2E_BM25_AncestorReuse_MCP(t *testing.T) {
	t.Run("bm25", func(t *testing.T) {
		session, trap, _ := startBM25Server(t)
		runAncestorReuse_MCP(t, session)
		trap.assertUnused(t)
	})
}

func runAncestorReuse_MCP(t *testing.T, session *mcp.ClientSession) {
	t.Parallel()

	dir := makeCanonicalDir(t)
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Place a file in sub/ so the pathPrefix filter has something to return.
	if err := os.WriteFile(filepath.Join(subDir, "util.go"), []byte(`package sub

// SubHelper is a helper in the subdirectory.
func SubHelper() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// First search indexes the root directory (including sub/).
	out1 := callSearch(t, session, map[string]any{
		"query":     "start server",
		"path":      dir,
		"limit":     10,
		"min_score": -1,
	})
	if !out1.Reindexed {
		t.Error("first search: expected Reindexed=true")
	}
	if out1.IndexedFiles < 3 {
		t.Errorf("first search: expected IndexedFiles >= 3, got %d", out1.IndexedFiles)
	}
	if findResult(out1.Results, "StartServer") == nil {
		t.Errorf("first search: expected StartServer in results, got: %v", resultSymbols(out1.Results))
	}

	// Second search from sub/ should reuse the parent's index (no re-index).
	out2 := callSearch(t, session, map[string]any{
		"query":     "sub helper",
		"path":      subDir,
		"limit":     10,
		"min_score": -1,
	})
	if out2.Reindexed {
		t.Error("sub/ search: expected Reindexed=false (parent index should be reused)")
	}
	if findResult(out2.Results, "SubHelper") == nil {
		t.Errorf("sub/ search: expected SubHelper in results (from parent index), got: %v", resultSymbols(out2.Results))
	}
}

// TestE2E_AncestorReuse_SkipDir verifies that ancestor walking does NOT cross
// skip directories (node_modules, testdata, etc.). A search from inside
// node_modules/ must create its own index, not reuse the parent's.
func TestE2E_AncestorReuse_SkipDir(t *testing.T) {
	t.Run("semantic", func(t *testing.T) { runAncestorReuse_SkipDir(t, startServer(t)) })
}

func TestE2E_BM25_AncestorReuse_SkipDir(t *testing.T) {
	t.Run("bm25", func(t *testing.T) {
		session, trap, _ := startBM25Server(t)
		runAncestorReuse_SkipDir(t, session)
		trap.assertUnused(t)
	})
}

func runAncestorReuse_SkipDir(t *testing.T, session *mcp.ClientSession) {
	t.Parallel()

	dir := makeCanonicalDir(t)

	// Add a Go file inside node_modules (a skip dir).
	nmDir := filepath.Join(dir, "node_modules", "somepkg")
	if err := os.MkdirAll(nmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "index.go"), []byte(`package somepkg

// NpmHelper is a helper function in node_modules.
func NpmHelper() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Index the root directory first.
	out1 := callSearch(t, session, map[string]any{
		"query":     "start server",
		"path":      dir,
		"limit":     10,
		"min_score": -1,
	})
	if !out1.Reindexed {
		t.Error("root search: expected Reindexed=true")
	}

	// Search from inside node_modules — must NOT reuse parent index.
	out2 := callSearch(t, session, map[string]any{
		"query":     "npm helper",
		"path":      nmDir,
		"limit":     10,
		"min_score": -1,
	})
	if !out2.Reindexed {
		t.Error("node_modules search: expected Reindexed=true (should NOT reuse parent)")
	}
	if findResult(out2.Results, "NpmHelper") == nil {
		t.Errorf("node_modules search: expected NpmHelper in results, got: %v", resultSymbols(out2.Results))
	}
	if findResult(out2.Results, "StartServer") != nil {
		t.Errorf("node_modules search: unexpected StartServer (parent index should NOT be reused): %v", resultSymbols(out2.Results))
	}
}
