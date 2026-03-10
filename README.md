<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Demo](#demo)
- [Quick Start](#quick-start)
- [What You Get](#what-you-get)
- [How It Works](#how-it-works)
- [Benchmarks](#benchmarks)
- [Supported Languages](#supported-languages)
- [Configuration](#configuration)
  - [Supported Embedding Models](#supported-embedding-models)
- [Controlling What Gets Indexed](#controlling-what-gets-indexed)
- [Database Location](#database-location)
- [CLI Reference](#cli-reference)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

![Ory Lumen: Semantic code search for AI agents](.github/lumen-banner.png)

[![CI](https://github.com/ory/lumen/actions/workflows/ci.yml/badge.svg)](https://github.com/ory/lumen/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ory/lumen)](https://goreportcard.com/report/github.com/ory/lumen)
[![Go Reference](https://pkg.go.dev/badge/github.com/ory/lumen.svg)](https://pkg.go.dev/github.com/ory/lumen)
[![Coverage Status](https://coveralls.io/repos/github/ory/lumen/badge.svg?branch=main)](https://coveralls.io/github/ory/lumen?branch=main)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Claude reads entire files to find what it needs. Lumen gives it a map.

Lumen is a 100% local semantic code search engine for AI coding agents. No API
keys, no cloud, no external database, just open-source embedding models
([Ollama](https://ollama.com/) or [LM Studio](https://lmstudio.ai/)), SQLite,
and your CPU. A single static binary, no runtime required.

The payoff is measurable and reproducible: across 7 languages and 42 real GitHub
bug-fix tasks, Lumen reduces tool calls by **27% on average**, cuts PHP tokens
by **33–86% depending on run**, and cuts session cost by up to **56% on
JavaScript** tasks — all verified with a
[transparent, open-source benchmark framework](docs/BENCHMARKS.md).

|                                 | With Lumen                   | Baseline (no Lumen) |
| ------------------------------- | ---------------------------- | ------------------- |
| Tool calls (7-language avg)     | **21.8 avg** (-27%)          | 29.8 avg            |
| PHP tokens (avg, excl. outlier) | **166K avg** (-33%)          | 247K avg            |
| Rust hard tasks                 | **Good** (3/3 runs)          | Poor/Good/Poor      |
| JavaScript best case            | **$0.32, 163s** (-56%, -51%) | $0.72, 336s         |
| PHP best case                   | **$0.12, 36s** (-52%, -59%)  | $0.25, 90s          |

## Demo

<img src="docs/demo/demo.gif" alt="Lumen demo" width="600"/>

_Claude Code asking about the
[Prometheus](https://github.com/prometheus/prometheus) codebase. Lumen's
`semantic_search` finds the relevant code without reading entire files._

## Quick Start

**Prerequisites:**

1. [Ollama](https://ollama.com/) installed and running, then pull the default
   embedding model:
   ```bash
   ollama pull ordis/jina-embeddings-v2-base-code
   ```
2. [Claude Code](https://code.claude.com/docs/en/quickstart) installed

**Install:**

```bash
/plugin marketplace add ory/claude-plugins
/plugin install lumen@ory
```

That's it. On first session start, Lumen:

1. Downloads the binary automatically from the
   [latest GitHub release](https://github.com/ory/lumen/releases)
2. Indexes your project in the background using Merkle tree change detection
3. Registers a `semantic_search` MCP tool that Claude uses automatically

Two skills are also available: `/lumen:doctor` (health check) and
`/lumen:reindex` (forced re-indexing).

## What You Get

- **Semantic vector search** — Claude finds relevant functions, types, and
  modules by meaning, not keyword matching
- **Auto-indexing** — indexes on session start, only re-processes changed files
  via Merkle tree diffing
- **Incremental updates** — re-indexes only what changed; large codebases
  re-index in seconds after the first run
- **15 language families** — Go, Python, TypeScript, JavaScript, Rust, Ruby,
  Java, PHP, C/C++, C#, Markdown, YAML, JSON, TOML, Go module
- **Zero cloud** — embeddings stay on your machine; no data leaves your network
- **Ollama and LM Studio** — works with either local embedding backend

## How It Works

Lumen sits between your codebase and Claude as an MCP server. When a session
starts, it walks your project and builds a **Merkle tree** over file hashes:
only changed files get re-chunked and re-embedded. Each file is split into
semantic chunks (functions, types, methods) using Go's native AST or tree-sitter
grammars for other languages. Chunks are embedded and stored in **SQLite +
sqlite-vec** using cosine-distance KNN for retrieval.

```
Files → semantic chunks → vector embeddings → SQLite/sqlite-vec → KNN search
```

When Claude needs to understand code, it calls `semantic_search` instead of
reading entire files. The index is stored outside your repo
(`~/.local/share/lumen/<hash>/index.db`), keyed by project path and model name —
different models never share an index.

## Benchmarks

Lumen is evaluated using **bench-swe**: a SWE-bench-style harness that runs
Claude on real GitHub bug-fix tasks and measures cost, time, tool usage, and
patch quality — with and without Lumen. All results are reproducible: raw JSONL
streams, patch diffs, and judge ratings are committed to this repository.

**Key results** — 7 languages, hard difficulty, 42 total runs
(`ordis/jina-embeddings-v2-base-code`, Ollama):

| Metric                          | With Lumen                   | Baseline             |
| ------------------------------- | ---------------------------- | -------------------- |
| Tool calls (7-language avg)     | **21.8** (-27%)              | 29.8                 |
| PHP tokens (avg, excl. outlier) | **166K avg** (-33%)          | 247K avg             |
| Rust patch quality              | **Good** (3/3 runs)          | Mixed (2/3 Poor)     |
| JavaScript best case            | **$0.32, 163s** (-56%, -51%) | $0.72, 336s          |
| PHP best case                   | **$0.12, 36s** (-52%, -59%)  | $0.25, 90s           |
| Average cost (all tasks)        | $0.52                        | $0.51 (near-neutral) |

The **tool call reduction** is the most consistent signal across all languages.
Language-specific wins (PHP tokens, Rust quality, JavaScript cost/time) are
where Lumen delivers clear per-task value today. TypeScript (complex chunking)
is an active area of improvement.

See [docs/BENCHMARKS.md](docs/BENCHMARKS.md) for the full results table, all 42
runs, per-language breakdowns, and reproduce instructions.

## Supported Languages

Supports **15 language families** with semantic chunking:

| Language         | Parser      | Extensions                                | Benchmark status                             |
| ---------------- | ----------- | ----------------------------------------- | -------------------------------------------- |
| Go               | Native AST  | `.go`, `.mod`                             | Benchmarked: -27% tool calls                 |
| Python           | tree-sitter | `.py`                                     | Benchmarked: Perfect quality both ways       |
| TypeScript / TSX | tree-sitter | `.ts`, `.tsx`                             | Benchmarked: chunker needs improvement       |
| JavaScript / JSX | tree-sitter | `.js`, `.jsx`, `.mjs`                     | Benchmarked: -56% cost, -39% tool calls      |
| Rust             | tree-sitter | `.rs`                                     | Benchmarked: quality uplift (Poor→Good)      |
| Ruby             | tree-sitter | `.rb`                                     | Benchmarked: -81% tool calls                 |
| PHP              | tree-sitter | `.php`                                    | Benchmarked: -33% tokens avg, -86% best case |
| Java             | tree-sitter | `.java`                                   | Supported                                    |
| C#               | tree-sitter | `.cs`                                     | Supported                                    |
| C / C++          | tree-sitter | `.c`, `.h`, `.cpp`, `.cc`, `.cxx`, `.hpp` | Supported                                    |
| Markdown / MDX   | tree-sitter | `.md`, `.mdx`                             | Supported                                    |
| YAML             | tree-sitter | `.yaml`, `.yml`                           | Supported                                    |
| JSON / TOML      | structured  | `.json`, `.toml`                          | Supported                                    |

Go uses the native Go AST parser for the most precise chunks. All other
languages use tree-sitter grammars. See [docs/BENCHMARKS.md](docs/BENCHMARKS.md)
for full per-language benchmark data.

## Configuration

All configuration is via environment variables:

| Variable                 | Default           | Description                                |
| ------------------------ | ----------------- | ------------------------------------------ |
| `LUMEN_EMBED_MODEL`      | see note ¹        | Embedding model (must be in registry)      |
| `LUMEN_BACKEND`          | `ollama`          | Embedding backend (`ollama` or `lmstudio`) |
| `OLLAMA_HOST`            | `localhost:11434` | Ollama server URL                          |
| `LM_STUDIO_HOST`         | `localhost:1234`  | LM Studio server URL                       |
| `LUMEN_MAX_CHUNK_TOKENS` | `512`             | Max tokens per chunk before splitting      |

¹ `ordis/jina-embeddings-v2-base-code` (Ollama),
`nomic-ai/nomic-embed-code-GGUF` (LM Studio)

### Supported Embedding Models

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

## Controlling What Gets Indexed

Lumen filters files through six layers: built-in directory and lock file skips →
`.gitignore` → `.lumenignore` → `.gitattributes` (`linguist-generated`) →
supported file extension. Only files that pass all layers are indexed.

**`.lumenignore`** uses `.gitignore` syntax. Place it in your project root (or
any subdirectory) to exclude files that aren't in `.gitignore` but are noise for
code search — generated protobuf files, test snapshots, vendored data, etc.

<details>
<summary>Built-in skips (always excluded)</summary>

**Directories:** `.git`, `node_modules`, `vendor`, `dist`, `.cache`, `.venv`,
`__pycache__`, `target`, `.gradle`, `_build`, `deps`, `.idea`, `.vscode`,
`.next`, `.nuxt`, `.build`, `.output`, `bower_components`, `.bundle`, `.tox`,
`.eggs`, `testdata`, `.hg`, `.svn`

**Lock files:** `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `bun.lock`,
`bun.lockb`, `go.sum`, `composer.lock`, `poetry.lock`, `Pipfile.lock`,
`Gemfile.lock`, `Cargo.lock`, `pubspec.lock`, `mix.lock`, `flake.lock`,
`packages.lock.json`

</details>

## Database Location

Index databases are stored outside your project:

```
~/.local/share/lumen/<hash>/index.db
```

Where `<hash>` is derived from the absolute project path and embedding model
name. No files are added to your repo, no `.gitignore` modifications needed.

You can safely delete the entire `lumen` directory to clear all indexes, or use
`lumen purge` to do it automatically.

## CLI Reference

Download the binary from the
[GitHub releases page](https://github.com/ory/lumen/releases) or let the plugin
install it automatically.

```bash
lumen help
```

## Troubleshooting

**Ollama not running / "connection refused"**

Start Ollama and verify the model is pulled:

```bash
ollama serve
ollama pull ordis/jina-embeddings-v2-base-code
```

Run `/lumen:doctor` inside Claude Code to confirm connectivity.

**Stale index after large refactor**

Run `/lumen:reindex` inside Claude Code to force a full re-index, or:

```bash
lumen purge && lumen index .
```

**Switching embedding models**

Set `LUMEN_EMBED_MODEL` to a model from the supported table above. Each model
gets its own database; the old index is not deleted automatically.

**Slow first indexing**

The first run embeds every file. Subsequent runs only process changed files
(typically a few seconds). For large projects (100k+ lines), first indexing can
take several minutes — this is a one-time cost.

## Development

```bash
git clone https://github.com/ory/lumen.git
cd lumen

# Build (CGO required for sqlite-vec)
make build

# Run tests
make test

# Run linter
make lint

# Load as a Claude Code plugin from source
make plugin-dev
```

See [CLAUDE.md](CLAUDE.md) for architecture details, design decisions, and
contribution guidelines.
