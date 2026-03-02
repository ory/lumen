# Benchmark Summary

Generated: 2026-03-01 20:25 UTC  |  Results: `20260301-205706-lmstudio-nomic-embed-code-GGUF`

| Scenario | Description |
|----------|-------------|
| **baseline** | All default Claude tools, no MCP |
| **mcp-only** | `semantic_search` MCP tool only |
| **mcp-full** | All default tools + MCP |

## Overall: Aggregated by Scenario

Totals across all 9 questions × 2 models.

| Model | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |
|-------|----------|------------|-----------------|------------------|------------------|
| **sonnet** | baseline | 848.3s | 284114 | 13445 | $9.0721 |
| **sonnet** | mcp-only | 530.6s | 627144 | 22145 | $3.6894 |
| **sonnet** | mcp-full | 411.0s | 687028 | 18255 | $4.1374 |
| **opus** | baseline | 623.4s | 579587 | 15376 | $7.0538 |
| **opus** | mcp-only | 399.2s | 649347 | 19962 | $3.7458 |
| **opus** | mcp-full | 482.8s | 501994 | 13799 | $4.8580 |

---

## go-label-matcher [go / easy]

> What label matcher types are available and how is a Matcher created? Show the type definitions and constructor.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 20.5s | 28071 | 28104 | 645 | $0.2373 |  |
| **sonnet** | mcp-only | 15.3s | 29526 | 0 | 835 | $0.1685 |  |
| **sonnet** | mcp-full | 16.1s | 47402 | 42156 | 920 | $0.2811 |  |
| **opus** | baseline | 17.4s | 44012 | 42345 | 762 | $0.2603 |  |
| **opus** | mcp-only | 10.8s | 18066 | 0 | 578 | $0.1048 | 🏆 Winner |
| **opus** | mcp-full | 17.2s | 47354 | 42345 | 855 | $0.2793 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

All six answers are correct and show the same core content (MatchType enum, Matcher struct, NewMatcher constructor). Differences are minor:

1. **opus/mcp-full** — Most complete: includes all type definitions, constructor code, notes the duplicated file, mentions the unexported `re` field explicitly, and references `MustNewMatcher`. Line references are accurate.
2. **sonnet/mcp-full** — Equally complete with `MustNewMatcher` code shown inline and good line references; the PromQL mention adds useful domain context.
3. **opus/baseline** — Concise yet thorough; uniquely mentions the `Matches()` method dispatch, adding extra value beyond what was asked.
4. **sonnet/mcp-only** — Complete with `MustNewMatcher` and the `matchTypeToStr` map (bonus detail), though the "two files that appear to be copies" note is slightly distracting.
5. **opus/mcp-only** — Accurate and well-structured but doesn't show the full constructor code inline, slightly less scannable.
6. **sonnet/baseline** — Correct and clean but omits `MustNewMatcher` entirely, making it the least complete.

## Efficiency

**opus/mcp-only** is the clear efficiency winner: 10.8s, 18k input tokens, $0.10 — roughly half the cost of most other runs and fastest overall. **sonnet/mcp-only** is second-best at $0.17 and 15.3s. The mcp-full and baseline runs for both models cluster around $0.24–$0.28, offering no meaningful quality improvement over the cheaper mcp-only runs.

## Verdict

**Winner: opus/mcp-only**

---

## go-histogram [go / medium]

> How does histogram bucket counting work? Show me the relevant function signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 112.8s | 31587 | 28104 | 906 | $1.6816 |  |
| **sonnet** | mcp-only | 11.2s | 17320 | 0 | 585 | $0.1012 |  |
| **sonnet** | mcp-full | 12.4s | 29467 | 28104 | 607 | $0.1766 |  |
| **opus** | baseline | 49.9s | 165099 | 98805 | 1997 | $0.9248 |  |
| **opus** | mcp-only | 12.3s | 17306 | 0 | 566 | $0.1007 |  |
| **opus** | mcp-full | 16.4s | 33203 | 28230 | 746 | $0.1988 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/baseline** — Most comprehensive and correct. Covers the full observation flow (`Observe` → `findBucket` → `observe`), native bucket key computation, bucket limiting strategies (widen zero, double width), and generation helpers, all with accurate line references.

