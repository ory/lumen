## Content Quality

**Ranking: opus/baseline ≈ opus/mcp-full > opus/mcp-only > sonnet/mcp-only > sonnet/mcp-full > sonnet/baseline**

**opus/baseline** — Excellent answer. Correct on all technical details: fields, manager, natural key, Meta constraints. Includes precise file/line references (`testdata/fixtures/python/django-models.py:27-36`, `39-85`). The "How it fits together" section adds genuine value by explaining how `Permission` connects to `PermissionsMixin`, `Group`, and the auto-created add/change/delete/view permissions — context that directly answers "how does it work" beyond just showing the class. References specific line numbers for related code (line 342, line 121, line 317). Thorough without being bloated.

**opus/mcp-full** — Nearly identical quality to opus/baseline. Same correct technical content, same code snippets, same "How it fits together" section connecting Permission to PermissionsMixin and Group. Line references are accurate. The only minor difference is slightly less specific line references for the related classes (mentions "line 317" for PermissionsMixin but not the M2M line numbers). Essentially tied with baseline.

**opus/mcp-only** — Correct and well-structured. Covers all requested components: PermissionManager, get_by_natural_key, Permission class, fields, Meta, and natural_key. Good file/line references. Mentions the docstring detail about auto-created permissions (add/change/delete/view) and that permissions are global per model type. However, it lacks the "how it fits together" synthesis that both other opus answers provide — no mention of PermissionsMixin, Group, or has_perm(), which means it answers the "show me" part well but less of the "how does it work" part.

**sonnet/mcp-only** — Correct and complete. Includes actual code blocks for both PermissionManager and Permission, a clean field table, and good explanations. Mentions the per-model-type vs per-object distinction from the docstring. File path is correctly identified as `testdata/fixtures/python/django-models.py`. Line references are accurate. Slightly less contextual than opus answers (no mention of how Permission connects to User/Group), but solid on everything explicitly asked for.

**sonnet/mcp-full** — Correct and concise. Covers all requested components with accurate code and explanations. Clean formatting with field table and bullet points. However, it's the most terse of the group — no mention of the docstring context about auto-created permissions or the per-model-type design, and no broader system context. The file reference says just "django-models.py" without the full path. Perfectly accurate but minimal.

**sonnet/baseline** — Correct on all technical details shown. Good field table, accurate code for PermissionManager. However, the file reference is just `django-models.py:27-36` without the `testdata/fixtures/python/` prefix. Doesn't include the full Permission class code block (only shows `natural_key`), which is a gap given the question explicitly asks to "show the Permission class." No broader context about how permissions connect to users/groups. Adequate but the least complete.

## Efficiency Analysis

| Scenario | Duration | Total Input | Output | Cost |
|---|---|---|---|---|
| sonnet/mcp-only | 17.1s | 18,694 | 980 | $0.118 |
| opus/mcp-only | 14.8s | 17,157 | 808 | $0.106 |
| sonnet/mcp-full | 15.8s | 31,618 | 872 | $0.194 |
| sonnet/baseline | 29.7s | 28,449 | 831 | $0.272 |
| opus/baseline | 21.8s | 49,503 | 934 | $0.292 |
| opus/mcp-full | 24.8s | 67,229 | 999 | $0.389 |

**Most efficient: opus/mcp-only** at $0.106 and 14.8s — fastest runtime, lowest cost, and still produced a high-quality answer. The MCP semantic search found the relevant file directly without needing multiple tool calls or large context windows.

**Best quality-to-cost ratio: opus/mcp-only** — It delivers ~90% of the quality of opus/baseline at 36% of the cost. The only thing missing is the broader system context (PermissionsMixin/Group connections), which is a nice-to-have rather than essential given the question.

**Surprising findings:**
- **opus/mcp-full is the most expensive** ($0.389) despite not producing noticeably better output than opus/baseline ($0.292). The "full" toolset seems to cause more exploratory tool calls that inflate input tokens (67K vs 49K) without quality gains.
- **sonnet/mcp-only is remarkably cheap** ($0.118) and produces better output than sonnet/baseline ($0.272) — half the cost, better quality. The baseline approach appears to waste tokens on broader file reads.
- **MCP-only consistently beats baseline on cost** for both models, suggesting semantic search is an efficient way to locate relevant code for targeted questions like this.
- **The "full" configuration hurts efficiency** in both models compared to mcp-only, adding cost without proportional quality improvement.

**Recommendation:** For factual code-lookup questions like this, **opus/mcp-only** is the clear winner — best speed, lowest cost, and quality that's close to the top. Reserve baseline/full configurations for questions requiring deep cross-file analysis where broader exploration justifies the cost.
