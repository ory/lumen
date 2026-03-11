title: Make Claude Code faster and cheaper in large codebases with Ory Lumen

description: Ory Lumen adds local semantic code search to Claude Code via MCP.
Index your codebase with local embeddings, cut runtime by up to 53% and API
costs by up to 39%.

slug: ory-lumen-semantic-search-claude-code

meta-desc: Claude Code getting slower as your codebase grows? Ory Lumen is a
local semantic search MCP server that cuts costs by up to 39% and sessions by up
to 53%.

meta-title: Faster and cheaper Claude Code with Ory Lumen | Ory

---

[Ory Lumen](https://github.com/ory/lumen) makes Claude Code faster and cheaper
by adding local semantic search through SQLite-vec.

As Ory's codebase has grown, I have noticed Claude Code getting slower and more
expensive to use. The reason is straightforward: Claude defaults to grep, glob,
and find, and those tools depend on exact matches. More code + LLM guesses create more
surface area, which leads to more tool calls, higher token costs, higher context use, and slower
task completion.

When you ask Claude to find a function or understand a module, it guesses file
and function names and then tries to match them exactly. That works in a small
codebase. In a larger one, it becomes expensive in both time and API costs. The
problem gets worse as the codebase grows, which is the point where you need it
to improve.

Want to try it now? Go to the README at
https://github.com/ory/lumen and, if it is useful, leave a star.

## The root cause

I wrote about this in more detail recently: agents struggle to build and keep a
durable mental model of a codebase. They repeatedly rediscover code by guessing
and reading files instead of keeping reference of where what lives. That is a constraint of
how LLMs work today, not a bug waiting for a patch. Ory Lumen
(https://github.com/ory/lumen) is a practical response to that constraint. It
improves discoverability in codebases large and small!

## What is Ory Lumen?

**[Ory Lumen](https://github.com/ory/lumen)** is a local semantic code search
engine that runs as an MCP server alongside Claude Code. It indexes your
codebase with local embedding models and exposes a `semantic_search` tool that
Claude can call instead of reading files directly. Claude can then find relevant
functions, types, and modules by meaning, without opening large numbers of files
to inspect them.

How it works:

1. On session start, Lumen walks your project and chunks each file into semantic
   units such as functions, methods, and types. Go uses the native AST parser.
   All other languages use tree-sitter grammars.
2. Those chunks are embedded with a local model and stored in SQLite with
   sqlite-vec for vector search.
3. When Claude needs relevant code, it calls `semantic_search` and gets back
   the relevant chunks without touching the files.

Everything runs on your machine. There are no API keys, cloud services, or
external dependencies. The embedding backend is
[Ollama](https://ollama.com/) or [LM Studio](https://lmstudio.ai/). The index is
stored at `~/.local/share/lumen/<hash>/index.db`, keyed by project path and
model name. Nothing is added to your repo.

### Re-indexing stays fast

On the first run, Lumen builds a Merkle tree over file hashes. On later
sessions, it re-chunks and re-embeds only changed files. In large codebases,
re-indexing after the first run takes seconds.

## The results

Lumen is evaluated with a SWE-bench-style harness: real GitHub bugs, real
codebases, Claude fixing them with and without Lumen. Patches are rated by a
blind judge against the known-correct fix. Full methodology, raw data, and
reproduction instructions are in
[docs/BENCHMARKS.md](https://github.com/ory/lumen/blob/main/docs/BENCHMARKS.md).

| Language   | Cost Reduction | Time Reduction | Output Token Reduction | Quality        |
| ---------- | -------------- | -------------- | ---------------------- | -------------- |
| Rust       | **39%**        | **34%**        | 31%                    | Same (Poor)    |
| JavaScript | **33%**        | **53%**        | **66%**                | Same (Perfect) |
| TypeScript | **27%**        | **33%**        | **64%**                | Same (Good)    |
| PHP        | **27%**        | **34%**        | **59%**                | Same (Good)    |
| Ruby       | **24%**        | 11%            | 9%                     | Same (Good)    |
| Python     | **20%**        | **29%**        | 36%                    | Same (Perfect) |
| Go         | **12%**        | 9%             | 10%                    | Same (Good)    |
| C++        | 8%             | 3%             | +42% (feature task)    | Same (Good)    |

**Cost was reduced in every language tested. Quality was maintained in every
task — zero regressions.** Across 8 benchmark runs on 8 languages, Lumen
reduced costs by 26% on average and time by 28% for bug-fix tasks. Output token
reduction is the most consistent signal: when Claude has semantic search, it
spends less effort exploring and more effort acting.

JavaScript stands out: the same Perfect-quality patches in about half the time,
with two-thirds fewer output tokens. Rust shows the largest cost reduction at
39%, even on a task that was too hard for either approach, which means Lumen
reduced the cost of failure.

The benchmark suite covers 8 languages: Go, JavaScript, PHP, Python, Ruby,
Rust, TypeScript, and C++. All tasks are hard-difficulty tasks from real
open-source projects. The results are fully reproducible — and we run them
repeatedly to confirm consistency.

## Installing it

We launched an
[Ory Claude plugin marketplace](https://github.com/ory/claude-plugins) alongside
Lumen today. Lumen is the first plugin in it. Inside Claude Code, run:

```
/plugin marketplace add ory/claude-plugins
/plugin install lumen@ory
```

Lumen downloads its binary automatically from the latest GitHub release, indexes
your project on the next session start, and registers the `semantic_search`
tool. Claude picks it up without additional configuration.

**Prerequisites:**

1. Install [Ollama](https://ollama.com/) and pull the default embedding model:

```bash
$ ollama pull ordis/jina-embeddings-v2-base-code
```

2. Have [Claude Code](https://code.claude.com/) installed.

The plugin also includes two skills: `/lumen:doctor` for a health check and
`/lumen:reindex` to force a full re-index after a large refactor.

## On keeping it local

One constraint I was not willing to drop was keeping code on your own machine.
Sending source code to an external embedding API should be a deliberate
engineering decision, not the default. Lumen runs entirely on local hardware
with open-source models. The embeddings do not leave your network.

That also makes it usable in air-gapped environments, which matters for
companies running Ory's self-hosted products.

## Contributing

Ory Lumen is a new project and still has rough edges. It will improve over time.
We want it to perform better on more benchmarks, and we welcome contributions
from anyone who wants to help make Claude Code faster and cheaper with
local-first tooling. If you find something, contribute on our
[GitHub](https://github.com/ory/lumen).

## Don't vibe code auth, use Ory

If you are thinking about login, user management, permissions, oauth2, oidc, or
sso, do not vibe code it. Use Ory's open source technology instead. It runs
anywhere, integrates with any stack, and is built for security and reliability.
Check out [Ory Kratos](https://www.ory.sh/kratos) for identity,
[Ory Keto](https://www.ory.sh/keto) for permissions,
[Ory Hydra](https://www.ory.sh/hydra) for oauth2 and oidc, and
[Ory Oathkeeper](https://www.ory.sh/oathkeeper) for access control. Identity is
infrastructure, not application code. Use the right tools for the job.