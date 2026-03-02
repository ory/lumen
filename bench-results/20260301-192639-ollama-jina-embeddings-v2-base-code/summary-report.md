# Benchmark Summary

Generated: 2026-03-01 18:52 UTC  |  Results: `20260301-192639-ollama-jina-embeddings-v2-base-code`

| Scenario | Description |
|----------|-------------|
| **baseline** | All default Claude tools, no MCP |
| **mcp-only** | `semantic_search` MCP tool only |
| **mcp-full** | All default tools + MCP |

## Overall: Aggregated by Scenario

Totals across all 9 questions × 2 models.

| Model | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |
|-------|----------|------------|-----------------|------------------|------------------|
| **sonnet** | baseline | 798.2s | 278324 | 13672 | $10.2157 |
| **sonnet** | mcp-only | 333.6s | 393920 | 17875 | $2.4165 |
| **sonnet** | mcp-full | 311.3s | 564240 | 16374 | $3.4273 |
| **opus** | baseline | 622.4s | 960358 | 17126 | $10.7016 |
| **opus** | mcp-only | 367.6s | 441572 | 18528 | $2.6711 |
| **opus** | mcp-full | 417.6s | 492449 | 14936 | $3.8292 |

---

## go-label-matcher [go / easy]

> What label matcher types are available and how is a Matcher created? Show the type definitions and constructor.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 20.6s | 28223 | 28104 | 602 | $0.2399 |  |
| **sonnet** | mcp-only | 11.8s | 19180 | 0 | 840 | $0.1169 |  |
| **sonnet** | mcp-full | 12.5s | 30175 | 28104 | 739 | $0.1834 |  |
| **opus** | baseline | 15.9s | 45388 | 42345 | 736 | $0.2665 |  |
| **opus** | mcp-only | 9.7s | 18058 | 0 | 486 | $0.1024 |  |
| **opus** | mcp-full | 10.5s | 29097 | 28230 | 531 | $0.1729 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

The files are identical. Now I can evaluate accurately.

## Content Quality

1. **sonnet/mcp-full** — Fully correct with exact code from `labels_matcher.go`, accurate line references, includes both constructors with complete source, and a clean summary. Notes the duplicate file correctly.
2. **opus/mcp-full** — Correct code, accurate line references, concise explanation. Mentions `MustNewMatcher` but doesn't show its source (minor omission). Slightly more compact than sonnet/mcp-full.
3. **opus/baseline** — Correct and complete, accurate line references, properly notes both files are duplicates, shows both constructors. Very solid.
4. **sonnet/mcp-only** — Correct and thorough with a nice operator table, but slightly verbose; correctly identifies both files and shows full constructor source.
5. **opus/mcp-only** — Correct, concise, accurate line references, covers all key elements but shows `NewMatcher` as signature-only without the body (the question asked to "show" the constructor).
6. **sonnet/baseline** — Correct but only references `matcher.go`, missing `labels_matcher.go`; line numbers are off by one for the const block (says 22-30 instead of 25-30 for the constants, though 22 is the `type` line so partially defensible). Doesn't show full constructor body.

## Efficiency

The mcp-only runs are dramatically cheaper ($0.10-0.12) and faster (9.7-11.8s) than baseline ($0.24-0.27, 15.9-20.6s) and mcp-full ($0.17-0.18, 10.5-12.5s). Opus/mcp-only delivers the best cost efficiency at $0.10 and 9.7s, though its answer is slightly less complete. Sonnet/mcp-only at $0.12 gives a more thorough answer for minimal extra cost. Among the full-quality answers, opus/mcp-full at $0.17 and 10.5s offers the best quality-to-cost ratio.

## Verdict

**Winner: opus/mcp-full**

---

## go-histogram [go / medium]

