# Semantic Summaries Design

**Date:** 2026-03-15
**Branch:** add-semantic-summary
**Status:** Draft

## Overview

Add LLM-generated semantic summaries to Lumen's indexing and search pipeline. Chunks and files get natural-language descriptions embedded with a text embedding model. Search queries all three vector indices (raw code, chunk summaries, file summaries) and merges results. When a file-level summary matches strongly, the MCP response includes a `<relevant_files>` hint tag nudging the agent to read the full file.

## Architecture

### New Package: `internal/summarizer/`

Wraps LLM chat APIs (Ollama + LM Studio) to generate natural-language summaries. Mirrors the structure of `internal/embedder/`:

- `summarizer.go` — `Summarizer` interface + factory
- `ollama.go` — Ollama chat completion client
- `lmstudio.go` — LM Studio chat completion client
- `models.go` — known summary model registry

To avoid coupling `summarizer` to the `chunker` package, `SummarizeChunk` takes a lightweight struct rather than `chunker.Chunk`:

```go
// ChunkInfo carries only the fields needed for summarization.
type ChunkInfo struct {
    Kind    string
    Symbol  string
    Content string
}

type Summarizer interface {
    SummarizeChunk(ctx context.Context, chunk ChunkInfo) (string, error)
    SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error)
}
```

`internal/index/index.go` maps `chunker.Chunk` → `summarizer.ChunkInfo` before calling the summarizer. This keeps the package dependency graph clean.

**Known summary models:**

| Model | Notes |
|---|---|
| `qwen2.5-coder:7b` | Default |
| `qwen2.5-coder:32b` | Higher quality |
| `qwen3-coder:30b` | Highest quality |

Unknown models are accepted without validation (same behavior as embedder registry).

### Modified: `internal/store/`

Four new tables added to the existing SQLite DB. The `Store` struct gains a `summaryDims int` field and `createSchema` is updated to accept it. The summary vec tables are only created when `summaryDims > 0` (i.e., when `LUMEN_SUMMARIES=true`).

A `vec_summary_dimensions` key is added to `project_meta` alongside the existing `vec_dimensions`, and the same dimension-mismatch guard (`ensureVecDimensions`) is applied to both vec table pairs.

`DeleteFileChunks` is extended to explicitly delete from `vec_chunk_summaries` before deleting from `chunks`, mirroring the existing explicit delete from `vec_chunks`. sqlite-vec virtual tables do not participate in SQLite FK cascades, so cascade cannot be relied on for vec table cleanup. `file_summaries` rows are cleaned up via their FK cascade on `files`, but `vec_file_summaries` must also be explicitly deleted when a file is removed.

### Modified: `internal/index/index.go`

Two new passes added after the existing embedding pass. The `Indexer` struct gains a `summarizer summarizer.Summarizer` field and a `summaryEmb embedder.Embedder` field — a second embedder instance pointed at the summary embedding model. Both are constructed in `runStdio` (or `runIndex`) alongside the existing embedder and injected into `Indexer`.

### Modified: `cmd/stdio.go`

Search makes three sequential vector searches (not parallel — `Store` uses `MaxOpenConns(1)`, so goroutine-based parallelism would serialize anyway). File-level hits produce a `<relevant_files>` section in the MCP response.

`findEffectiveRoot` calls `config.DBPathForProject` — this function must be updated to accept both `codeEmbedModel` and `summaryEmbedModel` (empty string when summaries disabled). All callers of `DBPathForProject` must be updated accordingly: `cmd/stdio.go`, `cmd/index.go`, `cmd/purge.go`, `cmd/hook.go`, `internal/config/config_test.go`.

`cmd/hook.go` also calls `store.New(dbPath, cfg.Dims)` directly (to read stats for the session-start hook). When `Store.New` gains a `summaryDims int` parameter, this call site must be updated to pass `cfg.SummaryEmbedDims` (zero when `Summaries=false`).

## Configuration

Three new environment variables in `internal/config/config.go`:

