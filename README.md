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

1. [Ollama](https://ollama.com/) installed and running, then pull the default
   embedding model:
   ```bash
   ollama pull ordis/jina-embeddings-v2-base-code
   ```
2. One of:
   [Claude Code](https://code.claude.com/docs/en/quickstart),
   [Cursor](https://cursor.com/),
   [Codex](https://developers.openai.com/codex/cli), or
   [OpenCode](https://opencode.ai/)

**Note:** Installation differs by platform. Claude Code is installed from a
plugin marketplace. Codex uses a local MCP server plus native skill discovery.
OpenCode installs from npm. Cursor packaging is shipped in this repository and
is ready for Cursor's plugin distribution workflow.

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

Quick install:

```text
Fetch and follow instructions from https://raw.githubusercontent.com/ory/lumen/refs/heads/main/.codex/INSTALL.md
```

Manual install:

```bash
CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"
git clone https://github.com/ory/lumen.git "$CODEX_HOME/lumen"
mkdir -p "$HOME/.agents/skills"
ln -s "$CODEX_HOME/lumen/skills" "$HOME/.agents/skills/lumen"
codex mcp add lumen -- "$CODEX_HOME/lumen/scripts/run.sh" stdio
```

Detailed docs: [.codex/INSTALL.md](.codex/INSTALL.md)

Verify with:

```bash
codex mcp get lumen
ls -la "$HOME/.agents/skills/lumen"
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
- **Codex** - `cd "${CODEX_HOME:-$HOME/.codex}/lumen" && git pull`
- **OpenCode** - update the version pin in `opencode.json` (e.g.
  `@ory/lumen-opencode@0.0.29`) and restart OpenCode

On first Claude Code or Cursor session start, Lumen:

1. Downloads the binary automatically from the
   [latest GitHub release](https://github.com/ory/lumen/releases)
2. Indexes your project in the background using Merkle tree change detection
3. Registers a `semantic_search` MCP tool that the host can use automatically

In Codex and OpenCode, the same binary download and index seeding happen on the
first `semantic_search` call.

Two shared skills are also available: `doctor` (health check) and `reindex`
(forced re-indexing). Claude exposes them as `/lumen:doctor` and
`/lumen:reindex`; the other hosts discover the same shared skill content
through their native skill systems.

The same `semantic_search`, `health_check`, and `index_status` MCP tools plus
the shared `doctor` and `reindex` skills are exposed through the Codex,
Cursor, and OpenCode surfaces as well. The first `semantic_search` call seeds
or refreshes the index automatically.

## What you get

- **Semantic vector search** — Claude finds relevant functions, types, and
  modules by meaning, not keyword matching
- **Auto-indexing** — indexes on session start, only re-processes changed files
  via Merkle tree diffing
- **Incremental updates** — re-indexes only what changed; large codebases
  re-index in seconds after the first run
- **12 language families** — Go, Python, TypeScript, JavaScript, Svelte, Rust,
  Ruby, Java, PHP, C/C++, C#, Dart
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
| TypeScript / TSX | tree-sitter | `.ts`, `.tsx`                             | Benchmarked: -64% tokens, -33% time           |
| JavaScript / JSX | tree-sitter | `.js`, `.jsx`, `.mjs`                     | Benchmarked: -66% tokens, -53% time           |
| Dart             | tree-sitter | `.dart`                                   | Benchmarked: -76% cost, -82% tokens, -79% time |
| Rust             | tree-sitter | `.rs`                                     | Benchmarked: -39% cost, -34% time             |
| Ruby             | tree-sitter | `.rb`                                     | Benchmarked: -24% cost, -11% time             |
| PHP              | tree-sitter | `.php`                                    | Benchmarked: -59% tokens, -34% time           |
| C / C++          | tree-sitter | `.c`, `.h`, `.cpp`, `.cc`, `.cxx`, `.hpp` | Benchmarked: -8% cost (C++ feature task)      |
| Svelte           | tree-sitter | `.svelte`                                 | Benchmarked: -26% cost, -31% time             |
| Java             | tree-sitter | `.java`                                   | Supported                                     |
| C#               | tree-sitter | `.cs`                                     | Supported                                     |

Go uses the native Go AST parser for the most precise chunks. All other
languages use tree-sitter grammars. See [docs/BENCHMARKS.md](docs/BENCHMARKS.md)
for all 10 per-language benchmark deep dives.

## Configuration

All configuration is via environment variables:

| Variable                 | Default                  | Description                                                   |
| ------------------------ | ------------------------ | ------------------------------------------------------------- |
| `LUMEN_EMBED_MODEL`      | see note ¹               | Embedding model; use with `LUMEN_EMBED_DIMS` for unlisted models |
| `LUMEN_BACKEND`          | `ollama`                 | Embedding backend (`ollama` or `lmstudio`)                    |
| `OLLAMA_HOST`            | `http://localhost:11434` | Ollama server URL                                             |
| `LM_STUDIO_HOST`         | `http://localhost:1234`  | LM Studio server URL                                          |
| `LUMEN_MAX_CHUNK_TOKENS` | `512`                    | Max tokens per chunk before splitting                         |
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

In Cursor, Codex, or OpenCode, use the shared `doctor` skill or call
`health_check` and `index_status` directly.

**Stale index after large refactor**

Run `/lumen:reindex` inside Claude Code to force a full re-index, or:

```bash
lumen purge && lumen index .
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
contribution guidelines, and [AGENTS.md](AGENTS.md) for repo-specific agent
instructions.
