# Benchmark Summary

Generated: 2026-03-01 19:48 UTC  |  Results: `20260301-202246-ollama-qwen3-embedding-4b`

| Scenario | Description |
|----------|-------------|
| **baseline** | All default Claude tools, no MCP |
| **mcp-only** | `semantic_search` MCP tool only |
| **mcp-full** | All default tools + MCP |

## Overall: Aggregated by Scenario

Totals across all 9 questions × 2 models.

| Model | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |
|-------|----------|------------|-----------------|------------------|------------------|
| **sonnet** | baseline | 772.4s | 881568 | 18131 | $9.4751 |
| **sonnet** | mcp-only | 383.1s | 651047 | 20464 | $3.7668 |
| **sonnet** | mcp-full | 404.3s | 778546 | 21380 | $4.6731 |
| **opus** | baseline | 602.7s | 856588 | 17678 | $8.2811 |
| **opus** | mcp-only | 439.4s | 820295 | 21711 | $4.6443 |
| **opus** | mcp-full | 403.7s | 583275 | 15999 | $3.9739 |

---

## go-label-matcher [go / easy]

> What label matcher types are available and how is a Matcher created? Show the type definitions and constructor.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 49.9s | 29281 | 28104 | 827 | $0.7489 |  |
| **sonnet** | mcp-only | 13.6s | 18378 | 0 | 917 | $0.1148 |  |
| **sonnet** | mcp-full | 14.4s | 47130 | 42156 | 829 | $0.2775 |  |
| **opus** | baseline | 16.9s | 43898 | 42345 | 746 | $0.2593 |  |
| **opus** | mcp-only | 11.5s | 18179 | 0 | 631 | $0.1067 | 🏆 Winner |
| **opus** | mcp-full | 15.9s | 45521 | 42345 | 628 | $0.2645 |  |

### Quality Ranking (Opus 4.6)

All six answers are correct and accurately reflect the source file. Ranking:

1. **sonnet/mcp-only** — Most thorough: includes the `matchTypeToStr` mapping table, operator symbols in a clear table, full constructor code, and `MustNewMatcher` code, all with accurate line references.
2. **opus/mcp-only** — Also very complete with full code for all three definitions plus `MustNewMatcher`, accurate line references, and a clean summary paragraph.
3. **sonnet/mcp-full** — Correct and well-structured with accurate line references; slightly less detail than the top two (omits `matchTypeToStr`, briefer on `MustNewMatcher`).
4. **opus/mcp-full** — Concise and accurate with a nice table format for match types, but shows less code (only signatures, not full constructor body).
5. **sonnet/baseline** — Correct with full code shown, but minor line number inaccuracies (says line 22 for `MatchType` when it's line 21-22, says line 56 for `NewMatcher` when it's 55-70); mentions `matcher.go` duplication which is slightly confusing.
6. **opus/baseline** — Correct and concise with good line references; mentions `Matches` method which adds value but slightly goes beyond the question scope.

## Efficiency

The mcp-only runs are dramatically cheaper ($0.11 each) than baseline and mcp-full ($0.25-$0.75), with faster runtimes (11-14s vs 15-50s). Sonnet/baseline is the outlier at $0.75 and 50s. Both mcp-only runs deliver top-quality answers at minimal cost, offering the best quality-to-cost ratio by far.

## Verdict

**Winner: opus/mcp-only**

---

## go-histogram [go / medium]

> How does histogram bucket counting work? Show me the relevant function signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 100.6s | 30271 | 28104 | 1064 | $2.0219 |  |
| **sonnet** | mcp-only | 15.9s | 22093 | 0 | 797 | $0.1304 |  |
| **sonnet** | mcp-full | 15.9s | 34267 | 28104 | 811 | $0.2057 |  |
| **opus** | baseline | 40.2s | 117790 | 70575 | 1555 | $0.6631 |  |
| **opus** | mcp-only | 17.7s | 22079 | 0 | 681 | $0.1274 | 🏆 Winner |
| **opus** | mcp-full | 20.1s | 34317 | 28230 | 936 | $0.2091 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most complete answer: covers findBucket, observe, histogramCounts.observe (with the actual native bucket key computation via math.Frexp), addToBucket, all three limitBuckets strategies, bucket creation helpers, and validation. Accurate line references and correct implementation details throughout.

