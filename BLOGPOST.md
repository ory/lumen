title: Make Claude Code faster and cheaper in large codebases with Ory Lumen
description: Ory Lumen adds local semantic code search to Claude Code via MCP.
Index your codebase with local embeddings, cut runtime by up to 53% and API
costs by up to 33%. slug: ory-lumen-semantic-search-claude-code meta-desc:
Claude Code getting slower as your codebase grows? Ory Lumen is a local semantic
search MCP server that makes Claude up to 2x faster and 33% cheaper. meta-title:
Faster and Cheaper Claude Code with Ory Lumen | Ory

[Ory Lumen](https://github.com/ory/lumen) makes Claude Code faster and cheaper
with local semantic search via SQLite-vec.

Ory's codebase is growing and over the past couple of weeks, I noticed Claude
Code was getting slower and more expensive to work with as a result. The issue
is simple: Claude's default to use grep/glob and find has limitations due to
exact match requirements. More code means more surface area, and surface area
means more tool use, higher token costs and slower task completion.

When you ask Claude to find a function or understand a module, it guesses file
and function names to find what is relevant with an exact match. In a small
codebase, this is fine. In a larger one, it becomes expensive, both in time and
in API costs. This problem compounds as the codebase grows, which means it gets
worse exactly when you need it to get better.

Want to try it out immediately? Jump to the README on
https://github.com/ory/lumen and if you like it, leave a star!

## The root cause

I wrote about this dynamic in more depth recently: agents struggle to build and
maintain a durable mental model of a codebase. They rediscover things repeatedly
through guessing and file reads instead of building on what they already know.
This is a fundamental constraint of how LLMs work today, not a bug that will get
patched. Ory Lumen (https://github.com/ory/lumen) is a direct, practical
response to that constraint which improves discoverability in code bases.

## What is Ory Lumen?

**[Ory Lumen](https://github.com/ory/lumen)** is a local semantic code search
engine that runs as an MCP server alongside Claude Code. It indexes your
codebase using local embedding models and exposes a `semantic_search` tool that
Claude calls instead of reading files directly. Claude can find relevant
functions, types, and modules by meaning, without opening everything to look.

How it works:

1. On session start, Lumen walks your project and chunks each file into semantic
   units: functions, methods, types. Go uses the native AST parser. All other
   languages use tree-sitter grammars.
2. Those chunks are embedded using a local model and stored in SQLite with
   sqlite-vec for vectorization.
3. When Claude needs to find relevant code, it calls `semantic_search` and gets
   back the relevant chunks without touching the files.

Everything runs on your machine. No API keys, no cloud, no external services.
The embedding backend is [Ollama](https://ollama.com/) or
[LM Studio](https://lmstudio.ai/). The index is stored at
`~/.local/share/lumen/<hash>/index.db`, keyed by project path and model name.
Nothing is added to your repo.

### Re-indexing stays fast

Lumen builds a Merkle tree over file hashes on the first run. On subsequent
sessions, only changed files get re-chunked and re-embedded. For large
codebases, re-indexing after the first run takes seconds.

## The results

Lumen is evaluated using a SWE-bench-style harness: real GitHub bugs, real
codebases, Claude fixing them with and without Lumen. Patches are rated by a
blind judge comparing against the known-correct fix. Full methodology, raw data,
and reproduce instructions are in
[docs/BENCHMARKS.md](https://github.com/ory/lumen/blob/main/docs/BENCHMARKS.md).

| Language   | Cost reduction | Time reduction | Quality        |
| ---------- | -------------- | -------------- | -------------- |
| PHP        | 52%            | 59%            | Good → Perfect |
| JavaScript | 56%            | 51%            | Same (Good)    |
| Go         | 30%            | 13%            | Same (Good)    |

Across all 7 languages tested, Lumen reduces tool calls by **27%** on average.
Claude navigates to the right code directly instead of reading files
systematically.

PHP is the standout: Lumen cuts token usage by up to 86% while also upgrading
patch quality from Good to Perfect on every run. JavaScript shows consistent
resource reduction at the same quality level. Rust shows a clear quality uplift.
Baseline produced mixed results while Lumen delivered Good on every run. Most
languages have potential to be further improved by tweaking the chunking
algorithm and adding additional discovery tools.

The benchmark suite covers Go, JavaScript, PHP, Python, Ruby, Rust, and
TypeScript, all hard-difficulty tasks from real open-source projects. Results
are fully reproducible.

## Installing it

We launched an
[Ory Claude plugin marketplace](https://github.com/ory/claude-plugins) alongside
Lumen today. It is the first plugin in it. Inside Claude Code, run:

```
/plugin marketplace add ory/claude-plugins
/plugin install lumen@ory
```

Lumen downloads its binary automatically from the latest GitHub release, indexes
your project on the next session start, and registers the `semantic_search`
tool. Claude picks it up without any additional configuration.

**Prerequisites:**

1. Install [Ollama](https://ollama.com/) and pull the default embedding model:

```bash
$ ollama pull ordis/jina-embeddings-v2-base-code
```

2. Have [Claude Code](https://code.claude.com/) installed.

Two skills come with the plugin: `/lumen:doctor` for a health check, and
`/lumen:reindex` to force a full re-index after a large refactor.

## On keeping it local

One constraint I was not willing to give up: your code stays on your machine.
Sending source code to an external embedding API is a decision engineering teams
should make deliberately, not by default. Lumen runs entirely on local hardware
with open-source models. The embeddings never leave your network.

This also makes it usable in air-gapped environments, which matters for the
companies running Ory's self-hosted products.

## Contributing

Ory Lumen is a new project with rough edges, and it will improve over time. We
want it to solve more benchmarks better and welcome anyone who wants to help
make Claude Code faster and cheaper with local-first tooling! If you find
something, please feel free to contribute on our
[GitHub](https://github.com/ory/lumen)!
