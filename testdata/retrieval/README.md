# Retrieval case catalog

One checked-in catalog drives every retrieval quality test, so relevance
judgments exist exactly once:

- `go test -tags fts5 ./internal/index -run TestRetrievalCatalogBM25` — offline
  BM25 through real indexing and search, no embedding service.
- `go test -tags fts5,e2e -run 'TestE2E_Catalog' .` — the same cases through the
  executable MCP server, in BM25 mode (no Ollama) and, when an embedding service
  is available, semantic mode (diagnostic only).
- `go test ./internal/evalcase` — validates the catalog against the source
  files: every judged string must exist in the named corpus file.

`LUMEN_EVAL_FILTER=<id prefix>` narrows a run; `LUMEN_EVAL_REPORT=<file>`
writes the Markdown report (per-language, per-kind hit@3/5/10, MRR and related
context recall, plus every case).

## Layout

- `cases/*.json` — arrays of cases, grouped by language or concern.
- `corpora/<name>/` — small connected fixtures written for this catalog.
  Vendored real-world fixtures stay in `testdata/fixtures/<language>` and
  `testdata/sample-project`; cases reference either by `corpus` (relative to
  `testdata/`).

## Case schema

| field         | meaning                                                                                                                       |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `id`          | stable, unique; `lang.<language>.<slug>` for transferred language-suite queries                                               |
| `corpus`      | directory under `testdata/` indexed as the project root                                                                       |
| `language`    | reporting group (`go`, `ts`, `cpp`, `svelte`, `markdown`, `multi`, ...)                                                       |
| `kind`        | `exact`, `control`, `syntax` (top 3); `task`, `structural`, `multistep`, `vocabulary` (top 5); `language` (top 10)            |
| `category`    | `supported` must pass; `diagnostic` is measured and reported but never fails                                                  |
| `origin`      | provenance: `lang-suite`, `sample-project`, `chunker-fixture`, `brainstorm-<n>`, ...                                          |
| `query`       | the query, unchanged from its origin                                                                                          |
| `path`        | optional corpus-relative path filter                                                                                          |
| `max_rank`    | overrides the kind default                                                                                                    |
| `expect`      | `"none"`: the query must return no direct hits                                                                                |
| `targets`     | any-of acceptable answers; the first matching **direct** result gives the rank                                                |
| `distractors` | results that must not rank above the best target                                                                              |
| `related`     | targets that must appear as one-hop related context (never counted as direct hits)                                            |
| `steps`       | multistep exploration; each later step's `requires` identifiers must appear in content returned by the previous step          |
| `note`        | why a case is diagnostic, or what it distinguishes                                                                            |

A target is `{file, symbol?, contains?, line?}`. `file` is corpus-relative with
forward slashes. A result matches when it is in that file, its symbol equals
`symbol` (or ends with `.symbol`), its span includes `line`, and its returned
content includes every `contains` string. Judgments name source content, not
chunk identities, so they hold regardless of how a retrieval mode splits or
merges declarations.

## Rules

- Freeze judgments against source before looking at retrieval output. Never
  edit a target, rank budget or category to hide a miss.
- A genuine vocabulary mismatch (no shared term between query and answer)
  stays in the catalog as `diagnostic` with a note.
- Every original language-suite query is present, even when a source audit
  shows the corpus has no valid answer (then it is `diagnostic`).
