# SWE-Bench Summary

Generated: 2026-05-04 22:09 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| dart-hard | dart | Poor | Good | $0.7414 | $0.4139 | 402.1s | 210.1s |
| python-hard | python | — | INVALID | — | — | — | — |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 0 | 1 | $0.7414 | 402.1s | 24180 |
| **with-lumen** | 0 | 1 | 0 | $0.4139 | 210.1s | 12910 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| dart | 0 | 1 |
| python | 0 | 0 |