2. **opus/mcp-full** — Very close to opus/baseline in coverage: observation path, bucket limiting (all three strategies), doubleBucketWidth, makeBuckets, histogramCounts struct, and bucket creation helpers. Slightly less detail on the native key computation but well-organized and accurate.

3. **opus/mcp-only** — Covers the same core flow correctly with accurate line references; includes limitBuckets strategies and makeBuckets serialization. Omits bucket creation helpers and validation but nails the essential counting mechanics.

4. **sonnet/mcp-only** — Clean, accurate coverage of findBucket, observe, addToBucket, makeBuckets, and addAndResetCounts with correct line references. Includes a good note on the double-buffer concurrency model. Misses bucket limiting strategies entirely.

5. **sonnet/mcp-full** — Similar content to sonnet/mcp-only but slightly less detailed explanations. Covers the same core functions with accurate references. Also omits bucket limiting.

6. **sonnet/baseline** — Broadest but least focused: pulls in histogram.go validation/iteration functions and bucket creation helpers, but some line references are slightly off, and the hot/cold merge section gets more attention than the core counting path. Correct but sprawling.

## Efficiency

The MCP-only runs are dramatically cheaper and faster: sonnet/mcp-only and opus/mcp-only both cost ~$0.13 and took 16-18s, versus baseline runs costing $0.66-$2.02 and taking 40-100s. The mcp-full runs sit in between at ~$0.21. Opus/mcp-only delivers near-top-tier quality at the lowest cost tier, making it the best quality-to-cost tradeoff.

## Verdict

**Winner: opus/mcp-only**

---

## go-tsdb-compaction [go / hard]

> How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 110.5s | 634346 | 252936 | 4864 | $3.6312 |  |
| **sonnet** | mcp-only | 79.5s | 252365 | 0 | 3571 | $1.3511 |  |
| **sonnet** | mcp-full | 61.7s | 191127 | 98364 | 2820 | $1.0753 |  |
| **opus** | baseline | 115.0s | 31871 | 28230 | 2192 | $1.0213 |  |
| **opus** | mcp-only | 47.4s | 61921 | 0 | 2177 | $0.3640 | 🏆 Winner |
| **opus** | mcp-full | 50.9s | 80226 | 42345 | 2516 | $0.4852 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most comprehensive: uniquely covers OOO compaction (`compactOOOHead`), stale series compaction trigger, and the 1.5× buffer explanation for `Head.compactable`, all with correct line references and a clean table for LeveledCompactor fields.
2. **sonnet/mcp-only** — Equally thorough with good `PopulateBlock` coverage, `cmtx` mutex details, and the buffered channel nuance; includes accurate line references throughout.
3. **opus/mcp-full** — Covers `EnableCompactions`/`DisableCompactions` uniquely and mentions `CompactionDelay` jitter, but slightly less detailed on the planning internals than the top two.
4. **sonnet/baseline** — Very detailed with `selectDirs` logic and exponential block ranges, but the sheer verbosity doesn't add proportional insight over more concise answers.
5. **opus/baseline** — Solid and well-structured with explicit `splitByRange` coverage, but misses OOO compaction and stale series handling.
6. **sonnet/mcp-full** — Covers all essential points cleanly but is the least detailed of the group, omitting OOO compaction and some planning nuances.

## Efficiency

opus/mcp-only is the standout: $0.36 and 47.4s — roughly 10× cheaper than sonnet/baseline and 2-3× cheaper than the next sonnet option, while producing one of the highest-quality answers. The opus runs consistently dominate on cost and time; among sonnet runs, mcp-full offers the best tradeoff but still costs 3× more than opus/mcp-only for comparable quality.

## Verdict

**Winner: opus/mcp-only**

---

## py-permissions [python / easy]

