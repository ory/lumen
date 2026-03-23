# Non-blocking semantic_search with partial results

**Date:** 2026-03-23
**Status:** Draft
**Branch:** `reindex-fixes`

## Problem

When the freshness TTL expires and no background indexer holds the flock,
`ensureIndexed()` calls `EnsureFresh()` synchronously. On large codebases with
many changed files this blocks the agent for minutes — it cannot respond, search,
or do anything else until reindexing completes.

The session-start hook already spawns a background indexer, but if that indexer
finishes before the first search and files change afterward, the next
`semantic_search` call pays the full synchronous reindex cost.

## Goal

`semantic_search` must never block the agent for more than 15 seconds waiting on
reindexing. If reindexing takes longer, return results from the stale index with
a warning that results may be incomplete, while reindexing continues in the
background.

## Design

### New output field

Add `StaleWarning` to `SemanticSearchOutput`:

```go
type SemanticSearchOutput struct {
    Results      []SearchResultItem `json:"results"`
    Reindexed    bool               `json:"reindexed"`
    IndexedFiles int                `json:"indexed_files,omitempty"`
    FilteredHint string             `json:"filtered_hint,omitempty"`
    SeedWarning  string             `json:"seed_warning,omitempty"`
    StaleWarning string             `json:"stale_warning,omitempty"` // NEW
}
```

When the 15s timeout fires, `StaleWarning` carries:

> "Index is being updated in the background. Results may be incomplete or
> outdated. A follow-up search in ~30s will return fresh results."

### Modified `ensureIndexed()` flow

Replace the synchronous `EnsureFresh()` call (lines 615-650) with a
timeout-guarded goroutine:

```
freshnessTTL miss AND flock NOT held:

  1. Create a buffered done channel (cap 1) and a result struct.
  2. Spawn goroutine:
     a. Try to acquire flock via TryAcquire().
     b. If flock acquired:
        - Run idx.EnsureFresh(bgCtx, projectDir, nil)
          (pass nil progress — the MCP request context may be gone
           by the time the goroutine runs, so progress notifications
           would fail)
        - On success: call ic.touchChecked(projectDir) so subsequent
          searches benefit from the freshness TTL cache.
        - On error: log the error at Warn level. Do NOT call
          touchChecked (next search retries).
        - Release flock (defer).
        - Send result (reindexed, stats, err) on done channel.
     c. If flock NOT acquired (race — another process grabbed it):
        - Send zero result on done channel (skip).
  3. Select on done channel with 15s timeout:
     a. Done received in time → process as today (touchChecked, set
        Reindexed/IndexedFiles, return).
     b. Timeout fires:
        - Log at Info level: "reindex timeout, returning stale results".
        - Set out.StaleWarning with the warning message.
        - Do NOT call touchChecked() — next search retries freshness.
        - Return immediately — search proceeds against stale index.
        - The goroutine's result is never read; the buffered channel
          ensures it does not block.
```

**Why buffered channel (cap 1):** If the timeout fires first, the caller never
reads from the done channel. An unbuffered channel would cause the goroutine to
block on send forever, leaking it. A buffered channel lets the goroutine send
and exit cleanly.

### Goroutine context and lifecycle

- The goroutine uses `context.Background()` with a 10-minute timeout as a safety
  net — NOT the request context, which would be cancelled when the response is
  sent.
- The flock prevents concurrent reindexing: subsequent `semantic_search` calls see
  `IsHeld() == true` and skip (existing fast-path at line 611).
- When the goroutine finishes, it releases the flock. The next search with an
  expired freshness TTL sees a fresh index.
- If the MCP server process exits, the OS releases the flock — no leaked locks.
- **Graceful shutdown**: `indexerCache` should track background goroutines via a
  `sync.WaitGroup`. `Close()` calls `wg.Wait()` before closing indexers, so a
  background `EnsureFresh` is not interrupted mid-write. The 10-minute context
  timeout is the upper bound — in practice reindexing finishes much sooner.

### `formatSearchResults` update

Render `StaleWarning` in the text output, following the existing pattern for
`SeedWarning` and `FilteredHint`:

```go
if out.StaleWarning != "" {
    fmt.Fprintf(&b, "\nWarning: %s", out.StaleWarning)
}
```

### What does NOT change

- **ForceReindex path** — stays synchronous. It is explicitly requested by the
  user via `/lumen:reindex`, so blocking is expected.
- **Session-start background indexer** — works as before, acquires flock.
- **Flock check fast-path** (line 611) — still skips when lock is held.
- **Freshness TTL** — still skips merkle walks within TTL window.

## Files touched

| File | Change |
|------|--------|
| `cmd/stdio.go` | `SemanticSearchOutput` struct: add `StaleWarning` field |
| `cmd/stdio.go` | `ensureIndexed()`: replace synchronous `EnsureFresh` with timeout-guarded goroutine + flock |
| `cmd/stdio.go` | `indexerCache` struct: add `sync.WaitGroup` for background goroutine tracking |
| `cmd/stdio.go` | `Close()`: wait for background goroutines before closing indexers |
| `cmd/stdio.go` | `formatSearchResults()`: render `StaleWarning` in output text |

No new files. No new packages.

## Testing

- **Unit test**: Mock `EnsureFresh` to sleep > 15s, verify `StaleWarning` is set
  and results are returned from stale index.
- **Unit test**: Mock `EnsureFresh` to complete in < 15s, verify no
  `StaleWarning` and `Reindexed` is true.
- **Unit test**: Verify flock is acquired by the goroutine (subsequent calls see
  `IsHeld() == true`).
- **E2E test** (if feasible): Trigger reindex on a large fixture, verify search
  returns within ~15s with warning.

## Timeout value

Hardcoded at 15 seconds. No env var for now — YAGNI. Can be made configurable
via `LUMEN_SEARCH_TIMEOUT` later if needed.
