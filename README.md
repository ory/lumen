# Lumen: semantic search for code agents

[![CI](https://github.com/aeneasr/lumen/actions/workflows/ci.yml/badge.svg)](https://github.com/aeneasr/lumen/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/aeneasr/lumen)](https://goreportcard.com/report/github.com/aeneasr/lumen)
[![Go Reference](https://pkg.go.dev/badge/github.com/aeneasr/lumen.svg)](https://pkg.go.dev/github.com/aeneasr/lumen)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A 100% local semantic code search engine for AI coding agents. No API keys, no
cloud, no external database. Just open-source embedding models (Ollama or LM
Studio), SQLite + sqlite-vec, and your CPU. Works on any developer machine
because of Golang.

Lumen lights up complex code bases and makes Claude Code **2.1-2.3x faster** and **63-81% cheaper**,
with reproducible [benchmarks](docs/BENCHMARKS.md) while **always** retaining
or exceeding answer quality over the baseline.

|                              | With lumen                  | Baseline (no MCP)           |
| ---------------------------- | --------------------------- | --------------------------- |
| Task completion              | **2.1-2.3x faster**        | baseline                    |
| API cost                     | **63-81% cheaper**          | baseline                    |
| Answer quality (blind judge) | **5/5 wins**                | 0/5 wins                    |

## Supported Languages

Supports **12 language families** with semantic chunking:

| Language         | Parser      | Extensions                                | Status                              |
| ---------------- | ----------- | ----------------------------------------- |-------------------------------------|
| Go               | Native AST  | `.go`                                     | Optimized: 3.8x faster, 90% cheaper |
| Python           | tree-sitter | `.py`                                     | Tested: 1.8x faster, 72% cheaper    |
| TypeScript / TSX | tree-sitter | `.ts`, `.tsx`                             | Tested: 1.4x faster, 48% cheaper    |
| JavaScript / JSX | tree-sitter | `.js`, `.jsx`, `.mjs`                     | Supported                           |
| Rust             | tree-sitter | `.rs`                                     | Supported                           |
| Ruby             | tree-sitter | `.rb`                                     | Supported                           |
| Java             | tree-sitter | `.java`                                   | Supported                           |
| PHP              | tree-sitter | `.php`                                    | Supported                           |
| C / C++          | tree-sitter | `.c`, `.h`, `.cpp`, `.cc`, `.cxx`, `.hpp` | Supported                           |
| Markdown / MDX   | tree-sitter | `.md`, `.mdx`                             | Supported                           |
| YAML             | tree-sitter | `.yaml`, `.yml`                           | Supported                           |
| JSON             | tree-sitter | `.json`                                   | Supported                           |

Go uses the native Go AST parser for the most precise chunks. All other
languages use tree-sitter grammars.

_Note: Golang is the best-supported language. Other languages work via
tree-sitter but may benefit from improved chunking strategies and require work to improve chunking algorithms._

## Why

Claude Code wastes context window tokens reading entire files when it only needs
one function. Semantic search fixes this: the agent describes what it's looking
for in natural language and gets back precise file paths and line ranges.

- **Local embeddings** via Ollama or LM Studio (no API keys, no network calls)
- **Local storage** via SQLite + sqlite-vec (no external database)
- **Incremental indexing** via Merkle tree change detection (only re-embeds
  changed files)
- **Auto-indexing** on search (no manual reindex step)

## Install

**Prerequisites:**

1. [Ollama](https://ollama.com/) or [LM Studio](https://lmstudio.ai/download)
   installed and running
2. Pull the default embedding model: `ollama pull ordis/jina-embeddings-v2-base-code`

### As a Claude Code plugin

```bash
# Install via the Claude Code marketplace
claude marketplace add github:aeneasr/lumen

# From source (development)
git clone https://github.com/aeneasr/lumen.git
cd lumen
make build
claude --plugin-dir .
```

The binary is downloaded automatically from the [latest GitHub release](https://github.com/aeneasr/lumen/releases)
on first use — no npm, no manual install step.

The plugin system handles everything automatically:
- **MCP server** registration (`.mcp.json`)
- **SessionStart hook** that directs the agent to prefer semantic search
- **PreToolUse hook** that intercepts natural language Grep/Glob patterns
- **Skills**: `/lumen:doctor` for health checks, `/lumen:reindex` for forced re-indexing

### Environment variables

| Variable            | Default                                | Description                        |
| ------------------- | -------------------------------------- | ---------------------------------- |
| `LUMEN_BACKEND`     | `ollama`                               | Backend: `ollama` or `lmstudio`    |
| `LUMEN_EMBED_MODEL` | `ordis/jina-embeddings-v2-base-code`   | Embedding model                    |
| `OLLAMA_HOST`       | `http://localhost:11434`               | Ollama server URL                  |
| `LM_STUDIO_HOST`    | `http://localhost:1234`                | LM Studio server URL               |

### Purge index data

To remove all lumen index databases:

```bash
lumen purge
```

This deletes `~/.local/share/lumen/`. Indexes are rebuilt automatically on the
next search.

## CLI

The `lumen index` command lets you pre-index a project from the terminal.

```bash
lumen index <project-path>
```

| Flag      | Short | Default                             | Description                                |
| --------- | ----- | ----------------------------------- | ------------------------------------------ |
| `--model` | `-m`  | `$LUMEN_EMBED_MODEL` or backend default | Embedding model to use                     |
| `--force` | `-f`  | false                               | Force full re-index (skip freshness check) |

**Examples:**

```bash
# Index using the default model
lumen index ~/workspace/myproject

# Force a full re-index
lumen index --force ~/workspace/myproject

# Use a specific model
lumen index -m nomic-embed-text ~/workspace/myproject
```

## MCP Tools

### `semantic_search`

Search indexed code using natural language. Auto-indexes if the index is stale.

| Parameter       | Type    | Required | Description                                                                   |
| --------------- | ------- | -------- | ----------------------------------------------------------------------------- |
| `query`         | string  | yes      | Natural language search query                                                 |
| `path`          | string  | yes      | Absolute path to the project root                                             |
| `limit`         | integer | no       | Max results (default: 50)                                                     |
| `min_score`     | float   | no       | Minimum score threshold (-1 to 1). Default 0.5. Use -1 to return all results. |
| `force_reindex` | boolean | no       | Force full re-index before searching                                          |

Returns file paths, symbol names, line ranges, and similarity scores (0-1).

### `index_status`

Check indexing status without triggering a reindex.

| Parameter | Type   | Required | Description                       |
| --------- | ------ | -------- | --------------------------------- |
| `path`    | string | yes      | Absolute path to the project root |

## Configuration

All configuration is via environment variables:

| Variable                  | Default              | Description                                |
| ------------------------- | -------------------- | ------------------------------------------ |
| `LUMEN_EMBED_MODEL`       | see note ¹           | Embedding model (must be in registry)      |
| `LUMEN_BACKEND`           | `ollama`             | Embedding backend (`ollama` or `lmstudio`) |
| `OLLAMA_HOST`             | `localhost:11434`    | Ollama server URL                          |
| `LM_STUDIO_HOST`          | `localhost:1234`     | LM Studio server URL                       |
| `LUMEN_MAX_CHUNK_TOKENS`  | `512`                | Max tokens per chunk before splitting      |

¹ `ordis/jina-embeddings-v2-base-code` (Ollama), `nomic-ai/nomic-embed-code-GGUF` (LM Studio)

### Supported embedding models

Dimensions and context length are configured automatically per model:

| Model                                | Backend   | Dims | Context | Recommended                                                           |
| ------------------------------------ | --------- | ---- | ------- |-----------------------------------------------------------------------|
| `ordis/jina-embeddings-v2-base-code` | Ollama    | 768  | 8192    | **Best default** — lowest cost, no over-retrieval                     |
| `qwen3-embedding:8b`                 | Ollama    | 4096 | 40960   | **Best quality** — strongest dominance (7/9 wins), very slow indexing |
| `nomic-ai/nomic-embed-code-GGUF`     | LM Studio | 3584 | 8192    | **Usable** — good quality, but TypeScript over-retrieval raises costs |
| `qwen3-embedding:4b`                 | Ollama    | 2560 | 40960   | **Not recommended** — highest costs, severe TypeScript over-retrieval |
| `nomic-embed-text`                   | Ollama    | 768  | 8192    | Untested                                                              |
| `qwen3-embedding:0.6b`              | Ollama    | 1024 | 32768   | Untested                                                              |
| `all-minilm`                         | Ollama    | 384  | 512     | Untested                                                              |

Switching models creates a separate index automatically. The model name is part
of the database path hash, so different models never collide.

## How It Works

1. **Change detection**: SHA-256 Merkle tree identifies added/modified/removed
   files. If nothing changed, search hits the existing index directly.
2. **AST chunking**: Changed files are parsed into semantic chunks. Go files use
   the native `go/ast` parser; other languages use tree-sitter grammars. Each
   function, method, type, interface, and const/var declaration becomes a chunk,
   including its doc comment.
3. **Embedding**: Chunks are batched (32 at a time) and sent to Ollama for
   embedding.
4. **Storage**: Vectors and metadata go into SQLite via sqlite-vec with cosine
   distance. Database lives in `$XDG_DATA_HOME/lumen/` — your project
   directory stays clean.
5. **Search**: Query is embedded with the same model, KNN search returns the
   closest matches.

## Storage

Index databases are stored outside your project:

```
~/.local/share/lumen/<hash>/index.db
```

Where `<hash>` is derived from the absolute project path and embedding model
name. No files are added to your repo, no `.gitignore` modifications needed.

You can safely delete the entire `lumen` directory to clear all indexes,
or use `lumen purge` to do it automatically.

## Benchmarks

Using Lumen is a clear win in speed, cost, and answer quality across both
embedding backends. The semantic search tool lets the agent find relevant code at
a fraction of the cost, significantly faster, and produces better answers that
win blind comparisons.

Key results (Ollama, jina-embeddings-v2-base-code):

| Model      | Speedup          | Cost Savings       | Quality       |
| ---------- | ---------------- | ------------------ | ------------- |
| Sonnet 4.6 | **2.2x faster**  | **63% cheaper**    | 5/5 MCP wins  |
| Opus 4.6   | **2.1x faster**  | **81% cheaper**    | 5/5 MCP wins  |

Results hold across LM Studio (nomic-embed-code) and across Go, Python, and
TypeScript in extended multi-model benchmarks.

See [docs/BENCHMARKS.md](docs/BENCHMARKS.md) for full speed/cost tables, answer
quality breakdowns, per-language results across 4 embedding models, and
reproduce instructions.

## Building from source

```bash
make build    # outputs bin/lumen-<os>-<arch>
```

## Contributing

PRs and issues welcome. Run `make lint test` before submitting.