> How does the Django Permission model work? Show the Permission class, its fields, the PermissionManager, and the get_by_natural_key method.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 32.0s | 28495 | 28104 | 1015 | $0.2772 |  |
| **sonnet** | mcp-only | 15.2s | 18426 | 0 | 888 | $0.1143 |  |
| **sonnet** | mcp-full | 13.7s | 30469 | 28104 | 852 | $0.1877 |  |
| **opus** | baseline | 20.8s | 49167 | 42345 | 894 | $0.2894 |  |
| **opus** | mcp-only | 16.7s | 17469 | 0 | 820 | $0.1078 | 🏆 Winner |
| **opus** | mcp-full | 19.2s | 44606 | 42345 | 837 | $0.2651 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Correct, complete, includes all requested components with accurate line references, and uniquely adds how Permission connects to `PermissionsMixin` and `Group` (lines 342, 121, 317), providing broader architectural context.
2. **opus/mcp-only** — Equally correct and complete with proper file:line references; explanation of the natural key round-trip is clear and concise, though slightly less contextual than baseline.
3. **opus/mcp-full** — Accurate and well-structured with line references, but the `__str__` example ("admin | log entry | Can add log entry") appears to add an extra segment not matching the code's format string.
4. **sonnet/baseline** — Correct with full code blocks, a nice fields table, and accurate line references; slightly verbose but thorough.
5. **sonnet/mcp-full** — Correct and well-organized with file:line references; the "Summary flow" paragraph is a nice touch, though it adds little beyond what's already shown.
6. **sonnet/mcp-only** — Correct but uses only the filename without line numbers (just "line 27–36"), slightly less precise for navigation; otherwise equivalent in content quality.

## Efficiency

The mcp-only runs are dramatically cheaper ($0.11 for both sonnet and opus) compared to baseline and mcp-full runs ($0.19–$0.29), while delivering comparable answer quality. Sonnet/mcp-full at $0.19 and 13.7s is the fastest overall, but opus/mcp-only at $0.11 and 16.7s delivers top-tier quality at the lowest cost among opus runs.

## Verdict

**Winner: opus/mcp-only**

---

## py-flask-config [python / medium]

> How does Flask configuration loading work? Explain the Config class, how it loads from files, environment variables, and Python objects. Show the key methods and class hierarchy.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 63.7s | 30648 | 28104 | 1570 | $0.3080 |  |
| **sonnet** | mcp-only | 32.6s | 51432 | 0 | 1690 | $0.2994 |  |
| **sonnet** | mcp-full | 35.7s | 60159 | 42156 | 1823 | $0.3674 |  |
| **opus** | baseline | 27.1s | 46521 | 42345 | 1183 | $0.2834 |  |
| **opus** | mcp-only | 34.5s | 39589 | 0 | 1731 | $0.2412 | 🏆 Winner |
| **opus** | mcp-full | 28.5s | 48689 | 42345 | 1116 | $0.2925 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most complete and well-organized. Covers all six loading methods plus `ConfigAttribute` and `get_namespace`, with accurate code snippets, correct line references, and a clear call chain summary. The descriptor explanation includes `get_converter` detail others miss.

2. **sonnet/mcp-full** — Equally thorough, covering all methods including `get_namespace`. Adds the `from_mapping` signature with `**kwargs` detail. Slightly more verbose without adding proportional value over opus/mcp-only.

3. **opus/mcp-full** — Concise and accurate with correct line references, covers all methods and both classes. Slightly less detailed on `from_prefixed_env` nested dict mechanics and omits `get_namespace` as a separate section (mentions it briefly at end).

4. **opus/baseline** — Clean and correct, covers all methods with good structure. Includes `get_namespace`. No line references to the actual file, which is expected without tool access but slightly less useful.

5. **sonnet/mcp-only** — Accurate and well-structured with line references. The `from_mapping` return value claim ("All methods return `bool`") is slightly imprecise since `from_object` returns `None` (which it does note). Good `ConfigAttribute` coverage.

6. **sonnet/baseline** — Correct and detailed with good code snippets and design decisions table. Lacks `ConfigAttribute` coverage and `get_namespace`, making it the least complete. No line references.

## Efficiency

Opus/mcp-only delivers the best answer at the lowest cost ($0.24) and moderate runtime (34.5s). Opus/baseline is fastest (27.1s) and cheap ($0.28) but lacks line references. Sonnet/mcp-full is the most expensive ($0.37) without being the best answer. The MCP-only scenarios generally offer better cost efficiency than mcp-full due to lower input token counts from skipping redundant tool calls.

