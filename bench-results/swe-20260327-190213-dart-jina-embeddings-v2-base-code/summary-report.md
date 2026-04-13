# SWE-Bench Summary

Generated: 2026-03-27 18:08 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| dart-hard | dart | Good | Good | $0.6342 | $0.1533 | 246.1s | 50.9s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 1 | 0 | $0.6342 | 246.1s | 21725 |
| **with-lumen** | 0 | 1 | 0 | $0.1533 | 50.9s | 3988 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| dart | 0 | 0 |

