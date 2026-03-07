# SWE-Bench Summary

Generated: 2026-03-07 09:53 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **mcp-full** | All default tools + Lumen |

## Results by Task

| Task | Lang | Diff | baseline Rating | mcp-full Rating | baseline Cost | mcp-full Cost | baseline Time | mcp-full Time |
|------|------|------|------------|------------|----------|----------|----------|----------|
| go-easy | go | easy | Good | Good | $0.2682 | $0.2605 | 85.4s | 67.9s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 1 | 0 | $0.2682 | 85.4s | 4394 |
| **mcp-full** | 0 | 1 | 0 | $0.2605 | 67.9s | 3003 |

## Aggregate by Language

| Language | baseline wins | mcp-full wins |
|----------|--------------|--------------|
| go | 0 | 0 |