## Verdict

**Winner: opus/mcp-only**

---

## py-django-queryset [python / hard]

> How does the Django QuerySet evaluation and filtering pipeline work? Explain QuerySet chaining, lazy evaluation, the Query class, how lookups and filters are compiled into SQL, and how the Manager ties it all together. Show key classes and method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 112.3s | 33068 | 28104 | 2706 | $0.8947 |  |
| **sonnet** | mcp-only | 63.9s | 80475 | 0 | 3911 | $0.5001 |  |
| **sonnet** | mcp-full | 60.7s | 105716 | 56208 | 3854 | $0.6530 |  |
| **opus** | baseline | 85.9s | 349377 | 141150 | 4038 | $1.9184 |  |
| **opus** | mcp-only | 73.9s | 84272 | 0 | 4481 | $0.5334 | 🏆 Winner |
| **opus** | mcp-full | 61.2s | 98116 | 56460 | 3369 | $0.6030 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most comprehensive: covers Manager descriptor, ManagerDescriptor blocking instance access, all evaluation triggers (including `exists()` and `count()` shortcuts), set operations (`&`, `|`, `^`), and the full Query class API surface with method signatures. Excellent file/line references throughout.

2. **opus/mcp-full** — Nearly as thorough as opus/baseline, with clear structure, accurate code quotes, and good line references. Covers the deferred filter mechanism and iterable classes well. Slightly less coverage of Query class methods and set operations.

3. **opus/mcp-only** — Very strong coverage with accurate code and references. Includes the ManagerDescriptor detail and `__getitem__` slicing logic. Slightly less polished organization than mcp-full but equally correct.

4. **sonnet/mcp-full** — Solid and well-structured with accurate code extractions and line references. Covers all major components but lacks the Query class method catalog and set operations found in opus answers.

5. **sonnet/baseline** — Good coverage with accurate code and a clean end-to-end trace. Includes the iterable classes table and Q object composition. Missing some depth on the Query class internals and `__getitem__` behavior.

6. **sonnet/mcp-only** — Accurate and well-organized but slightly less detailed than sonnet/baseline on iterable classes and evaluation triggers. The full call-stack summary at the end is a nice touch.

## Efficiency

The MCP-only runs deliver the best efficiency: sonnet/mcp-only ($0.50, 64s) and opus/mcp-only ($0.53, 74s) cost 55-72% less than their baseline counterparts while producing answers of comparable or near-comparable quality. The baseline runs are dramatically more expensive — opus/baseline at $1.92 is 3.6x the cost of opus/mcp-only for a modest quality improvement. The mcp-full runs sit in between, adding cache read overhead without meaningfully improving over mcp-only.

## Verdict

**Winner: opus/mcp-only**

---

## ts-disposable [typescript / easy]

