# SWE-Bench Summary

Generated: 2026-03-10 19:12 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| javascript-hard | javascript | Perfect | Perfect | $0.4821 | $0.3248 | 254.7s | 119.3s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 0 | 0 | $0.4821 | 254.7s | 14395 |
| **with-lumen** | 1 | 0 | 0 | $0.3248 | 119.3s | 4898 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| javascript | 0 | 0 |

