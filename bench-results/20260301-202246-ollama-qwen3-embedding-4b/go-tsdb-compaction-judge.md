## Content Quality

**Ranking: opus/mcp-full > opus/mcp-only > sonnet/mcp-full ≈ opus/baseline > sonnet/mcp-only > sonnet/baseline**

**opus/mcp-full** — The most complete and well-structured answer. Correctly identifies the `Compactor` interface at `compact.go:52-77`, `LeveledCompactor` at `compact.go:79-93`, and all three planning strategies (overlapping, leveled, tombstone). Uniquely includes `EnableCompactions`/`DisableCompactions` toggle details (`db.go:2242-2258`) and the `Head.compactable` logic with the 1.5x explanation. Line references are precise and consistent. Covers the full trigger chain from `Commit()` through the run loop to both head and block compaction phases. The end-to-end summary is clean and accurate.

**opus/mcp-only** — Nearly as complete as opus/mcp-full. Covers the same core material with accurate line references. Uniquely mentions OOO compaction (`db.go:1534-1585`) and stale series compaction as separate paths, which adds real value. The table format for `LeveledCompactor` fields is a nice touch. Slightly less detailed on the `write` internals and `DefaultBlockPopulator` compared to the mcp-full variant, but the overall accuracy is excellent.

**sonnet/mcp-full** — Solid coverage with correct line references (`compact.go:52`, `compact.go:79`, `compact.go:790` for `PopulateBlock`). Includes the appender commit trigger path and `Head.compactable` logic. Covers `DefaultBlockPopulator.PopulateBlock` which some others gloss over. The summary flow diagram is concise and accurate. Slightly less detailed on edge cases (no mention of `Compaction.Failed` marking, no OOO path).

**opus/baseline** — Strong on correctness and includes `splitByRange` explanation which others omit. Good coverage of the `write` internals and `DefaultBlockPopulator`. Line references are accurate (`compact.go:485-584`, `db.go:1390-1402`). The end-to-end flow diagram showing level progression (2h→10h→50h) is pedagogically effective. Misses OOO compaction and stale series paths.

**sonnet/mcp-only** — Comprehensive and well-organized. Correctly identifies all major components with accurate line references. Includes `DefaultBlockPopulator.PopulateBlock` at `compact.go:790` and the `Head.compactable` formula. The "Key design choices" section at the end adds useful context (head priority, channel buffering, mutex separation). However, some line references feel slightly imprecise (e.g., `compact.go:489` vs the more common `483-487` split), and it's somewhat verbose without proportionally more insight.

**sonnet/baseline** — Correct overall but the least precise on some details. References `compact.go:54-77` and `compact.go:80-93` which are slightly off from other answers' consensus of `:52-77` and `:79-93`, suggesting less precise tool usage. The `db.run` code block shows a simplified/paraphrased version that conflates the two select arms. Coverage of `selectDirs` and `splitByRange` internals is good. Missing OOO compaction entirely.

---

## Efficiency Analysis

| Scenario | Duration | Total Input | Cost | Quality Rank |
|---|---|---|---|---|
| opus/mcp-only | 47.4s | 61.9K | $0.36 | 2nd |
| opus/mcp-full | 50.9s | 80.2K+42.3K cache | $0.49 | 1st |
| sonnet/mcp-full | 61.7s | 191K+98K cache | $1.08 | 3rd |
| sonnet/mcp-only | 79.5s | 252K | $1.35 | 5th |
| opus/baseline | 115.0s | 31.9K+28.2K cache | $1.02 | 4th |
| sonnet/baseline | 110.5s | 634K+253K cache | $3.63 | 6th |

**Key observations:**

- **Opus dominates on efficiency.** All three opus runs cost under $1.05, while all three sonnet runs cost over $1.05. Opus with MCP tools is remarkably cheap — the mcp-only run at $0.36 delivers the second-best answer at 1/10th the cost of sonnet/baseline.

- **MCP tools dramatically reduce sonnet's costs.** Sonnet/baseline consumed 634K input tokens at $3.63 — nearly 6x more expensive than sonnet/mcp-full ($1.08). The semantic search index lets it skip reading large swaths of code, which is especially impactful for sonnet's apparently more verbose exploration strategy.

- **Opus is inherently more token-efficient.** Even opus/baseline used only 31.9K input tokens vs sonnet/baseline's 634K — a 20x difference. Opus appears to make far more targeted tool calls regardless of available tooling.

- **Speed correlates with MCP usage.** The two fastest runs (opus/mcp-only at 47.4s, opus/mcp-full at 50.9s) both used MCP. Baseline runs for both models exceeded 110s.

- **The surprising result:** opus/mcp-only slightly outperforms opus/mcp-full on cost ($0.36 vs $0.49) and speed (47.4s vs 50.9s) while delivering comparable quality. The cache reads in mcp-full suggest redundant re-reads that the mcp-only run avoided.

**Recommendation:** **opus/mcp-only** is the clear best quality-to-cost tradeoff — second-best answer at the lowest cost ($0.36) and fastest runtime (47.4s). For maximum quality with modest cost increase, **opus/mcp-full** at $0.49 is the premium pick. Sonnet/baseline should be avoided entirely for this type of deep codebase exploration — it costs 10x more than opus/mcp-only for a worse result.
