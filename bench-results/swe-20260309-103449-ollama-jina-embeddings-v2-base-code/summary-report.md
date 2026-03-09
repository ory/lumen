# SWE-Bench Summary

Generated: 2026-03-09 09:59 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | Diff | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------|------------|------------|----------|----------|----------|----------|
| go-easy | go | easy | — | — | — | — | — | — |
| go-hard | go | hard | — | Poor | $0.2991 | — | 173.1s | — |
| go-medium | go | medium | — | — | $0.1317 | $0.2691 | 60.2s | 121.2s |
| php-easy | php | easy | — | Poor | — | — | — | — |
| php-hard | php | hard | — | — | $0.7139 | — | 295.3s | — |
| php-medium | php | medium | — | INVALID | $0.2865 | — | 128.6s | — |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 0 | 0 | $0.3578 | 164.3s | 14589 |
| **with-lumen** | 0 | 0 | 2 | $0.2691 | 121.2s | 9996 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| go | 0 | 1 |
| php | 0 | 2 |

