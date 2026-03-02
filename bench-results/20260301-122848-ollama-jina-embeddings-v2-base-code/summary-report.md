# Benchmark Summary

Generated: 2026-03-01 11:49 UTC | Results: `20260301-122848`

| Scenario     | Description                      |
| ------------ | -------------------------------- |
| **baseline** | All default Claude tools, no MCP |
| **mcp-only** | `semantic_search` MCP tool only  |
| **mcp-full** | All default tools + MCP          |

## Overall: Aggregated by Scenario

Totals across all 5 questions × 2 models.

| Model      | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |
| ---------- | -------- | ---------- | --------------- | ---------------- | ---------------- |
| **sonnet** | baseline | 496.8s     | 159194          | 8234             | $5.9746          |
| **sonnet** | mcp-only | 228.9s     | 376922          | 12595            | **$2.1995**      |
| **sonnet** | mcp-full | **216.9s** | 424465          | 11758            | $2.5427          |
| **opus**   | baseline | 478.0s     | 579141          | 10829            | $9.6577          |
| **opus**   | mcp-only | 229.9s     | 300322          | 11692            | **$1.7939**      |
| **opus**   | mcp-full | **422.6s** | 383725          | 10307            | $6.4194          |

---

## label-matcher [easy]

> What label matcher types are available and how is a Matcher created? Show the
> type definitions and constructor.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | ------ |
| **sonnet** | baseline | 26.6s    | 29344     | 28104      | 667        | $0.2669    |        |
| **sonnet** | mcp-only | 12.3s    | 18929     | 0          | 715        | $0.1125    |        |
| **sonnet** | mcp-full | 9.4s     | 29935     | 28104      | 574        | $0.1781    |        |
| **opus**   | baseline | 17.2s    | 46482     | 42345      | 537        | $0.2670    |        |
| **opus**   | mcp-only | 11.7s    | 18953     | 0          | 504        | $0.1074    |        |
| **opus**   | mcp-full | 12.7s    | 31142     | 28230      | 559        | $0.1838    |        |

### Quality Ranking (Opus 4.6)

## Content Quality

**sonnet/mcp-full** — Correct, complete, includes all type definitions and full
constructor source with accurate line references. Clean presentation with key
points summarized concisely.

**opus/mcp-full** — Equally correct and complete with full constructor source
and accurate references. Slightly more compact.

**sonnet/mcp-only** — Correct and complete but introduces confusion by noting
"two files with identical definitions" (labels_matcher.go and matcher.go), which
is a distraction and potentially misleading about the codebase structure.

**sonnet/baseline** — Correct and complete, references `matcher.go` rather than
`labels_matcher.go` but otherwise solid. Good note about eager regex
compilation.

**opus/baseline** — Correct and concise with accurate references, though
slightly less detailed (no full constructor body shown).

**opus/mcp-only** — Correct and well-structured with accurate references,
comparable to opus/baseline in detail level.

## Efficiency

MCP-only runs are cheapest (~$0.11) and fastest (~12s), while baseline runs are
most expensive (~$0.27). The mcp-full runs sit in between (~$0.18). For this
straightforward lookup question, mcp-only provides the best cost efficiency with
minimal quality tradeoff.

## Verdict

**Winner: opus/mcp-full**

---

## histogram [medium]

> How does histogram bucket counting work? Show me the relevant function
> signatures.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | ------ |
| **sonnet** | baseline | 119.6s   | 31571     | 28104      | 925        | $2.6238    |        |
| **sonnet** | mcp-only | 19.1s    | 23359     | 0          | 929        | $0.1400    |        |
| **sonnet** | mcp-full | 16.3s    | 35499     | 28104      | 742        | $0.2101    |        |
| **opus**   | baseline | 53.1s    | 181341    | 98805      | 2112       | $1.0089    |        |
| **opus**   | mcp-only | 22.2s    | 20842     | 0          | 957        | $0.1281    |        |
| **opus**   | mcp-full | 22.6s    | 33036     | 28230      | 840        | $0.2003    |        |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most comprehensive: covers bucket creation functions
   (LinearBuckets, ExponentialBuckets), core observation, bucket
   limiting/resolution reduction, validation, and iteration with accurate line
   references throughout. Minor excess detail but nothing incorrect.

2. **sonnet/baseline** — Strong coverage of classic and native paths, hot/cold
   scheme, TSDB iteration layer, and validation. Includes `addToBucket` and
   `validateCount` with line references. Slightly less organized but very
   thorough.

3. **opus/mcp-full** — Clean, well-structured explanation of both bucket systems
   with accurate code snippets for the key math (Frexp decomposition, schema
   branching). Covers cumulative conversion and delta-encoded iteration. Concise
   without sacrificing correctness.

4. **opus/mcp-only** — Comparable to opus/mcp-full with good coverage of both
   systems, hot/cold swap, and iteration types. Slightly more verbose on the
   native bucket routing but accurate throughout.

5. **sonnet/mcp-full** — Accurate and well-organized with clear sections. Covers
   findBucket, observe, hot/cold dispatch, and cumulative write. Slightly less
   detail on iteration and validation than top answers.

