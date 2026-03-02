## Content Quality

1. **opus/baseline** — Most complete: correctly identifies all fields, methods, and meta constraints with accurate line references, and uniquely explains how `PermissionsMixin` and `Group` connect via M2M relationships, giving the fullest picture of the permission system.

2. **opus/mcp-only** — Equally accurate on the core Permission model, adds valuable detail about `ModelBackend` permission checking and caching as `"app_label.codename"` strings, though the `__str__` example is slightly off (includes an extra segment).

3. **opus/mcp-full** — Correct and well-structured, references `_user_has_perm` and `_user_get_permissions` helper functions with line numbers, providing good architectural context; minor issue with paraphrasing the `ForeignKey` signature.

4. **sonnet/mcp-full** — Accurate with proper file path (`testdata/fixtures/python/django-models.py`), good summary section, but stays surface-level compared to opus answers — no mention of how permissions connect to users/groups.

5. **sonnet/mcp-only** — Solid coverage with a nice flow diagram showing the lookup chain; accurate details throughout but lacks the broader system context (backends, M2M relationships).

6. **sonnet/baseline** — Correct and concise but the most minimal; uses generic file reference (`django-models.py:27-36`) and doesn't explore how permissions integrate with the wider auth system.

## Efficiency

The mcp-only runs are dramatically cheaper: sonnet/mcp-only ($0.12) and opus/mcp-only ($0.11) cost 2-3× less than their baseline/mcp-full counterparts while delivering comparable or better quality. Opus/mcp-full is the most expensive at $0.39 with no proportional quality gain over opus/mcp-only. Sonnet/mcp-full offers a good middle ground at $0.19 but doesn't match opus quality.

## Verdict

**Winner: opus/mcp-only**
