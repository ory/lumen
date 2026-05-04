# SWE-Bench Summary

Generated: 2026-05-04 19:11 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| svelte-hard | svelte | Poor | Good | $0.3430 | $0.1302 | 160.7s | 41.5s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 0 | 1 | $0.3430 | 160.7s | 8923 |
| **with-lumen** | 0 | 1 | 0 | $0.1302 | 41.5s | 2006 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| svelte | 0 | 1 |