6. **sonnet/mcp-only** — Solid coverage of the core observation path with good
   detail on the double-buffering scheme and implicit +Inf bucket. Misses
   iteration and validation; narrower scope than others.

## Efficiency

The MCP-only runs (sonnet at $0.14/19s, opus at $0.13/22s) are 7-20× cheaper
than baselines while delivering answers of comparable quality. MCP-full runs add
~$0.07-0.08 for cache-read tokens with minimal quality gain over MCP-only. The
baselines are dramatically more expensive (sonnet baseline at $2.62 is an
outlier, opus baseline at $1.01 with 181K input tokens), making them poor value
propositions despite slightly more comprehensive answers.

## Verdict

**Winner: opus/mcp-only**

---

## tsdb-compaction [hard]

> How does TSDB compaction work end-to-end? Explain the Compactor interface,
> LeveledCompactor, and how the DB triggers compaction. Show relevant types,
> interfaces, and key method signatures.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | ------ |
| **sonnet** | baseline | 96.3s    | 30539     | 28104      | 1877       | $1.1318    |        |
| **sonnet** | mcp-only | 45.9s    | 49394     | 0          | 2299       | $0.3044    |        |
| **sonnet** | mcp-full | 53.1s    | 87711     | 56208      | 2532       | $0.5300    |        |
| **opus**   | baseline | 81.4s    | 285260    | 155265     | 3211       | $1.5842    |        |
| **opus**   | mcp-only | 49.5s    | 42393     | 0          | 2241       | $0.2680    |        |
| **opus**   | mcp-full | 38.3s    | 51437     | 28230      | 1785       | $0.3159    |        |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most comprehensive: uniquely covers all three trigger
   paths (appender commit, periodic tick, stale series), compactor
   initialization, internal `write` temp-dir naming convention, and planning
   strategies with clear enumeration. Excellent structure and specific line
   references throughout.
2. **opus/mcp-full** — Nearly as complete, includes initialization code and
   control mechanisms (EnableCompactions/DisableCompactions), mentions
   exponential backoff on failure, but slightly less detailed on write
   internals.
3. **sonnet/baseline** — Uniquely includes `BlockMeta`/`CompactionMeta` type
   definitions and `reloadBlocks` detail with retention/deletion tracking,
   making it the most complete on the data model side.
4. **opus/baseline** — Good coverage of `DefaultBlockPopulator.PopulateBlock`
   and control mechanisms, but at extreme cost; the content doesn't justify the
   5-6x price premium.
5. **sonnet/mcp-full** — Solid planning algorithm breakdown with three
   strategies clearly enumerated, good `compactHead` code snippet, but less
   coverage of trigger paths.
6. **sonnet/mcp-only** — Covers the core flow well with the background loop code
   and three phases, but slightly less detailed on initialization, control
   mechanisms, and write internals.

## Efficiency

The baseline runs are dramatically more expensive ($1.13–$1.58) and slower
(81–96s) than the MCP variants ($0.27–$0.53, 38–53s), with no meaningful quality
advantage — opus/baseline's 285K input tokens and $1.58 cost is particularly
wasteful. Among MCP runs, opus/mcp-only ($0.27, 50s) and opus/mcp-full ($0.32,
38s) offer the best quality-to-cost ratio, with opus/mcp-full being fastest and
opus/mcp-only being cheapest while producing the most comprehensive answer.

## Verdict

**Winner: opus/mcp-only**

---

## promql-engine [very-hard]

> How does PromQL query evaluation work? Explain the evaluation engine, how
> functions are registered and called, how the AST nodes are evaluated, and how
> alert and recording rules trigger evaluations. Show key interfaces, types, and
> function signatures.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | ------ |
| **sonnet** | baseline | 133.3s   | 33526     | 28104      | 2108       | $1.1311    |        |
| **sonnet** | mcp-only | 85.0s    | 193983    | 0          | 4701       | $1.0874    |        |
| **sonnet** | mcp-full | 75.3s    | 158657    | 84312      | 4247       | $0.9416    |        |
| **opus**   | baseline | 225.7s   | 33429     | 28230      | 2866       | $6.1429    |        |
| **opus**   | mcp-only | 75.5s    | 117776    | 0          | 4107       | $0.6916    |        |
| **opus**   | mcp-full | 89.0s    | 232269    | 84690      | 4661       | $1.3202    |        |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Most complete and well-structured. Covers all four
   requested topics with accurate file:line references, includes the engine
   comment about rules not being handled directly, explains panic-based error
   propagation, and provides clear code signatures. The EvalStmt, Call node, and
   StepInvariantExpr explanations are particularly precise.

2. **opus/mcp-only** — Nearly as thorough as opus/mcp-full with accurate
   references and good coverage of the alert state machine. Slightly less
   polished organization but includes the important engine comment about rules.
   The ChildrenIter mention and EngineOpts coverage add useful detail.

