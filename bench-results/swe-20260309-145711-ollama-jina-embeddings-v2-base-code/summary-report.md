# SWE-Bench Summary

Generated: 2026-03-09 14:15 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | Diff | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------|------------|------------|----------|----------|----------|----------|
| go-easy | go | easy | Poor | INVALID | — | $0.6723 | — | 235.6s |
| go-hard | go | hard | Good | INVALID | $0.9738 | $0.4909 | 238.7s | 118.5s |
| go-medium | go | medium | Perfect | INVALID | $0.1522 | $0.1628 | 24.2s | 29.3s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 1 | 1 | $0.5630 | 131.4s | 6441 |
| **with-lumen** | 0 | 0 | 0 | — | — | — |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| go | 1 | 0 |

