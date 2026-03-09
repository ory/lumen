# SWE-Bench Summary

Generated: 2026-03-09 13:56 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | Diff | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------|------------|------------|----------|----------|----------|----------|
| go-easy | go | easy | Good | Perfect | $0.4016 | $0.4759 | 130.7s | 204.3s |
| go-hard | go | hard | Good | Good | $0.5876 | $0.4146 | 273.6s | 179.4s |
| go-medium | go | medium | Good | Good | $0.4692 | $0.2485 | 169.1s | 141.8s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 3 | 0 | $0.4861 | 191.2s | 15400 |
| **with-lumen** | 1 | 2 | 0 | $0.3796 | 175.2s | 16569 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| go | 0 | 1 |

