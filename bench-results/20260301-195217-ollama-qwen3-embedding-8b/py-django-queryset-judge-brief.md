## Content Quality

1. **opus/mcp-full** — Most thorough and well-structured; covers Manager installation via `contribute_to_class` and `ManagerDescriptor.__get__` that others gloss over, includes the deferred filter property, combinator queries, and a comprehensive summary table of chaining methods with their Query mutations. All file:line references are precise.

2. **opus/mcp-only** — Nearly identical coverage to opus/mcp-full with excellent structure and a detailed summary table of classes/signatures at the end; slightly less detail on `contribute_to_class` mechanics but adds the set operations section and has the most complete reference table.

3. **opus/baseline** — Strong coverage with good explanations of `ManagerDescriptor`, the deferred filter property, and iterable class variants; slightly less organized than the MCP runs but still accurate and complete with correct line references.

4. **sonnet/mcp-full** — Solid and accurate with good coverage of the deferred filter optimization and combinator queries; slightly less polished organization and missing the iterable class variant table that other answers include.

5. **sonnet/baseline** — Good coverage with a useful evaluation triggers table and clear end-to-end example; slightly more surface-level on the Query class internals since it presents them as a table rather than explaining the compilation pipeline.

6. **sonnet/mcp-only** — Accurate but the thinnest of the six; covers all major topics but with less depth on Manager internals and fewer concrete line references for the iterable classes.

## Efficiency

The MCP-only runs offer dramatically better cost efficiency: sonnet/mcp-only at $0.49 and opus/mcp-only at $0.53 are 2-3x cheaper than their baseline counterparts while producing comparable or better answers. The mcp-full runs sit in between at $0.68-$0.82. Runtime is comparable across all runs (63-73s) except sonnet/baseline which is an outlier at 162s.

## Verdict

**Winner: opus/mcp-only**
