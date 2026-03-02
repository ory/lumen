# Benchmark Summary

Generated: 2026-03-01 13:16 UTC | Results: `20260301-135442`

| Scenario     | Description                      |
| ------------ | -------------------------------- |
| **baseline** | All default Claude tools, no MCP |
| **mcp-only** | `semantic_search` MCP tool only  |
| **mcp-full** | All default tools + MCP          |

## Overall: Aggregated by Scenario

Totals across all 5 questions × 2 models.

| Model      | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |
| ---------- | -------- | ---------- | --------------- | ---------------- | ---------------- |
| **sonnet** | baseline | 478.4s     | 166035          | 8625             | $5.0423          |
| **sonnet** | mcp-only | 326.3s     | 801470          | 17730            | $4.4506          |
| **sonnet** | mcp-full | 286.7s     | 1049775         | 15182            | $5.9446          |
| **opus**   | baseline | 675.2s     | 266022          | 9043             | $13.3065         |
| **opus**   | mcp-only | 238.4s     | 327155          | 11802            | $1.9308          |
| **opus**   | mcp-full | 439.7s     | 179671          | 8865             | $6.1377          |

---

## label-matcher [easy]

> What label matcher types are available and how is a Matcher created? Show the
> type definitions and constructor.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner    |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | --------- |
| **sonnet** | baseline | 20.7s    | 27763     | 28104      | 727        | $0.2411    |           |
| **sonnet** | mcp-only | 10.9s    | 25075     | 0          | 759        | $0.1444    |           |
| **sonnet** | mcp-full | 13.1s    | 44008     | 42156      | 769        | $0.2603    |           |
| **opus**   | baseline | 14.9s    | 44898     | 42345      | 712        | $0.2635    |           |
| **opus**   | mcp-only | 10.4s    | 17478     | 0          | 576        | $0.1018    | 🏆 Winner |
| **opus**   | mcp-full | 15.9s    | 46124     | 42345      | 674        | $0.2686    |           |

### Quality Ranking (Opus 4.6)

## Content Quality

All six answers are substantively correct and nearly identical in content — they
all show the four `MatchType` constants, the `Matcher` struct, `NewMatcher`, and
mention `MustNewMatcher`. Differences are minor formatting choices.

1. **sonnet/baseline** — Most complete: shows full code for both `NewMatcher`
   and `MustNewMatcher`, accurate line references, clear explanatory text.
2. **opus/mcp-full** — Correct, notes the duplicate file issue, mentions
   `String()` method and `matchTypeToStr` which adds context others omit;
   doesn't inline `MustNewMatcher` code but references it.
3. **sonnet/mcp-full** — Clean, accurate line references, good explanatory
   notes; essentially equivalent to sonnet/mcp-only.
4. **sonnet/mcp-only** — Accurate with line references and includes the
   `matchTypeToStr` array which is a nice extra; very close to sonnet/mcp-full.
5. **opus/mcp-only** — Correct and concise with a nice table for enum values;
   accurate line references.
6. **opus/baseline** — Correct, notes the duplicate file, good line references;
   slightly less polished formatting.

All answers are high quality with no meaningful errors. The differences are
cosmetic.

## Efficiency

**opus/mcp-only** is the clear efficiency winner: fastest runtime (10.4s),
lowest token usage (17,478 input), and lowest cost ($0.10). **sonnet/mcp-only**
is a close second (10.9s, $0.14). The baseline and mcp-full runs for both models
cost 1.5–2.6× more with no meaningful quality improvement. Cache reads inflate
token counts and costs for the baseline/mcp-full scenarios.

## Verdict

**Winner: opus/mcp-only**

---

## histogram [medium]

