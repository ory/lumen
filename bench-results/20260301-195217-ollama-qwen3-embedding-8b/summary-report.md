# Benchmark Summary

Generated: 2026-03-01 19:16 UTC  |  Results: `20260301-195217-ollama-qwen3-embedding-8b`

| Scenario | Description |
|----------|-------------|
| **baseline** | All default Claude tools, no MCP |
| **mcp-only** | `semantic_search` MCP tool only |
| **mcp-full** | All default tools + MCP |

## Overall: Aggregated by Scenario

Totals across all 9 questions × 2 models.

| Model | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |
|-------|----------|------------|-----------------|------------------|------------------|
| **sonnet** | baseline | 723.9s | 294725 | 13305 | $6.5200 |
| **sonnet** | mcp-only | 382.5s | 493992 | 19771 | $2.9642 |
| **sonnet** | mcp-full | 328.7s | 594256 | 16287 | $3.5822 |
| **opus** | baseline | 538.5s | 775762 | 16903 | $5.9717 |
| **opus** | mcp-only | 377.2s | 457254 | 18817 | $2.7567 |
| **opus** | mcp-full | 442.6s | 625404 | 16119 | $4.6602 |

---

## go-label-matcher [go / easy]

> What label matcher types are available and how is a Matcher created? Show the type definitions and constructor.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 19.1s | 28076 | 28104 | 615 | $0.2367 |  |
| **sonnet** | mcp-only | 12.9s | 18112 | 0 | 739 | $0.1090 |  |
| **sonnet** | mcp-full | 14.2s | 46938 | 42156 | 645 | $0.2719 |  |
| **opus** | baseline | 16.7s | 45628 | 42345 | 782 | $0.2689 |  |
| **opus** | mcp-only | 11.4s | 16954 | 0 | 529 | $0.0980 | 🏆 Winner |
| **opus** | mcp-full | 12.2s | 30121 | 28230 | 603 | $0.1798 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

All six answers are essentially identical in correctness and completeness — they all correctly identify the four `MatchType` constants, the `Matcher` struct, the `NewMatcher` constructor, and the `MustNewMatcher` helper. Differences are purely presentational.

1. **opus/baseline** — Correct, complete, nice table format for the constants, accurate line references.
2. **sonnet/mcp-only** — Correct, complete, adds useful detail about `matchTypeToStr` and `FastRegexMatcher` optimizations, good line references.
3. **sonnet/baseline** — Correct, complete, clean presentation with accurate line references and a good design observation about zero regex overhead.
4. **opus/mcp-only** — Correct, complete, concise, accurate line references.
5. **opus/mcp-full** — Correct, complete, concise, accurate line references, mentions FastRegexMatcher optimization.
6. **sonnet/mcp-full** — Correct, complete but slightly less detailed (describes constructor behavior in prose rather than showing full code), accurate line references.

## Efficiency

The mcp-only runs are dramatically cheaper and faster: opus/mcp-only at $0.10 in 11.4s and sonnet/mcp-only at $0.11 in 12.9s, versus baseline/mcp-full runs costing $0.17–$0.27. For a straightforward lookup question where all answers converge on the same content, the mcp-only scenarios offer the best quality-to-cost ratio.

## Verdict

**Winner: opus/mcp-only**

---

## go-histogram [go / medium]

> How does histogram bucket counting work? Show me the relevant function signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 53.0s | 43079 | 42156 | 984 | $0.7076 |  |
| **sonnet** | mcp-only | 20.4s | 22639 | 0 | 1067 | $0.1399 |  |
| **sonnet** | mcp-full | 18.0s | 34762 | 28104 | 885 | $0.2100 |  |
| **opus** | baseline | 47.6s | 207245 | 112920 | 1975 | $1.1421 |  |
| **opus** | mcp-only | 21.8s | 22594 | 0 | 937 | $0.1364 |  |
| **opus** | mcp-full | 20.3s | 34832 | 28230 | 892 | $0.2106 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Most precise and well-structured: correctly covers both histogram systems, provides accurate file:line references (prom_histogram.go:652-706, histogram.go:481, etc.), explains the three-way routing for native buckets, and includes the completion-signal detail about count being incremented last. Concise without sacrificing depth.