> How does histogram bucket counting work? Show me the relevant function signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 127.4s | 31154 | 28104 | 935 | $2.8066 |  |
| **sonnet** | mcp-only | 16.2s | 22458 | 0 | 757 | $0.1312 |  |
| **sonnet** | mcp-full | 16.3s | 34606 | 28104 | 753 | $0.2059 |  |
| **opus** | baseline | 60.0s | 261106 | 155265 | 2680 | $1.4502 |  |
| **opus** | mcp-only | 20.1s | 22426 | 0 | 918 | $0.1351 | 🏆 Winner |
| **opus** | mcp-full | 18.9s | 32132 | 28230 | 710 | $0.1925 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most comprehensive: covers classic buckets, native/sparse buckets with schema details, `makeBuckets` span/delta encoding, bucket limiting, and iteration with delta accumulation. Accurate file:line references throughout. Includes `makeBuckets` and `limitBuckets` signatures that others miss or only mention in passing.

2. **sonnet/baseline** — Strong coverage of both classic and sparse paths, includes bucket construction helpers (`LinearBuckets`, `ExponentialBuckets`), `addToBucket`, validation via `Validate()`, and iterator delta decoding. Good breadth but slightly less precise on line references.

3. **opus/mcp-only** — Thorough and well-structured with accurate line references, covers classic, native, cumulative read path, iteration, and bucket limiting. Slightly less detail on `makeBuckets` encoding than opus/baseline.

4. **opus/mcp-full** — Correct and concise, covers both bucket mechanisms, key computation, and cumulative counting. Includes type/iterator signatures but less detail on limiting and encoding than the other opus answers.

5. **sonnet/mcp-full** — Accurate with good line references, covers the core four functions cleanly, and adds the double-buffer explanation. Narrower scope than the opus answers — omits `addToBucket`, `makeBuckets`, and bucket limiting.

6. **sonnet/mcp-only** — Correct and well-referenced for the four functions covered, but narrowest scope — misses `addToBucket`, `makeBuckets`, iterator delta decoding, and bucket limiting.

## Efficiency

The MCP-only runs are dramatically cheaper ($0.13) and faster (16–20s) than baseline runs, with opus/baseline being the most expensive at $1.45/60s and sonnet/baseline extreme at $2.81/127s. The MCP scenarios deliver 80–90% of the content quality at ~10% of the cost, making them far superior on efficiency. Among MCP runs, opus/mcp-only edges out on quality for essentially the same cost as sonnet/mcp-only.

## Verdict

**Winner: opus/mcp-only**

---

## go-tsdb-compaction [go / hard]

> How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 123.9s | 31215 | 28104 | 1927 | $1.4836 |  |
| **sonnet** | mcp-only | 41.4s | 49590 | 0 | 2154 | $0.3018 |  |
| **sonnet** | mcp-full | 48.3s | 93954 | 56208 | 2341 | $0.5564 |  |
| **opus** | baseline | 187.8s | 33041 | 28230 | 1726 | $4.3928 |  |
| **opus** | mcp-only | 42.4s | 39284 | 0 | 1959 | $0.2454 |  |
| **opus** | mcp-full | 42.5s | 50538 | 28230 | 1985 | $0.3164 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Most comprehensive: uniquely identifies all three trigger paths (background loop, appender commit, Compact itself), includes the `dbAppender.Commit()` code snippet, covers stale series compaction, OOO WBL truncation, mmap GC, and `EnableCompactions()`/`DisableCompactions()`, all with precise file:line references.
2. **opus/baseline** — Very thorough with the `head.compactable()` threshold formula (`MaxTime - MinTime > chunkRange * 1.5`), `EnableDelayedCompaction` detail, and a useful method summary table, though at extreme cost.
3. **opus/mcp-only** — Clean and comprehensive with good `reloadBlocks()` explanation and concurrency control summary; slightly less detailed than mcp-full on trigger paths and edge cases.
4. **sonnet/baseline** — Detailed with both trigger paths, the `dbAppender.Commit()` snippet, and a solid ASCII flow diagram, but misses some nuances like stale series compaction.
5. **sonnet/mcp-full** — Adds valuable detail on `DefaultBlockPopulator.PopulateBlock` merge engine and locking, but the four-phase breakdown is slightly less crisp than opus variants.
6. **sonnet/mcp-only** — Solid foundational coverage with good design-choice notes at the end, but least detailed on concurrency controls and edge cases.

## Efficiency

The MCP scenarios are 3-14x cheaper and 2.5-4.5x faster than baselines, with opus/mcp-only the cheapest ($0.25, 42.4s) and opus/baseline the most expensive ($4.39, 187.8s). opus/mcp-full delivers the highest-quality answer at only $0.32 and 42.5s — a 14x cost reduction over opus/baseline with arguably better output, making it the clear quality-to-cost winner.

