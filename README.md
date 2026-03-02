# agent-index

[![CI](https://github.com/aeneasr/agent-index/actions/workflows/ci.yml/badge.svg)](https://github.com/aeneasr/agent-index/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/aeneasr/agent-index)](https://goreportcard.com/report/github.com/aeneasr/agent-index)
[![Go Reference](https://pkg.go.dev/badge/github.com/aeneasr/agent-index.svg)](https://pkg.go.dev/github.com/aeneasr/agent-index)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A 100% local semantic code search engine (think Claude Context, Augment Code, Cursor) using open-source models, SQLite + sqlite-vec
and using only your CPU time. Works on any developer machine because of Golang.

Code Agent Index makes Claude Code **2.1–2.3× faster**
and **63–81% cheaper**, with reproducible [benchmarks](#its-a-game-changer-benchmarks).

This service is exposed as an
[MCP](https://modelcontextprotocol.io/) server. It parses your codebase into
semantic chunks (functions, methods, types, interfaces, constants), embeds them
via a local Ollama model, and exposes search over MCP. Your code never leaves
your machine.

Semantic search is the **fastest and most cost-efficient way** to work with
Claude Code in large code bases. Instead of reading whole files, the agent
describes what it needs and gets back exact file paths and line ranges. In
[benchmarks](#its-a-game-changer-benchmarks) against Prometheus/TSDB source code
across 5 questions of increasing difficulty, `semantic_search` completed tasks
**2.1–2.3× faster** and **63–81% cheaper** than the default file-read tools of
Claude Code — confirmed independently with both Ollama and LM Studio embedding
backends. Agent Index won 5 out of 5 blind quality comparisons. Baseline Claude
Code (only default tools) won zero.

Everything runs locally no API keys, no code sent to external services, no cloud
dependency. Using open source embedding models via Ollama or LM Studio, and
storing vectors in a local SQLite database. Your code stays on your machine,
indexed and searchable without any network calls. It's fast and reliable.

|                              | With `semantic_search`       | Default tools (baseline) |
| ---------------------------- | ---------------------------- | ------------------------ |
| Task completion              | **2.1–2.3× faster**          | baseline                 |
| API cost                     | **63–81% cheaper**           | baseline                 |
| Answer quality (blind judge) | **5/5 wins** (both backends) | 0/5 wins                 |

Supports **12 language families** with semantic chunking:

| Language         | Extensions                                | Chunking strategy                                                   |
| ---------------- | ----------------------------------------- | ------------------------------------------------------------------- |
| Go               | `.go`                                     | Native Go AST — functions, methods, types, interfaces, consts, vars |
| TypeScript / TSX | `.ts`, `.tsx`                             | tree-sitter — functions, classes, interfaces, type aliases, methods |
| JavaScript / JSX | `.js`, `.jsx`, `.mjs`                     | tree-sitter — functions, classes, methods, generators               |
| Python           | `.py`                                     | tree-sitter — function definitions, class definitions               |
| Rust             | `.rs`                                     | tree-sitter — functions, structs, enums, traits, impls, consts      |
| Ruby             | `.rb`                                     | tree-sitter — methods, singleton methods, classes, modules          |
| Java             | `.java`                                   | tree-sitter — methods, classes, interfaces, constructors, enums     |
| PHP              | `.php`                                    | tree-sitter — functions, classes, interfaces, traits, methods       |
| C / C++          | `.c`, `.h`, `.cpp`, `.cc`, `.cxx`, `.hpp` | tree-sitter — function definitions, structs, enums, classes         |
| Markdown / MDX   | `.md`, `.mdx`                             | Heading-based — each `#` / `##` / `###` section is one chunk        |
| YAML             | `.yaml`, `.yml`                           | Key-based — each top-level key and its value block is one chunk     |
| JSON             | `.json`                                   | Key-based — each top-level key and its value block is one chunk     |

## Why

Claude Code is good at writing code but wasteful and slow at navigating large
codebases. It wastes context window tokens reading entire files when it only
needs one function. Semantic search fixes this: the code agent describes what
it's looking for in natural language and gets back precise file paths and line
ranges.

Cloud-hosted vector databases solve this, but they're expensive, intransparent,
and require sending your code to a third party. agent-index gives you the same
capability with everything running locally for free:

- **Local embeddings** via Ollama (no API keys, no network calls to external
  services) or LM Studio
- **Local storage** via SQLite + sqlite-vec (no external database)
- **Incremental indexing** via Merkle tree change detection (only re-embeds
  changed files)
- **Auto-indexing** on search (no manual reindex step)

## Install

**Prerequisites:**

1. [Ollama](https://ollama.com/) or [LM Studio](https://lmstudio.ai/download)
   installed and running
2. [Go](https://go.dev/) 1.26+

```bash
# Install the binary
CGO_ENABLED=1 go install github.com/aeneasr/agent-index@latest
```

> `CGO_ENABLED=1` is required — sqlite-vec compiles from C source.

## Setup with Claude Code

### Best practice configuration

The default configuration yielded 2.15x faster indexing and 72% less cost in
benchmarks. This configuration uses Ollama +
`ordis/jina-embeddings-v2-base-code` for fast, efficient indexing. It's the
default configuration and works out of the box with Claude Code if you have
Ollama installed.

```bash
# Pull the default embedding model
ollama pull ordis/jina-embeddings-v2-base-code

# Add as an MCP server (defaults work out of the box)
claude mcp add --scope user \
  agent-index "$(go env GOPATH)/bin/agent-index" -- stdio
```

That's it. Claude Code will now have access to `semantic_search` and
`index_status` tools. On the first search against a project, it auto-indexes the
codebase.

### Alternative: LM Studio + nomic-embed-code

An experimental configuration with higher-quality 3584-dim embeddings via LM
Studio. Expect significantly slower indexing times, especially on large
codebases. This configuration excels when using Opus 4.6 but is not as good as
the default configuration for Sonnet 4.6 in benchmarks.

[LM Studio](https://lmstudio.ai/) exposes an OpenAI-compatible `/v1/embeddings`
endpoint at `http://localhost:1234` by default. `nomic-embed-code` is a
code-optimized model with 3584 dimensions.

> [!WARNING]  
> `nomic-ai/nomic-embed-code-GGUF` is significantly more resource intense than
> the default Ollama model. Expect higher CPU usage and longer indexing times,
> especially on large codebases. Consider using
> `agent-index index path/to/source` to pre-index your codebase.

```bash
# Download and load the model via lms CLI
lms get nomic-ai/nomic-embed-code-GGUF
lms load nomic-ai/nomic-embed-code-GGUF

# Add as MCP server using the lmstudio backend
claude mcp add --scope user \
  -eAGENT_INDEX_BACKEND=lmstudio \
  -eAGENT_INDEX_EMBED_MODEL=nomic-ai/nomic-embed-code-GGUF \
  agent-index "$(go env GOPATH)/bin/agent-index" -- stdio
```

### Switching models (Ollama)

To use a different Ollama model, set `AGENT_INDEX_EMBED_MODEL` — dims and
context are looked up automatically:

```bash
claude mcp remove --scope user agent-index
claude mcp add --scope user \
  -eAGENT_INDEX_EMBED_MODEL=nomic-embed-text \
  agent-index "$(go env GOPATH)/bin/agent-index" -- stdio
```

## CLI

The `agent-index index` command lets you pre-index a project from the terminal.
This is useful for large codebases where you want indexing to happen in the
background before the first MCP search.

```bash
agent-index index <project-path>
```

| Flag      | Short | Default                                       | Description                                |
| --------- | ----- | --------------------------------------------- | ------------------------------------------ |
| `--model` | `-m`  | `$AGENT_INDEX_EMBED_MODEL` or backend default | Embedding model to use                     |
| `--force` | `-f`  | false                                         | Force full re-index (skip freshness check) |

**Examples:**

```bash
# Index using the default model
agent-index index ~/workspace/myproject

# Force a full re-index
agent-index index --force ~/workspace/myproject

# Use a specific model
agent-index index -m nomic-embed-text ~/workspace/myproject
```

Progress is printed to stderr. When done, the command outputs:

```
Done. Indexed 42 files, 318 chunks in 4.231s.
```

If the index is already up to date and `--force` is not set:

```
Index is already up to date.
```

> `agent-index stdio` starts the MCP server on stdin/stdout. This is invoked
> automatically by Claude Code — you don't need to run it manually.

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

Returns file paths, symbol names, line ranges, and similarity scores (0–1).

### `index_status`

Check indexing status without triggering a reindex.

| Parameter | Type   | Required | Description                       |
| --------- | ------ | -------- | --------------------------------- |
| `path`    | string | yes      | Absolute path to the project root |

## Configuration

All configuration is via environment variables:

| Variable                       | Default                                                                                      | Description                                |
| ------------------------------ | -------------------------------------------------------------------------------------------- | ------------------------------------------ |
| `AGENT_INDEX_EMBED_MODEL`      | `ordis/jina-embeddings-v2-base-code` (Ollama) / `nomic-ai/nomic-embed-code-GGUF` (LM Studio) | Embedding model (must be in registry)      |
| `AGENT_INDEX_BACKEND`          | `ollama`                                                                                     | Embedding backend (`ollama` or `lmstudio`) |
| `OLLAMA_HOST`                  | `http://localhost:11434`                                                                     | Ollama server URL                          |
| `LM_STUDIO_HOST`               | `http://localhost:1234`                                                                      | LM Studio server URL                       |
| `AGENT_INDEX_MAX_CHUNK_TOKENS` | `512`                                                                                        | Max tokens per chunk before splitting      |

### Supported embedding models

Dimensions and context length are configured automatically per model:

| Model                                | Backend   | Dims | Context | Size   | Notes                                        | Recommended                             |
| ------------------------------------ | --------- | ---- | ------- | ------ | -------------------------------------------- | --------------------------------------- |
| `ordis/jina-embeddings-v2-base-code` | Ollama    | 768  | 8192    | ~323MB | Default. Code-optimized, fast, balanced      | **Best default** — lowest MCP cost, no over-retrieval |
| `qwen3-embedding:8b`                 | Ollama    | 4096 | 40960   | ~4.7GB | Highest retrieval quality, very slow to load | **Best quality** — strongest MCP dominance (7/9 wins), requires 4.7 GB |
| `nomic-ai/nomic-embed-code-GGUF`     | LM Studio | 3584 | 8192    | ~274MB | Code-optimized, high-dim, slow               | **Usable** — good quality, but TypeScript over-retrieval raises costs |
| `qwen3-embedding:4b`                 | Ollama    | 2560 | 40960   | ~2.6GB | High-dim, moderate quality                   | **Not recommended** — highest MCP costs, severe TypeScript over-retrieval |
| `nomic-embed-text`                   | Ollama    | 768  | 8192    | ~274MB | Fast, good general quality                   | Untested                                |
| `qwen3-embedding:0.6b`               | Ollama    | 1024 | 32768   | ~522MB | Lightweight                                  | Untested                                |
| `all-minilm`                         | Ollama    | 384  | 512     | ~33MB  | Tiny, CI use, fast                           | Untested                                |

Switching models creates a separate index automatically. The model name is part
of the database path hash, so different models never collide. Models perform differently across languages.

## Supported Languages

| Language         | Parser          | Status            |
| ---------------- | --------------- | ----------------- |
| Go               | Native `go/ast` | Thoroughly tested |
| TypeScript / TSX | tree-sitter     | Supported         |
| JavaScript / JSX | tree-sitter     | Supported         |
| Python           | tree-sitter     | Supported         |
| Rust             | tree-sitter     | Supported         |
| Ruby             | tree-sitter     | Supported         |
| Java             | tree-sitter     | Supported         |
| C                | tree-sitter     | Supported         |
| C++              | tree-sitter     | Supported         |

Go uses the native Go AST parser, which produces the most precise chunks and has
comprehensive test coverage. All other languages use tree-sitter grammars — they
work but have less test coverage and may miss some language-specific constructs.

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
   distance. Database lives in `$XDG_DATA_HOME/agent-index/` — your project
   directory stays clean.
5. **Search**: Query is embedded with the same model, KNN search returns the
   closest matches.

## Storage

Index databases are stored outside your project:

```
~/.local/share/agent-index/<hash>/index.db
```

Where `<hash>` is derived from the absolute project path and embedding model
name. No files are added to your repo, no `.gitignore` modifications needed.

You can safely delete the entire `agent-index` directory to clear all indexes,
or delete specific subdirectories to clear indexes for specific projects/models.

## It's A Game Changer: Benchmarks

`bench-mcp.sh` runs 5 questions of increasing difficulty against
[Prometheus/TSDB Go fixtures](testdata/fixtures/go), across 2 models (Sonnet
4.6, Opus 4.6) and 3 scenarios:

- **baseline** — default tools only (grep, file reads), no MCP
- **mcp-only** — `semantic_search` only, no file reads
- **mcp-full** — all tools + `semantic_search`

Answers are ranked blind by an LLM judge (Opus 4.6). Benchmarks are transparent
(check bench-results) and reproducible. Please note that **mcp-only** disables built-in tools from Claude Code
which could impact tool performance, even though benchmarks show no sign of it.

## Results

Using Agent Index is a clear win in speed, cost, and answer quality across both
embedding backends. The semantic search tool lets the agent find relevant code a
fraction of the cost of the baseline, significantly faster, and produces better
answers that win blind comparisons, confirmed independently with Ollama and LM
Studio.

### Speed & cost — Ollama (jina-embeddings-v2-base-code, 768-dim)

Totals across all 5 questions × 2 models:

| Model      | Scenario | Total Time               | Total Cost              |
| ---------- | -------- | ------------------------ | ----------------------- |
| Sonnet 4.6 | baseline | 496.8s                   | $5.97                   |
| Sonnet 4.6 | mcp-only | 228.9s (**2.2× faster**) | $2.20 (**63% cheaper**) |
| Opus 4.6   | baseline | 478.0s                   | $9.66                   |
| Opus 4.6   | mcp-only | 229.9s (**2.1× faster**) | $1.79 (**81% cheaper**) |

### Answer quality — Ollama

Baseline never wins. `mcp-only` wins all medium/hard/very-hard questions at a
fraction of the cost.

| Question        | Difficulty | Winner          | Judge summary                                                                                                                           |
| --------------- | ---------- | --------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| label-matcher   | easy       | opus / mcp-full | Correct, complete; full type definitions and constructor source with accurate line references                                           |
| histogram       | medium     | opus / mcp-only | Good coverage of both bucket systems (classic + native), hot/cold swap, and iteration; 7–20× cheaper than baseline                      |
| tsdb-compaction | hard       | opus / mcp-only | Uniquely covers all three trigger paths, compactor initialization, and planning strategies; 5–6× cheaper than baseline                  |
| promql-engine   | very-hard  | opus / mcp-only | Thorough coverage of all four topics (engine, functions, AST, rules) with accurate file:line references; half the cost of opus/baseline |
| scrape-pipeline | very-hard  | opus / mcp-only | Best Registry coverage; unique dual data-flow summary for scraping and exposition paths                                                 |

`mcp-only` wins 4/5, `mcp-full` wins 1/5, `baseline` wins 0/5.

### Speed & cost — LM Studio (nomic-embed-code, 3584-dim)

Totals across all 5 questions × 2 models. Opus shows even stronger gains with
this backend: 2.8× speedup and 86% cost reduction. Sonnet's benefits are more
modest due to embedding model quality differences (see note below):

| Model      | Scenario | Total Time               | Total Cost              |
| ---------- | -------- | ------------------------ | ----------------------- |
| Sonnet 4.6 | baseline | 478.4s                   | $5.04                   |
| Sonnet 4.6 | mcp-only | 326.4s (**1.5× faster**) | $4.45 (**12% cheaper**) |
| Opus 4.6   | baseline | 675.3s                   | $13.31                  |
| Opus 4.6   | mcp-only | 238.5s (**2.8× faster**) | $1.93 (**86% cheaper**) |

**Why Sonnet shows smaller gains with nomic-embed-code:** Nomic's embeddings
score below the default `min_score=0.5` threshold on several Go code queries
(e.g. "RecordingRule eval", "PromQL AST eval switch"). Sonnet receives "No
results found" and retries with alternative query phrasings — each attempt
consuming tokens without payoff. Opus makes fewer, better-targeted searches and
is largely unaffected. The underlying issue is retrieval quality:
`jina-embeddings-v2-base-code` (Ollama default) is simply performing better in
this scenario then `nomic-embed-code`. If you use LM Studio, Opus is the better
choice.

### Answer quality — LM Studio

The higher-dimensional embeddings produce quality results that match or exceed
the Ollama run:

| Question        | Difficulty | Winner          | Judge summary                                                                                        |
| --------------- | ---------- | --------------- | ---------------------------------------------------------------------------------------------------- |
| label-matcher   | easy       | opus / mcp-only | All answers correct; mcp-only fastest (10.4s) and cheapest ($0.10) at equal quality                  |
| histogram       | medium     | opus / mcp-full | Full observation flow, function signatures, schema-based key computation; ~15× cheaper than baseline |
| tsdb-compaction | hard       | opus / mcp-only | Covers all 3 trigger paths, planning priority order, early-abort logic; 6× cheaper at $0.42          |
| promql-engine   | very-hard  | opus / mcp-only | Function safety sets, storage interfaces, full eval pipeline; $0.67 vs $7.16 baseline                |
| scrape-pipeline | very-hard  | opus / mcp-only | Best registry coverage; Register 5-step validation, Gatherers merging, ApplyConfig hot-reload        |

`mcp-only` wins 4/5, `mcp-full` wins 1/5, `baseline` wins 0/5.

### Extended benchmarks: Results by Language

A comprehensive benchmark comparing 4 embedding models across 9 questions of varying difficulty in Go, Python, and TypeScript (36 question/model combinations, 216 total runs). **Embedding model performance varies significantly by programming language.** Python shows uniform MCP-only dominance, Go shows strong MCP performance, and TypeScript reveals over-retrieval issues with larger-dimension models.

**Why language matters:** Larger-dimension models (qwen3-8b, qwen3-4b, nomic) embed more semantic detail but retrieve redundant chunks for simple TypeScript questions. This drives up token costs without improving answer quality. Jina's 768-dim embeddings avoid over-retrieval entirely while maintaining strong quality across all languages.

#### Go Results

| Model | baseline<br/>Cost | baseline<br/>Time | mcp-only<br/>Cost | mcp-only<br/>Time | mcp-only<br/>Speedup | mcp-only<br/>Savings | mcp-full<br/>Cost | mcp-full<br/>Time | Wins (base / mcp-o / mcp-f) |
|-------|---|---|---|---|---|---|---|---|---|
| jina-v2 | $10.64 | 536s | $1.03 | 142s | 3.8x | 90% | $1.63 | 149s | 0/3 / 1/3 / 2/3 |
| qwen3-8b | $4.59 | 421s | $1.05 | 165s | 2.6x | 77% | $1.84 | 168s | 0/3 / 2/3 / 1/3 |
| qwen3-4b | $8.35 | 433s | $2.19 | 186s | 2.3x | 74% | $2.52 | 179s | 0/3 / 3/3 / 0/3 |
| nomic | $5.46 | 469s | $1.55 | 280s | 1.7x | 72% | $1.96 | 229s | 0/3 / 1/3 / 2/3 |

**Insight:** Qwen3-4b wins the most scenarios (3/3 mcp-only), but **jina achieves 90% cost savings and 3.8× speedup**—by far the most efficient. No baseline wins on Go questions across any model.

#### Python Results

| Model | baseline<br/>Cost | baseline<br/>Time | mcp-only<br/>Cost | mcp-only<br/>Time | mcp-only<br/>Speedup | mcp-only<br/>Savings | mcp-full<br/>Cost | mcp-full<br/>Time | Wins (base / mcp-o / mcp-f) |
|-------|---|---|---|---|---|---|---|---|---|
| jina-v2 | $5.41 | 406s | $1.53 | 226s | 1.8x | 72% | $1.75 | 206s | 0/3 / 2/3 / 1/3 |
| qwen3-8b | $3.78 | 373s | $1.69 | 235s | 1.6x | 55% | $2.59 | 224s | 0/3 / 3/3 / 0/3 |
| qwen3-4b | $3.97 | 342s | $1.80 | 237s | 1.4x | 55% | $2.37 | 219s | 0/3 / 3/3 / 0/3 |
| nomic | $5.82 | 483s | $1.99 | 238s | 2.0x | 66% | $3.20 | 278s | 0/3 / 3/3 / 0/3 |

**Insight:** MCP-only dominates universally (all models 2-3/3 wins). Qwen3-8b, qwen3-4b, and nomic achieve 3/3 mcp-only wins. However, **jina remains cost-optimal at 72% savings** and lowest baseline cost ($5.41).

#### TypeScript Results

| Model | baseline<br/>Cost | baseline<br/>Time | mcp-only<br/>Cost | mcp-only<br/>Time | mcp-only<br/>Speedup | mcp-only<br/>Savings | mcp-full<br/>Cost | mcp-full<br/>Time | Wins (base / mcp-o / mcp-f) |
|-------|---|---|---|---|---|---|---|---|---|
| jina-v2 | $4.86 | 478s | $2.53 | 332s | 1.4x | 48% | $3.88 | 373s | 1/3 / 1/3 / 1/3 |
| qwen3-8b | $4.12 | 468s | $2.98 | 359s | 1.3x | 28% | $3.81 | 378s | 1/3 / 2/3 / 0/3 |
| qwen3-4b | $5.44 | 600s | $4.42 | 399s | 1.5x | 19% | $3.76 | 409s | 2/3 / 1/3 / 0/3 |
| nomic | $4.84 | 519s | $3.89 | 411s | 1.3x | 20% | $3.84 | 386s | 0/3 / 2/3 / 1/3 |

**Insight:** The **over-retrieval problem appears here**. Jina stays cheapest for mcp-only at $2.53 with 48% savings. Qwen3-4b suffers worst degradation (19% savings, 2/3 baseline wins). This exemplifies why dimensionality matters: larger models retrieve too many redundant chunks for simple lifecycle questions, nullifying cost advantages.

#### Summary: Why Jina Remains the Default

| Metric | jina-v2 | qwen3-8b | qwen3-4b | nomic |
|--------|---------|----------|----------|-------|
| **Best Go cost** | ✓ 90% | 77% | 74% | 72% |
| **Best Python cost** | ✓ 72% | 55% | 55% | 66% |
| **Best TypeScript cost** | ✓ 48% | 28% | 19% | 20% |
| **Consistent across languages** | ✓ | — | — | — |
| **No over-retrieval** | ✓ | Limited | Severe | Moderate |
| **Verdict** | **Default** | Best quality (Go/Py) | Not recommended | Usable (Opus) |

Full question-level analysis available in [`detail-report.md` per benchmark](bench-results/)

### Reproduce

Requires Ollama, the `claude` CLI, `jq`, and `bc`.

```bash
./bench-mcp.sh                                          # all questions, all models
./bench-mcp.sh --model sonnet                           # filter by model
./bench-mcp.sh --question tsdb-compaction               # filter by question
./bench-mcp.sh --model opus --question label-matcher    # combine
```

Results land in `bench-results/<timestamp>/`. The script runs an LLM judge at
the end to rank answers.

## Building from source

```bash
CGO_ENABLED=1 go build -o agent-index .
```

## Contributing

This project was created within a couple of days using Claude Code. The code
base will contain tech debt and some slop as well.