2. **sonnet/baseline** — Strong breadth covering `findBucket`, `observe`, `addToBucket`, validation, iteration (PromQL), and bucket boundary creators. Includes accurate signatures and line references. Minor inaccuracy: `addToBucket` signature shows `*sync.Map` parameters, not `*[]uint64`.

3. **opus/mcp-full** — Good coverage of both bucket systems, `addToBucket` with correct `sync.Map` signature, bucket limiting, validation via `validateCount`, and iteration types. Accurate line references throughout.

4. **sonnet/mcp-full** — Correct two-track explanation with accurate detail on native bucket key computation (schema branching, `math.Frexp`). Fewer function signatures than peers but what's there is accurate.

5. **opus/mcp-only** — Solid explanation of the core mechanism with correct detail on native bucket key computation and validation. Fewer signatures shown; `regularBucketIterator` inclusion is less relevant to the "counting" question.

6. **sonnet/mcp-only** — Accurate core explanation but `addToBucket` signature is wrong (`*[]uint64` instead of `*sync.Map`). Least breadth — no bucket limiting, no boundary generators, no validation.

## Efficiency

The MCP-only runs are dramatically cheaper ($0.10) and faster (11-16s) than baseline runs ($0.92-$1.68, 50-113s), with opus/baseline being the most expensive at nearly 10× the cost of MCP runs. The mcp-full runs add ~$0.07-0.10 over mcp-only for cache-read tokens but provide modestly richer answers. For this question, opus/mcp-full delivers strong quality at $0.20 — roughly 4.5× cheaper than opus/baseline with comparable depth.

## Verdict

**Winner: opus/mcp-full**

---

## go-tsdb-compaction [go / hard]

> How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 137.3s | 33349 | 28104 | 1946 | $1.0575 |  |
| **sonnet** | mcp-only | 178.2s | 118954 | 0 | 3331 | $0.6780 |  |
| **sonnet** | mcp-full | 119.9s | 119360 | 84312 | 3103 | $0.7165 |  |
| **opus** | baseline | 130.8s | 31670 | 28230 | 2063 | $1.3000 |  |
| **opus** | mcp-only | 52.3s | 67332 | 0 | 2408 | $0.3969 |  |
| **opus** | mcp-full | 46.8s | 47393 | 28230 | 2074 | $0.3029 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Most complete and accurate: covers the Compactor interface, LeveledCompactor fields, all three planning strategies, the compact/write pipeline, DB triggering (including the append-driven `dbAppender.Commit` path with actual code), OOO compaction, and enable/disable controls, all with precise file:line references.

2. **opus/mcp-only** — Nearly identical coverage to opus/mcp-full, also includes the `dbAppender.Commit` trigger and initialization details; slightly more verbose but equally accurate with good line references.

3. **opus/baseline** — Covers all key areas correctly with accurate line references and includes the `CompactionMeta` struct (unique detail), but misses the append-driven compaction trigger and has slightly less precise line references than the MCP variants.

4. **sonnet/mcp-full** — Solid coverage with correct details on planning, writing, and the DB loop; includes the atomic rename detail and head compactability check, but slightly less precise on some line references and misses the `dbAppender.Commit` trigger.

5. **sonnet/baseline** — Impressively detailed with the `PopulateBlock` flow and specialized entry point table, but includes some questionable details (e.g., `compactc` signaling from `dbAppender.Commit` alongside the timer — correct but described ambiguously) and the line references are approximate.

6. **sonnet/mcp-only** — Accurate but the DB triggering section slightly mischaracterizes the flow (shows `compactc` only from the timer, omits the append-driven signal), and the summary is less detailed than other answers.

## Efficiency

Opus/mcp-full is the clear efficiency winner: fastest runtime (46.8s), lowest cost ($0.30), and lowest input tokens among complete answers, while producing the highest-quality result. Sonnet/baseline is the most expensive ($1.06) and slowest, with opus/mcp-only close behind opus/mcp-full on quality but slightly more expensive and slower.

## Verdict

**Winner: opus/mcp-full**

---

## py-permissions [python / easy]

