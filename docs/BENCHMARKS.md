# Benchmarks

`bench-mcp.sh` runs 5 questions of increasing difficulty against
[Prometheus/TSDB Go fixtures](../testdata/fixtures/go), across 2 models (Sonnet
4.6, Opus 4.6) and 3 scenarios:

- **baseline** — default tools only (grep, file reads), no MCP
- **mcp-only** — `semantic_search` only, no file reads
- **with-lumen** — all tools + `semantic_search`

Answers are ranked blind by an LLM judge (Opus 4.6). Benchmarks are transparent
(check bench-results) and reproducible. Please note that **mcp-only** disables
built-in tools from Claude Code which could impact tool performance, even though
benchmarks show no sign of it.

## Speed & cost — Ollama (jina-embeddings-v2-base-code, 768-dim)

Totals across all 5 questions × 2 models:

| Model      | Scenario | Total Time               | Total Cost              |
| ---------- | -------- | ------------------------ | ----------------------- |
| Sonnet 4.6 | baseline | 496.8s                   | $5.97                   |
| Sonnet 4.6 | mcp-only | 228.9s (**2.2× faster**) | $2.20 (**63% cheaper**) |
| Opus 4.6   | baseline | 478.0s                   | $9.66                   |
| Opus 4.6   | mcp-only | 229.9s (**2.1× faster**) | $1.79 (**81% cheaper**) |

## Answer quality — Ollama

Baseline never wins. `mcp-only` wins all medium/hard/very-hard questions at a
fraction of the cost.

| Question        | Difficulty | Winner          | Judge summary                                                                                                                           |
| --------------- | ---------- | --------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| label-matcher   | easy       | opus / with-lumen | Correct, complete; full type definitions and constructor source with accurate line references                                           |
| histogram       | medium     | opus / mcp-only | Good coverage of both bucket systems (classic + native), hot/cold swap, and iteration; 7–20× cheaper than baseline                      |
| tsdb-compaction | hard       | opus / mcp-only | Uniquely covers all three trigger paths, compactor initialization, and planning strategies; 5–6× cheaper than baseline                  |
| promql-engine   | very-hard  | opus / mcp-only | Thorough coverage of all four topics (engine, functions, AST, rules) with accurate file:line references; half the cost of opus/baseline |
| scrape-pipeline | very-hard  | opus / mcp-only | Best Registry coverage; unique dual data-flow summary for scraping and exposition paths                                                 |

`mcp-only` wins 4/5, `with-lumen` wins 1/5, `baseline` wins 0/5.

## Speed & cost — LM Studio (nomic-embed-code, 3584-dim)

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

## Answer quality — LM Studio

The higher-dimensional embeddings produce quality results that match or exceed
the Ollama run:

| Question        | Difficulty | Winner          | Judge summary                                                                                        |
| --------------- | ---------- | --------------- | ---------------------------------------------------------------------------------------------------- |
| label-matcher   | easy       | opus / mcp-only | All answers correct; mcp-only fastest (10.4s) and cheapest ($0.10) at equal quality                  |
| histogram       | medium     | opus / with-lumen | Full observation flow, function signatures, schema-based key computation; ~15× cheaper than baseline |
| tsdb-compaction | hard       | opus / mcp-only | Covers all 3 trigger paths, planning priority order, early-abort logic; 6× cheaper at $0.42          |
| promql-engine   | very-hard  | opus / mcp-only | Function safety sets, storage interfaces, full eval pipeline; $0.67 vs $7.16 baseline                |
| scrape-pipeline | very-hard  | opus / mcp-only | Best registry coverage; Register 5-step validation, Gatherers merging, ApplyConfig hot-reload        |

`mcp-only` wins 4/5, `with-lumen` wins 1/5, `baseline` wins 0/5.

## Extended benchmarks: Results by Language

A comprehensive benchmark comparing 4 embedding models across 9 questions of
varying difficulty in Go, Python, and TypeScript (36 question/model
combinations, 216 total runs). **Embedding model performance varies
significantly by programming language.** Python shows uniform MCP-only
dominance, Go shows strong MCP performance, and TypeScript reveals
over-retrieval issues with larger-dimension models.

**Why language matters:** Larger-dimension models (qwen3-8b, qwen3-4b, nomic)
embed more semantic detail but retrieve redundant chunks for simple TypeScript
questions. This drives up token costs without improving answer quality. Jina's
768-dim embeddings avoid over-retrieval entirely while maintaining strong
quality across all languages.

### Go Results