## Verdict

**Winner: opus/mcp-full**

---

## py-permissions [python / easy]

> How does the Django Permission model work? Show the Permission class, its fields, the PermissionManager, and the get_by_natural_key method.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 54.6s | 28715 | 28104 | 871 | $0.9874 |  |
| **sonnet** | mcp-only | 15.0s | 17634 | 0 | 888 | $0.1104 |  |
| **sonnet** | mcp-full | 15.9s | 31077 | 28104 | 860 | $0.1909 |  |
| **opus** | baseline | 24.5s | 49449 | 42345 | 1136 | $0.2968 |  |
| **opus** | mcp-only | 16.4s | 17642 | 0 | 833 | $0.1090 | 🏆 Winner |
| **opus** | mcp-full | 18.9s | 44362 | 42345 | 805 | $0.2631 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most complete: covers all fields, PermissionManager, natural_key, and uniquely explains how Permission connects to users via PermissionsMixin (line 317), with accurate file/line references throughout.
2. **opus/mcp-full** — Equally correct and well-structured, also mentions PermissionsMixin connection, but slightly more terse on the manager explanation.
3. **sonnet/mcp-only** — Correct and thorough, includes a nice summary of design points (auto-created verbs, global per model type), good line references, but misses the PermissionsMixin connection.
4. **sonnet/mcp-full** — Correct with a clean "how it fits together" diagram showing natural_key round-tripping, but no deeper context about user/group integration.
5. **opus/mcp-only** — Accurate and concise, mentions object-level permission limitation which is a nice touch, but slightly less structured than the sonnet variants.
6. **sonnet/baseline** — Correct but the most bare-bones of the six; covers the essentials without adding broader context.

## Efficiency

The mcp-only runs are dramatically cheaper ($0.11) and faster (15-16s) than baseline and mcp-full runs ($0.19-$0.99, 18-55s), while delivering comparable or equal quality. Sonnet/baseline is the worst value at $0.99 for the least detailed answer. Opus/baseline delivers the best content but at 2.7× the cost of opus/mcp-only.

## Verdict

**Winner: opus/mcp-only**

---

## py-flask-config [python / medium]

> How does Flask configuration loading work? Explain the Config class, how it loads from files, environment variables, and Python objects. Show the key methods and class hierarchy.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 58.2s | 29782 | 28104 | 1511 | $0.3496 |  |
| **sonnet** | mcp-only | 28.9s | 26731 | 0 | 1416 | $0.1691 |  |
| **sonnet** | mcp-full | 25.8s | 34857 | 28104 | 1327 | $0.2215 |  |
| **opus** | baseline | 42.7s | 29547 | 28230 | 935 | $0.2839 |  |
| **opus** | mcp-only | 27.5s | 22908 | 0 | 1242 | $0.1456 | 🏆 Winner |
| **opus** | mcp-full | 25.6s | 35105 | 28230 | 1250 | $0.2209 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Most logically organized, presenting methods from core primitive outward with a clear delegation chain diagram. Correct throughout, includes line references, code snippets, and the nested env var mechanism. Concise without sacrificing completeness.

2. **opus/mcp-only** — Nearly identical quality to opus/mcp-full with accurate line references, all six methods covered, and good code excerpts. Slightly more verbose in prose but the "funnel" design summary is a nice touch.

3. **sonnet/mcp-full** — Comprehensive and correct with all six methods, line references, and the uppercase invariant clearly stated. Slightly more verbose than the opus answers without adding proportional value.

4. **sonnet/mcp-only** — Correct and complete with a useful method call chain diagram. Good line references and code snippets. The explanation of `from_prefixed_env` nesting is clear. Comparable to sonnet/mcp-full.

5. **sonnet/baseline** — Correct and well-structured with a helpful precedence pattern example, but only covers 6 methods as a table without showing `from_prefixed_env`'s nesting logic in code. Line references are absent (uses line numbers but not file references consistently).

