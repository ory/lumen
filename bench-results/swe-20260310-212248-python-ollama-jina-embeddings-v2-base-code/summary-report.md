# SWE-Bench Summary

Generated: 2026-03-10 20:25 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| python-hard | python | Perfect | Perfect | $0.1188 | $0.0956 | 43.0s | 30.6s |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 1 | 0 | 0 | $0.1188 | 43.0s | 1719 |
| **with-lumen** | 1 | 0 | 0 | $0.0956 | 30.6s | 1101 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| python | 0 | 0 |

