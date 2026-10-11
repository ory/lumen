# SWE-Bench Summary

Generated: 2026-10-11 07:54 UTC | Retrieval: `bm25` | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `claude-sonnet-5-5`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| c-hard | c | Perfect | INVALID | $0.1413 | $0.1620 | 104.0s | 45.5s |
| cpp-hard | cpp | Good | INVALID | $0.2856 | $0.2474 | 90.4s | 69.2s |
| csharp-hard | csharp | — | INVALID | — | — | — | — |
| dart-hard | dart | Good | INVALID | $0.1598 | $0.2260 | 41.4s | 67.4s |
| go-hard | go | Good | INVALID | $0.1194 | $0.1328 | 23.1s | 27.9s |
| java-hard | java | Good | INVALID | $0.2324 | $0.1833 | 84.2s | 73.2s |
| javascript-hard | javascript | Good | INVALID | $0.1146 | $0.1438 | 34.2s | 61.5s |
| php-hard | php | Perfect | INVALID | $0.0884 | $0.0932 | 20.7s | 14.5s |
| python-hard | python | Perfect | INVALID | $0.0984 | $0.1137 | 19.7s | 24.0s |
| ruby-hard | ruby | Good | INVALID | $0.1268 | $0.1782 | 26.4s | 67.0s |
| rust-hard | rust | Good | INVALID | $0.2113 | $0.2371 | 91.8s | 89.1s |
| svelte-hard | svelte | Good | INVALID | $0.1010 | $0.0926 | 16.8s | 13.1s |
| swift-hard | swift | Good | INVALID | $0.1513 | $0.1687 | 98.7s | 100.0s |
| typescript-hard | typescript | — | INVALID | — | — | — | — |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 3 | 9 | 0 | $0.1525 | 54.3s | 3367 |
| **with-lumen** | 0 | 0 | 0 | — | — | — |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| c | 1 | 0 |
| cpp | 0 | 0 |
| csharp | 0 | 0 |
| dart | 0 | 0 |
| go | 0 | 0 |
| java | 1 | 0 |
| javascript | 0 | 0 |
| php | 0 | 0 |
| python | 1 | 0 |
| ruby | 0 | 1 |
| rust | 1 | 0 |
| svelte | 0 | 0 |
| swift | 0 | 0 |
| typescript | 0 | 0 |

