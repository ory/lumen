---
name: doctor
description: Run a health check on the bundled Lumen semantic search setup for the current project, verify backend reachability and index freshness, and summarize remediation steps.
---

# Lumen Doctor

Run a health check on the bundled Lumen semantic search setup for the current
project.

## Steps

1. Call the Lumen `health_check` tool to verify the embedding backend is ready:
   a reachable Ollama or LM Studio service, or, for the `local` backend, an
   in-process model that is cached or will be downloaded on first use.
2. Call the Lumen `index_status` tool with `path` or `cwd` set to the current
   working directory to check index freshness.
3. Report a concise summary:
   - Embedding backend status, backend, host (`in-process` for `local`), and
     model
   - Index totals: files, chunks, last indexed time, stale or fresh
   - Storage totals: unique vectors, shared references, deduplication ratio,
     vector precision, database bytes, and reclaimable bytes
   - Any MCP or plugin setup issue that blocks the tools
4. If no index exists yet, explain that the Lumen `semantic_search` tool seeds
   the index on first use.
5. If the user wants eager indexing instead of waiting for the next search,
   suggest running `lumen index .` in the repository root.
6. Interpret storage totals as collection-wide values shared by compatible Git
   worktrees; file, chunk, freshness, and indexing values are project-local.
   A high deduplication ratio is expected and is not an index-health problem.
