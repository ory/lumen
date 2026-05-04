# SWE-Bench Summary

Generated: 2026-05-04 20:03 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| ruby-hard | ruby | Good | Good | $0.5357 | $0.3705 | 248.3s | 149.1s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 1 | 0 | $0.5357 | 248.3s | 10907 |
| **with-lumen** | 0 | 1 | 0 | $0.3705 | 149.1s | 8433 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| ruby | 0 | 0 |

