## Content Quality

**Ranking: opus/mcp-full > sonnet/mcp-full > opus/mcp-only > sonnet/mcp-only > opus/baseline > sonnet/baseline**

**opus/mcp-full** — The best answer. It cleanly separates the two bucket mechanisms (classic vs native/sparse), explains the key computation for both schema ranges, and provides accurate function signatures with file:line references. The explanation of `addToBucket`, zero-threshold routing, and cumulative counting in `Write` is precise and well-structured. It includes the important struct/iterator type signatures without overloading the response. Concise and complete.

**sonnet/mcp-full** — Very close to opus/mcp-full in quality. It correctly identifies all four key functions (`findBucket`, `histogramCounts.observe`, `histogram.observe`, `histogram.Write`) with accurate line references. The explanation of the double-buffer pattern is a nice addition that opus/mcp-full omits. Slightly less detailed on the native bucket key computation (doesn't explain schema > 0 vs <= 0 paths), but the overall flow is clearly presented.

**opus/mcp-only** — Comprehensive and well-organized with accurate line references. Covers classic buckets, native buckets, cumulative counting on read, sparse iteration, and bucket limiting. The native bucket key computation explanation is thorough. Slightly more verbose than the mcp-full answers without adding proportional value, and the section headers make it feel more like documentation than a focused answer.

**sonnet/mcp-only** — Accurate and focused on the right functions with correct line references. Good explanation of the hot path and double-buffering. Missing some depth on the native bucket key computation (no schema > 0 vs <= 0 distinction) and doesn't cover `addToBucket` or `makeBuckets`. Still a solid answer that hits the core mechanics.

**opus/baseline** — The most comprehensive answer overall, covering `findBucket`, `observe`, `addToBucket`, `limitBuckets`, `makeBuckets`, and the iterator side. However, some line references are slightly off (e.g., 866 vs 864 for `findBucket`, 900 vs 899 for `observe`). The breadth is impressive but comes at 10-30x the cost of MCP answers — a poor tradeoff given the marginal quality gain.

**sonnet/baseline** — Covers both classic and sparse/native histograms, includes bucket construction helpers (`LinearBuckets`, `ExponentialBuckets`), and explains the delta-encoded iteration. However, it's the least focused answer — the bucket construction functions (`LinearBuckets`, etc.) aren't really about "bucket counting" per se, and the organization by file rather than by flow makes it harder to follow. Line references are absent (only file names). The $2.80 cost is hard to justify.

## Efficiency Analysis

| Scenario | Duration | Cost | Quality Rank |
|----------|----------|------|-------------|
| sonnet/mcp-only | 16.2s | $0.131 | 4th |
| sonnet/mcp-full | 16.3s | $0.206 | 2nd |
| opus/mcp-full | 18.9s | $0.193 | 1st |
| opus/mcp-only | 20.1s | $0.135 | 3rd |
| opus/baseline | 60.0s | $1.450 | 5th |
| sonnet/baseline | 127.4s | $2.807 | 6th |

**Key observations:**

- **MCP variants are 3-15x cheaper and 3-8x faster** than baselines across both models, while producing equal or better quality. The semantic search tool clearly finds the right code quickly.
- **sonnet/baseline is the outlier** at $2.81 and 127s — it consumed 31K input tokens with 28K cache reads, suggesting extensive file reading. The cost is 21x the cheapest option for the worst-ranked answer.
- **opus/mcp-full is the best quality-to-cost tradeoff** at $0.19 for the top-ranked answer. It's only $0.06 more than the cheapest option (sonnet/mcp-only) but delivers notably better depth and accuracy.
- **Cache reads don't help baselines much** — opus/baseline had 155K cache-read tokens but still cost $1.45, showing that brute-force file reading is fundamentally wasteful even with caching.
- **mcp-only vs mcp-full** shows minimal difference in duration (~2-3s) and cost (~$0.06), but mcp-full answers tend to be better organized, likely because the model has broader context from additional tools.

**Recommendation:** **opus/mcp-full** offers the best balance — top-quality answer at $0.19 in 19 seconds. For budget-conscious use, **sonnet/mcp-only** at $0.13 delivers a good answer at the lowest cost.