| Variable | Default (Ollama) | Default (LM Studio) | Description |
|---|---|---|---|
| `LUMEN_SUMMARIES` | `false` | `false` | Enable semantic summarization |
| `LUMEN_SUMMARY_MODEL` | `qwen2.5-coder:7b` | `qwen2.5-coder:7b` | LLM for generating summaries |
| `LUMEN_SUMMARY_EMBED_MODEL` | `nomic-embed-text` | `nomic-ai/nomic-embed-text-GGUF` | Embedding model for summary vectors |

`LUMEN_SUMMARY_EMBED_MODEL` defaults differ by backend (analogous to how `LUMEN_EMBED_MODEL` has separate Ollama/LM Studio defaults) to ensure the default model is compatible with whichever backend is active.

`nomic-ai/nomic-embed-text-GGUF` does not yet exist in `KnownModels`. It must be added to `internal/embedder/models.go` as part of this work:

```go
"nomic-ai/nomic-embed-text-GGUF": {Dims: 768, CtxLength: 8192, MinScore: 0.30, Backend: "lmstudio"},
```

**Config struct additions:**

```go
type Config struct {
    // ... existing fields ...
    Summaries         bool
    SummaryModel      string
    SummaryEmbedModel string
    SummaryEmbedDims  int  // resolved from SummaryEmbedModel in KnownModels
}
```

`SummaryEmbedDims` is resolved the same way as `Dims` — looked up from `KnownModels`. `nomic-embed-text` is already in the registry (768 dims). If the model is unknown, a sensible fallback (768) is used and a warning logged.

## Data Model

```sql
-- Summary text per chunk
CREATE TABLE IF NOT EXISTS chunk_summaries (
    chunk_id TEXT PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
    summary  TEXT NOT NULL
);

-- Summary text per file (derived from chunk summaries)
CREATE TABLE IF NOT EXISTS file_summaries (
    file_path TEXT PRIMARY KEY REFERENCES files(path) ON DELETE CASCADE,
    summary   TEXT NOT NULL
);

-- Vector index for chunk summaries (only created when summaryDims > 0)
CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunk_summaries USING vec0(
    id        TEXT PRIMARY KEY,
    embedding float[{summary_dims}] distance_metric=cosine
);

-- Vector index for file summaries (only created when summaryDims > 0)
CREATE VIRTUAL TABLE IF NOT EXISTS vec_file_summaries USING vec0(
    id        TEXT PRIMARY KEY,  -- file_path used as ID
    embedding float[{summary_dims}] distance_metric=cosine
);
```

### Cleanup on File Removal

sqlite-vec virtual tables do not support FK cascade deletes. Cleanup must be explicit:

sqlite-vec virtual tables do not participate in FK cascades, so all four new tables require explicit cleanup. `DeleteFileChunks` in `store.go` is extended with the following three-phase ordering within a single transaction:

1. **Fetch chunk IDs** — `SELECT id FROM chunks WHERE file_path = ?` (must happen before chunks are deleted)
2. **Explicit vec deletes** — `DELETE FROM vec_chunk_summaries WHERE id IN (...)`, `DELETE FROM vec_file_summaries WHERE id = ?`, `DELETE FROM vec_chunks WHERE id IN (...)` (existing)
3. **Explicit row deletes** — `DELETE FROM chunks WHERE file_path = ?` (cascades to `chunk_summaries`), `DELETE FROM files WHERE path = ?` (cascades to `file_summaries`)

This order ensures chunk IDs are available for vec cleanup before the chunk rows are gone.

### Dimension Mismatch Reset

The existing `resetAndRecreateVecTable` / `ensureVecDimensions` path executes `DELETE FROM project_meta` as part of a full reset when code vector dimensions change. With summary vec tables also tracked in `project_meta` (via `vec_summary_dimensions`), a code-model dimension change that triggers a total reset must also drop and recreate `vec_chunk_summaries` and `vec_file_summaries`, and clear `vec_summary_dimensions` from `project_meta`. The implementation must expand the reset path to handle both vec table pairs atomically, so the DB is never left with orphaned summary vec tables after a reset.

### DB Path Hashing

`config.DBPathForProject` gains a `summaryEmbedModel string` parameter:

- **Summaries disabled:** `summaryEmbedModel` is passed as `""` → hash is `SHA-256(projectPath + codeEmbedModel + "" + IndexVersion)` — functionally identical to the current hash (backward compatible, no re-index for existing users)
- **Summaries enabled:** `summaryEmbedModel` is passed as e.g. `"nomic-embed-text"` → different hash → different DB path → clean full re-index (expected)

