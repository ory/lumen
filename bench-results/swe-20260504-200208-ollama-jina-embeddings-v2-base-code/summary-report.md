# SWE-Bench Summary

Generated: 2026-05-05 00:32 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| c-hard | c | — | INVALID | — | — | — | — |
| cpp-hard | cpp | Good | Good | $1.1953 | $0.3724 | 374.1s | 769.7s |
| csharp-hard | csharp | — | INVALID | — | — | — | — |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 1 | 0 | $1.1953 | 374.1s | 14599 |
| **with-lumen** | 0 | 1 | 0 | $0.3724 | 769.7s | 12750 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| c | 0 | 0 |
| cpp | 0 | 0 |
| csharp | 0 | 0 |

