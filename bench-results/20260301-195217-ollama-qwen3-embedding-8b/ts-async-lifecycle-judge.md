## Content Quality

**Ranking: opus/mcp-full > opus/mcp-only > sonnet/mcp-full ≈ sonnet/mcp-only > opus/baseline > sonnet/baseline**

**opus/mcp-full** — The most complete and well-structured answer. It correctly covers all five subsystems (disposables, cancellation, CancelablePromise, async utilities, events) with accurate file:line references (e.g., `lifecycle.ts:312-314`, `async.ts:34-81`, `cancellation.ts:144-148`). The composition diagram at the end clearly shows how the layers interconnect. It accurately describes `cancelOnDispose`, `thenRegisterOrDispose`, lazy token creation, the auto-dispose-on-cancel behavior in `createCancelablePromise`, and event combinator lifecycle hooks. The `retry` implementation is shown with actual code. Every claim maps to a specific location.

**opus/mcp-only** — Nearly as strong as opus/mcp-full, with precise line references and accurate descriptions of internal mechanics like `MutableToken` wrapping an `Emitter<void>`. It uniquely highlights `thenIfNotDisposed` and `thenRegisterOrDispose` as bridges between promises and disposables — details other answers miss. The layered structure (disposable → cancellation → async) is pedagogically clear. Coverage of `TaskSequentializer`, `RunOnceScheduler`, and `ThrottledWorker` goes deeper than most answers. Slightly less polished composition diagram than opus/mcp-full.

**sonnet/mcp-full** — Solid coverage with good file references. Correctly describes the three-layer architecture and most key classes. The composition diagram is serviceable. It includes `TaskSequentializer` and `SequencerByKey` which some answers miss. However, it's slightly less precise than the opus answers — for example, it describes `AsyncEmitter` more briefly and doesn't mention `thenRegisterOrDispose`. The "How They Compose" section is more of a dependency list than a true explanation of integration patterns.

**sonnet/mcp-only** — Impressively detailed, with the best coverage of `AsyncEmitter` and the `IWaitUntil` pattern among all answers. The `ThrottledWorker` section correctly identifies the `MutableDisposable<RunOnceScheduler>` pattern. The composition diagram is the most detailed, showing the full flow from events through cancelable promises to disposable stores. However, some line references (e.g., `cancellation.ts:144`, `async.ts:573`) appear accurate but aren't as consistently provided as the opus answers. `raceCancellablePromises` is correctly described.

**opus/baseline** — Well-organized with correct descriptions of all major components. Covers `DisposableMap`, `DisposableSet`, `RefCountedDisposable`, and leak detection that some MCP-assisted answers skip. The five integration patterns at the end are clearly articulated. However, lacking tool access means no file:line references, and some internal details (like `MutableToken` internals, `cancelOnDispose` implementation) are described at a higher level without code evidence. The `CancellationTokenPool` description as an "AND-gate" is a nice conceptual framing.

**sonnet/baseline** — Covers the right topics but is the least precise. No file:line references. Some descriptions are slightly vague (e.g., `Throttler` description says "holds a `CancellationTokenSource` internally" without showing how). The composition diagram is useful but simpler. Missing `TaskSequentializer`, `SequencerByKey`, `RunOnceScheduler`. The `EmitterOptions` lifecycle hooks section is good but brief. Overall correct but thinnest on implementation detail.

## Efficiency Analysis

| Scenario | Duration | Total Input | Output | Cost |
|----------|----------|-------------|--------|------|
| sonnet/baseline | 144.5s | 65K | 1,970 | $1.32 |
| sonnet/mcp-only | 81.1s | 94K | 3,962 | $0.57 |
| sonnet/mcp-full | 65.0s | 189K | 3,192 | $0.71 |
| opus/baseline | 116.1s | 60K | 2,939 | $0.73 |
| opus/mcp-only | 75.3s | 80K | 3,739 | $0.49 |
| opus/mcp-full | 154.4s | 64K | 2,932 | $1.18 |

**Most efficient: opus/mcp-only** at $0.49 — lowest cost, strong quality, reasonable 75s runtime. It found the right information through semantic search without needing full tool access, and opus produced a thorough answer in fewer tokens than sonnet/mcp-only.

**Surprising findings:**
- **sonnet/baseline is the most expensive** ($1.32) despite producing the weakest answer. The 144s runtime and low output (1,970 tokens) suggest it spent most of its budget on internal reasoning rather than retrieval, yet still produced less detail.
- **opus/mcp-full is unexpectedly slow** (154s) and expensive ($1.18) — the slowest of all runs. This suggests opus with full tools may over-explore, reading more files than necessary. The quality improvement over opus/mcp-only is marginal.
- **sonnet/mcp-only vs sonnet/mcp-full**: mcp-only produced more output (3,962 vs 3,192 tokens) at lower cost ($0.57 vs $0.71), and the quality is comparable. The additional tools didn't clearly help sonnet.
- **Cache reads** dramatically affect cost — sonnet/baseline's 28K cache read still resulted in the highest cost due to output pricing and reasoning overhead.

**Best quality-to-cost tradeoff: opus/mcp-only** ($0.49, rank #2 quality). For just $0.05 more than the cheapest run, you get the second-best answer with precise line references and deep architectural understanding. opus/mcp-full is the top-quality answer but at 2.4x the cost and 2x the runtime — a poor marginal return.