3. **opus/baseline** — Comprehensive and well-organized with correct file
   references. Covers all four areas with good depth. The function dispatch
   section clearly explains the three paths (special, no-matrix, matrix).
   Slightly less detailed on the AST node types than the MCP variants.

4. **sonnet/mcp-full** — Strong coverage with inline code snippets showing the
   actual eval() switch logic. Good flow diagram at the end. Accurate file
   references throughout. Slightly less precise on some line numbers compared to
   opus variants.

5. **sonnet/mcp-only** — Very detailed with the most extensive AST node table
   (lists all concrete types with line references). Good coverage of the
   QueryEngine interface. The "inferred from usage" comment on QueryFunc is
   slightly less authoritative. Includes ChildrenIter detail.

6. **sonnet/baseline** — Solid and correct but slightly less structured. Missing
   the QueryEngine interface definition. The function registry section is
   accurate. The data flow diagram at the end is helpful but the answer feels
   marginally less polished than the others.

## Efficiency

Opus/mcp-only is the standout for efficiency: $0.69 cost and 75.5s runtime while
delivering a top-tier answer — roughly half the cost of opus/baseline ($6.14,
225.7s) and half the cost of opus/mcp-full ($1.32, 89s). Among sonnet runs,
mcp-full ($0.94, 75.3s) offers the best cost-to-quality ratio, but sonnet
answers are a tier below opus in depth. The baseline runs for both models show
the highest costs with opus/baseline being dramatically expensive at $6.14.

## Verdict

**Winner: opus/mcp-only**

---

## scrape-pipeline [very-hard]

> How does Prometheus metrics scraping and collection work? Explain how the
> scrape manager coordinates scrapers, how metrics are parsed from the text
> format, how counters and gauges are tracked internally, and how the registry
> manages metric families. Show the key types and the data flow from scrape to
> in-memory storage.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | ------ |
| **sonnet** | baseline | 120.8s   | 34214     | 28104      | 2657       | $0.8210    |        |
| **sonnet** | mcp-only | 66.4s    | 91257     | 0          | 3951       | $0.5551    |        |
| **sonnet** | mcp-full | 62.6s    | 112663    | 56208      | 3663       | $0.6830    |        |
| **opus**   | baseline | 100.3s   | 32629     | 28230      | 2103       | $0.6546    |        |
| **opus**   | mcp-only | 70.8s    | 100358    | 0          | 3883       | $0.5989    |        |
| **opus**   | mcp-full | 259.8s   | 35841     | 28230      | 2462       | $4.3992    |        |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Deepest architectural insights: identifies the
   three-layer caching strategy, StaleNaN hex value, FNV-64a jitter, xxhash
   deduplication, and CounterVec; best end-to-end diagram with specific
   implementation details.
2. **opus/mcp-only** — Most comprehensive Registry coverage (Registration +
   Gathering + Gatherers composition), unique dual data flow summary covering
   both server-side scraping and client-side exposition paths, with solid line
   references throughout.
3. **sonnet/baseline** — Very thorough with a useful design decisions table,
   detailed staleness/scrapeCache section, and consistent file:line references;
   slightly formulaic but highly accurate.
4. **sonnet/mcp-full** — Good detail on the actual parse loop code and a clear
   "Key difference" callout for counter vs gauge; solid scrapeLoopAppender
   coverage.
5. **sonnet/mcp-only** — Structurally sound with good processMetric breakdown
   and summary table, but offers fewer unique insights than the top entries.
6. **opus/baseline** — Correct and well-organized but notably less detailed than
   other opus answers; mentions pooling/symbol tables/staleness markers briefly
   without the depth of competitors.

## Efficiency

opus/mcp-full delivers the highest quality but at a catastrophic $4.40 and 260s
— 7× the cost and 4× the runtime of peers. The mcp-only scenarios for both
models cluster around $0.56–$0.60 and 66–71s, offering excellent quality-to-cost
ratios. sonnet/baseline is surprisingly expensive ($0.82) and slow (121s) for a
no-tool run due to high cache-read tokens. opus/mcp-only stands out as
near-top-tier quality at the second-lowest cost ($0.60).

## Verdict

**Winner: opus/mcp-only**

---

## Overall: Algorithm Comparison

| Question        | Difficulty | 🏆 Winner     | Runner-up       |
| --------------- | ---------- | ------------- | --------------- |
| label-matcher   | easy       | opus/mcp-full | opus/mcp-only   |
| histogram       | medium     | opus/mcp-only | opus/baseline   |
| tsdb-compaction | hard       | opus/mcp-only | opus/mcp-full   |
| promql-engine   | very-hard  | opus/mcp-only | opus/mcp-full   |
| scrape-pipeline | very-hard  | opus/mcp-only | sonnet/mcp-only |

**Scenario Win Counts** (across all 5 questions):

| Scenario | Wins |
| -------- | ---- |
| baseline | 0    |
| mcp-only | 4    |
| mcp-full | 1    |

**Overall winner: opus/mcp-only** (4/5 questions).

_Full answers and detailed analysis: `detail-report.md`_