> How does the Django Permission model work? Show the Permission class, its fields, the PermissionManager, and the get_by_natural_key method.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 29.7s | 28449 | 28104 | 831 | $0.2715 |  |
| **sonnet** | mcp-only | 17.1s | 18694 | 0 | 980 | $0.1180 |  |
| **sonnet** | mcp-full | 15.8s | 31618 | 28104 | 872 | $0.1939 |  |
| **opus** | baseline | 21.8s | 49503 | 42345 | 934 | $0.2920 |  |
| **opus** | mcp-only | 14.8s | 17157 | 0 | 808 | $0.1060 | 🏆 Winner |
| **opus** | mcp-full | 24.8s | 67229 | 56460 | 999 | $0.3893 |  |

### Quality Ranking (Opus 4.6)

1. **opus/baseline** — Most complete: covers all requested elements with accurate code/fields, includes how permissions connect to the broader system (PermissionsMixin, Groups, auto-created permissions), and provides specific line references.
2. **opus/mcp-full** — Equally accurate and well-structured, includes the broader system context (PermissionsMixin, Groups), with good line references and code snippets.
3. **opus/mcp-only** — Accurate and thorough, includes code snippets and explains the docstring's note about auto-created permissions, but slightly less context about the broader permission system (no mention of Groups/PermissionsMixin).
4. **sonnet/mcp-only** — Most detailed of the sonnet answers, includes the per-model-type-not-per-instance design point from the docstring, accurate code and line references throughout.
5. **sonnet/mcp-full** — Accurate and well-organized with good line references, but omits the broader system context and the docstring insight about per-model-type permissions.
6. **sonnet/baseline** — Correct and concise but the least complete: omits `__str__`, doesn't show full code blocks, and lacks broader system context.

## Efficiency

The mcp-only runs are dramatically cheaper ($0.11 for both sonnet and opus) with the fastest runtimes (14-17s), while baseline and mcp-full runs cost 2-3× more due to higher token usage. Opus/mcp-only delivers near-top-tier quality at the lowest cost and second-fastest time, offering the best quality-to-cost ratio.

## Verdict

**Winner: opus/mcp-only**

---

## py-flask-config [python / medium]

> How does Flask configuration loading work? Explain the Config class, how it loads from files, environment variables, and Python objects. Show the key methods and class hierarchy.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 63.8s | 31358 | 28104 | 1118 | $0.3541 |  |
| **sonnet** | mcp-only | 45.2s | 89041 | 0 | 2216 | $0.5006 |  |
| **sonnet** | mcp-full | 26.1s | 35330 | 28104 | 1301 | $0.2232 |  |
| **opus** | baseline | 47.2s | 43615 | 42345 | 1193 | $0.3418 |  |
| **opus** | mcp-only | 30.1s | 22676 | 0 | 1281 | $0.1454 | 🏆 Winner |
| **opus** | mcp-full | 27.0s | 48697 | 42345 | 1197 | $0.2946 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most complete and well-organized: covers all six methods with accurate code snippets, includes the call graph, explains ConfigAttribute thoroughly, and references specific line numbers. Clean structure with no errors.

2. **sonnet/mcp-only** — Very thorough with accurate line references and good code examples. Covers all methods with clear explanations. Slightly more verbose than opus/mcp-only without adding substance.

3. **opus/mcp-full** — Equally accurate and well-structured as opus/mcp-only, with correct line references and the same call chain diagram. Marginally less detail on `from_prefixed_env` nesting.

4. **sonnet/mcp-full** — Correct and complete with good line references, but the constructor section feels like padding and the overall structure is slightly less polished than the opus answers.

5. **sonnet/baseline** — Accurate with a nice loading chain diagram and table summary. Good use of line references. Slightly less detailed code snippets for some methods.

6. **opus/baseline** — Correct and concise with a useful table format, but provides the least code detail of all six. The table-driven approach sacrifices depth for brevity.

## Efficiency