6. **opus/baseline** — Most concise but sacrifices depth — presents methods as a table without code snippets for most, missing the `from_prefixed_env` nesting detail. Still correct and includes line references.

## Efficiency

The mcp-only runs for both models are the cheapest ($0.15–0.17) and fastest (27–29s), while baseline and mcp-full runs cost $0.22–0.35. Opus/mcp-only delivers top-tier quality at the lowest cost ($0.146, 27.5s), making it the clear efficiency winner. The mcp-full runs add cache read tokens without meaningfully improving answer quality over mcp-only.

## Verdict

**Winner: opus/mcp-only**

---

## py-django-queryset [python / hard]

> How does the Django QuerySet evaluation and filtering pipeline work? Explain QuerySet chaining, lazy evaluation, the Query class, how lookups and filters are compiled into SQL, and how the Manager ties it all together. Show key classes and method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 145.7s | 32429 | 28104 | 2437 | $2.1727 |  |
| **sonnet** | mcp-only | 59.0s | 77994 | 0 | 3349 | $0.4737 |  |
| **sonnet** | mcp-full | 48.5s | 57719 | 42156 | 2996 | $0.3846 |  |
| **opus** | baseline | 80.1s | 234779 | 112920 | 3758 | $1.3243 |  |
| **opus** | mcp-only | 79.7s | 80319 | 0 | 4771 | $0.5209 |  |
| **opus** | mcp-full | 70.9s | 62994 | 42345 | 3159 | $0.4675 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most thorough and well-organized; covers all requested topics (Manager, QuerySet, Query class, lookups, chaining, lazy evaluation) with accurate line references, a comprehensive Query method table, and clear iterable class explanation. Excellent structure with the complete pipeline summary.

2. **opus/mcp-full** — Nearly as complete as opus/mcp-only; adds a useful `get()` method walkthrough and a chaining methods table, but the Query class section is slightly less detailed since it relies more on inference from usage patterns rather than direct method enumeration.

3. **sonnet/baseline** — Strong coverage with accurate code excerpts and line references; the three-layer iterator explanation and lookup pipeline (`QuerySet.filter → add_q → build_filter → Lookup.as_sql → WhereNode.as_sql → compiler.execute_sql`) is uniquely detailed, though some sections are slightly less polished.

4. **opus/baseline** — Very comprehensive with good set operations coverage and Query class method table, but verbose at ~4000 output tokens; the additional detail doesn't substantially improve understanding over the more concise answers.

5. **sonnet/mcp-full** — Accurate and well-structured but slightly less complete than the top answers; the Query class section is thinner and the deferred filter explanation, while present, is briefer.

