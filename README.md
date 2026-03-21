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
and your CPU. A single static binary and your own local embedding server.

The payoff is measurable and reproducible: across 8 benchmark runs on 8
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
| Patch quality          | **Maintained in all 8 tasks** | —                    |

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

> **Platform support:** Linux and macOS only. Windows is not currently supported
> because background indexing coordination uses `flock(2)`, a Unix advisory
> file-locking primitive not available on Windows.

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

## What you get

- **Semantic vector search** — Claude finds relevant functions, types, and
  modules by meaning, not keyword matching
- **Auto-indexing** — indexes on session start, only re-processes changed files
  via Merkle tree diffing
- **Incremental updates** — re-indexes only what changed; large codebases
  re-index in seconds after the first run
- **11 language families** — Go, Python, TypeScript, JavaScript, Rust, Ruby,
  Java, PHP, C/C++, C#
- **Git worktree support** — worktrees share index data automatically; a new
  worktree seeds from a sibling's index and only re-indexes changed files,
  turning minutes of embedding into seconds
- **Zero cloud** — embeddings stay on your machine; no data leaves your network
- **Ollama and LM Studio** — works with either local embedding backend

## How it works

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
Claude on real GitHub bug-fix tasks and measures cost, time, output tokens, and
patch quality — with and without Lumen. All results are reproducible: raw JSONL
streams, patch diffs, and judge ratings are committed to this repository.

**Key results** — 8 runs across 8 languages, hard difficulty, real GitHub
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

**Cost was reduced in every language tested. Quality was maintained in every
task — zero regressions.** JavaScript and TypeScript show the most dramatic
efficiency gains: same quality fixes in half the time with two-thirds fewer
tokens. Even on tasks too hard for either approach (Rust), Lumen cuts the cost
of failure by 39%.

See [docs/BENCHMARKS.md](docs/BENCHMARKS.md) for all 8 per-language deep dives,
judge rationales, and reproduce instructions.

## Supported languages

Supports **11 language families** with semantic chunking (8 benchmarked):

| Language         | Parser      | Extensions                                | Benchmark status                              |
| ---------------- | ----------- | ----------------------------------------- | --------------------------------------------- |
| Go               | Native AST  | `.go`                                     | Benchmarked: -12% cost, Good quality          |
| Python           | tree-sitter | `.py`                                     | Benchmarked: Perfect quality, -36% tokens     |
| TypeScript / TSX | tree-sitter | `.ts`, `.tsx`                             | Benchmarked: -64% tokens, -33% time           |
| JavaScript / JSX | tree-sitter | `.js`, `.jsx`, `.mjs`                     | Benchmarked: -66% tokens, -53% time           |
| Rust             | tree-sitter | `.rs`                                     | Benchmarked: -39% cost, -34% time             |
| Ruby             | tree-sitter | `.rb`                                     | Benchmarked: -24% cost, -11% time             |
| PHP              | tree-sitter | `.php`                                    | Benchmarked: -59% tokens, -34% time           |
| C / C++          | tree-sitter | `.c`, `.h`, `.cpp`, `.cc`, `.cxx`, `.hpp` | Benchmarked: -8% cost (C++ feature task)      |
| Java             | tree-sitter | `.java`                                   | Supported                                     |
| C#               | tree-sitter | `.cs`                                     | Supported                                     |

Go uses the native Go AST parser for the most precise chunks. All other
languages use tree-sitter grammars. See [docs/BENCHMARKS.md](docs/BENCHMARKS.md)
for all 8 per-language benchmark deep dives.

## Configuration

All configuration is via environment variables:

| Variable                 | Default                  | Description                                |
| ------------------------ | ------------------------ | ------------------------------------------ |
| `LUMEN_EMBED_MODEL`      | see note ¹               | Embedding model (must be in registry)      |
| `LUMEN_BACKEND`          | `ollama`                 | Embedding backend (`ollama` or `lmstudio`) |
| `OLLAMA_HOST`            | `http://localhost:11434` | Ollama server URL                          |
| `LM_STUDIO_HOST`         | `http://localhost:1234`  | LM Studio server URL                       |
| `LUMEN_MAX_CHUNK_TOKENS` | `512`                    | Max tokens per chunk before splitting      |

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

Where `<hash>` is derived from the absolute project path, embedding model name,
and binary version. Different models or Lumen versions automatically get
separate indexes. No files are added to your repo, no `.gitignore` modifications
needed.

You can safely delete the entire `lumen` directory to clear all indexes, or use
`lumen purge` to do it automatically.

**Git worktrees** are detected automatically. When you create a new worktree
(`git worktree add` or `claude --worktree`), Lumen finds a sibling worktree's
existing index and copies it as a seed. The Merkle tree diff then re-indexes
only the files that actually differ — typically a handful of files instead of
the entire codebase. No configuration needed; it just works.

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
contribution guidelines.
