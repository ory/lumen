---
name: doctor
description: Run a health check on the bundled Lumen semantic search setup for the current project, verify backend reachability and index freshness, and summarize remediation steps.
---

# Lumen Doctor

Run a health check on the bundled Lumen semantic search setup for the current
project.

## Steps

1. Call the Lumen `health_check` tool to verify the embedding service is
   reachable.
2. Call the Lumen `index_status` tool with `path` or `cwd` set to the current
   working directory to check index freshness.
3. Report a concise summary:
   - Embedding service status, backend, host, and model
   - Index totals: files, chunks, last indexed time, stale or fresh
   - Any MCP or plugin setup issue that blocks the tools
4. If no index exists yet, explain that the Lumen `semantic_search` tool seeds
   the index on first use.
5. If the user wants eager indexing instead of waiting for the next search,
   suggest running `lumen index .` in the repository root.