| Model    | baseline<br/>Cost | baseline<br/>Time | mcp-only<br/>Cost | mcp-only<br/>Time | mcp-only<br/>Speedup | mcp-only<br/>Savings | with-lumen<br/>Cost | with-lumen<br/>Time | Wins (base / mcp-o / mcp-f) |
| -------- | ----------------- | ----------------- | ----------------- | ----------------- | -------------------- | -------------------- | ----------------- | ----------------- | --------------------------- |
| jina-v2  | $10.64            | 536s              | $1.03             | 142s              | 3.8x                 | 90%                  | $1.63             | 149s              | 0/3 / 1/3 / 2/3             |
| qwen3-8b | $4.59             | 421s              | $1.05             | 165s              | 2.6x                 | 77%                  | $1.84             | 168s              | 0/3 / 2/3 / 1/3             |
| qwen3-4b | $8.35             | 433s              | $2.19             | 186s              | 2.3x                 | 74%                  | $2.52             | 179s              | 0/3 / 3/3 / 0/3             |
| nomic    | $5.46             | 469s              | $1.55             | 280s              | 1.7x                 | 72%                  | $1.96             | 229s              | 0/3 / 1/3 / 2/3             |

**Insight:** Qwen3-4b wins the most scenarios (3/3 mcp-only), but **jina
achieves 90% cost savings and 3.8× speedup**—by far the most efficient. No
baseline wins on Go questions across any model.

### Python Results

| Model    | baseline<br/>Cost | baseline<br/>Time | mcp-only<br/>Cost | mcp-only<br/>Time | mcp-only<br/>Speedup | mcp-only<br/>Savings | with-lumen<br/>Cost | with-lumen<br/>Time | Wins (base / mcp-o / mcp-f) |
| -------- | ----------------- | ----------------- | ----------------- | ----------------- | -------------------- | -------------------- | ----------------- | ----------------- | --------------------------- |
| jina-v2  | $5.41             | 406s              | $1.53             | 226s              | 1.8x                 | 72%                  | $1.75             | 206s              | 0/3 / 2/3 / 1/3             |
| qwen3-8b | $3.78             | 373s              | $1.69             | 235s              | 1.6x                 | 55%                  | $2.59             | 224s              | 0/3 / 3/3 / 0/3             |
| qwen3-4b | $3.97             | 342s              | $1.80             | 237s              | 1.4x                 | 55%                  | $2.37             | 219s              | 0/3 / 3/3 / 0/3             |
| nomic    | $5.82             | 483s              | $1.99             | 238s              | 2.0x                 | 66%                  | $3.20             | 278s              | 0/3 / 3/3 / 0/3             |

**Insight:** MCP-only dominates universally (all models 2-3/3 wins). Qwen3-8b,
qwen3-4b, and nomic achieve 3/3 mcp-only wins. However, **jina remains
cost-optimal at 72% savings** and lowest baseline cost ($5.41).

### TypeScript Results

| Model    | baseline<br/>Cost | baseline<br/>Time | mcp-only<br/>Cost | mcp-only<br/>Time | mcp-only<br/>Speedup | mcp-only<br/>Savings | with-lumen<br/>Cost | with-lumen<br/>Time | Wins (base / mcp-o / mcp-f) |
| -------- | ----------------- | ----------------- | ----------------- | ----------------- | -------------------- | -------------------- | ----------------- | ----------------- | --------------------------- |
| jina-v2  | $4.86             | 478s              | $2.53             | 332s              | 1.4x                 | 48%                  | $3.88             | 373s              | 1/3 / 1/3 / 1/3             |
| qwen3-8b | $4.12             | 468s              | $2.98             | 359s              | 1.3x                 | 28%                  | $3.81             | 378s              | 1/3 / 2/3 / 0/3             |
| qwen3-4b | $5.44             | 600s              | $4.42             | 399s              | 1.5x                 | 19%                  | $3.76             | 409s              | 2/3 / 1/3 / 0/3             |
| nomic    | $4.84             | 519s              | $3.89             | 411s              | 1.3x                 | 20%                  | $3.84             | 386s              | 0/3 / 2/3 / 1/3             |

**Insight:** The TypeScript chunker is not properly optimized yet and returns
redundant chunks or misses important ones.

### Summary: Why Jina Remains the Default

| Metric                          | jina-v2                        | qwen3-8b             | qwen3-4b        | nomic         |
| ------------------------------- | ------------------------------ | -------------------- | --------------- | ------------- |
| **Best Go cost**                | ✓ 90%                          | 77%                  | 74%             | 72%           |
| **Best Python cost**            | ✓ 72%                          | 55%                  | 55%             | 66%           |
| **Best TypeScript cost**        | ✓ 48%                          | 28%                  | 19%             | 20%           |
| **Consistent across languages** | ✓                              | —                    | —               | —             |
| **No over-retrieval**           | ✓                              | Limited              | Severe          | Moderate      |
| **Verdict**                     | **State of the Art (Default)** | Best quality (Go/Py) | Not recommended | Usable (Opus) |

Full question-level analysis available in
[`detail-report.md` per benchmark](../bench-results/)

## Reproduce

Requires Ollama, the `claude` CLI, `jq`, and `bc`.

```bash
./bench-mcp.sh                                          # all questions, all models
./bench-mcp.sh --model sonnet                           # filter by model
./bench-mcp.sh --question tsdb-compaction               # filter by question
./bench-mcp.sh --model opus --question label-matcher    # combine
```

Results land in `bench-results/<timestamp>/`. The script runs an LLM judge at
the end to rank answers.
