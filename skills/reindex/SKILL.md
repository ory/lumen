# Lumen Reindex

Reindexing is handled automatically by the SessionStart hook. To trigger a
fresh index, ask the user to open a new terminal session in the project
directory. The hook will detect stale state and reindex in the background.
