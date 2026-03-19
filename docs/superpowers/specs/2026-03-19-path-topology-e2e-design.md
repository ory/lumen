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
- Optionally declares a second search call (with its own query, path, and symbol assertions) to verify cache reuse or index sharing

All entries share one MCP server session for performance. Each entry uses a different temp dir, producing a different DB path hash, so there is no cache interference between entries. `LUMEN_FRESHNESS_TTL=1s` (set by `startServer`) means TTL interactions between sequentially-run entries are bounded to 1 second and do not affect correctness.

## Data Types

```go
type pathTopologyCase struct {
    name          string
    setup         func(t *testing.T) topologySetup
    query         string
    wantReindexed bool
    wantMinFiles  int       // 0 = unchecked
    wantSymbols   []string  // all must appear in results
    wantNoSymbols []string  // none must appear (verifies pathPrefix scoping)
    second        *secondCall
}

type topologySetup struct {
    searchPath string
    cwd        string // empty = omit from MCP request
}

// secondCall describes an optional second search call on the same repo.
// Used to verify cache/index sharing or sibling-directory scoping.
type secondCall struct {
    query         string
    searchPath    string
    wantReindexed bool
    wantSymbols   []string
    wantNoSymbols []string  // verifies pathPrefix on second call too
}
```

## Repo layout used by git-based topologies

Topologies 2–10 use a shared canonical layout (each gets its own fresh temp dir):

```
repo/               ← git root (git init)
  pkg/
    server.go       ← defines func StartServer()
  api/
    handler.go      ← defines func HandleLogin()
```

Worktree topologies additionally create:

```
# external worktree (topologies 6, 7):
worktree/           ← git worktree of repo (git worktree add)
  pkg/
    server.go       ← defines func StartServer() (same content)
  api/
    handler.go      ← defines func HandleLogin() (same content)

# internal worktree (topology 8):
repo/.worktrees/feat/   ← internal worktree (git worktree add repo/.worktrees/feat -b feat)
  pkg/
    worker.go       ← defines func RunWorker()
  api/
    auth.go         ← defines func AuthenticateUser()
```

## Topologies

| # | Name | searchPath | cwd | query (1st) | wantMinFiles | wantSymbols | wantNoSymbols | second call |
|---|------|-----------|-----|-------------|--------------|-------------|----------------|-------------|
| 1 | `plain-dir` | repo root (no git) | — | "start server" | 2 | `StartServer` | — | — |
| 2 | `git-root` | repo root | — | "start server" | 2 | `StartServer` | — | — |
| 3 | `git-subdir` | `repo/pkg/` | — | "start server" | 2 | `StartServer` | `HandleLogin` | — |
| 4 | `git-subdir-sibling` | `repo/pkg/` | — | "start server" | 2 | `StartServer` | `HandleLogin` | path=`repo/api/`, query="login handler", Reindexed=false, wantSymbols=`HandleLogin`, wantNoSymbols=`StartServer` |
| 5 | `git-subdir-cwd` | `repo/pkg/` | `repo/` | "start server" | 2 | `StartServer` | — | — |
| 6 | `worktree-root` | worktree root | — | "start server" | 2 | `StartServer` | — | — |
| 7 | `worktree-subdir` | `worktree/pkg/` | — | "start server" | 2 | `StartServer` | `HandleLogin` | — |
| 8 | `internal-worktree-subdir` | `repo/.worktrees/feat/pkg/` | — | "run worker" | 2 | `RunWorker` | `AuthenticateUser` | — |
| 9 | `symlink-root` *(skip if symlink unavailable)* | symlink → repo root | — | "start server" | 2 | `StartServer` | — | — |
| 10 | `symlink-subdir` *(skip if symlink unavailable)* | symlink → `repo/pkg/` | — | "start server" | 2 | `StartServer` | `HandleLogin` | — |

**Topology 1 (`plain-dir`)** has `wantMinFiles: 2` because the plain-dir layout has both `pkg/server.go` and `api/handler.go` and no path filtering — both must be indexed.

**Topology 5 (`git-subdir-cwd`)** adds `wantMinFiles: 2` to verify that even with `cwd=repo/` (where no DB exists yet), the git root fallback indexes both subdirectories — not just `pkg/`. The existing `TestE2E_CwdNotAdoptedWhenNoDBExists` only verifies one symbol is findable; this topology adds the complementary file-count assertion.

**Topology 8 (`internal-worktree-subdir`)** works because git worktrees have their own git root: `git rev-parse --show-toplevel` from inside `repo/.worktrees/feat/` returns `repo/.worktrees/feat/`, not `repo/`. Therefore `findEffectiveRoot(repo/.worktrees/feat/pkg/)` walks up to `repo/.worktrees/feat/` (the worktree's git root) and uses it as `effectiveRoot`. Both `feat/pkg/worker.go` and `feat/api/auth.go` are indexed at that root. Searching with `pathPrefix="pkg"` returns `RunWorker` but not `AuthenticateUser`. This is distinct from `TestE2E_InternalWorktreeSearchNoReindex`, which only verifies `Reindexed=false` and does not test pathPrefix scoping within the worktree's own index.

**Topologies 9 and 10 (symlink):** Each topology creates a secondary temp dir outside the repo, then calls `os.Symlink(repoDir, symlinkDir)` where `symlinkDir` is a path inside that secondary temp dir. This guarantees the symlink path and the resolved path are different strings on all platforms: on macOS, `t.TempDir()` already returns a symlink path (`/var/...` → `/private/var/...`), so both `symlinkDir` and `repoDir` differ from their resolved forms; on Linux, `t.TempDir()` returns real paths and `os.Symlink` creates a genuinely different string path. If `os.Symlink` fails (e.g., some restricted environments), `t.Skip` is called rather than failing.

## Key Assertions

**`wantNoSymbols`** is the critical new assertion. For topologies 3, 4 (second call), 7, 8, and 10, symbols from sibling subdirectories must not appear in results. This verifies that pathPrefix filtering actively excludes out-of-scope results — not just that it returns something. This assertion was absent in all tests before this PR.

**Second call `wantReindexed=false`** in topology 4 verifies that sibling subdirectory searches share the git-root index. The second call also uses `wantNoSymbols: ["StartServer"]` to confirm that searching `api/` with pathPrefix="api" excludes `pkg/` symbols even when using the shared index.

**`wantMinFiles: 2`** is required for all topologies. It distinguishes between "indexed only the searched subdirectory" (wrong: 1 file) and "indexed the full root" (correct: 2 files), catching any regression where effectiveRoot is scoped too narrowly.

## File Structure

- **New file**: `e2e_topology_test.go` (build tag `e2e`)
- Shares existing helpers from `e2e_test.go`: `startServer`, `callSearch`, `findResult`, `resultSymbols`, `gitE2ERun`
- Does not modify existing tests

## Out of Scope

- Nested git repos (submodules) — separate concern
- Windows path separators — no Windows CI currently
- Non-`all-minilm` embedding models — topology test uses same model as all other e2e tests