> How does histogram bucket counting work? Show me the relevant function
> signatures.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner    |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | --------- |
| **sonnet** | baseline | 103.7s   | 42625     | 42156      | 1022       | $1.5573    |           |
| **sonnet** | mcp-only | 12.4s    | 16741     | 0          | 634        | $0.0996    |           |
| **sonnet** | mcp-full | 13.3s    | 28888     | 28104      | 571        | $0.1728    |           |
| **opus**   | baseline | 36.1s    | 122700    | 84690      | 1639       | $0.6968    |           |
| **opus**   | mcp-only | 20.4s    | 20398     | 0          | 885        | $0.1241    |           |
| **opus**   | mcp-full | 17.9s    | 32614     | 28230      | 776        | $0.1966    | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Most complete and well-structured answer. Covers the full
   observation flow (entry point → counting → storage → limiting → validation →
   iteration) with accurate file:line references, correct function signatures,
   and clear explanation of the schema-based key computation. Nothing
   extraneous.

2. **opus/mcp-only** — Nearly as thorough as opus/mcp-full, with excellent
   coverage of `limitBuckets` strategies and the `addAndReset` function not
   mentioned elsewhere. Slightly less polished organization but includes
   `validateCount` and `addToBucket` signatures accurately.

3. **sonnet/baseline** — Covers both the model-layer iterators
   (`regularBucketIterator`, `cumulativeBucketIterator`) and the client-side
   `prom_histogram.go` functions well, but some signatures look reconstructed
   rather than precisely quoted, and it spreads across many sections without a
   clear flow narrative.

4. **opus/baseline** — Strong on the model-layer `histogram.go` side (delta
   encoding, spans, validation) with a useful table of signatures, but
   completely misses the `prom_histogram.go` observation/counting path, which is
   arguably the core of "how bucket counting works." Opens with a confused
   sentence about missing helper functions.

5. **sonnet/mcp-full** — Correct but notably thinner than peers. Mentions
   `observe`, key computation, and validation but provides fewer concrete
   signatures and less detail on limiting strategies or iteration.

6. **sonnet/mcp-only** — Accurate on the `observe` method and key computation
   logic, but omits iteration, `addToBucket`, and bucket limiting entirely. No
   function signatures shown despite the question asking for them.

## Efficiency

The MCP-backed runs are dramatically cheaper and faster: sonnet/mcp-only ($0.10,
12s) and opus/mcp-only ($0.12, 20s) deliver strong answers at ~7-15% the cost of
their baseline counterparts. Opus/mcp-full ($0.20, 18s) delivers the
highest-quality answer at under 30% of opus/baseline's cost. Sonnet/baseline is
the outlier at $1.56 — expensive for a mid-tier answer.

## Verdict

**Winner: opus/mcp-full**

---

## tsdb-compaction [hard]

> How does TSDB compaction work end-to-end? Explain the Compactor interface,
> LeveledCompactor, and how the DB triggers compaction. Show relevant types,
> interfaces, and key method signatures.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner    |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | --------- |
| **sonnet** | baseline | 120.4s   | 30099     | 28104      | 1954       | $1.6949    |           |
| **sonnet** | mcp-only | 87.8s    | 305968    | 0          | 4529       | $1.6431    |           |
| **sonnet** | mcp-full | 45.2s    | 80082     | 56208      | 2328       | $0.4867    |           |
| **opus**   | baseline | 189.8s   | 32841     | 28230      | 1950       | $2.6172    |           |
| **opus**   | mcp-only | 53.2s    | 70935     | 0          | 2570       | $0.4189    | 🏆 Winner |
| **opus**   | mcp-full | 93.2s    | 33914     | 28230      | 1611       | $0.6412    |           |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most thorough and well-structured. Covers all three
   trigger paths (appender commit, periodic reload, compactc consumer)
   distinctly, includes `compactHead` signature, explains the planning priority
   order with line references, and details the early-abort logic in
   `compactBlocks`. Excellent balance of depth and clarity.

2. **sonnet/mcp-only** — Equally comprehensive with the best end-to-end flow
   diagram. Uniquely includes the `dbAppender.Commit` code, `DB.run` loop code,
   and the `DB.Compact` orchestration with all four sub-phases spelled out.
   Slightly more verbose but no less accurate; includes `compactOOOHead` detail
   others miss.

