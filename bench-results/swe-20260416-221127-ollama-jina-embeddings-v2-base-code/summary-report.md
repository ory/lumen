# SWE-Bench Summary

Generated: 2026-04-17 02:17 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| svelte-hard | svelte | Poor | Poor | $0.1355 | $0.1004 | 79.9s | 55.5s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 0 | 1 | $0.1355 | 79.9s | 4136 |
| **with-lumen** | 0 | 0 | 1 | $0.1004 | 55.5s | 3056 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| svelte | 0 | 0 |

