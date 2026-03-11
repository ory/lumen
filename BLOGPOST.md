title: Make Claude Code faster and cheaper in large codebases with Ory Lumen
description: Ory Lumen adds local semantic code search to Claude Code through MCP. It indexes your codebase with local embeddings and reduced runtime by up to 53% and API cost by up to 33% in our benchmarks.
slug: ory-lumen-semantic-search-claude-code
meta-desc: Claude Code slows down as codebases grow. Ory Lumen adds local semantic search through MCP and reduced runtime by up to 53% and API cost by up to 33% in our benchmarks.
meta-title: Make Claude Code faster and cheaper with Ory Lumen | Ory

[Ory Lumen](https://github.com/ory/lumen) makes Claude Code faster and cheaper in larger codebases by adding local semantic code search through SQLite-vec.

As Ory's codebase grew, Claude Code got slower and more expensive to use. The main reason was code discovery. Claude mostly relies on grep, glob, and file reads. Those tools work well when you already know the exact file name, symbol name, or string to search for. They work less well when you only know what the code does.

That gap gets larger as the codebase grows. More files means more guesses, more tool calls, more file reads, more tokens, and more time spent finding the right place to edit.

With Ory Lumen, Claude can search by meaning instead of exact text. That cuts down on blind exploration and gets it to the relevant code faster.

Want to try it now? The README is here: https://github.com/ory/lumen

## Why Claude Code slows down in larger repos

Claude does not keep a stable working model of a large codebase. It reconstructs context as it goes. In practice, that means it often has to rediscover the same code by reading files, searching for symbols, and following references.

In a small repository, that cost is limited. In a large one, it adds up fast. Claude may look through several files before it finds the function, type, or module that actually matters. That increases latency and token usage, and it raises the chance that it edits the wrong place or misses a related file.

Ory Lumen addresses that problem at the discovery step.

## What Ory Lumen does

**[Ory Lumen](https://github.com/ory/lumen)** is a local semantic code search engine that runs as an MCP server next to Claude Code. It indexes your repository with local embedding models and exposes a `semantic_search` tool that Claude can call during a session.

Instead of opening many files to hunt for the right code, Claude can ask for code that matches the intent of the task. Lumen returns the most relevant chunks, such as functions, methods, and types.

The flow is simple:

1. On session start, Lumen walks the project and splits files into semantic units. For Go, it uses the native AST parser. For other languages, it uses tree-sitter grammars.
2. Lumen embeds those chunks with a local model and stores them in SQLite using sqlite-vec.
3. When Claude needs relevant code, it calls `semantic_search` and gets back the highest-matching chunks.

Everything runs locally. There are no API keys, cloud services, or external embedding calls. The embedding backend is [Ollama](https://ollama.com/) or [LM Studio](https://lmstudio.ai/). The index lives at `~/.local/share/lumen/<hash>/index.db`, keyed by project path and model name. Nothing is written into your repository.

### Re-indexing is incremental

The first run builds a Merkle tree over file hashes. After that, Lumen only re-chunks and re-embeds files that changed. In large codebases, re-indexing after the first run usually takes seconds.

## Benchmark results

We evaluated Lumen with a SWE-bench-style harness using real GitHub issues and real codebases. Claude solves the task once with Lumen and once without it. A blind judge compares both patches against the known-correct fix. The full method, raw data, and reproduction steps are in [docs/BENCHMARKS.md](https://github.com/ory/lumen/blob/main/docs/BENCHMARKS.md).

| Language   | Cost reduction | Time reduction | Output token reduction | Quality        |
| ---------- | -------------- | -------------- | ---------------------- | -------------- |
| JavaScript | 33%            | 53%            | 66%                    | Perfect (both) |
| TypeScript | 27%            | 33%            | 64%                    | Good (both)    |
| PHP        | 27%            | 34%            | 59%                    | Good (both)    |
| Python     | 20%            | 29%            | 36%                    | Perfect (both) |
| Ruby       | 24%            | 11%            | 9%                     | Good (both)    |
| Go         | 4%             | 5%             | 30%                    | Good (both)    |
| C++\*      | -20%           | 14%            | 11%                    | Good (both)    |

\* The C++ task is a feature implementation task rather than a bug fix. It is the only case where cost increased.

Across the six bug-fix tasks, Lumen reduced output tokens by **46% on average**.

The clearest gains were in JavaScript, TypeScript, and PHP. In those tasks, Claude spent less time exploring and reached the relevant code faster. Go showed smaller gains in time and cost, but Lumen still helped produce a more complete patch that included a test file the baseline missed.

The current benchmark suite covers Go, JavaScript, PHP, Python, Ruby, TypeScript, and C++. All tasks are hard-difficulty tasks from real open source projects, and the results are reproducible.

## Install Ory Lumen

We also launched the [Ory Claude plugin marketplace](https://github.com/ory/claude-plugins). Lumen is the first plugin in it.

Inside Claude Code, run:

```bash
/plugin marketplace add ory/claude-plugins
/plugin install lumen@ory