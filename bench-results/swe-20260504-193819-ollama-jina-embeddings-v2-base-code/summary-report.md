# SWE-Bench Summary

Generated: 2026-05-04 23:58 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `haiku`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| java-hard | java | Poor | Poor | $0.4108 | $0.3617 | 171.2s | 165.8s |
| rust-hard | rust | Poor | Good | $0.8494 | $0.9237 | 449.1s | 539.4s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 0 | 0 | 2 | $0.6301 | 310.2s | 26256 |
| **with-lumen** | 0 | 1 | 1 | $0.6427 | 352.6s | 33205 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| java | 0 | 0 |
| rust | 0 | 1 |