`IndexVersion` stays at `"2"` — new tables are additive and only exist in summary-enabled DBs.

## Indexing Pipeline

The existing pipeline is unchanged. Two new passes run after raw embedding, only when `LUMEN_SUMMARIES=true`:

```
Existing:
  file → chunk → split → merge → embed (code model) → store vec_chunks

New pass 1 — chunk summaries:
  chunk → filter (≥3 lines) → LLM summarize → embed (summary model) → store vec_chunk_summaries

New pass 2 — file summaries:
  per file: concatenate chunk summaries → LLM summarize → embed (summary model) → store vec_file_summaries
  (skip if file has no chunk summaries)
```

**Details:**

- **Chunk threshold:** chunks covering fewer than 3 lines are skipped — no summary generated, no summary embedding stored. The condition is `EndLine - StartLine < 2` (since a chunk on lines 5–7 has `EndLine - StartLine = 2` and covers 3 lines inclusively)
- **File summarization is hierarchical:** raw file content is never sent to the LLM. The file summary is produced by summarizing the chunk summaries for that file. This avoids context window limits entirely and produces accurate high-level descriptions
- **Files with no chunk summaries** (all chunks <3 lines) get no file summary — they are trivially simple
- **Incremental:** only chunks/files added or modified (per Merkle diff) are summarized. Unchanged files skip summarization entirely
- **LLM calls are sequential** — local models are not safely parallelizable
- **Summary embedding batching:** 32 texts per request (same as raw embedder)
- **Error handling:** if an LLM call fails for a chunk or file, log a warning and skip. Raw code search continues to work. Partial summary coverage is acceptable

**LLM prompts:**

```
Chunk:
  "Summarize what this {kind} '{symbol}' does in 2-3 sentences,
   focusing on its purpose and behavior:\n\n{content}"

File:
  "Summarize what this file does in 3-5 sentences, covering its
   main purpose, key types/functions, and role in the codebase:\n\n{chunk_summaries}"
```

## Search

When `LUMEN_SUMMARIES=true`, the search handler in `cmd/stdio.go` runs an expanded pipeline:

1. **Embed query twice** — once with the code embedder, once with the summary embedder (two sequential embed calls)
2. **Three sequential vector searches** (sequential because `Store` uses `MaxOpenConns(1)`):
   - `vec_chunks` with code query vector (existing)
   - `vec_chunk_summaries` with summary query vector, using `nomic-embed-text`'s `MinScore` from `KnownModels` as the distance threshold
   - `vec_file_summaries` with summary query vector, using the same `MinScore`
3. **Merge chunk results** — union of raw + chunk-summary hits, deduplicate by chunk ID, take max score
4. **Expand file hits** — for each file-level hit above `MinScore`, fetch top 4 chunks from that file by raw-code distance (using `vec_chunks`), convert each chunk's cosine distance to a score (`1.0 - distance`) and inject into result set using their own raw-code scores (not the file summary score); collect the file path for `<relevant_files>`
5. **Final dedup + re-rank** — deduplicate all chunk results by chunk ID, sort by score descending
6. **Return results** — existing format plus `<relevant_files>` for file-level hits

## MCP Response Format

When file-level summary hits are found, a `<relevant_files>` section is appended after search results:

```xml
<relevant_files>
  <file path="internal/auth/middleware.go" score="0.91"/>
  <file path="internal/auth/token.go" score="0.87"/>
</relevant_files>
```

This signals to the agent that reading these files in full may provide deeper context beyond the returned snippets.

## Testing

- Unit tests for `internal/summarizer/` — mock LLM responses, verify prompt construction and error handling
- Unit tests for new store methods — verify upsert, explicit cascade cleanup for vec tables, search queries, `vec_summary_dimensions` mismatch guard
- Integration tests for the indexing pipeline — verify chunk filtering (≥3 lines), hierarchical file summary generation, incremental summarization, correct cleanup of summary rows on file re-index
- E2E test — index a small fixture with `LUMEN_SUMMARIES=true`, run a search, verify `<relevant_files>` appears in response