2. **opus/mcp-only** — Nearly identical content to opus/mcp-full with accurate line references and good structural organization; slightly more verbose with the PromQL section which adds marginal value, but otherwise excellent coverage of both observation and iteration paths.

3. **sonnet/mcp-full** — Correct and focused with good line references; uniquely includes the actual `math.Frexp` key computation code inline, which directly answers "how does it work"; slightly less complete on iteration/span-based encoding than the opus answers.

4. **sonnet/mcp-only** — Strong coverage with accurate line references and a helpful summary flow diagram; includes the `histogramCounts` struct definition which adds context, though it's somewhat long and the cumulative iterator section is thin.

5. **opus/baseline** — Comprehensive and correct, covering classic buckets, native buckets, bucket limiting, and iteration; good function signatures with line numbers, but spread across more categories than necessary, making it harder to follow the core counting flow.

6. **sonnet/baseline** — Covers the right concepts but lacks specific file:line references, mixes in bucket creation helpers (LinearBuckets, ExponentialBuckets) that aren't central to "how counting works," and the function signatures for iterators lack file locations.

## Efficiency

The MCP-only runs (both sonnet and opus) are the cheapest at ~$0.14 and fastest at ~20s, while baseline runs cost 5-8× more ($0.71-$1.14) and take 2-3× longer. The mcp-full runs sit in between at ~$0.21. Given that mcp-only and mcp-full produce answers of comparable or better quality than baseline, the MCP scenarios offer dramatically better cost efficiency.

## Verdict

**Winner: opus/mcp-full**

---

## go-tsdb-compaction [go / hard]

> How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 124.6s | 31872 | 28104 | 2343 | $1.0474 |  |
| **sonnet** | mcp-only | 53.8s | 52649 | 0 | 2716 | $0.3311 |  |
| **sonnet** | mcp-full | 48.6s | 74036 | 42156 | 2501 | $0.4538 |  |
| **opus** | baseline | 160.1s | 39605 | 28230 | 2197 | $1.1887 |  |
| **opus** | mcp-only | 44.6s | 35597 | 0 | 2239 | $0.2340 | 🏆 Winner |
| **opus** | mcp-full | 55.0s | 84305 | 42345 | 2663 | $0.5093 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most thorough and precise. Covers all three Plan strategies (overlapping, leveled, tombstone) in a clear table, explains the `write()` crash-safety pattern, details `DefaultBlockPopulator` series merging with chunk pool reuse, and includes error handling (marking blocks `Failed`/`Deletable`). Specific line references throughout.

2. **sonnet/mcp-only** — Excellent structure with accurate line references. Covers planning strategies well, explains the `BlockPopulator` seam for downstream overrides (Thanos, Mimir), and notes key invariants (`cmtx`, `autoCompactMtx`, compaction delay). Slightly more architectural context than other sonnet runs.

3. **opus/mcp-full** — Very accurate and well-organized. Covers all phases clearly with good code excerpts. Slightly less detail than opus/baseline on error handling paths and the `DefaultBlockPopulator` internals, but still comprehensive.

4. **opus/mcp-only** — Solid coverage with correct line references and good structure. Mentions the appender trigger path with actual code. Slightly less detail on `selectDirs` mechanics than the top answers.

5. **sonnet/mcp-full** — Good accuracy and structure. Covers `selectDirs` well with the range tier explanation. Slightly less detail on error handling and OOO compaction than the opus answers.

6. **sonnet/baseline** — Correct and comprehensive with good ASCII flow diagram. Some line references are absent (uses function names instead). Slightly more verbose without proportionally more insight; the `CompactStaleHead` detail is a nice touch but the planning section is less precise than others.

## Efficiency

The mcp-only runs dominate on cost: opus/mcp-only is the cheapest at $0.23 and fastest at 44.6s, while sonnet/mcp-only is $0.33 at 53.8s. Baseline runs are 3-5x more expensive ($1.05-$1.19) and 2-3x slower. The mcp-full runs sit in between on cost ($0.45-$0.51) without clearly better quality than mcp-only. Opus/mcp-only delivers near-top-tier quality at the lowest cost and fastest time.

## Verdict

**Winner: opus/mcp-only**

