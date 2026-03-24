# Lumen Doctor

Run a health check on the Lumen semantic search setup for the current project.

## Steps

1. Call mcp**lumen**health_check to verify the embedding service is reachable
2. Call mcp**lumen**index_status with path set to the current working directory
   to check index freshness
3. Report a summary:
   - Embedding service: status, backend, host, model
   - Index: total files, indexed files, chunks, stale or fresh, last indexed
     time
   - If MCP or plugin issues found (not index issues), suggest remediation (e.g.
     "reinstall the lumen plugin")
   - If the index is stale or does not exist, inform the user that reindexing
     is triggered automatically by the SessionStart hook — ask them to open a
     new terminal session in the project directory to kick it off.