Opus/mcp-only is the clear efficiency winner at $0.15 and 30s — cheapest and second-fastest while producing a top-quality answer. Sonnet/mcp-full is also efficient at $0.22 and 26s but with slightly lower quality. The baseline runs are surprisingly expensive (sonnet/baseline at $0.35, opus/baseline at $0.34) given they don't produce better answers, likely due to cache read costs. Sonnet/mcp-only is the most expensive at $0.50 with high input tokens and no cache hits.

## Verdict

**Winner: opus/mcp-only**

---

## py-django-queryset [python / hard]

> How does the Django QuerySet evaluation and filtering pipeline work? Explain QuerySet chaining, lazy evaluation, the Query class, how lookups and filters are compiled into SQL, and how the Manager ties it all together. Show key classes and method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 221.8s | 35742 | 28104 | 2469 | $3.6896 |  |
| **sonnet** | mcp-only | 74.7s | 113031 | 0 | 4304 | $0.6728 |  |
| **sonnet** | mcp-full | 62.2s | 127431 | 84312 | 3462 | $0.7659 |  |
| **opus** | baseline | 99.1s | 32282 | 28230 | 2724 | $0.8717 |  |
| **opus** | mcp-only | 56.4s | 73222 | 0 | 3419 | $0.4516 | 🏆 Winner |
| **opus** | mcp-full | 121.7s | 33502 | 28230 | 1953 | $1.3303 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/mcp-full** — Most thorough and well-structured: covers all five requested topics (chaining, lazy evaluation, Query class, lookup compilation, Manager), includes the iterable class hierarchy table, set operations, deferred filter pattern, and consistent file:line references throughout.

2. **opus/mcp-only** — Equally correct and complete with excellent code excerpts and line references; the six-section structure maps cleanly to the question's sub-topics; slightly less detail on set operations and iterable variants than sonnet/mcp-full.

3. **sonnet/mcp-only** — Strong coverage including the three-iterator protocol, set operations, and a clear end-to-end flow diagram; occasionally verbose but accurate with good line references.

4. **opus/mcp-full** — Correct and concise with good structure, but noticeably shorter than peers; the iterable class table and deferred filter coverage are nice touches, though SQL compilation depth is thinner.

5. **opus/baseline** — Solid and accurate with proper line references and a clean summary table; covers all major topics but less depth on deferred filters and set operations.

6. **sonnet/baseline** — Correct and detailed with good tables, but occasionally hedges about code "not in fixtures"; the end-to-end flow diagram is excellent, though it's slightly less precise on line references than the MCP variants.

## Efficiency

The opus/mcp-only run delivers a top-tier answer at the lowest cost ($0.45) and fastest time (56.4s), using moderate tokens. Sonnet/baseline is the most expensive at $3.69 with the slowest runtime (221.8s) — poor value. Sonnet/mcp-full offers strong quality at $0.77 and 62.2s, making it competitive, while opus/mcp-full is surprisingly expensive ($1.33) for a shorter answer.

## Verdict

**Winner: opus/mcp-only**

---

## ts-disposable [typescript / easy]

> What is the IDisposable interface and how does the Disposable base class work? Show the interface, the base class, and how DisposableStore manages multiple disposables.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 32.7s | 28796 | 28104 | 905 | $0.2794 |  |
| **sonnet** | mcp-only | 36.9s | 76589 | 0 | 2072 | $0.4347 |  |
| **sonnet** | mcp-full | 32.4s | 113129 | 84312 | 1587 | $0.6475 |  |
| **opus** | baseline | 27.4s | 53100 | 42345 | 1049 | $0.3129 |  |
| **opus** | mcp-only | 75.7s | 272171 | 0 | 3586 | $1.4505 |  |
| **opus** | mcp-full | 27.7s | 56290 | 42345 | 1221 | $0.3331 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-full** — Correct, complete, and well-structured with accurate line references (312, 416, 526). Includes all key methods, safety features, error aggregation, and a clear composition example. Concise without sacrificing detail.

2. **opus/baseline** — Equally accurate and complete, with the same line references and a nice composition example. Virtually identical quality to opus/mcp-full; slightly less detail on `deleteAndLeak`.

3. **sonnet/mcp-full** — Correct and thorough with accurate line references. Includes the ASCII composition diagram and mentions error aggregation. Slightly more verbose than opus variants but no less accurate.