3. **opus/baseline** — Concise yet complete. Includes the `head.compactable()`
   threshold formula (`chunkRange/2*3`), which no other answer provides. Good
   tabular format for LeveledCompactor methods. Slightly less detail on the
   planning strategies than the mcp variants.

4. **sonnet/baseline** — Strong coverage with accurate code snippets and a clean
   ASCII flow diagram. Covers `CompactBlockMetas` parent tracking and the
   `compactBlocks` loop well. Minor issue: the `ranges` example says "2h, 6h,
   24h" but actual Prometheus defaults use exponential ranges; this is a
   simplification, not an error.

5. **opus/mcp-full** — Clean four-phase breakdown and good synchronization notes
   (CompactionDelay, cmtx). However, less code shown than peers — uses tables
   and summaries more than actual signatures. The flow diagram is simpler and
   less informative than others.

6. **sonnet/mcp-full** — Shortest and most superficial. Covers the basics
   correctly but omits `compactOOOHead` detail, the trigger mechanism
   (`dbAppender.Commit`), and the `DB.run` loop structure. The `selectDirs` and
   planning logic is under-explained compared to all other answers.

## Efficiency

The opus/mcp-only run stands out: 53.2s runtime and $0.42 cost — the fastest and
cheapest of all six runs while producing the highest-quality answer.
Sonnet/mcp-full is cheapest at $0.49 but delivers the weakest content. The
baseline runs are dramatically more expensive ($1.69–$2.62) due to high
cache-read tokens, with opus/baseline being the costliest at $2.62 for a
mid-tier answer.

## Verdict

**Winner: opus/mcp-only**

---

## promql-engine [very-hard]

> How does PromQL query evaluation work? Explain the evaluation engine, how
> functions are registered and called, how the AST nodes are evaluated, and how
> alert and recording rules trigger evaluations. Show key interfaces, types, and
> function signatures.

### Time & Tokens

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner    |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | --------- |
| **sonnet** | baseline | 139.3s   | 33957     | 28104      | 3218       | $0.8228    |           |
| **sonnet** | mcp-only | 96.2s    | 241469    | 0          | 5708       | $1.3500    |           |
| **sonnet** | mcp-full | 115.6s   | 617837    | 337248     | 6126       | $3.4110    |           |
| **opus**   | baseline | 273.2s   | 33552     | 28230      | 2602       | $7.1551    |           |
| **opus**   | mcp-only | 71.9s    | 114751    | 0          | 3856       | $0.6702    | 🏆 Winner |
| **opus**   | mcp-full | 161.8s   | 34782     | 28230      | 3546       | $1.8265    |           |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Most comprehensive and best-structured: includes a
   detailed AST node table with example PromQL and key fields, safety function
   sets (AtModifierUnsafeFunctions, AnchoredSafeFunctions), VectorMatching
   struct, Alert struct fields, and a clear evaluator methods table with line
   references.
2. **opus/mcp-only** — Nearly as thorough, uniquely covers safety sets and the
   three core AST interfaces (Node/Statement/Expr), with good detail on the
   StepInvariantExpr preprocessing optimization and error handling via panics;
   slightly less polished tables than mcp-full.
3. **opus/baseline** — Well-organized with correct line references, good
   coverage of EvalNodeHelper and the alert state machine including
   keepFiringFor, but lacks the safety sets and VectorMatching struct details.
4. **sonnet/mcp-only** — Clear three-step lifecycle explanation, correctly
   identifies EngineQueryFunc and rule group scheduling, but less precise on
   struct fields and misses unique details like safety sets.
5. **sonnet/mcp-full** — Similar depth to sonnet/mcp-only with good alert state
   machine coverage, but no meaningfully new information for 2.5x the cost.
6. **sonnet/baseline** — Solid coverage including the AST Walk/Visitor (unique
   detail), but least precise on internal struct fields and evaluation paths
   compared to others.

## Efficiency

opus/mcp-only is the standout: it delivers the second-best answer at the
**lowest cost ($0.67)** and **fastest runtime (72s)** — 10x cheaper than
opus/baseline and nearly half the cost of sonnet/baseline, while producing a
higher-quality answer than both. sonnet/mcp-full is the worst value proposition
at $3.41 for quality comparable to sonnet/mcp-only at $1.35.

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

