# SWE-Bench Summary

Generated: 2026-03-16 08:04 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| javascript-hard | javascript | Perfect | Good | $0.5037 | $0.6944 | 300.8s | 580.6s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 0 | 0 | $0.5037 | 300.8s | 9929 |
| **with-lumen** | 0 | 1 | 0 | $0.6944 | 580.6s | 13309 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| javascript | 1 | 0 |

