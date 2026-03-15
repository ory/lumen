# Semantic Summaries Design

**Date:** 2026-03-15
**Branch:** add-semantic-summary
**Status:** Approved

## Overview

Add LLM-generated semantic summaries to Lumen's indexing and search pipeline. Chunks and files get natural-language descriptions embedded with a text embedding model. Search queries all three vector indices (raw code, chunk summaries, file summaries) and merges results. When a file-level summary matches strongly, the MCP response includes a `<relevant_files>` hint tag nudging the agent to read the full file.

## Architecture

### New Package: `internal/summarizer/`

Wraps LLM chat APIs (Ollama + LM Studio) to generate natural-language summaries. Mirrors the structure of `internal/embedder/`:

- `summarizer.go` — `Summarizer` interface + factory
- `ollama.go` — Ollama chat completion client
- `lmstudio.go` — LM Studio chat completion client
- `models.go` — known summary model registry

```go
type Summarizer interface {
    SummarizeChunk(ctx context.Context, chunk chunker.Chunk) (string, error)
    SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error)
}
```

**Known summary models:**

| Model | Notes |
|---|---|
| `qwen2.5-coder:7b` | Default |
| `qwen2.5-coder:32b` | Higher quality |
| `qwen3-coder:30b` | Highest quality |

Unknown models are accepted without validation (same behavior as embedder registry).

### Modified: `internal/store/`

Four new tables added to the existing SQLite DB via `CREATE TABLE IF NOT EXISTS` — fully additive, no migration needed.

### Modified: `internal/index/index.go`

Two new passes added after the existing embedding pass.

### Modified: `cmd/stdio.go`

Search merges results from all three vec tables. File-level hits produce a `<relevant_files>` section in the MCP response.

## Configuration

Three new environment variables in `internal/config/config.go`:

| Variable | Default | Description |
|---|---|---|
| `LUMEN_SUMMARIES` | `false` | Enable semantic summarization |
| `LUMEN_SUMMARY_MODEL` | `qwen2.5-coder:7b` | LLM for generating summaries |
| `LUMEN_SUMMARY_EMBED_MODEL` | `nomic-embed-text` | Embedding model for summary vectors |

**Config struct additions:**

```go
type Config struct {
    // ... existing fields ...
    Summaries         bool
    SummaryModel      string
    SummaryEmbedModel string
    SummaryEmbedDims  int  // resolved from SummaryEmbedModel registry
}
```

`nomic-embed-text` is already in the embedder model registry (768 dims), so `SummaryEmbedDims` resolves automatically.

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

-- Vector index for chunk summaries
CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunk_summaries USING vec0(
    id        TEXT PRIMARY KEY,
    embedding float[{summary_dims}] distance_metric=cosine
);

-- Vector index for file summaries
CREATE VIRTUAL TABLE IF NOT EXISTS vec_file_summaries USING vec0(
    id        TEXT PRIMARY KEY,  -- file_path used as ID
    embedding float[{summary_dims}] distance_metric=cosine
);
```

### DB Path Hashing

- **Summaries disabled (default):** `SHA-256(projectPath + codeEmbedModel + IndexVersion)` — unchanged, backward compatible, no re-index
- **Summaries enabled:** `SHA-256(projectPath + codeEmbedModel + summaryEmbedModel + IndexVersion)` — different DB path, triggers clean full re-index (expected)

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

- **Chunk threshold:** chunks with fewer than 3 lines are skipped — no summary generated, no summary embedding stored
- **File summarization is hierarchical:** raw file content is never sent to the LLM. The file summary is produced by summarizing the chunk summaries for that file. This avoids context window limits entirely and produces accurate high-level descriptions
- **Files with no chunk summaries** (all chunks <3 lines) get no file summary — they are trivially simple
- **Incremental:** only chunks/files added or modified (per Merkle diff) are summarized. Unchanged files skip summarization
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

1. **Embed query twice** — once with the code embedder, once with the summary embedder (two embed calls)
2. **Three parallel vector searches:**
   - `vec_chunks` with code query vector (existing)
   - `vec_chunk_summaries` with summary query vector
   - `vec_file_summaries` with summary query vector
3. **Merge chunk results** — union of raw + chunk-summary hits, deduplicate by chunk ID, take max score
4. **Expand file hits** — for each file-level hit above MinScore, fetch top 4 chunks from that file by raw-code score, inject into result set with the file's summary score
5. **Final dedup + re-rank** — deduplicate, sort by score descending, apply existing 4-chunks-per-file cap
6. **Return results** — existing format plus `<relevant_files>` for file-level hits

## MCP Response Format

When file-level summary hits are found, a `<relevant_files>` section is appended after search results:

```xml
<relevant_files>
  <file path="internal/auth/middleware.go" reason="File-level semantic match" score="0.91"/>
  <file path="internal/auth/token.go" reason="File-level semantic match" score="0.87"/>
</relevant_files>
```

This signals to the agent that reading these files in full may provide deeper context beyond the returned snippets.

## Testing

- Unit tests for `internal/summarizer/` — mock LLM responses, verify prompt construction and error handling
- Unit tests for new store methods — verify upsert, cascade delete, search queries
- Integration tests for the indexing pipeline — verify chunk filtering (≥3 lines), hierarchical file summary generation, incremental summarization
- E2E test — index a small fixture with `LUMEN_SUMMARIES=true`, run a search, verify `<relevant_files>` appears in response