4. **sonnet/baseline** — Accurate with correct line references and good table format for DisposableStore methods. Covers all key points including `DISABLE_DISPOSED_WARNING` and the `clear() ≠ dispose()` distinction.

5. **sonnet/mcp-only** — Correct and detailed with good line references and composition diagram. Slightly more verbose than needed but no errors. Mentions `AggregateError` and all safety guards.

6. **opus/mcp-only** — Accurate content but includes visible "thinking out loud" artifacts ("I have all the pieces now", "Let me also check if there's a `Disposable[1/N]` chunk"). Reconstructs the class from chunks rather than showing it cleanly. Labels `_store` as `private` when it's `protected` in some renderings. Otherwise complete.

## Efficiency

Opus/baseline and opus/mcp-full are nearly identical in cost (~$0.31-0.33) and time (~27s), delivering top-tier answers. Sonnet/baseline is comparable in cost ($0.28) and time (33s). The mcp-only runs are dramatically more expensive — opus/mcp-only is 4.6× the cost of opus/baseline at 75.7s for a lower-quality answer, and sonnet/mcp-only is 1.6× sonnet/baseline cost. The best quality-to-cost tradeoff is opus/baseline or opus/mcp-full, both delivering excellent answers at ~$0.32 in ~27s.

## Verdict

**Winner: opus/mcp-full**

---

## ts-event-emitter [typescript / medium]

> How does the event emitter system work? Explain the Event interface, the Emitter class, event composition (map, filter, debounce), and how events integrate with disposables. Show key types and patterns.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 100.0s | 33021 | 28104 | 1666 | $0.6185 |  |
| **sonnet** | mcp-only | 68.1s | 85742 | 0 | 3295 | $0.5111 |  |
| **sonnet** | mcp-full | 48.2s | 66782 | 42156 | 2624 | $0.4206 |  |
| **opus** | baseline | 57.2s | 127384 | 84690 | 2265 | $0.7359 |  |
| **opus** | mcp-only | 59.2s | 79003 | 0 | 3031 | $0.4708 | 🏆 Winner |
| **opus** | mcp-full | 57.0s | 130541 | 84690 | 2358 | $0.7540 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **opus/mcp-only** — Most comprehensive and well-structured; covers single-listener optimization, sparse array compaction, all major combinators with line numbers, ChainableSynthesis with HaltChainable sentinel, all emitter variants, and a six-point disposable integration section with a clean summary diagram.
2. **sonnet/mcp-only** — Very detailed with extensive inline code (snapshot, once, latch, ChainableSynthesis evaluate loop), good coverage of Relay and EventMultiplexer, and a strong typical-usage pattern at the end; slightly less organized than opus/mcp-only.
3. **opus/mcp-full** — Strong coverage with unique safety infrastructure section (LeakageMonitor, ListenerRefusalError, Stacktrace) and good debounce detail, but slightly thinner on the chainable API and variant emitters compared to the top two.
4. **opus/baseline** — Good unique details about EventDeliveryQueuePrivate reentrant safety and leak detection at threshold², solid overall but less structured and missing ChainableSynthesis coverage.
5. **sonnet/mcp-full** — Competent coverage with a unique mention of EmitterObserver bridging observables to events, but less depth on internals and fewer code samples than the top entries.
6. **sonnet/baseline** — Good tables of specialized emitters (7 variants) and shows MutableDisposable, but least deep on Emitter internals and composition mechanics.

## Efficiency

sonnet/mcp-full is the fastest (48.2s) and cheapest ($0.42), while opus/mcp-only is nearly as fast (59.2s) at the second-lowest cost ($0.47) — both dramatically cheaper than the baseline opus runs (~$0.74). The sonnet/baseline run is the slowest at 100s and mid-range cost, offering the worst efficiency overall.

## Verdict

**Winner: opus/mcp-only**

---

## ts-async-lifecycle [typescript / hard]

