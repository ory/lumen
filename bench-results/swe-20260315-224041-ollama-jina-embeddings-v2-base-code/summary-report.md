# SWE-Bench Summary

Generated: 2026-03-15 21:44 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| javascript-hard | javascript | Perfect | Good | $0.2348 | $0.2426 | 111.6s | 96.4s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 0 | 0 | $0.2348 | 111.6s | 7121 |
| **with-lumen** | 0 | 1 | 0 | $0.2426 | 96.4s | 7494 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| javascript | 1 | 0 |

