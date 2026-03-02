## Content Quality

**Ranking: opus/baseline > opus/mcp-full > opus/mcp-only > sonnet/mcp-full > sonnet/mcp-only > sonnet/baseline**

**opus / baseline** — The most complete answer. Correct on all fields, meta constraints, methods, and line references. Uniquely mentions the broader integration: `PermissionsMixin` (line 317) adding `user_permissions` as M2M, and `Group` (line 102) having its own `permissions` M2M. This gives the reader genuine understanding of how Permission fits into Django's auth system, not just the class in isolation. File path includes the `testdata/fixtures/python/` prefix, which is precise. Used more tokens reading surrounding context but leveraged it well.

**opus / mcp-full** — Also highly accurate with correct line references. Mentions `PermissionsMixin` (line 342) and `Group.permissions` (line 121), plus references helper functions `_user_has_perm` (line 261) and `_user_get_permissions` (line 243). This broader context is valuable. One minor nit: the `__str__` example shows `"blog | Can add article"` which is reasonable but the actual format includes the model name in the content_type string representation. Overall excellent.

**opus / mcp-only** — Correct and well-structured. Mentions the `ModelBackend` integration (`django-backends.py:104`) and how permissions are cached as `"app_label.codename"` strings — a useful detail no other answer includes. Slightly less precise on file paths (uses `django-models.py` without the full testdata path). The `__str__` example includes three parts (`"admin | log entry | Can add log entry"`) which is slightly off — `ContentType.__str__` typically returns `"app_label | model"`, making the Permission `__str__` a two-part format.

**sonnet / mcp-full** — Accurate, clean, well-organized. Correctly identifies the full file path `testdata/fixtures/python/django-models.py`. Good summary section explaining how content_type + codename form the unique identity. Doesn't go beyond the Permission class itself to discuss integration with User/Group models, which limits completeness for the "how does it work" part of the question.

**sonnet / mcp-only** — Very similar quality to mcp-full. Includes a nice flow diagram showing the `get_by_natural_key` call chain. Correct on all technical details. Uses `django-models.py` without full path. The multi-db note (`self.db`) is a good detail. Slightly verbose but accurate.

**sonnet / baseline** — Correct but the most concise of the six. Covers all asked-for elements (class, fields, manager, `get_by_natural_key`). File references use `django-models.py` without line numbers in the table. Mentions auto-created permissions. Doesn't discuss integration with User/Group or auth backends. Adequate but least informative.

All six answers are fundamentally correct — no factual errors on the core Permission model. The differentiation comes from completeness of the broader auth system context and precision of references.

## Efficiency Analysis

| Scenario | Duration | Output Tok | Cost |
|----------|----------|------------|------|
| sonnet / mcp-only | 18.1s | 975 | $0.122 |
| opus / mcp-only | 18.4s | 844 | $0.113 |
| sonnet / mcp-full | 16.3s | 807 | $0.191 |
| opus / baseline | 22.9s | 1035 | $0.295 |
| sonnet / baseline | 39.1s | 851 | $0.364 |
| opus / mcp-full | 25.7s | 1032 | $0.390 |

**Most efficient: opus/mcp-only** at $0.113 — lowest cost, fast runtime, and a high-quality answer with backend integration details. Zero cache reads suggest it found the relevant code quickly through semantic search alone.

**Best quality-to-cost: opus/mcp-only.** It produced the 3rd-ranked answer at the lowest cost. The opus/baseline answer is marginally better in quality but costs 2.6× more.

**Surprising findings:**
- **sonnet/baseline is the most expensive** despite producing the least complete answer — 39.1s and $0.364. The high cache read (28K tokens) suggests it read lots of context but didn't synthesize it as effectively.
- **opus/mcp-full is the costliest opus run** ($0.390) with 67K input tokens, yet its answer isn't meaningfully better than opus/mcp-only. The full toolset led to over-reading.
- **mcp-only runs for both models** are consistently the cheapest and fastest. The semantic search alone was sufficient to locate the Permission model code without needing file exploration tools.
- **sonnet/mcp-full** was the fastest at 16.3s but its quality doesn't justify the $0.07 premium over sonnet/mcp-only.

**Recommendation:** For factual code comprehension questions like this, **opus/mcp-only** offers the best tradeoff — top-tier quality at minimum cost. The semantic search tool alone provides enough context for the model to produce a thorough answer. Adding baseline tools or the full toolset increases cost without proportional quality gains.
