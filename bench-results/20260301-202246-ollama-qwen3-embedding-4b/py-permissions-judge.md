## Content Quality

**Ranking: 1st (tie) — opus/baseline, opus/mcp-only, opus/mcp-full**

All three Opus answers are essentially equivalent in quality. They are fully correct against the source (lines 27–85 verified). Line references are accurate. All three correctly identify the three fields, `unique_together`, ordering, `__str__`, `natural_key()`, and `get_by_natural_key`. They add useful contextual explanation (Django auto-creates four permissions per model, how natural keys enable fixture serialization). The opus/baseline answer goes slightly further by mentioning `PermissionsMixin` (line 317/342) and how permissions connect to users/groups — genuinely relevant context. The opus/mcp-full answer is the most concise while still covering everything. All use precise `file:line` references.

**Ranking: 4th (tie) — sonnet/baseline, sonnet/mcp-only, sonnet/mcp-full**

All three Sonnet answers are also correct and complete. The code snippets match the source. The differences from the Opus answers are minor: Sonnet answers are slightly more verbose in formatting (bigger tables, more horizontal rules) without adding proportionally more insight. The sonnet/baseline says "lines 39–86" when the class actually ends at line 85 (line 86 is empty) — a trivial inaccuracy. The sonnet/mcp-only omits the full file path (`django-models.py` without `testdata/fixtures/python/` prefix). The sonnet/mcp-full includes a good summary flow paragraph at the end. All Sonnet answers lack the broader context about `PermissionsMixin` and how permissions connect to the user model that opus/baseline provides. Overall the quality gap between Opus and Sonnet is small — all six answers are good.

## Efficiency Analysis

| Scenario | Duration | Input Tok | Output Tok | Cost |
|---|---|---|---|---|
| sonnet/mcp-only | 15.2s | 18,426 | 888 | **$0.114** |
| opus/mcp-only | 16.7s | 17,469 | 820 | **$0.108** |
| sonnet/mcp-full | 13.7s | 30,469 | 852 | $0.188 |
| opus/mcp-full | 19.2s | 44,606 | 837 | $0.265 |
| sonnet/baseline | 32.0s | 28,495 | 1,015 | $0.277 |
| opus/baseline | 20.8s | 49,167 | 894 | $0.289 |

**Key observations:**

- **MCP-only is the clear efficiency winner.** Both sonnet/mcp-only ($0.114) and opus/mcp-only ($0.108) are 2–2.5× cheaper than their baseline counterparts, with comparable or better quality. They use dramatically fewer input tokens (~18k vs ~28–49k) because semantic search returns targeted chunks rather than requiring full file reads.

- **Baseline is the most expensive across the board.** The sonnet/baseline is the slowest at 32s and opus/baseline uses the most input tokens (49k). The baseline approach presumably reads more of the file or surrounding context to find the relevant code.

- **MCP-full offers no advantage over MCP-only here.** It costs 1.6–2.5× more than MCP-only due to higher input tokens (likely the full CLAUDE.md context), with no quality improvement. The "full" toolset is overkill for a targeted lookup question.

- **sonnet/mcp-full is the fastest** at 13.7s, likely benefiting from cache reads (28k cached) plus Sonnet's inherently faster generation.

- **Opus/mcp-only is the best quality-to-cost tradeoff** at $0.108 — the cheapest run overall, with top-tier answer quality. For this type of targeted code comprehension question, semantic search alone is sufficient and the most efficient approach.

**Recommendation:** For focused code lookup questions, **mcp-only** is the optimal configuration regardless of model. It delivers equivalent quality at ~40–60% lower cost and comparable latency. Use opus/mcp-only when answer quality matters most, sonnet/mcp-only when speed is the priority.
