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
   - If the index is empty or has never been indexed, tell the user to run
     `/lumen:reindex` to build the index
   - If the index is stale (out of date), tell the user to run
     `/lumen:reindex` to update it
   - If any issues found, suggest remediation (e.g. "reinstall the lumen
     plugin")
