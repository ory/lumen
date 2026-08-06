# Index storage and lifecycle

Lumen stores indexes outside the source repository in repository-scoped,
content-addressed SQLite collections. This lets Git worktrees reuse unchanged
file revisions and embeddings without copying a complete database per worktree.

## Collection identity

The default location is:

```text
~/.local/share/lumen/<profile-hash>/index.db
```

`XDG_DATA_HOME` replaces `~/.local/share` when it is set. The profile hash is
derived from:

- the resolved Git common directory, or the absolute project path for a non-Git
  project;
- the indexed scope;
- embedding model and vector dimensions;
- vector storage (`int8` or `float32`);
- maximum chunk size; and
- Lumen's index-format version.

All worktrees whose settings resolve to the same profile use one physical
collection. Each worktree still has an independent project membership, Merkle
state, access time, and set of files. A different model, dimension count, vector
precision, chunk size, repository, or format version selects a different
collection automatically.

The embedding backend is not part of the profile. If the same model name is
served by both Ollama and LM Studio, use distinct configured model names unless
the services return compatible embeddings.

## What is shared

Lumen deduplicates at two levels inside a collection:

1. A file revision is identified by its relative path and content hash. When
   another worktree contains the same revision, Lumen attaches it directly.
2. A vector is identified by the SHA-256 hash of the exact embedding input:

   ```text
   // <relative-path>
   <chunk-content>
   ```

Including the relative path preserves Lumen's existing filepath-aware search
semantics. Identical text at different paths is therefore not assumed to have
the same vector.

When a file changes, Lumen chunks the new revision, checks which exact inputs
already exist, and sends only missing inputs to the embedding service. Unique
constraints make concurrent worktree indexing safe; two indexers may compute the
same embedding during a race, but only one stored vector remains.

Search remains project-local. Lumen scans collection vectors adaptively, then
joins candidates through the requested project's memberships and optional path
filter. Results from another worktree are never returned merely because the
underlying storage is shared.

## Vector storage

`int8` is the default:

```bash
export LUMEN_VECTOR_STORAGE=int8
```

Lumen max-absolute-normalizes each vector before quantizing it to signed bytes.
Cosine KNN then operates directly on sqlite-vec's int8 representation. This
substantially reduces vector storage while retaining the vector's direction.

Use float32 when you need the unquantized representation:

```bash
export LUMEN_VECTOR_STORAGE=float32
```

Changing this setting creates a separate profile rather than rewriting the
active collection in place. The same applies to the embedding model, dimensions,
and `LUMEN_MAX_CHUNK_TOKENS`.

## Reading `index_status`

The MCP `index_status` tool reports:

| Field                 | Scope           | Meaning                                                      |
| --------------------- | --------------- | ------------------------------------------------------------ |
| `total_files`         | Current project | Files found by the last completed indexing walk              |
| `indexed_files`       | Current project | File memberships stored for this project                     |
| `total_chunks`        | Current project | Chunk references reachable by this project                   |
| `unique_vectors`      | Collection      | Physical vectors stored once in the collection               |
| `shared_references`   | Collection      | Chunk references beyond the unique-vector count              |
| `deduplication_ratio` | Collection      | `shared_references / (unique_vectors + shared_references)`   |
| `vector_storage`      | Collection      | `int8` or `float32`                                          |
| `database_bytes`      | Collection      | Allocated SQLite database pages                              |
| `reclaimable_bytes`   | Collection      | Free SQLite pages that incremental vacuum may reclaim        |
| `last_indexed_at`     | Current project | Last completed indexing timestamp                            |
| `stale`               | Current project | Whether the source tree differs from the stored Merkle state |

For example, `10,000` unique vectors and `15,000` shared references means
`25,000` total chunk references are represented by `10,000` physical vectors,
for a deduplication ratio of 60%.

Collection-wide values can stay unchanged after indexing a second worktree even
though that worktree's project-local counts increase. That is the expected sign
that its revisions and vectors were reused.

## Legacy migration

Indexes created before shared collections used one float32 database per
worktree. Migration is lazy and automatic when Lumen first opens a project with
the new index format:

1. Lumen opens the legacy database read-only.
2. It verifies each candidate file still matches its stored content hash.
3. It re-chunks matching files to reconstruct the exact filepath-aware input
   hashes.
4. Matching vectors are reused, and quantized when the destination uses int8;
   only missing or changed inputs are embedded.
5. The legacy database is removed only after the new project's file hashes have
   been verified.

If legacy recovery cannot be completed, Lumen leaves the source database in
place and embeds the missing inputs normally.

## Cleanup and disk reclamation

Run cleanup manually with:

```bash
lumen clean            # memberships unused for 30 days or missing projects
lumen clean --days 7   # use a seven-day inactivity cutoff
lumen clean --days 0   # remove all cached indexes not actively being written
```

For shared collections, cleanup first removes stale project memberships, then
garbage-collects file revisions, chunks, and vectors that no remaining project
references. It incrementally vacuums free SQLite pages. The collection directory
is deleted when no projects remain. Legacy index directories use the same age
policy.

Opening a project for search, indexing, status, or session startup refreshes its
access time. An active indexer lock prevents cleanup from deleting a collection
being written.

The MCP server also performs this cleanup on startup, throttled to at most once
every 24 hours. The throttle stamp is stored at
`~/.local/share/lumen/.last-cleanup` (or below `XDG_DATA_HOME`).

`reclaimable_bytes` is a point-in-time estimate from SQLite's freelist. It may
be nonzero until incremental vacuum can truncate pages, and it does not include
WAL or filesystem allocation details.

## Reindexing versus wiping

Use the narrowest operation that matches the problem:

```bash
lumen index .           # refresh changed files
lumen index --force .   # reprocess every file in the current project
lumen clean             # reclaim stale memberships and unreferenced data
lumen clean --days 0    # wipe every cached index on the host
```

`--force` does not wipe other worktrees from a shared collection. It rebuilds
the current project's memberships and chunk definitions while the collection
continues to deduplicate physical vectors. Use the full wipe only when you
intend to rebuild all Lumen indexes on the machine.

To delete indexes manually, stop active Lumen indexers and remove the Lumen data
directory. No source-tree files are stored there, and no files are added to the
project repository.
