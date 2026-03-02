## Content Quality

**Ranking: opus/mcp-full > opus/baseline > sonnet/mcp-only > sonnet/mcp-full > opus/mcp-only > sonnet/baseline**

**opus/mcp-full** — The strongest answer overall. Correct throughout, with precise line references (e.g., line 37, 1093, 1174, 1315, 858, 204, 1347). Covers all requested topics: Event interface, Emitter internals (single-listener optimization, sparse arrays, compaction, leak detection, delivery queue), composition operators with the crucial `snapshot` pattern explained with actual code, disposable integration including the ownership/leak warning about composed events needing a store, and specialized emitters. The explanation of `snapshot` as the core primitive is a key architectural insight that some answers miss or underemphasize. Concise without sacrificing depth.

**opus/baseline** — Very close to opus/mcp-full in quality. Correct and thorough, with good line references. Includes the canonical usage pattern (private Emitter, public Event) which is pedagogically valuable. Covers `snapshot` with code, `EmitterOptions` hooks, delivery queue, leak detection, and specialized emitters including `AsyncEmitter`. Slightly less polished on disposable integration (fewer concrete details about the ownership chain) but adds the `AsyncEmitter` coverage that opus/mcp-full includes. Essentially tied with opus/mcp-full.

**sonnet/mcp-only** — Excellent depth and structure. Provides precise line references (e.g., `event.ts:37`, `event.ts:1093`, `event.ts:260`). The `snapshot` explanation with code is clear. The disposable integration section is the most thorough of all answers, covering five distinct integration patterns with code examples and explaining the `fromNodeEventEmitter` lazy bridging pattern. The ASCII diagram at the end is a nice summary. Covers `ChainableSynthesis` internals well. Minor: doesn't cover `AsyncEmitter`.

**sonnet/mcp-full** — Solid and correct. Good line references. Covers all major topics including `ChainableSynthesis` with `HaltChainable`. The composition section is well-organized with individual operator descriptions. Disposable integration is clear but slightly less detailed than sonnet/mcp-only. Covers `EventMultiplexer` which some others miss. The explanation of `snapshot` is present but less prominent than in the top answers.

**opus/mcp-only** — Correct and well-structured with precise line references. Good coverage of `EmitterOptions` hooks, the `snapshot` pattern, `ChainableSynthesis`, and all emitter variants including `EventMultiplexer`. Disposable integration is thorough with five numbered points. However, it's slightly drier and more catalog-like than the top answers — less architectural insight woven through the explanation. The `debounce` section could use more detail given its complexity.

**sonnet/baseline** — Correct and impressively comprehensive — covers the most operators of any answer (including `throttle`, `split`, `defer`, `runAndSubscribe`). The table of composition operators is the most complete. Covers `MutableDisposable`, `Relay`, `EventBufferer`, and `EventMultiplexer` which most others skip. However, it has **no line references at all**, which is a significant weakness for a codebase explanation task. The `snapshot` pattern — arguably the most important architectural detail — is never mentioned. Breadth over depth: it lists many things but explains fewer of the underlying mechanisms.

## Efficiency Analysis

| Scenario | Duration | Total Input | Output | Cost |
|----------|----------|-------------|--------|------|
| sonnet/baseline | 87.6s | 59,697 | 1,940 | $0.72 |
| sonnet/mcp-only | 66.8s | 114,798 | 2,938 | $0.65 |
| sonnet/mcp-full | 46.2s | 104,879 | 2,129 | $0.39 |
| opus/baseline | 55.8s | 212,060 | 2,350 | $0.74 |
| opus/mcp-only | 51.4s | 28,087 | 2,330 | $0.20 |
| opus/mcp-full | 53.8s | 215,247 | 2,029 | $0.75 |

**Most efficient: opus/mcp-only** at $0.20 — by far the cheapest run, with the lowest input tokens (28K) and fast runtime (51.4s). It produced a good answer, though not the best. This is the standout efficiency winner.

**Best cost for quality: sonnet/mcp-full** at $0.39 — half the cost of the baseline runs, fastest wall-clock time (46.2s), and produced a solid answer. The cache hit (42K tokens) helped significantly.

**Surprising findings:**
- **opus/mcp-only vs opus/mcp-full**: The mcp-only run used 7.6x fewer input tokens and cost 3.7x less, yet produced a comparable quality answer. The mcp-full run's cache reads (84K) inflated its cost without proportional quality gain.
- **sonnet/baseline was the slowest and most expensive sonnet run** despite having the fewest input tokens — suggesting it spent more turns reading files directly rather than leveraging search tools.
- **opus/baseline and opus/mcp-full cost nearly the same** ($0.74 vs $0.75) with similar token profiles, suggesting the MCP tools didn't save overhead when cache was already warm for opus.

**Recommended tradeoff:** For this type of architectural comprehension question, **opus/mcp-only** offers the best quality-to-cost ratio — a strong answer at $0.20. If you need top-tier quality regardless of cost, **opus/mcp-full** or **opus/baseline** at ~$0.75 deliver the most insightful answers. The sweet spot is opus/mcp-only at roughly 4x cheaper than the premium options with only marginally lower quality.
