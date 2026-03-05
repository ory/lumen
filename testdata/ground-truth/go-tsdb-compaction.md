<!-- Source: https://prometheus.io/docs/prometheus/latest/storage/ -->
<!-- Validated against fixtures: 2026-03-05 -->

## Reference Documentation

Prometheus TSDB stores time series data in 2-hour blocks that are progressively
compacted into larger blocks spanning up to 10% of the retention time or 31
days (whichever is smaller). The `Compactor` interface defines three operations:
`Plan` (select blocks for compaction), `Write` (persist a block reader to disk),
and `Compact` (merge multiple blocks into one). `LeveledCompactor` implements
this using time-range-based level selection. The `DB` type orchestrates
compaction via `Compact()`, which handles head compaction (in-memory to disk),
out-of-order head compaction, and block-level compaction in sequence.

## Key Types in Fixtures

**compact.go:**
- `Compactor` — interface with Plan, Write, Compact methods
- `LeveledCompactor` — struct implementing Compactor with time-range-based levels
- `LeveledCompactorOptions` — configuration for LeveledCompactor
- `CompactorMetrics` — Prometheus metrics for compaction operations
- `BlockPopulator` — interface for merging series/chunks during compaction
- `DefaultBlockPopulator` — default implementation of BlockPopulator
- `CompactBlockMetas` — function that merges block metadata and increments level

**db.go:**
- `DB` — main database struct, holds a `compactor Compactor` field
- `NewCompactorFunc` — type for injecting custom Compactor implementations
- `Options` — DB options including compaction settings
- `DB.Compact` — orchestrates head, OOO, and block compaction
- `DB.compactHead` — persists head data to disk via compactor.Write
- `DB.compactBlocks` — iterates Plan/Compact until no work remains
- `DB.CompactStaleHead` — compacts stale series separately
- `DB.DisableCompactions`, `DB.EnableCompactions` — control auto-compaction
- `BlockExcludeFilterFunc` — type for filtering blocks from compaction

**head.go:**
- `Head` — in-memory TSDB head block
- `Head.compactable` — checks if head has enough data for compaction

## Required Facts

1. The `Compactor` interface defines exactly 3 methods: `Plan(dir string) ([]string, error)`, `Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)`, and `Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)`.
2. `LeveledCompactor` uses a `ranges []int64` field for time-range-based level selection.
3. `LeveledCompactor.Plan()` first checks `selectOverlappingDirs()` (if enabled), then falls back to `selectDirs()` for same-range blocks, then checks tombstone ratios (>5%).
4. `selectDirs()` requires at least 2 configured ranges and skips blocks marked with `Compaction.Failed`.
5. `selectDirs()` avoids compacting the most recent block prematurely — it uses `highTime` from the last block's MinTime.
6. `selectOverlappingDirs()` returns all blocks with overlapping time ranges when `enableOverlappingCompaction` is true.
7. `CompactBlockMetas()` merges metadata from input blocks and increments `Compaction.Level`, tracking parents and sources.
8. `Compact()` reuses already-open blocks (from the `open []*Block` parameter) to avoid loading duplicate indexes.
9. `Compact()` returns nil UIDs and marks source blocks deletable if the resulting block has zero samples.
10. `DB.Compact()` orchestrates three phases in sequence: head compaction, OOO head compaction, and block compaction.
11. `DB.compactHead()` calls `db.compactor.Write(db.dir, head, head.MinTime(), head.BlockMaxTime(), nil)` to persist in-memory data.
12. `DB.compactBlocks()` loops calling `compactor.Plan()` then `compactor.Compact()` until Plan returns no blocks.
13. Compaction is triggered by appender commit when `head.compactable()` returns true, sending a non-blocking signal to the `compactc` channel.
14. `DB.DisableCompactions()` and `DB.EnableCompactions()` control the `autoCompact` flag via `autoCompactMtx` mutex.
15. `NewCompactorFunc` type allows injecting custom Compactor implementations instead of the default LeveledCompactor.
16. `Options` includes `CompactionDelay`, `CompactionDelayMaxPercent`, and `EnableDelayedCompaction` for timing control.
17. `DefaultBlockPopulator.PopulateBlock()` merges series and chunks from multiple input blocks, writing to IndexWriter and ChunkWriter.
18. After head compaction, `head.truncateMemory()` and `head.RebuildSymbolTable()` are called to free in-memory data.

## Hallucination Traps

- There is NO `SimpleCompactor` type — only `LeveledCompactor` implements the `Compactor` interface in these fixtures.
- The `Compactor` interface does NOT handle WAL truncation — WAL management is handled by `Head` separately.
- There is NO remote storage compaction — all Plan/Write/Compact methods work on local directory paths.
- The `Compactor` does NOT decide which blocks to delete — `DB.reloadBlocks()` and `BlocksToDeleteFunc` handle deletion.
- Compaction is NOT triggered on every commit — only when `head.compactable()` returns true AND the channel send is non-blocking.
- There is NO separate background scheduler thread — compaction runs in a single goroutine with a channel-based trigger.
- `autoCompact` is NOT always enabled — it can be disabled via `DB.DisableCompactions()`.
- The `Compactor` does NOT handle block retention — retention is managed by `DB` separately.
