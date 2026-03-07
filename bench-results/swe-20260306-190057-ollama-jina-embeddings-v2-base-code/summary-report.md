# SWE-Bench Summary

Generated: 2026-03-06 18:04 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **mcp-full** | All default tools + Lumen |

## Results by Task

| Task | Lang | Diff | baseline Rating | mcp-full Rating | baseline Cost | mcp-full Cost | baseline Time | mcp-full Time |
|------|------|------|------------|------------|----------|----------|----------|----------|
| go-easy | go | easy | Good | Good | $0.2250 | $0.1550 | 110.9s | 93.0s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 1 | 0 | $0.2250 | 110.9s | 8735 |
| **mcp-full** | 0 | 1 | 0 | $0.1550 | 93.0s | 8206 |

## Aggregate by Language

| Language | baseline wins | mcp-full wins |
|----------|--------------|--------------|
| go | 0 | 0 |

