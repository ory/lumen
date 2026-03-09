# SWE-Bench Summary

Generated: 2026-03-09 00:32 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | Diff | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------|------------|------------|----------|----------|----------|----------|
| go-easy | go | easy | Good | Good | $0.7711† | $0.2975† | 148.5s† | 74.4s† |
| go-hard | go | hard | Perfect | Perfect | $0.1077† | $0.1278† | 14.3s† | 14.7s† |
| go-medium | go | medium | Good | Perfect | $0.1593† | $0.1600† | 23.0s† | 23.6s† |
| php-easy | php | easy | Perfect | Perfect | $0.1329† | $0.1160† | 15.4s† | 12.5s† |
| php-hard | php | hard | — | — | — | — | — | — |
| php-medium | php | medium | Good | Good | $0.1490† | $0.1667† | 18.7s† | 21.3s† |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 2 | 3 | 0 | $0.2640 | 44.0s | 2411 |
| **with-lumen** | 3 | 2 | 0 | $0.1736 | 29.3s | 1503 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| go | 0 | 1 |
| php | 0 | 0 |

