## Content Quality

**Ranking: opus/baseline > opus/mcp-full > sonnet/mcp-full ≈ sonnet/mcp-only ≈ opus/mcp-only > sonnet/baseline**

**1. opus/baseline** — The most complete answer. Correctly covers all requested elements (Permission class, fields, PermissionManager, get_by_natural_key) with accurate code snippets and explanations. Uniquely goes beyond the question to explain how Permission connects to users via `PermissionsMixin` and `ManyToManyField`, which adds genuine context. File references are precise (`testdata/fixtures/python/django-models.py`, lines 27-36, 39-85, 317). The `__str__` method, CASCADE behavior, and auto-created permissions are all mentioned. Used a straightforward approach reading the file directly.

**2. opus/mcp-full** — Nearly as complete as opus/baseline. Covers all requested elements accurately with correct line references. Includes the connection to `PermissionsMixin` at line 317, matching baseline's breadth. Slightly more concise in the code presentation — it doesn't inline the full Permission model code block, instead using a table for fields. This is a stylistic tradeoff; some users might prefer seeing the actual code. All technical details are correct.

**3. sonnet/mcp-full** — Correct and well-structured. Covers all requested elements with accurate code and line references. The "How it all fits together" section with the arrow diagram showing the natural_key round-trip is a nice touch for comprehension. Mentions auto-created permissions. Doesn't go as deep as opus/baseline on the broader auth system connection, but fully answers the question asked.

**4. sonnet/mcp-only** — Very similar quality to sonnet/mcp-full. All technical details are correct. Includes a good note about the four built-in permission verbs. The "Key design points" section is well-organized. File references use `django-models.py` without the full path, which is slightly less precise. Essentially equivalent to sonnet/mcp-full in content.

**5. opus/mcp-only** — Correct and concise. Covers all requested elements accurately. Adds a useful clarification about object-level permissions being outside the scope of the built-in system. Slightly less detailed in code presentation than the opus/baseline — doesn't show the full model class definition inline. File references are present but use the short form.

**6. sonnet/baseline** — Correct but the least detailed of the group. Covers all the core elements requested but is the most terse in explanations. Doesn't mention the `__str__` method or the connection to the broader auth system. The "In summary" paragraph is useful but brief. Still a solid answer — the gap between all six is relatively small.

## Efficiency Analysis

| Scenario | Duration | Total Input | Output | Cost |
|---|---|---|---|---|
| sonnet/baseline | 54.6s | 28.7K | 871 | $0.987 |
| sonnet/mcp-only | 15.0s | 17.6K | 888 | $0.110 |
| sonnet/mcp-full | 15.9s | 31.1K | 860 | $0.191 |
| opus/baseline | 24.5s | 49.4K | 1136 | $0.297 |
| opus/mcp-only | 16.4s | 17.6K | 833 | $0.109 |
| opus/mcp-full | 18.9s | 44.4K | 805 | $0.263 |

**Key observations:**

- **sonnet/baseline is the outlier on cost** at $0.99 — nearly 9x more expensive than sonnet/mcp-only for marginally worse quality. The 54.6s runtime is also by far the slowest. This is likely due to multiple tool calls reading large files without cache hits.

- **mcp-only is the efficiency winner** for both models. opus/mcp-only ($0.109, 16.4s) and sonnet/mcp-only ($0.110, 15.0s) are nearly identical in cost and speed, with minimal input tokens (17.6K). The semantic search tool returned targeted results without needing the full conversation context.

- **Cache reads dramatically affect cost.** The baseline and mcp-full runs show large cache read columns (28-42K tokens), meaning they're reading substantial file content. The mcp-only runs avoid this entirely.

- **opus/baseline delivers the best quality** at a moderate cost ($0.297) — roughly 3x the mcp-only runs but with noticeably richer content.

- **mcp-full adds cost without proportional quality gain.** Comparing sonnet/mcp-only ($0.110) to sonnet/mcp-full ($0.191), the extra ~$0.08 buys a slightly nicer "fits together" diagram but no meaningful accuracy improvement.

**Recommendation:** For this type of "explain this code" question, **opus/mcp-only** offers the best quality-to-cost tradeoff — good depth at $0.109 and 16.4s. If maximum completeness matters (e.g., understanding how Permission connects to the broader auth system), **opus/baseline** at $0.297 is worth the premium. The **sonnet/baseline** run at $0.987 should be avoided — it's the most expensive with the least detailed answer.