---

## py-permissions [python / easy]

> How does the Django Permission model work? Show the Permission class, its fields, the PermissionManager, and the get_by_natural_key method.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 39.1s | 28562 | 28104 | 851 | $0.3644 |  |
| **sonnet** | mcp-only | 18.1s | 19548 | 0 | 975 | $0.1221 |  |
| **sonnet** | mcp-full | 16.3s | 31428 | 28104 | 807 | $0.1914 |  |
| **opus** | baseline | 22.9s | 49503 | 42345 | 1035 | $0.2946 |  |
| **opus** | mcp-only | 18.4s | 18326 | 0 | 844 | $0.1127 | 🏆 Winner |
| **opus** | mcp-full | 25.7s | 67229 | 56460 | 1032 | $0.3902 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most complete: correctly identifies all fields, methods, and meta constraints with accurate line references, and uniquely explains how `PermissionsMixin` and `Group` connect via M2M relationships, giving the fullest picture of the permission system.

2. **opus/mcp-only** — Equally accurate on the core Permission model, adds valuable detail about `ModelBackend` permission checking and caching as `"app_label.codename"` strings, though the `__str__` example is slightly off (includes an extra segment).

3. **opus/mcp-full** — Correct and well-structured, references `_user_has_perm` and `_user_get_permissions` helper functions with line numbers, providing good architectural context; minor issue with paraphrasing the `ForeignKey` signature.

4. **sonnet/mcp-full** — Accurate with proper file path (`testdata/fixtures/python/django-models.py`), good summary section, but stays surface-level compared to opus answers — no mention of how permissions connect to users/groups.

5. **sonnet/mcp-only** — Solid coverage with a nice flow diagram showing the lookup chain; accurate details throughout but lacks the broader system context (backends, M2M relationships).

6. **sonnet/baseline** — Correct and concise but the most minimal; uses generic file reference (`django-models.py:27-36`) and doesn't explore how permissions integrate with the wider auth system.

## Efficiency

The mcp-only runs are dramatically cheaper: sonnet/mcp-only ($0.12) and opus/mcp-only ($0.11) cost 2-3× less than their baseline/mcp-full counterparts while delivering comparable or better quality. Opus/mcp-full is the most expensive at $0.39 with no proportional quality gain over opus/mcp-only. Sonnet/mcp-full offers a good middle ground at $0.19 but doesn't match opus quality.

## Verdict

**Winner: opus/mcp-only**

---

## py-flask-config [python / medium]

> How does Flask configuration loading work? Explain the Config class, how it loads from files, environment variables, and Python objects. Show the key methods and class hierarchy.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 52.9s | 29200 | 28104 | 1104 | $0.3498 |  |
| **sonnet** | mcp-only | 28.3s | 30465 | 0 | 1506 | $0.1900 |  |
| **sonnet** | mcp-full | 25.7s | 35453 | 28104 | 1205 | $0.2214 |  |
| **opus** | baseline | 27.5s | 47351 | 42345 | 1230 | $0.2887 |  |
| **opus** | mcp-only | 31.5s | 41261 | 0 | 1444 | $0.2424 | 🏆 Winner |
| **opus** | mcp-full | 25.5s | 48689 | 42345 | 1081 | $0.2916 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most complete and well-structured: covers all six loading methods with accurate code snippets, includes `ConfigAttribute` descriptor, `get_namespace()`, the loading chain summary, and the JSON parsing fallback detail for `from_prefixed_env`. Line references are present and accurate.

2. **sonnet/mcp-only** — Very thorough with a useful method dependency map, covers `ConfigAttribute`, all six methods, and the nested dict `__` separator. Slightly more verbose than needed but highly accurate with good line references.

