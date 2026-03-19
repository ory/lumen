# Path Topology E2E Test Matrix Design

## Goal

Add a table-driven `TestE2E_PathTopologies` test that systematically exercises all combinations of path topology (plain dir, git repo, subdirectory, worktree, symlink) to prevent regressions like the macOS symlink/pathPrefix bug found in PR #61.

## Background

Two structural gaps were identified in the existing e2e test suite:

**Gap A — Symlinks not tested:** On macOS, `t.TempDir()` returns symlink paths (`/var/folders/...`) while `git.RepoRoot()` resolves them via `EvalSymlinks` (`/private/var/...`). When the resolved git root was used as `effectiveRoot` but the raw symlink path was used as `input.Path`, `filepath.Rel(effectiveRoot, input.Path)` produced paths with `..` components, causing pathPrefix filtering to match nothing and return empty results. The fix (`EvalSymlinks` in `validateSearchInput`) had no dedicated regression test.

**Gap B — Git-repo + subdirectory not systematically covered:** Existing core search tests (`TestE2E_IndexAndSearchResults`, etc.) use `sampleProjectPath` — a plain directory with no `.git`. None exercise pathPrefix filtering. If pathPrefix filtering broke, those tests would not catch it.

## Architecture

A single `TestE2E_PathTopologies` test function iterates over a table of `pathTopologyCase` entries. Each entry:

- Has a self-contained `setup` function that creates an isolated temp dir and repo layout
- Declares search parameters (path, cwd, query)
- Declares expectations: reindexed flag, minimum file count, symbols that must appear (`wantSymbols`), symbols that must not appear (`wantNoSymbols`)
- Optionally declares a second search call to verify cache reuse or index sharing

All entries share one MCP server session for performance. Each entry uses a different temp dir, producing a different DB path hash, so there is no cache interference between entries.

## Data Types

```go
type pathTopologyCase struct {
    name          string
    setup         func(t *testing.T) topologySetup
    query         string
    wantReindexed bool
    wantMinFiles  int
    wantSymbols   []string // all must appear in results
    wantNoSymbols []string // none must appear (verifies pathPrefix scoping)
    second        *secondCall
}

type topologySetup struct {
    searchPath string
    cwd        string // empty = omit from MCP request
}

type secondCall struct {
    query         string
    searchPath    string
    wantReindexed bool
    wantSymbols   []string
}
```

## Topologies

| # | Name | Setup | Path | wantNoSymbols | Notes |
|---|------|-------|------|---------------|-------|
| 1 | `plain-dir` | Temp dir, one Go file, no git | root | — | Baseline; `findEffectiveRoot` falls back to input path |
| 2 | `git-root` | Git repo, `pkg/` + `api/` subdirs | repo root | — | No pathPrefix; both subdirs indexed |
| 3 | `git-subdir` | Same repo as #2 layout (new temp) | `pkg/` | `api/` symbols | pathPrefix="pkg" must exclude `api/` |
| 4 | `git-subdir-sibling` | Same; second call from `api/` | `pkg/` → `api/` | — | Second call: Reindexed=false; `api/` symbols found via shared index |
| 5 | `git-subdir-cwd` | Git repo; `path=pkg/, cwd=root` | `pkg/` | — | cwd not adopted (no DB); git root fallback used |
| 6 | `worktree-root` | Git repo + external worktree | worktree root | — | Worktree uses its own index root |
| 7 | `worktree-subdir` | Git repo + worktree with `pkg/` subdir | `worktree/pkg/` | symbols outside `pkg/` | pathPrefix inside worktree |
| 8 | `internal-worktree-subdir` | Git repo + internal worktree at `.worktrees/feat/` with `pkg/` | `.worktrees/feat/pkg/` | — | Internal worktree treated as own root; pathPrefix within it |
| 9 | `symlink-root` | Git repo; symlink → repo root | symlink path | — | Gap A: symlinks resolved; results found |
| 10 | `symlink-subdir` | Git repo with `pkg/`; symlink → repo | `symlink/pkg/` | symbols outside `pkg/` | Gap A+B combined: symlink + pathPrefix |

## Key Assertions

**`wantNoSymbols`** is the critical assertion that was missing before. For topologies 3, 7, and 10, symbols from sibling subdirectories must not appear in results. This verifies that pathPrefix filtering actually excludes out-of-scope results — not just that it returns something.

**Second call `wantReindexed=false`** in topology 4 verifies that sibling subdirectory searches share the git-root index rather than creating per-subdirectory indexes.

**Topologies 9 and 10** are explicit regression tests for the macOS symlink bug. They use `os.Symlink` to create an actual symlink, pass the symlink path to the server, and assert that results are correct — so any future removal of `EvalSymlinks` from `validateSearchInput` will cause these tests to fail.

## File Structure

- **New file**: `e2e_topology_test.go` (build tag `e2e`)
- Shares existing helpers from `e2e_test.go`: `startServer`, `callSearch`, `findResult`, `resultSymbols`, `gitE2ERun`
- Does not modify existing tests

## Out of Scope

- Nested git repos (submodules) — separate concern
- Windows path separators — no Windows CI currently
- Non-`all-minilm` embedding models — topology test uses same model as all other e2e tests
