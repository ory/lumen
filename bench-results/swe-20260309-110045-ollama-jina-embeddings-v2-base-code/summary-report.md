# SWE-Bench Summary

Generated: 2026-03-09 10:11 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | Diff | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------|------------|------------|----------|----------|----------|----------|
| go-easy | go | easy | Perfect | INVALID | $0.6055 | $0.4036 | 215.4s | 154.7s |
| go-hard | go | hard | Good | INVALID | $0.5244 | $0.1358 | 196.8s | 76.2s |
| go-medium | go | medium | Good | Good | $0.3344 | $0.3430 | 141.8s | 128.3s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 2 | 0 | $0.4881 | 184.7s | 15355 |
| **with-lumen** | 0 | 1 | 0 | $0.3430 | 128.3s | 9994 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| go | 1 | 0 |