> How do async operations, cancellation, and resource lifecycle management work together? Explain CancelablePromise, CancellationToken, the async utilities (throttle, debounce, retry), how they integrate with the disposable lifecycle system, and how event-driven patterns compose with async flows. Show key interfaces and class relationships.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **sonnet** | baseline | 129.5s | 33741 | 28104 | 2959 | $0.8826 |  |
| **sonnet** | mcp-only | 83.6s | 78247 | 0 | 4527 | $0.5044 | 🏆 Winner |
| **sonnet** | mcp-full | 77.4s | 116509 | 70260 | 3779 | $0.7122 |  |
| **opus** | baseline | 172.2s | 32922 | 28230 | 2389 | $2.0144 |  |
| **opus** | mcp-only | 87.2s | 82414 | 0 | 4285 | $0.5192 |  |
| **opus** | mcp-full | 143.7s | 37785 | 28230 | 2396 | $0.9756 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **sonnet/mcp-only** — Most comprehensive and well-structured answer. Covers all requested topics (CancelablePromise, CancellationToken, async utilities, disposable lifecycle, event-async bridges) with accurate code snippets, specific file/line references (e.g., `cancellation.ts:144`, `async.ts:224`), and includes advanced topics like `AsyncIterableObject`, `Relay`, and `cancelOnDispose`. The relationship diagram and integration patterns are clear.

2. **sonnet/baseline** — Equally thorough with excellent architectural explanations and accurate code. Strong on design principles (lazy subscription propagation, settlement cleanup). Slightly less structured than mcp-only but covers `AsyncEmitter`, `Event.toPromise`, and composition patterns well with good line references.

3. **opus/mcp-full** — Correct and well-organized with a clean cascade diagram showing how `dispose()` propagates. Covers all key components but is somewhat shorter than the top sonnet answers, missing some details like `CancellationTokenPool`, `AsyncIterableSource`, and `Sequencer`.

4. **opus/mcp-only** — Strong coverage with good table-based summaries and accurate class hierarchy. Includes `AsyncIterableObject`/`AsyncIterableSource` and `DeferredPromise` which some others miss. The relationship summary is clean but the prose explanations are slightly less detailed than the top answers.

5. **opus/baseline** — Accurate and concise but noticeably shorter. Covers all major components correctly but with less depth — e.g., `AsyncEmitter` gets one sentence, `Throttler` internals are briefly mentioned. Good integration diagram but fewer code snippets.

6. **sonnet/mcp-full** — Correct and covers the core systems well, but is the least detailed of the six. Missing some advanced topics like `CancellationTokenPool`, `Sequencer`, `AsyncIterableObject`. The relationship diagram is simpler than others.

## Efficiency

The mcp-only runs for both models are the fastest and cheapest (sonnet: 83.6s/$0.50, opus: 87.2s/$0.52), while baseline runs are slowest (sonnet: 129.5s/$0.88, opus: 172.2s/$2.01). Sonnet/mcp-only delivers the highest-quality answer at the lowest cost, making it the clear efficiency winner; opus/baseline is the worst value at $2.01 for a shorter answer.

## Verdict

**Winner: sonnet/mcp-only**

---

## Overall: Algorithm Comparison

| Question | Language | Difficulty | 🏆 Winner | Runner-up |
|----------|----------|------------|-----------|-----------|
| go-label-matcher | go | easy | opus/mcp-only | sonnet/mcp-only |
| go-histogram | go | medium | opus/mcp-full | opus/mcp-only |
| go-tsdb-compaction | go | hard | opus/mcp-full | opus/mcp-only |
| py-permissions | python | easy | opus/mcp-only | sonnet/mcp-only |
| py-flask-config | python | medium | opus/mcp-only | sonnet/mcp-full |
| py-django-queryset | python | hard | opus/mcp-only | sonnet/mcp-only |
| ts-disposable | typescript | easy | opus/mcp-full | sonnet/baseline |
| ts-event-emitter | typescript | medium | opus/mcp-only | sonnet/mcp-full |
| ts-async-lifecycle | typescript | hard | sonnet/mcp-only | opus/mcp-only |

**Scenario Win Counts** (across all 9 questions):

| Scenario | Wins |
|----------|------|
| baseline | 0 |
| mcp-only | 6 |
| mcp-full | 3 |

**Overall winner: mcp-only** — won 6 of 9 questions.

_Full answers and detailed analysis: `detail-report.md`_
