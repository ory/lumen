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