> What is the IDisposable interface and how does the Disposable base class work? Show the interface, the base class, and how DisposableStore manages multiple disposables.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 39.5s | 29084 | 28104 | 934 | $0.2929 | 🏆 Winner |
| **sonnet** | mcp-only | 34.1s | 51485 | 0 | 1796 | $0.3023 |  |
| **sonnet** | mcp-full | 29.6s | 49213 | 42156 | 1353 | $0.3010 |  |
| **opus** | baseline | 27.6s | 54472 | 42345 | 1147 | $0.3222 |  |
| **opus** | mcp-only | 65.8s | 195589 | 0 | 2681 | $1.0450 |  |
| **opus** | mcp-full | 27.3s | 66977 | 56460 | 1417 | $0.3985 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/mcp-full** — Correct and complete with accurate line references, includes the `remove` method (though named differently from other answers' `deleteAndLeak`), mentions AggregateError handling, and provides a clear compositional diagram. Minor issue: shows a `remove` method that other answers call `deleteAndLeak`, suggesting possible inaccuracy in method naming.

2. **opus/mcp-full** — Accurate with good line references, includes the `isDisposable` type guard that others miss, explains the standalone `dispose()` function's error aggregation, and has a clean flow summary. Slightly less detailed on DisposableStore internals than some others.

3. **opus/baseline** — Correct, well-structured with a useful table for DisposableStore methods, mentions AggregateError and idempotency, includes a practical usage example showing the pattern in action. Good balance of detail.

4. **sonnet/mcp-only** — Most comprehensive answer with accurate code, a relationship diagram, standalone usage example (`disposeOnReturn`), and correct line references. The extra `disposeOnReturn` example adds genuine value showing standalone DisposableStore usage.

5. **sonnet/baseline** — Accurate and concise with correct line references, covers all three components well, explains the ownership model clearly. Slightly less detailed on error handling.

6. **opus/mcp-only** — Correct content but presented awkwardly — opens with "I have all the pieces" meta-commentary about chunking, which is noise. The actual technical content is solid with accurate line references and good explanation of error handling.

## Efficiency

Opus/mcp-only is a clear outlier at $1.04 and 66s for content that isn't meaningfully better than cheaper runs. The baseline runs and mcp-full runs cluster around $0.29–$0.40 with 27–40s runtimes. Sonnet/baseline offers the cheapest run at $0.29 with strong quality; opus/mcp-full and sonnet/mcp-full deliver slightly richer answers for modest cost increases (~$0.30–$0.40).

## Verdict

**Winner: sonnet/baseline**

---

## ts-event-emitter [typescript / medium]

> How does the event emitter system work? Explain the Event interface, the Emitter class, event composition (map, filter, debounce), and how events integrate with disposables. Show key types and patterns.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 133.4s | 31750 | 28104 | 2006 | $0.6907 |  |
| **sonnet** | mcp-only | 53.1s | 57450 | 0 | 2855 | $0.3586 |  |
| **sonnet** | mcp-full | 62.9s | 143564 | 84312 | 3119 | $0.8380 |  |
| **opus** | baseline | 55.3s | 129502 | 84690 | 2341 | $0.7484 |  |
| **opus** | mcp-only | 48.1s | 46615 | 0 | 2271 | $0.2898 | 🏆 Winner |
| **opus** | mcp-full | 55.9s | 130564 | 84690 | 2273 | $0.7520 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most insightful internals: correctly identifies sparse array compaction at 50%, leak detection refusing listeners at threshold², AsyncEmitter's thenables-freeze pattern, and re-entrant delivery queue mechanics. Precise line references throughout.
2. **opus/mcp-only** — Strong on the `snapshot` bridge pattern with code, excellent 6-point disposable integration breakdown, and good coverage of `ChainableSynthesis` with `HaltChainable` sentinel; line references are accurate and consistent.
3. **sonnet/mcp-only** — Very thorough with accurate internals (UniqueContainer optimization, sparse array), good `snapshot` code example, and covers external event adapters (`fromNodeEventEmitter`/`fromDOMEventEmitter`) that others miss.
4. **opus/mcp-full** — Clean table of all combinators with line numbers, explains the "public events MUST pass DisposableStore" warning others omit, and covers AsyncEmitter's `waitUntil` pattern concisely.
5. **sonnet/mcp-full** — Solid coverage including external adapters and `chain` API, but slightly more verbose without proportionally deeper insight compared to peers.
6. **sonnet/baseline** — Broadest surface coverage (Relay, ValueWithChangeEvent, EventBufferer) but reads more like a reference catalog than an explanation; some internal details are less precise.

## Efficiency

opus/mcp-only is the clear efficiency leader at $0.29 and 48.1s — roughly 60% cheaper and 13% faster than opus/baseline ($0.75, 55.3s) while delivering nearly comparable quality. sonnet/mcp-full is the worst value at $0.84 for a mid-tier answer, and sonnet/baseline is the slowest at 133.4s. The mcp-only scenario consistently outperforms both baseline and mcp-full on cost across both models.

## Verdict

**Winner: opus/mcp-only**

---

## ts-async-lifecycle [typescript / hard]

> How do async operations, cancellation, and resource lifecycle management work together? Explain CancelablePromise, CancellationToken, the async utilities (throttle, debounce, retry), how they integrate with the disposable lifecycle system, and how event-driven patterns compose with async flows. Show key interfaces and class relationships.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 130.2s | 34625 | 28104 | 3145 | $0.6095 | 🏆 Winner |
| **sonnet** | mcp-only | 74.7s | 98943 | 0 | 4039 | $0.5957 |  |
| **sonnet** | mcp-full | 109.2s | 116901 | 70260 | 5919 | $0.7676 |  |
| **opus** | baseline | 213.5s | 33990 | 28230 | 3582 | $2.7757 |  |
| **opus** | mcp-only | 123.3s | 334582 | 0 | 6238 | $1.8289 |  |
| **opus** | mcp-full | 124.3s | 34259 | 28230 | 2907 | $0.7039 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/baseline** — Exceptionally thorough and accurate; covers MutableToken lazy emitter, CancelablePromise auto-dispose of IDisposable results, AsyncEmitter's IWaitUntil with frozen thenables, event combinators, CancellationTokenPool, and parent propagation — all with clear code snippets and a well-structured integration patterns table and relationship diagram.

2. **opus/baseline** — Equally strong on architecture; uniquely highlights bridge functions (`cancelOnDispose`, `thenIfNotDisposed`, `thenRegisterOrDispose`) and provides an excellent cascade diagram showing what happens on `store.dispose()`; slightly less internal detail on MutableToken and AsyncEmitter than sonnet/baseline.

3. **sonnet/mcp-full** — Very thorough with file:line references throughout (e.g., `lifecycle.ts:416`, `cancellation.ts:60-95`); covers AsyncEmitter's `waitUntil` freezing, PauseableEmitter, and DebounceEmitter; the full lifecycle teardown diagram is strong but overall somewhat more verbose.

4. **opus/mcp-only** — Comprehensive with good line references and unique coverage of `thenIfNotDisposed` and `RefCountedDisposable`; the numbered layered structure reads well but the integration section is slightly less polished than top entries.

5. **sonnet/mcp-only** — Solid coverage with line references; uniquely mentions `TaskSequentializer`, `Sequencer`, and `SequencerByKey`; the "Key Design Principles" summary is clear but the class relationship diagram is less detailed.

6. **opus/mcp-full** — Notably shorter and less detailed than all other answers; missing CancellationTokenPool, thin on event combinators, and AsyncEmitter coverage is abbreviated despite having full tool access.

## Efficiency

Opus/baseline delivers arguably the best architectural narrative but at $2.78 and 214s — 4.5× the cost of sonnet/baseline ($0.61, 130s) for marginal quality gain. Sonnet/mcp-only is the fastest (75s) and cheapest ($0.60) but sacrifices some depth. Opus/mcp-full is surprisingly cheap for opus ($0.70) but produced the weakest answer, suggesting the tools didn't help. The best quality-to-cost tradeoff is sonnet/baseline: top-tier content at the lowest cost tier.

## Verdict

**Winner: sonnet/baseline**

---

## Overall: Algorithm Comparison

| Question | Language | Difficulty | 🏆 Winner | Runner-up |
|----------|----------|------------|-----------|-----------|
| go-label-matcher | go | easy | opus/mcp-only | sonnet/mcp-only |
| go-histogram | go | medium | opus/mcp-only | sonnet/mcp-only |
| go-tsdb-compaction | go | hard | opus/mcp-only | opus/mcp-full |
| py-permissions | python | easy | opus/mcp-only | sonnet/mcp-only |
| py-flask-config | python | medium | opus/mcp-only | opus/baseline |
| py-django-queryset | python | hard | opus/mcp-only | sonnet/mcp-only |
| ts-disposable | typescript | easy | sonnet/baseline | sonnet/mcp-full |
| ts-event-emitter | typescript | medium | opus/mcp-only | sonnet/mcp-only |
| ts-async-lifecycle | typescript | hard | sonnet/baseline | sonnet/mcp-only |

**Scenario Win Counts** (across all 9 questions):

| Scenario | Wins |
|----------|------|
| baseline | 2 |
| mcp-only | 7 |
| mcp-full | 0 |

**Overall winner: mcp-only** — won 7 of 9 questions.

_Full answers and detailed analysis: `detail-report.md`_
