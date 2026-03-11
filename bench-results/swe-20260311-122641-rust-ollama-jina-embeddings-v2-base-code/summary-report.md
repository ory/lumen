# SWE-Bench Summary

Generated: 2026-03-11 11:37 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| rust-hard | rust | Poor | Poor | $0.6113 | $0.3748 | 309.7s | 204.0s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 0 | 1 | $0.6113 | 309.7s | 17742 |
| **with-lumen** | 0 | 0 | 1 | $0.3748 | 204.0s | 12308 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| rust | 0 | 0 |