6. **sonnet/mcp-only** — Solid and correct with good deferred filter coverage, but the Query class section is the weakest ("While the full Query class implementation isn't in these fixtures" — actually it's referenced sufficiently) and lacks the iterable class table other answers provide.

## Efficiency

Opus/baseline is by far the most expensive ($1.32) and token-heavy (234K input), while opus/mcp-only delivers comparable or better quality at $0.52 (60% cheaper) in similar time. Sonnet/mcp-full is the cheapest at $0.38 and fastest at 48.5s but sacrifices some depth. The MCP scenarios consistently outperform baselines on cost efficiency, with opus/mcp-full ($0.47, 71s) offering strong quality at low cost.

## Verdict

**Winner: opus/mcp-full**

---

## ts-disposable [typescript / easy]

> What is the IDisposable interface and how does the Disposable base class work? Show the interface, the base class, and how DisposableStore manages multiple disposables.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 38.9s | 28860 | 28104 | 1008 | $0.3238 |  |
| **sonnet** | mcp-only | 27.9s | 35557 | 0 | 1550 | $0.2165 |  |
| **sonnet** | mcp-full | 24.3s | 74146 | 56208 | 1313 | $0.4317 |  |
| **opus** | baseline | 27.4s | 53094 | 42345 | 1184 | $0.3162 | 🏆 Winner |
| **opus** | mcp-only | 27.9s | 38621 | 0 | 1371 | $0.2274 |  |
| **opus** | mcp-full | 30.4s | 70643 | 56460 | 1434 | $0.4173 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most complete and accurate: includes the self-registration guard in `_register`, mentions `AggregateError` handling in the standalone `dispose()` function, notes `Set` dedup behavior, and correctly identifies the file as `testdata/fixtures/ts/lifecycle.ts`. Precise line references.

2. **sonnet/mcp-full** — Excellent structure with the `dispose()` → `clear()` cascade diagram, correctly notes the "warn not throw" design rationale for add-after-dispose, and includes a practical subclass example. Accurate line references.

3. **opus/mcp-full** — Very similar quality to sonnet/mcp-full, correctly covers all key methods and the AggregateError detail, good practical example, but slightly less detailed on the design rationale for warn-vs-throw.

4. **sonnet/baseline** — Accurate and well-structured with correct line references, covers all key methods including `deleteAndLeak`, but presents method signatures in summary form rather than showing actual code for DisposableStore.

5. **opus/mcp-only** — Good coverage including AggregateError and FinalizationRegistry leak tracking details, but reconstructs code from search results rather than showing exact source, and some line references are approximate.

6. **sonnet/mcp-only** — Thorough with a nice table and composition diagram, mentions FinalizationRegistry leak tracking, but labels the class as non-abstract (`export class Disposable` instead of `export abstract class Disposable`) which is incorrect.

## Efficiency

The mcp-only runs for both models offer the lowest costs ($0.22-0.23) with competitive quality and fast runtimes (~28s). The baseline runs vary widely in cost ($0.31-0.32) but opus/baseline delivers top quality at moderate cost. The mcp-full runs are the most expensive ($0.42-0.43) without proportional quality gains over baseline or mcp-only.

## Verdict

**Winner: opus/baseline**

---

## ts-event-emitter [typescript / medium]

> How does the event emitter system work? Explain the Event interface, the Emitter class, event composition (map, filter, debounce), and how events integrate with disposables. Show key types and patterns.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 104.3s | 31613 | 28104 | 2010 | $0.8593 |  |
| **sonnet** | mcp-only | 64.5s | 87966 | 0 | 3072 | $0.5166 |  |
| **sonnet** | mcp-full | 51.8s | 94670 | 56208 | 2816 | $0.5719 |  |
| **opus** | baseline | 62.9s | 221581 | 98805 | 2497 | $1.2197 |  |
| **opus** | mcp-only | 53.8s | 55388 | 0 | 2576 | $0.3413 | 🏆 Winner |
| **opus** | mcp-full | 54.2s | 130604 | 84690 | 2259 | $0.7518 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/mcp-only** — Most comprehensive answer: uniquely covers the chainable API (`ChainableSynthesis`), `Relay<T>`, `EventBufferer`, `EmitterObserver`, and provides a full subscription lifecycle walkthrough with concrete composed-disposal examples; all with accurate line references.
2. **opus/baseline** — Strong coverage with the idiomatic private-emitter/public-event pattern, `AsyncEmitter` with `waitUntil`, leak detection (`LeakageMonitor` + `ListenerRefusalError`), and clear `DisposableStore` integration; minor gap on the chainable API.
3. **opus/mcp-only** — Thorough treatment of internals (sparse array compaction, `UniqueContainer` optimization), covers `Relay`, `EventMultiplexer`, and leak detection well; slightly less polished presentation than opus/baseline.
4. **sonnet/baseline** — Covers all specialized emitters and the `snapshot` pattern well with accurate line references; slightly less depth on leak detection and the chainable API.
5. **opus/mcp-full** — Best explanation of the `chain` API with `HaltChainable` sentinel, but truncates specialized emitters to just two (AsyncEmitter, PauseableEmitter), making it less complete overall.
6. **sonnet/mcp-full** — Solid and accurate but the least detailed of the six; omits chainable API, `Relay`, and `AsyncEmitter`.

## Efficiency

opus/mcp-only is the clear efficiency leader at $0.34 and 53.8s — roughly 40% cheaper than the next-cheapest sonnet run and 72% cheaper than opus/baseline, while delivering quality comparable to the top answers. sonnet/baseline is the worst performer at $0.86 and 104.3s for a mid-tier answer. The MCP scenarios consistently outperform their baseline counterparts in both cost and runtime.

## Verdict

**Winner: opus/mcp-only**

---

## ts-async-lifecycle [typescript / hard]

> How do async operations, cancellation, and resource lifecycle management work together? Explain CancelablePromise, CancellationToken, the async utilities (throttle, debounce, retry), how they integrate with the disposable lifecycle system, and how event-driven patterns compose with async flows. Show key interfaces and class relationships.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 124.1s | 36333 | 28104 | 2371 | $0.9928 |  |
| **sonnet** | mcp-only | 68.6s | 56810 | 0 | 3849 | $0.3803 |  |
| **sonnet** | mcp-full | 67.4s | 113036 | 70260 | 3229 | $0.6810 | 🏆 Winner |
| **opus** | baseline | 120.8s | 32373 | 28230 | 2474 | $1.1511 |  |
| **opus** | mcp-only | 89.5s | 146926 | 0 | 4372 | $0.8439 |  |
| **opus** | mcp-full | 145.3s | 36974 | 28230 | 2803 | $1.0267 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/mcp-full** — Most thorough and precise: includes specific file:line references (e.g., `lifecycle.ts:312`, `async.ts:34`, `cancellation.ts:97`), correctly explains lazy token creation optimization, covers `thenRegisterOrDispose` for async-dispose races, and provides a clear integration table showing how all systems connect.

2. **opus/mcp-full** — Excellent structure with accurate details on single-listener fast path optimization and `AsyncIterableObject` bridge that others miss, good file references, but slightly less precise on some line numbers and the composition diagram is more schematic than explanatory.

3. **sonnet/baseline** — Impressively comprehensive without tool access: covers `CancellationTokenPool`, `AsyncEmitter.fireAsync` internals, `MicrotaskDelay`, and provides a clear composition hierarchy; lacks file:line references but compensates with accurate code snippets.

4. **opus/baseline** — Clean and accurate with good coverage of `DeferredPromise`, `raceCancellablePromises`, and the lazy token optimization; slightly less detailed on event system composition and missing some integration patterns like `thenRegisterOrDispose`.

5. **sonnet/mcp-only** — Solid coverage with file:line references and correct technical details including `thenRegisterOrDispose` and `AsyncEmitter` internals; the integration diagram is effective but the overall answer is slightly more verbose without proportional depth gain over the baseline.

6. **opus/mcp-only** — Most detailed and longest answer with good accuracy on lazy token creation and `ThrottledWorker`, but somewhat sprawling; the integration section, while correct, doesn't synthesize as crisply as the mcp-full variants.

## Efficiency

Sonnet/mcp-only delivers strong quality at the lowest cost ($0.38) and fastest time (68.6s), making it the clear efficiency leader. Sonnet/mcp-full matches it in speed but costs nearly double ($0.68) for a marginal quality improvement. The opus runs are 1.5-3x more expensive with the baseline and mcp-full variants exceeding $1.00; opus/mcp-only is mid-range in cost ($0.84) but took 89.5s with the highest input token count (147K).

## Verdict

**Winner: sonnet/mcp-full**

---

## Overall: Algorithm Comparison

| Question | Language | Difficulty | 🏆 Winner | Runner-up |
|----------|----------|------------|-----------|-----------|
| go-label-matcher | go | easy | opus/mcp-full | opus/mcp-only |
| go-histogram | go | medium | opus/mcp-only | sonnet/mcp-only |
| go-tsdb-compaction | go | hard | opus/mcp-full | opus/mcp-only |
| py-permissions | python | easy | opus/mcp-only | sonnet/mcp-only |
| py-flask-config | python | medium | opus/mcp-only | sonnet/mcp-only |
| py-django-queryset | python | hard | opus/mcp-full | sonnet/mcp-full |
| ts-disposable | typescript | easy | opus/baseline | sonnet/mcp-only |
| ts-event-emitter | typescript | medium | opus/mcp-only | sonnet/mcp-only |
| ts-async-lifecycle | typescript | hard | sonnet/mcp-full | sonnet/mcp-only |

**Scenario Win Counts** (across all 9 questions):

| Scenario | Wins |
|----------|------|
| baseline | 1 |
| mcp-only | 4 |
| mcp-full | 4 |

**Overall winner: mcp-only** — won 4 of 9 questions.

_Full answers and detailed analysis: `detail-report.md`_
