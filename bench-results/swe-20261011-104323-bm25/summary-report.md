# SWE-Bench Summary

Generated: 2026-10-11 09:54 UTC | Retrieval: `bm25` | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `claude-sonnet-5-5`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| c-hard | c | Good | INVALID | $0.1076 | $0.1673 | 22.9s | 59.4s |
| cpp-hard | cpp | Good | INVALID | $0.2834 | $0.3074 | 100.9s | 184.1s |
| csharp-hard | csharp | — | INVALID | — | — | — | — |
| dart-hard | dart | Good | INVALID | $0.1326 | $0.1704 | 37.2s | 36.0s |
| go-hard | go | Good | INVALID | $0.1108 | $0.1370 | 23.7s | 73.5s |
| java-hard | java | Poor | Poor | $0.1746 | $0.1923 | 88.8s | 81.4s |
| javascript-hard | javascript | Good | INVALID | $0.1154 | $0.1253 | 49.1s | 39.5s |
| php-hard | php | Perfect | INVALID | $0.1002 | $0.0880 | 22.5s | 22.4s |
| python-hard | python | Good | INVALID | $0.1199 | $0.0997 | 61.3s | 33.2s |
| ruby-hard | ruby | Good | INVALID | $0.1416 | $0.1724 | 59.9s | 101.4s |
| rust-hard | rust | Poor | Poor | $0.1499 | $0.2534 | 74.4s | 748.8s |
| svelte-hard | svelte | Good | INVALID | $0.1000 | $0.1058 | 21.4s | 23.5s |
| swift-hard | swift | Good | Good | $0.1436 | $0.2017 | 128.5s | 113.4s |
| typescript-hard | typescript | — | INVALID | — | — | — | — |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 9 | 2 | $0.1400 | 57.6s | 2980 |
| **with-lumen** | 0 | 1 | 2 | $0.2158 | 314.5s | 5173 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| c | 0 | 0 |
| cpp | 0 | 0 |
| csharp | 0 | 0 |
| dart | 0 | 0 |
| go | 0 | 0 |
| java | 0 | 0 |
| javascript | 0 | 0 |
| php | 0 | 0 |
| python | 0 | 0 |
| ruby | 0 | 1 |
| rust | 0 | 0 |
| svelte | 0 | 0 |
| swift | 0 | 0 |
| typescript | 0 | 0 |

