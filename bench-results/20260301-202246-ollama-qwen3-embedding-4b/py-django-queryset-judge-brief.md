## Content Quality

1. **opus/baseline** — Most comprehensive: covers Manager descriptor, ManagerDescriptor blocking instance access, all evaluation triggers (including `exists()` and `count()` shortcuts), set operations (`&`, `|`, `^`), and the full Query class API surface with method signatures. Excellent file/line references throughout.

2. **opus/mcp-full** — Nearly as thorough as opus/baseline, with clear structure, accurate code quotes, and good line references. Covers the deferred filter mechanism and iterable classes well. Slightly less coverage of Query class methods and set operations.

3. **opus/mcp-only** — Very strong coverage with accurate code and references. Includes the ManagerDescriptor detail and `__getitem__` slicing logic. Slightly less polished organization than mcp-full but equally correct.

4. **sonnet/mcp-full** — Solid and well-structured with accurate code extractions and line references. Covers all major components but lacks the Query class method catalog and set operations found in opus answers.

5. **sonnet/baseline** — Good coverage with accurate code and a clean end-to-end trace. Includes the iterable classes table and Q object composition. Missing some depth on the Query class internals and `__getitem__` behavior.

6. **sonnet/mcp-only** — Accurate and well-organized but slightly less detailed than sonnet/baseline on iterable classes and evaluation triggers. The full call-stack summary at the end is a nice touch.

## Efficiency

The MCP-only runs deliver the best efficiency: sonnet/mcp-only ($0.50, 64s) and opus/mcp-only ($0.53, 74s) cost 55-72% less than their baseline counterparts while producing answers of comparable or near-comparable quality. The baseline runs are dramatically more expensive — opus/baseline at $1.92 is 3.6x the cost of opus/mcp-only for a modest quality improvement. The mcp-full runs sit in between, adding cache read overhead without meaningfully improving over mcp-only.

## Verdict

**Winner: opus/mcp-only**
