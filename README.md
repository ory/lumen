![Ory Lumen: Semantic code search for AI agents](.github/lumen-banner.png)

[![CI](https://github.com/ory/lumen/actions/workflows/ci.yml/badge.svg)](https://github.com/ory/lumen/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ory/lumen)](https://goreportcard.com/report/github.com/ory/lumen)
[![Go Reference](https://pkg.go.dev/badge/github.com/ory/lumen.svg)](https://pkg.go.dev/github.com/ory/lumen)
[![Coverage Status](https://coveralls.io/repos/github/ory/lumen/badge.svg?branch=main)](https://coveralls.io/github/ory/lumen?branch=main)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Claude reads entire files to find what it needs. Lumen gives it a map.

Lumen is a local code search engine for AI coding agents. Its default uses
code-aware BM25 in SQLite: no API keys, model download, or embedding server.
It returns symbol-sized snippets and a small amount of related local code.
Embedding-based semantic search remains an explicit opt-in.

The benchmark figures below describe the previous **semantic retrieval** setup,
not the new BM25 default. See the [retrieval evaluation](docs/BENCHMARKS.md#retrieval-evaluation)
for a reproducible lexical baseline and structural-context comparison.

The payoff is measurable and reproducible: across 9 benchmark runs on 9
languages and real GitHub bug-fix tasks, Lumen cuts cost in **every single
language** — up to 39%. Output tokens drop by up to 66%, sessions complete up to
53% faster, and patch quality is maintained in every task. All verified with a
[transparent, open-source benchmark framework](docs/BENCHMARKS.md) that you can
run yourself.

|                        | With Lumen                    | Baseline (no Lumen)  |
| ---------------------- | ----------------------------- | -------------------- |
| Cost (avg, bug-fix)    | **$0.29** (-26%)              | $0.40                |
| Time (avg, bug-fix)    | **125s** (-28%)               | 174s                 |
| Output tokens (avg)    | **5,247** (-37%)              | 8,323                |
| JavaScript (marked)    | **$0.32, 119s** (-33%, -53%)  | $0.48, 255s          |
| Rust (toml)            | **$0.38, 204s** (-39%, -34%)  | $0.61, 310s          |
| PHP (monolog)          | **$0.14, 34s** (-27%, -34%)   | $0.19, 52s           |
| TypeScript (commander) | **$0.14, 56s** (-27%, -33%)   | $0.19, 84s           |
| Svelte (chat-ui)       | **$0.10, 56s** (-26%, -31%)   | $0.14, 80s           |
| Patch quality          | **Maintained in all 9 tasks** | —                    |

## Table of contents

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Demo](#demo)
- [Quick start](#quick-start)
- [What you get](#what-you-get)
- [How it works](#how-it-works)
- [Benchmarks](#benchmarks)
- [Supported languages](#supported-languages)
- [Configuration](#configuration)
  - [Supported embedding models](#supported-embedding-models)
- [Controlling what gets indexed](#controlling-what-gets-indexed)
- [Database location](#database-location)
- [CLI Reference](#cli-reference)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Demo

<img src="docs/demo/demo.gif" alt="Lumen demo" width="600"/>

_Claude Code asking about the
[Prometheus](https://github.com/prometheus/prometheus) codebase. Lumen's
`semantic_search` finds the relevant code without reading entire files._

## Quick start

**Prerequisites:**

> **Platform support:** Linux, macOS, and Windows. File locking for background
> indexing coordination uses `flock(2)` on Unix and `LockFileEx` on Windows
> (via [gofrs/flock](https://github.com/gofrs/flock)).

Install one of [Claude Code](https://code.claude.com/docs/en/quickstart),
[Cursor](https://cursor.com/), [Codex](https://developers.openai.com/codex/cli),
or [OpenCode](https://opencode.ai/). No embedding service is needed for the default.

**Note:** Installation differs by platform. Claude Code and Codex install from
plugin marketplaces. OpenCode installs from npm. Cursor packaging is shipped
in this repository and is ready for Cursor's plugin distribution workflow.

**Install:**

**Claude Code**

```bash
/plugin marketplace add ory/claude-plugins
/plugin install lumen@ory
```

Verify by starting a new Claude session and running `/lumen:doctor`.

**Cursor**

Lumen ships a native Cursor plugin bundle in this repository:

- `.cursor-plugin/plugin.json` - plugin manifest
- `mcp.json` - local `lumen` MCP server wiring
- `hooks/hooks-cursor.json` - SessionStart hook
- `skills/` - shared `doctor` and `reindex` skills

Use Cursor's plugin installation or distribution workflow with this bundle.
Detailed packaging notes: [.cursor-plugin/INSTALL.md](.cursor-plugin/INSTALL.md)

Verify by opening a new Cursor agent session and asking it to use the `doctor`
skill or the Lumen `semantic_search` tool.

**Codex**

Codex CLI 0.147.0 or newer installs Lumen as a native plugin:

```bash
codex plugin marketplace add ory/claude-plugins
codex plugin add lumen@ory
```

If the marketplace already exists, run `codex plugin marketplace upgrade ory`
before installing. Legacy manual-clone and broken-plugin repair instructions:
[.codex/INSTALL.md](.codex/INSTALL.md).

Verify with:

```bash
codex mcp get lumen --json
```

**OpenCode**

Add `@ory/lumen-opencode` to the `plugin` array in your `opencode.json`:

```json
{
  "plugin": ["@ory/lumen-opencode"]
}
```

Detailed docs: [.opencode/INSTALL.md](.opencode/INSTALL.md)

Verify with:

```bash
opencode mcp list
```

**Updating**

- **Claude Code** - update through Claude's plugin marketplace
- **Cursor** - refresh or reinstall the bundled plugin through Cursor after
  updating this repository or the published package
- **Codex** - upgrade the `ory` marketplace, reinstall `lumen@ory`, and restart
- **OpenCode** - update the version pin in `opencode.json` (e.g.
  `@ory/lumen-opencode@0.0.29`) and restart OpenCode

On first Claude Code or Cursor session start, Lumen:

1. Downloads the binary automatically from the
   [latest GitHub release](https://github.com/ory/lumen/releases)
2. Indexes your project in the background using Merkle tree change detection
3. Registers a `semantic_search` MCP tool that the host can use automatically

In Codex and OpenCode, the same binary download and index seeding happen on the
first `semantic_search` call. Codex stores the downloaded binary in the
plugin's writable data directory rather than the read-only package cache.

Two shared skills are also available: `doctor` (health check) and `reindex`
(forced re-indexing). Claude exposes them as `/lumen:doctor` and
`/lumen:reindex`; the other hosts discover the same shared skill content
through their native skill systems.

The same `semantic_search`, `health_check`, and `index_status` MCP tools plus
the shared `doctor` and `reindex` skills are exposed through the Codex,
Cursor, and OpenCode surfaces as well. The first `semantic_search` call seeds
or refreshes the index automatically.

## What you get

- **Code-aware BM25** — weighted paths, symbols, signatures, comments and bodies;
  complete identifiers are preserved and also split into words
- **Bounded related context** — at most two extra local snippets from one-hop
  direct calls, within 6000 bytes, starting from the top three direct matches.
  Calls resolve only through syntax: same-file functions, Go module imports,
  and relative JS/TS imports of named, `export { … }` or default exports
  (`./x.js` resolves to `x.ts`; ambiguous modules are skipped)
- **Incremental indexing** — Merkle hashes identify changed files
- **12 language families** — Go, Python, TypeScript, JavaScript, Svelte, Rust,
  Ruby, Java, PHP, C/C++, C#, Dart
- **Git worktree support** — unchanged revisions share index entries
- **Optional semantic retrieval** — explicitly select Ollama or LM Studio embeddings

## How it works

Lumen splits source into functions, types and methods using native Go AST and
tree-sitter parsers. SQLite FTS5 ranks fields with BM25 and exact symbol/path
boosts. Fresh snippets are read from the active worktree after hash validation.

```text
Files → symbol chunks + identifiers → shared FTS5 index → BM25 matches → bounded related context
```

The MCP tool retains the name `semantic_search` for compatibility. Default
searches need no model or backend connection. `lumen search --lexical-only "query"`
or MCP `lexical_only: true` disables related context for a lexical baseline.
Related snippets carry `related_to` and `relation`, not a relevance score.
BM25 scores are positive ranking values, not cosine similarity or probabilities;
omit `min_score` (positive thresholds require semantic mode).

Expansion conservatively follows unambiguous direct functions in the same file,
plus directly exported named local JavaScript/TypeScript functions and exported
Go functions reached through local-package aliases.
It is not a full call graph: ambiguous dispatch, external dependencies and
unproven reexports are skipped. Structured data and Markdown use lexical search
only. No Go-to-JavaScript HTTP traversal is inferred.

Vocabulary mismatch remains a limitation: a query about duplicate payments may
miss `IdempotencyStore` without matching words or a discoverable calling function.
The new baseline is evaluated rather than assumed equivalent to embeddings.

Indexes stay outside the repository. Compatible Git worktrees share a collection
but retain separate memberships; lexical profiles are independent of models.

## Benchmarks

Lumen is evaluated using **bench-swe**: a SWE-bench-style harness that runs
Claude on real GitHub bug-fix tasks and measures cost, time, output tokens, and
patch quality — with and without Lumen. All results are reproducible: raw JSONL
streams, patch diffs, and judge ratings are committed to this repository.

**Key results** — 9 runs across 9 languages, hard difficulty, real GitHub
issues (`ordis/jina-embeddings-v2-base-code`, Ollama):

| Language   | Cost Reduction | Time Reduction | Output Token Reduction  | Quality        |
| ---------- | -------------- | -------------- | ----------------------- | -------------- |
| Rust       | **-39%**       | **-34%**       | **-31%** (18K → 12K)    | Poor (both)    |
| JavaScript | **-33%**       | **-53%**       | **-66%** (14K → 5K)     | Perfect (both) |
| TypeScript | **-27%**       | **-33%**       | **-64%** (5K → 1.8K)    | Good (both)    |
| PHP        | **-27%**       | **-34%**       | **-59%** (1.9K → 0.8K)  | Good (both)    |
| Ruby       | **-24%**       | **-11%**       | -9% (6.1K → 5.6K)       | Good (both)    |
| Python     | **-20%**       | **-29%**       | **-36%** (1.7K → 1.1K)  | Perfect (both) |
| Go         | **-12%**       | -9%            | -10% (11K → 10K)         | Good (both)    |
| C++        | **-8%**        | -3%            | +42% (feature task)      | Good (both)    |
| Svelte     | **-26%**       | **-31%**       | **-26%** (4.0K → 3.0K)  | Poor (both)    |

**Cost was reduced in every language tested. Quality was maintained in every
task — zero regressions.** JavaScript and TypeScript show the most dramatic
efficiency gains: same quality fixes in half the time with two-thirds fewer
tokens. Even on tasks too hard for either approach (Rust, Svelte), Lumen cuts
the cost of failure by 26–39%.

See [docs/BENCHMARKS.md](docs/BENCHMARKS.md) for all 9 per-language deep dives,
judge rationales, and reproduce instructions.

## Supported languages

Supports **12 language families** with semantic chunking (10 benchmarked):

| Language         | Parser      | Extensions                                | Benchmark status                              |
| ---------------- | ----------- | ----------------------------------------- | --------------------------------------------- |
| Go               | Native AST  | `.go`                                     | Benchmarked: -12% cost, Good quality          |
| Python           | tree-sitter | `.py`                                     | Benchmarked: Perfect quality, -36% tokens     |
| TypeScript / TSX | tree-sitter | `.ts`, `.tsx`, `.mts`, `.cts`             | Benchmarked: -64% tokens, -33% time           |
| JavaScript / JSX | tree-sitter | `.js`, `.jsx`, `.mjs`, `.cjs`             | Benchmarked: -66% tokens, -53% time           |
| Dart             | tree-sitter | `.dart`                                   | Benchmarked: -76% cost, -82% tokens, -79% time |
| Rust             | tree-sitter | `.rs`                                     | Benchmarked: -39% cost, -34% time             |
| Ruby             | tree-sitter | `.rb`                                     | Benchmarked: -24% cost, -11% time             |
| PHP              | tree-sitter | `.php`                                    | Benchmarked: -59% tokens, -34% time           |
| C / C++          | tree-sitter | `.c`, `.h`, `.cpp`, `.cc`, `.cxx`, `.hpp` | Benchmarked: -8% cost (C++ feature task)      |
| Svelte           | tree-sitter | `.svelte`                                 | Benchmarked: -26% cost, -31% time             |
| Java             | tree-sitter | `.java`                                   | Supported                                     |
| C#               | tree-sitter | `.cs`                                     | Supported                                     |

Go uses the native Go AST parser for the most precise chunks. All other
languages use tree-sitter grammars. Svelte components contribute script
declarations and statements plus template and style regions. Markdown/MDX
files are split into fence-aware H1–H6 sections, and YAML/JSON into key-path
chunks with source-accurate line ranges. Retrieval quality is tracked by a
shared, source-judged case catalog — see
[testdata/retrieval/README.md](testdata/retrieval/README.md). See [docs/BENCHMARKS.md](docs/BENCHMARKS.md)
for all 10 per-language benchmark deep dives.

## Configuration

Set `retrieval: bm25` or `retrieval: semantic` in configuration, use
`LUMEN_RETRIEVAL`, or pass `--retrieval` on the CLI. Default: `bm25`.
Restart the MCP server after changing retrieval mode.

For optional semantic mode, run your embedding service and pull its model, then:

```bash
ollama pull ordis/jina-embeddings-v2-base-code
lumen --retrieval semantic index .
lumen --retrieval semantic search "refresh access token"
```

Embedding settings below apply only to semantic retrieval:

| Variable                 | Default                  | Description                                                   |
| ------------------------ | ------------------------ | ------------------------------------------------------------- |
| `LUMEN_RETRIEVAL`       | `bm25`                   | Retrieval mode: `bm25` or `semantic`                           |
| `LUMEN_EMBED_MODEL`      | see note ¹               | Embedding model; use with `LUMEN_EMBED_DIMS` for unlisted models |
| `LUMEN_BACKEND`          | `ollama`                 | Embedding backend (`ollama` or `lmstudio`)                    |
| `OLLAMA_HOST`            | `http://localhost:11434` | Ollama server URL                                             |
| `LM_STUDIO_HOST`         | `http://localhost:1234`  | LM Studio server URL                                          |
| `LUMEN_MAX_CHUNK_TOKENS` | `512`                    | Max tokens per chunk before splitting                         |
| `LUMEN_VECTOR_STORAGE`   | `int8`                   | Vector precision (`int8` or `float32`)                         |
| `LUMEN_EMBED_DIMS`       | —                        | Override embedding dimensions (required for unlisted models)  |
| `LUMEN_EMBED_CTX`        | `8192` (unlisted models) | Override context window length                                |

¹ `ordis/jina-embeddings-v2-base-code` (Ollama),
`nomic-ai/nomic-embed-code-GGUF` (LM Studio)

### Supported embedding models

Dimensions and context length are configured automatically per model:

| Model                                | Backend   | Dims | Context | Recommended                                                           |
| ------------------------------------ | --------- | ---- | ------- | --------------------------------------------------------------------- |
| `ordis/jina-embeddings-v2-base-code` | Ollama    | 768  | 8192    | **Best default** — lowest cost, no over-retrieval                     |
| `qwen3-embedding:8b`                 | Ollama    | 4096 | 40960   | **Best quality** — strongest dominance (7/9 wins), very slow indexing |
| `nomic-ai/nomic-embed-code-GGUF`     | LM Studio | 3584 | 8192    | **Usable** — good quality, but TypeScript over-retrieval raises costs |
| `qwen3-embedding:4b`                 | Ollama    | 2560 | 40960   | **Not recommended** — highest costs, severe TypeScript over-retrieval |
| `nomic-embed-text`                   | Ollama    | 768  | 8192    | Untested                                                              |
| `qwen3-embedding:0.6b`               | Ollama    | 1024 | 32768   | Untested                                                              |
| `all-minilm`                         | Ollama    | 384  | 512     | Untested                                                              |

Switching models creates a separate index automatically. The model name is part
of the database path hash, so different models never collide.

> **Caveat**: the DB path hash includes the model name but not the backend. If
> the same model name is configured on two backends (e.g. an Ollama and an LM
> Studio entry both named `foo`), they share the same index — use distinct
> model names per backend to avoid collisions.

### Selecting a server per invocation

`lumen index` and `lumen search` accept `--model`/`-m` and `--backend`/`-b`
to pick from a multi-server `config.yaml`. The selection filters the
configured servers to those matching both fields; failover still works
within the filtered subset. Both flags require `--retrieval semantic`; in the
default BM25 mode they are rejected instead of silently ignored.

```sh
# Index with the Ollama server matching this model name.
lumen --retrieval semantic index --model ordis/jina-embeddings-v2-base-code .

# Same model name hosted on LM Studio (present in YAML, not in the
# static registry) — accepted because the name is configured.
lumen --retrieval semantic index --model text-embedding-jina-embeddings-v2-base-code .

# Disambiguate when the same model is configured on two backends.
lumen --retrieval semantic index --model my-embed --backend lmstudio .

# Pick the first configured Ollama server regardless of model.
lumen --retrieval semantic search --backend ollama "…"
```

If `--model` is not configured in YAML but is a known registry model (and
`--backend` is unset), Lumen falls back to mutating the default server's
model — preserving `lumen --retrieval semantic index --model all-minilm .` for users with no YAML.

### Using a custom or unlisted model

If your model is not in the registry above, set `LUMEN_EMBED_DIMS` to bypass the
registry check. `LUMEN_EMBED_CTX` is optional and defaults to `8192`.

Both variables can also override values for _known_ models — useful when running
a model variant with a longer context window or different output dimensions.

```sh
LUMEN_BACKEND=lmstudio
LM_STUDIO_HOST=http://localhost:8801
LUMEN_EMBED_MODEL=mlx-community/Qwen3-Embedding-8B-4bit-DWQ
LUMEN_EMBED_DIMS=4096
LUMEN_EMBED_CTX=40960   # optional, defaults to 8192
```

## Controlling what gets indexed

Lumen filters files through six layers: built-in directory and lock file skips →
`.gitignore` → `.lumenignore` → `.gitattributes` (`linguist-generated`) →
supported file extension. Only files that pass all layers are indexed.

**`.lumenignore`** uses `.gitignore` syntax. Place it in your project root (or
any subdirectory) to exclude files that aren't in `.gitignore` but are noise for
code search — generated protobuf files, test snapshots, vendored data, etc.

<details>
<summary>Built-in skips (always excluded)</summary>

**Directories:** `.git`, `node_modules`, `vendor`, `dist`, `.cache`, `.venv`,
`venv`, `__pycache__`, `target`, `.gradle`, `_build`, `deps`, `.idea`,
`.vscode`, `.next`, `.nuxt`, `.build`, `.output`, `bower_components`, `.bundle`,
`.tox`, `.eggs`, `testdata`, `.hg`, `.svn`

**Lock files:** `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `bun.lock`,
`bun.lockb`, `go.sum`, `composer.lock`, `poetry.lock`, `Pipfile.lock`,
`Gemfile.lock`, `Cargo.lock`, `pubspec.lock`, `mix.lock`, `flake.lock`,
`packages.lock.json`

</details>

## Database location

Index databases are stored outside your project:

```
~/.local/share/lumen/<hash>/index.db
```

The hash identifies repository, indexed scope, chunking profile and format.
BM25 uses a model-independent profile. Semantic profiles additionally include
model, dimensions and vector precision (`int8` by default). Worktrees share
content-addressed revisions while retaining separate memberships. No files are
added to your repository. Existing vector collections remain available in
semantic mode; selecting BM25 does not destructively migrate them.

You can safely delete the entire `lumen` directory to clear all indexes, or let
Lumen reclaim the space for you:

```bash
lumen clean            # remove indexes unused for 30 days or whose project is gone
lumen clean --days 7   # tighten the cutoff to a week
lumen clean --days 0   # remove every eligible index except actively locked indexes
```

An index counts as used every time Lumen opens it (search, indexing, status, or
session start), so indexes for projects you still work on are never removed.
Indexes with an indexer currently running are always kept.

**Git worktrees** attach unchanged path-and-content revisions directly from the
repository collection. Removing an old worktree drops its memberships; shared
entries remain until their final reference disappears. Semantic mode retains
its existing legacy vector migration path.

For the complete storage key, sharing rules, status metrics, migration process,
and cleanup lifecycle, see [Index storage and lifecycle](docs/INDEX_STORAGE.md).

## CLI Reference

Download the binary from the
[GitHub releases page](https://github.com/ory/lumen/releases) or let the plugin
install it automatically.

```bash
lumen help
```

## Troubleshooting

**Ollama not running / "connection refused"**

This applies only to `--retrieval semantic`. For the default BM25 mode, no
Ollama service is needed. In semantic mode, start Ollama and pull the model:

```bash
ollama serve
ollama pull ordis/jina-embeddings-v2-base-code
```

Run `/lumen:doctor` inside Claude Code to confirm connectivity.

In Cursor, Codex, or OpenCode, use the shared `doctor` skill or call
`health_check` and `index_status` directly.

**Stale index after large refactor**

Run `/lumen:reindex` inside Claude Code to force a full re-index, or:

```bash
lumen index --force .
```

In Codex, use the bundled `reindex` skill to refresh the index through the MCP
server, or run the same CLI commands for a clean rebuild. The same shared
`reindex` skill is available in Cursor and OpenCode as well.

**LM Studio: embedding model appears under LLMs instead of Embeddings**

LM Studio classifies embedding models by matching the GGUF `arch` field against
a hardcoded allowlist (`bert`, `nomic-bert`). Models built on other
architectures — including Qwen2-based models like `nomic-embed-code` — are
misclassified as LLMs. This affects `lms ls` output and the `/v1/embeddings`
REST endpoint.

**Fix (GGUF, v0.3.16+):** Open LM Studio → My Models, click the gear icon next
to the model, set **Override Domain Type** → **Text Embedding**.

> **macOS / Apple Silicon:** MLX format models are significantly faster on Apple
> Silicon. However, LM Studio removed the domain type override for MLX in
> v0.3.30+, so MLX embedding models cannot be reclassified. Use the GGUF
> variant to retain the override option, or switch to Ollama
> (`ordis/jina-embeddings-v2-base-code` or `qwen3-embedding:8b`).

**Switching embedding models**

Set `LUMEN_EMBED_MODEL` to a model from the supported table above. Each model
gets its own database; the old index is not deleted automatically.

Changing `LUMEN_VECTOR_STORAGE`, `LUMEN_EMBED_DIMS`, or `LUMEN_MAX_CHUNK_TOKENS`
also selects a separate collection. Run `lumen clean` after the old profile is
no longer in use if you want to reclaim its disk space.

**Understanding index size and deduplication**

Call `index_status` for the project. It reports project-local file and chunk
counts alongside collection-wide lexical entries (or semantic vectors), shared
references, deduplication ratio, database size, and currently reclaimable
SQLite pages. See
[Index storage and lifecycle](docs/INDEX_STORAGE.md#reading-index-status) for
definitions and examples.

**Slow first indexing**

The first run parses and indexes every eligible file. Subsequent runs process
changed files. Semantic mode additionally embeds new chunks and can take
considerably longer depending on the model and hardware.

## Development

```bash
git clone https://github.com/ory/lumen.git
cd lumen

# Build locally (CGO required for sqlite-vec)
make build-local

# Run tests
make test

# Run linter
make lint

# Load as a Claude Code plugin from source
make plugin-dev
```

See [CLAUDE.md](CLAUDE.md) for architecture details, design decisions, and
contribution guidelines, and [AGENTS.md](AGENTS.md) for repo-specific agent
instructions.
