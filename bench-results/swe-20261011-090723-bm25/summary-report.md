# SWE-Bench Summary

Generated: 2026-10-11 07:33 UTC | Retrieval: `bm25` | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `claude-sonnet-5-5`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| c-hard | c | Good | INVALID | $0.1237 | $0.1343 | 94.0s | 50.0s |
| cpp-hard | cpp | Good | INVALID | $0.2758 | $0.4102 | 156.8s | 145.3s |
| csharp-hard | csharp | — | INVALID | — | — | — | — |
| dart-hard | dart | Good | INVALID | $0.1262 | $0.1420 | 41.2s | 93.8s |
| go-hard | go | Good | INVALID | $0.1175 | $0.1227 | 36.2s | 46.6s |
| java-hard | java | Poor | INVALID | $0.1644 | $0.1832 | 71.8s | 80.6s |
| javascript-hard | javascript | Good | INVALID | $0.1126 | $0.1365 | 33.1s | 85.4s |
| php-hard | php | Good | INVALID | $0.0784 | $0.0807 | 21.7s | 11.5s |
| python-hard | python | Good | INVALID | $0.1034 | $0.1067 | 36.5s | 22.6s |
| ruby-hard | ruby | Perfect | INVALID | $0.1197 | $0.1264 | 31.5s | 33.4s |
| rust-hard | rust | Poor | INVALID | $0.2080 | $0.2160 | 205.5s | 70.1s |
| svelte-hard | svelte | Good | INVALID | $0.1094 | $0.1275 | 18.8s | 21.1s |
| swift-hard | swift | Good | INVALID | $0.1806 | $0.1386 | 78.8s | 96.0s |
| typescript-hard | typescript | — | INVALID | — | — | — | — |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 9 | 2 | $0.1433 | 68.8s | 2857 |
| **with-lumen** | 0 | 0 | 0 | — | — | — |

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
| python | 0 | 1 |
| ruby | 1 | 0 |
| rust | 0 | 0 |
| svelte | 0 | 0 |
| swift | 0 | 0 |
| typescript | 0 | 0 |