3. **opus/mcp-full** — Covers all methods accurately with code snippets and a clean loading chain diagram. Slightly less detailed than mcp-only (e.g., doesn't mention JSON parsing fallback behavior) but still very complete with line references.

4. **opus/baseline** — Strong coverage including `get_namespace()` and the loading chain, with accurate code. Comparable to mcp-full but slightly less polished in structure.

5. **sonnet/mcp-full** — Accurate and well-organized, covers `ConfigAttribute` and all methods. Slightly less detailed on `from_prefixed_env` nuances but solid overall.

6. **sonnet/baseline** — Accurate and covers the key methods well, but misses `ConfigAttribute` descriptor entirely and doesn't mention `get_namespace()`. Still a good answer but least complete.

## Efficiency

Sonnet/mcp-only ($0.19, 28.3s) and sonnet/mcp-full ($0.22, 25.7s) are the cheapest and fastest runs. Opus/mcp-only ($0.24, 31.5s) delivers the highest quality at moderate cost. The baseline runs for both models are comparable in cost to their mcp variants but sonnet/baseline is notably slower at 52.9s. Opus/mcp-only offers the best quality-to-cost ratio given its superior answer at only $0.05 more than the cheapest run.

## Verdict

**Winner: opus/mcp-only**

---

## py-django-queryset [python / hard]

> How does the Django QuerySet evaluation and filtering pipeline work? Explain QuerySet chaining, lazy evaluation, the Query class, how lookups and filters are compiled into SQL, and how the Manager ties it all together. Show key classes and method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 162.3s | 35878 | 28104 | 2614 | $1.4710 |  |
| **sonnet** | mcp-only | 65.1s | 77788 | 0 | 3950 | $0.4877 |  |
| **sonnet** | mcp-full | 63.9s | 111514 | 70260 | 3568 | $0.6819 |  |
| **opus** | baseline | 68.1s | 173896 | 112920 | 3462 | $1.0125 |  |
| **opus** | mcp-only | 73.2s | 85190 | 0 | 4360 | $0.5350 | 🏆 Winner |
| **opus** | mcp-full | 67.2s | 137924 | 70575 | 3645 | $0.8160 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Most thorough and well-structured; covers Manager installation via `contribute_to_class` and `ManagerDescriptor.__get__` that others gloss over, includes the deferred filter property, combinator queries, and a comprehensive summary table of chaining methods with their Query mutations. All file:line references are precise.

2. **opus/mcp-only** — Nearly identical coverage to opus/mcp-full with excellent structure and a detailed summary table of classes/signatures at the end; slightly less detail on `contribute_to_class` mechanics but adds the set operations section and has the most complete reference table.

3. **opus/baseline** — Strong coverage with good explanations of `ManagerDescriptor`, the deferred filter property, and iterable class variants; slightly less organized than the MCP runs but still accurate and complete with correct line references.

4. **sonnet/mcp-full** — Solid and accurate with good coverage of the deferred filter optimization and combinator queries; slightly less polished organization and missing the iterable class variant table that other answers include.

5. **sonnet/baseline** — Good coverage with a useful evaluation triggers table and clear end-to-end example; slightly more surface-level on the Query class internals since it presents them as a table rather than explaining the compilation pipeline.

6. **sonnet/mcp-only** — Accurate but the thinnest of the six; covers all major topics but with less depth on Manager internals and fewer concrete line references for the iterable classes.

## Efficiency

The MCP-only runs offer dramatically better cost efficiency: sonnet/mcp-only at $0.49 and opus/mcp-only at $0.53 are 2-3x cheaper than their baseline counterparts while producing comparable or better answers. The mcp-full runs sit in between at $0.68-$0.82. Runtime is comparable across all runs (63-73s) except sonnet/baseline which is an outlier at 162s.

## Verdict

**Winner: opus/mcp-only**

---

## ts-disposable [typescript / easy]

> What is the IDisposable interface and how does the Disposable base class work? Show the interface, the base class, and how DisposableStore manages multiple disposables.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 40.3s | 29195 | 28104 | 884 | $0.3063 |  |
| **sonnet** | mcp-only | 35.5s | 64396 | 0 | 1918 | $0.3699 |  |
| **sonnet** | mcp-full | 30.3s | 78857 | 56208 | 1355 | $0.4563 |  |
| **opus** | baseline | 23.3s | 53090 | 42345 | 933 | $0.3099 | 🏆 Winner |
| **opus** | mcp-only | 49.0s | 128986 | 0 | 2395 | $0.7048 |  |
| **opus** | mcp-full | 28.2s | 56290 | 42345 | 1242 | $0.3337 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most comprehensive: uniquely includes the `isDisposable` type guard, the standalone `dispose()` utility with `AggregateError` handling, and a concrete subclass usage example; all line references are accurate.
2. **opus/mcp-full** — Clean and thorough with accurate line references, full method signatures for `DisposableStore`, and a practical usage example showing the composition pattern.
3. **sonnet/mcp-only** — Excellent structure with code for all three components, a clear relationship diagram, and correct mention of error collection in `dispose()`; slightly verbose.
4. **opus/baseline** — Concise and accurate with a well-organized table for `DisposableStore` methods and mention of `AggregateError`; covers all key safety guards.
5. **sonnet/mcp-full** — Good relationship diagram and table, accurate line references, mentions `deleteAndLeak` and leak tracking, but slightly less detailed on error handling.
6. **sonnet/baseline** — Correct and clear but omits the `DisposableStore` code listing and the standalone `dispose()` helper; least detailed of the group.

## Efficiency

opus/baseline ($0.31, 23.3s) and sonnet/baseline ($0.31, 40.3s) tie on cost, but opus/baseline is nearly twice as fast. opus/mcp-only delivers the richest answer but at 2.3× the cost ($0.70) and the slowest runtime (49s), making it the worst efficiency tradeoff. opus/mcp-full ($0.33, 28.2s) offers near-opus/mcp-only quality for less than half the cost.

## Verdict

**Winner: opus/baseline**

---

## ts-event-emitter [typescript / medium]

> How does the event emitter system work? Explain the Event interface, the Emitter class, event composition (map, filter, debounce), and how events integrate with disposables. Show key types and patterns.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 87.6s | 31593 | 28104 | 1940 | $0.7216 |  |
| **sonnet** | mcp-only | 66.8s | 114798 | 0 | 2938 | $0.6474 |  |
| **sonnet** | mcp-full | 46.2s | 62723 | 42156 | 2129 | $0.3879 |  |
| **opus** | baseline | 55.8s | 127370 | 84690 | 2350 | $0.7379 |  |
| **opus** | mcp-only | 51.4s | 28087 | 0 | 2330 | $0.1987 | 🏆 Winner |
| **opus** | mcp-full | 53.8s | 130557 | 84690 | 2029 | $0.7459 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/mcp-only** — Most thorough and best-organized answer. Covers `Event<T>`, `Emitter<T>` internals, all major composition operators with line references, `ChainableSynthesis`, emitter variants, and a clear 5-point disposable integration section with code examples showing `snapshot`, `fromNodeEventEmitter`, and the lazy subscription pattern. The ASCII diagram at the end is a nice touch.

2. **sonnet/baseline** — Impressively comprehensive without tool use: covers all operators in a table, all emitter variants, key design patterns (reentrancy, leak detection, error isolation), and disposable integration with three subscription patterns. Slightly less precise on line references since it didn't read the file, but content is accurate and well-structured.

3. **opus/mcp-full** — Accurate and well-structured with good line references. Covers `snapshot` pattern with code, composition operators table, delivery queue, leak detection. Slightly less complete on emitter variants (missing `MicrotaskEmitter`, `EventMultiplexer`) and disposable integration is briefer than the top answers.

4. **opus/baseline** — Solid coverage with correct internals (sparse arrays, compaction, `UniqueContainer`). Good `EmitterOptions` table. Misses some emitter variants and the `chain` API explanation is absent. Line references present but less precise without tool verification.

5. **opus/mcp-only** — Very thorough on all sections with accurate line references. Covers `ChainableSynthesis`, all emitter subclasses, and 5-point disposable integration. Content quality is on par with sonnet/mcp-only but slightly more verbose without adding proportional insight.

6. **sonnet/mcp-full** — Correct and well-organized but slightly less detailed than peers. The `ChainableSynthesis` section mentions `HaltChainable` which is good, but disposable integration and emitter variants sections are more compressed.

## Efficiency

The opus/mcp-only run stands out at $0.20 and 51.4s — by far the cheapest run while still delivering a high-quality, comprehensive answer. Sonnet/mcp-full is also efficient at $0.39 and 46.2s (fastest) but with slightly less complete content. The baseline runs and opus/mcp-full are all in the $0.72–$0.75 range, making them 3–4× more expensive than opus/mcp-only for comparable or marginally better quality.

## Verdict

**Winner: opus/mcp-only**

---

## ts-async-lifecycle [typescript / hard]

> How do async operations, cancellation, and resource lifecycle management work together? Explain CancelablePromise, CancellationToken, the async utilities (throttle, debounce, retry), how they integrate with the disposable lifecycle system, and how event-driven patterns compose with async flows. Show key interfaces and class relationships.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 144.5s | 37270 | 28104 | 1970 | $1.3152 |  |
| **sonnet** | mcp-only | 81.1s | 93597 | 0 | 3962 | $0.5670 |  |
| **sonnet** | mcp-full | 65.0s | 118545 | 70260 | 3192 | $0.7077 |  |
| **opus** | baseline | 116.1s | 32074 | 28230 | 2939 | $0.7285 |  |
| **opus** | mcp-only | 75.3s | 80259 | 0 | 3739 | $0.4948 | 🏆 Winner |
| **opus** | mcp-full | 154.4s | 35457 | 28230 | 2932 | $1.1831 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/mcp-only** — Most comprehensive answer with precise line references throughout (e.g., `cancellation.ts:144`, `async.ts:573`), uniquely covers `cancelOnDispose`, `disposableTimeout`, `ThrottledWorker` with `MutableDisposable` internals, and the `AsyncEmitter`/`IWaitUntil` pattern in full detail.
2. **opus/mcp-full** — Very thorough with consistent line references, excellent event combinator table, clear `createCancelablePromise` step-by-step breakdown, and a strong compositional diagram showing subsystem relationships.
3. **opus/mcp-only** — Strong line references, uniquely covers `thenRegisterOrDispose` and `thenIfNotDisposed` lifecycle-async bridges, and the `AsyncEmitter.fireAsync` cancellation-aware delivery loop; composition section is clear and actionable.
4. **sonnet/mcp-full** — Good line references and covers `TaskSequentializer`/`LimitedQueue` that others miss, but the composition diagram is slightly less detailed than the top answers.
5. **opus/baseline** — Comprehensive without line references, good coverage of `CancellationTokenPool` and `RefCountedDisposable`, solid event combinators table, but lacks the specificity that file:line references provide.
6. **sonnet/baseline** — Mentions unique details like `ResourceQueue`, `GCBasedDisposableTracker` leak detection, and `EmitterOptions` lazy hooks, but organization is slightly looser and lacks line references.

## Efficiency

opus/mcp-only is the cheapest ($0.49) and second fastest (75.3s), while sonnet/mcp-only delivers the highest quality at a modest premium ($0.57, 81.1s). The baseline runs are the most expensive (sonnet/baseline at $1.32, opus/baseline at $0.73) with slower runtimes and no line references. opus/mcp-full is the slowest and second most expensive ($1.18, 154.4s) despite strong quality, making it a poor efficiency tradeoff.

**Winner: opus/mcp-only**

---

## Overall: Algorithm Comparison

| Question | Language | Difficulty | 🏆 Winner | Runner-up |
|----------|----------|------------|-----------|-----------|
| go-label-matcher | go | easy | opus/mcp-only | sonnet/mcp-only |
| go-histogram | go | medium | opus/mcp-full | opus/mcp-only |
| go-tsdb-compaction | go | hard | opus/mcp-only | sonnet/mcp-only |
| py-permissions | python | easy | opus/mcp-only | sonnet/mcp-only |
| py-flask-config | python | medium | opus/mcp-only | sonnet/mcp-only |
| py-django-queryset | python | hard | opus/mcp-only | sonnet/mcp-only |
| ts-disposable | typescript | easy | opus/baseline | sonnet/baseline |
| ts-event-emitter | typescript | medium | opus/mcp-only | sonnet/mcp-full |
| ts-async-lifecycle | typescript | hard | opus/mcp-only | sonnet/mcp-only |

**Scenario Win Counts** (across all 9 questions):

| Scenario | Wins |
|----------|------|
| baseline | 1 |
| mcp-only | 7 |
| mcp-full | 1 |

**Overall winner: mcp-only** — won 7 of 9 questions.

_Full answers and detailed analysis: `detail-report.md`_
