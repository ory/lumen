# SWE-Bench Summary

Generated: 2026-05-05 01:05 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| swift-hard | swift | Poor | Good | — | $0.2873 | — | 570.9s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 0 | 1 | — | — | — |
| **with-lumen** | 0 | 1 | 0 | $0.2873 | 570.9s | 9230 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| swift | 0 | 1 |

