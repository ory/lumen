# SWE-Bench Summary

Generated: 2026-05-04 19:42 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| javascript-hard | javascript | Perfect | Perfect | $0.6171 | $0.6090 | 328.4s | 219.0s |
| ruby-hard | ruby | — | INVALID | — | — | — | — |
| typescript-hard | typescript | Perfect | Perfect | $0.2812 | $0.2436 | 161.1s | 110.9s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 2 | 0 | 0 | $0.4492 | 244.7s | 11386 |
| **with-lumen** | 2 | 0 | 0 | $0.4263 | 164.9s | 8777 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| javascript | 0 | 0 |
| ruby | 0 | 0 |
| typescript | 0 | 0 |

