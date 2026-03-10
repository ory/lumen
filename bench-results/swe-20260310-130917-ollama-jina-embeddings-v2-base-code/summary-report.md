# SWE-Bench Summary

Generated: 2026-03-10 15:54 UTC | Embed: `ordis/jina-embeddings-v2-base-code` | Claude: `sonnet`

| Scenario | Description |
|----------|-------------|
| **baseline** | Default Claude tools, no Lumen |
| **with-lumen** | All default tools + Lumen |

## Results by Task

| Task | Lang | baseline Rating | with-lumen Rating | baseline Cost | with-lumen Cost | baseline Time | with-lumen Time |
|------|------|------------|------------|----------|----------|----------|----------|
| go-hard | go | Good | Good | $0.4048† | $0.4492† | 203.9s† | 195.8s† |
| javascript-hard | javascript | Perfect | Good | $0.6328† | $0.5458† | 336.4s† | 264.7s† |
| php-hard | php | Good | Perfect | $0.1825† | $0.1903† | 71.9s† | 74.6s† |
| python-hard | python | Perfect | Perfect | $0.0746† | $0.0827† | 25.2s† | 31.3s† |
| ruby-hard | ruby | Perfect | Poor | $0.7836† | $0.2565† | 502.4s† | 142.4s† |
| rust-hard | rust | Good | Good | $0.4490† | $0.5407† | 207.1s† | 279.4s† |
| typescript-hard | typescript | Good | Poor | $1.0498† | $1.5452† | 602.1s† | 840.3s† |

## Aggregate by Scenario

| Scenario | Perfect | Good | Poor | Avg Cost | Avg Time | Avg Tokens |
|----------|---------|------|------|----------|----------|------------|
| **baseline** | 3 | 4 | 0 | $0.5110 | 278.4s | 13641 |
| **with-lumen** | 2 | 3 | 2 | $0.5158 | 261.2s | 13509 |

## Aggregate by Language

| Language | baseline wins | with-lumen wins |
|----------|--------------|--------------|
| go | 0 | 0 |
| javascript | 1 | 0 |
| php | 0 | 1 |
| python | 0 | 0 |
| ruby | 1 | 0 |
| rust | 0 | 0 |
| typescript | 1 | 0 |

