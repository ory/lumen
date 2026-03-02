# Benchmark Summary

Generated: 2026-03-01 14:07 UTC  |  Results: `20260301-144739-ollama-qwen3-embedding-8b`

| Scenario | Description |
|----------|-------------|
| **baseline** | All default Claude tools, no MCP |
| **mcp-only** | `semantic_search` MCP tool only |
| **mcp-full** | All default tools + MCP |

## Overall: Aggregated by Scenario

Totals across all 5 questions × 2 models.

| Model | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |
|-------|----------|------------|-----------------|------------------|------------------|
| **sonnet** | baseline | 455.1s | 155608 | 7941 | $3.8839 |
| **sonnet** | mcp-only | 706.9s | 453436 | 13713 | $2.6100 |
| **sonnet** | mcp-full | 254.0s | 430358 | 11737 | $2.5647 |
| **opus** | baseline | 271.3s | 215041 | 5232 | $4.5449 |
| **opus** | mcp-only | 170.1s | 236734 | 9017 | $1.4091 |
| **opus** | mcp-full | 185.6s | 169958 | 4070 | $3.8918 |

---

## label-matcher [easy]

> What label matcher types are available and how is a Matcher created? Show the type definitions and constructor.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 20.0s | 27666 | 28104 | 615 | $0.2308 |  |
| **sonnet** | mcp-only | 10.8s | 17455 | 0 | 622 | $0.1028 |  |
| **sonnet** | mcp-full | 9.8s | 29607 | 28104 | 610 | $0.1773 |  |
| **opus** | baseline | 16.1s | 43036 | 42345 | 728 | $0.2546 |  |
| **opus** | mcp-only | 10.7s | 17489 | 0 | 551 | $0.1012 |  |
| **opus** | mcp-full | 16.4s | 45056 | 42345 | 697 | $0.2639 |  |

### Quality Ranking (Opus 4.6)

_Judge unavailable_

---

## histogram [medium]

> How does histogram bucket counting work? Show me the relevant function signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 62.6s | 29439 | 28104 | 885 | $0.6564 |  |
| **sonnet** | mcp-only | 19.8s | 22162 | 0 | 1016 | $0.1362 |  |
| **sonnet** | mcp-full | 18.4s | 34307 | 28104 | 825 | $0.2062 |  |
| **opus** | baseline | 49.6s | 128149 | 84690 | 1989 | $0.7328 |  |
| **opus** | mcp-only | 17.6s | 22148 | 0 | 899 | $0.1332 |  |
| **opus** | mcp-full | 20.3s | 34377 | 28230 | 765 | $0.2051 |  |

### Quality Ranking (Opus 4.6)

_Judge unavailable_

---

## tsdb-compaction [hard]

> How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 94.7s | 31919 | 28104 | 1596 | $0.5161 |  |
| **sonnet** | mcp-only | 520.5s | 225054 | 0 | 4227 | $1.2309 |  |
| **sonnet** | mcp-full | 84.9s | 69575 | 42156 | 2609 | $0.4342 |  |
| **opus** | baseline | 114.5s | 30450 | 28230 | 2051 | $1.1394 |  |
| **opus** | mcp-only | 43.1s | 35146 | 0 | 2143 | $0.2293 |  |
| **opus** | mcp-full | 57.9s | 76587 | 42345 | 2239 | $0.4601 |  |

### Quality Ranking (Opus 4.6)

_Judge unavailable_

---

## promql-engine [very-hard]

> How does PromQL query evaluation work? Explain the evaluation engine, how functions are registered and called, how the AST nodes are evaluated, and how alert and recording rules trigger evaluations. Show key interfaces, types, and function signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 149.6s | 32766 | 28104 | 2315 | $1.6553 |  |
| **sonnet** | mcp-only | 79.6s | 85716 | 0 | 3908 | $0.5263 |  |
| **sonnet** | mcp-full | 77.0s | 194676 | 84312 | 4296 | $1.1229 |  |
| **opus** | baseline | 90.7s | 13406 | 14115 | 464 | $2.4181 |  |
| **opus** | mcp-only | 98.2s | 161951 | 0 | 5424 | $0.9454 |  |
| **opus** | mcp-full | 90.3s | 13938 | 14115 | 369 | $2.9627 |  |

### Quality Ranking (Opus 4.6)

_Judge unavailable_

---

## scrape-pipeline [very-hard]

> How does Prometheus metrics scraping and collection work? Explain how the scrape manager coordinates scrapers, how metrics are parsed from the text format, how counters and gauges are tracked internally, and how the registry manages metric families. Show the key types and the data flow from scrape to in-memory storage.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 128.1s | 33818 | 28104 | 2530 | $0.8253 |  |
| **sonnet** | mcp-only | 76.1s | 103049 | 0 | 3940 | $0.6137 |  |
| **sonnet** | mcp-full | 63.7s | 102193 | 56208 | 3397 | $0.6240 |  |
| **opus** | baseline | .3s | 0 | 0 | 0 | $0.0000 |  |
| **opus** | mcp-only | .3s | 0 | 0 | 0 | $0.0000 |  |
| **opus** | mcp-full | .5s | 0 | 0 | 0 | $0.0000 |  |

### Quality Ranking (Opus 4.6)

_Judge unavailable_

---

## Overall: Algorithm Comparison

| Question | Difficulty | 🏆 Winner | Runner-up |
|----------|------------|-----------|-----------|
| label-matcher | easy | unknown | opus/mcp-only |
| histogram | medium | unknown | opus/mcp-only |
| tsdb-compaction | hard | unknown | opus/mcp-only |
| promql-engine | very-hard | unknown | sonnet/mcp-only |
| scrape-pipeline | very-hard | unknown | opus/baseline |

**Scenario Win Counts** (across all 5 questions):

| Scenario | Wins |
|----------|------|
| baseline | 0 |
| mcp-only | 0 |
| mcp-full | 0 |

**Overall winner: undetermined** (no judge results available).

_Full answers and detailed analysis: `detail-report.md`_
