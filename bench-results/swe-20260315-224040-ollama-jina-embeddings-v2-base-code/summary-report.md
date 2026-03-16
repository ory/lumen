# SWE-Bench Summary

Generated: 2026-03-15 21:43 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| php-hard | php | Perfect | Good | $0.1760 | $0.1964 | 66.8s | 74.0s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 0 | 0 | $0.1760 | 66.8s | 6343 |
| **with-lumen** | 0 | 1 | 0 | $0.1964 | 74.0s | 7300 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| php | 1 | 0 |

