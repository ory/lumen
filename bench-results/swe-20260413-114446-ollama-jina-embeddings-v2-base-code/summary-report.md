# SWE-Bench Summary

Generated: 2026-04-13 10:01 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| kotlin-hard | kotlin | Good | Perfect | $0.5606 | $0.3516 | 350.1s | 231.4s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 1 | 0 | $0.5606 | 350.1s | 15816 |
| **with-lumen** | 1 | 0 | 0 | $0.3516 | 231.4s | 14167 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| kotlin | 0 | 1 |