| Model      | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner    |
| ---------- | -------- | -------- | --------- | ---------- | ---------- | ---------- | --------- |
| **sonnet** | baseline | 94.1s    | 31591     | 28104      | 1704       | $0.7261    |           |
| **sonnet** | mcp-only | 118.8s   | 212217    | 0          | 6100       | $1.2136    |           |
| **sonnet** | mcp-full | 99.4s    | 278960    | 168624     | 5388       | $1.6138    |           |
| **opus**   | baseline | 161.1s   | 32031     | 28230      | 2140       | $2.5738    |           |
| **opus**   | mcp-only | 82.4s    | 103593    | 0          | 3915       | $0.6158    | 🏆 Winner |
| **opus**   | mcp-full | 150.7s   | 32237     | 28230      | 2258       | $3.2048    |           |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/mcp-full** — Most comprehensive: covers both the scrape engine and
   client library as distinct subsystems, includes the
   `Manager.Run → reloader → reload` chain, detailed `scrapeAndReport` 5-step
   flow, parser selection table, counter dual-variable design, and full
   end-to-end diagram showing both halves connected at `/metrics`.

2. **opus/mcp-only** — Nearly as thorough with the best registry coverage
   (includes `Gatherers` multi-registry merging, `Register` 5-step validation,
   `Gather` fan-out/fan-in), plus solid line references (e.g.,
   `prom_registry.go:269-324`), `scrapeCache` ref caching, and the
   `sampleMutator` relabeling step.

3. **sonnet/mcp-only** — Very detailed with good two-system framing, explicit
   `Manager.ApplyConfig` GOMAXPROCS semaphore detail, clear `scrapePool.sync`
   reconciliation logic, and `processMetric` type-inference switch statement;
   slightly more verbose than needed.

4. **opus/mcp-full** — Concise and accurate, includes `scrapeCache` ping-pong
   `seriesCur/seriesPrev` swap, `StaleNaN` hex value, suffix collision detection
   in `processMetric`, and `updateStaleMarkers` step; good density but covers
   slightly fewer sub-topics.

5. **opus/baseline** — Impressively compact while covering all major components
   including `TSDB Head` with `stripeSeries` and `MemPostings` (unique among
   answers), `scrapeCache` details, and stale markers; correct throughout but
   sparser on code examples.

6. **sonnet/baseline** — Correct and well-structured with good `SeriesRef`
   optimization explanation and `processMetric` consistency validation section,
   but lacks some depth on the Manager coordination chain and has less precise
   line references.

## Efficiency

opus/mcp-only is the standout: lowest cost ($0.62), fastest runtime (82s), and
fewest input tokens (104K) while producing a top-tier answer — roughly 4×
cheaper and 2× faster than opus/baseline or opus/mcp-full. sonnet/baseline
($0.73, 94s) is similarly cheap but produces a noticeably weaker answer. The
mcp-full runs for both models are the most expensive without proportional
quality gains.

## Verdict

**Winner: opus/mcp-only**

---

## Overall: Algorithm Comparison

| Question        | Difficulty | 🏆 Winner     | Runner-up       |
| --------------- | ---------- | ------------- | --------------- |
| label-matcher   | easy       | opus/mcp-only | sonnet/mcp-only |
| histogram       | medium     | opus/mcp-full | sonnet/mcp-only |
| tsdb-compaction | hard       | opus/mcp-only | sonnet/mcp-full |
| promql-engine   | very-hard  | opus/mcp-only | sonnet/baseline |
| scrape-pipeline | very-hard  | opus/mcp-only | sonnet/baseline |

**Scenario Win Counts** (across all 5 questions):

| Scenario | Wins |
| -------- | ---- |
| baseline | 0    |
| mcp-only | 4    |
| mcp-full | 1    |

**Overall winner: mcp-only** — won 4 of 5 questions.

_Full answers and detailed analysis: `detail-report.md`_
