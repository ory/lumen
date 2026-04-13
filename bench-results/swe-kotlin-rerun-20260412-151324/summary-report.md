# SWE-Bench Summary

Generated: 2026-04-12 13:25 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| kotlin-hard | kotlin | Good | Perfect | $0.7397 | $0.5098 | 308.3s | 195.8s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 1 | 0 | $0.7397 | 308.3s | 14840 |
| **with-lumen** | 1 | 0 | 0 | $0.5098 | 195.8s | 11807 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| kotlin | 0 | 1 |

