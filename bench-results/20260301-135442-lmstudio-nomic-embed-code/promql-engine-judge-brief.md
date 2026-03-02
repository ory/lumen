## Content Quality

1. **opus/mcp-full** — Most comprehensive and best-structured: includes a
   detailed AST node table with example PromQL and key fields, safety function
   sets (AtModifierUnsafeFunctions, AnchoredSafeFunctions), VectorMatching
   struct, Alert struct fields, and a clear evaluator methods table with line
   references.
2. **opus/mcp-only** — Nearly as thorough, uniquely covers safety sets and the
   three core AST interfaces (Node/Statement/Expr), with good detail on the
   StepInvariantExpr preprocessing optimization and error handling via panics;
   slightly less polished tables than mcp-full.
3. **opus/baseline** — Well-organized with correct line references, good
   coverage of EvalNodeHelper and the alert state machine including
   keepFiringFor, but lacks the safety sets and VectorMatching struct details.
4. **sonnet/mcp-only** — Clear three-step lifecycle explanation, correctly
   identifies EngineQueryFunc and rule group scheduling, but less precise on
   struct fields and misses unique details like safety sets.
5. **sonnet/mcp-full** — Similar depth to sonnet/mcp-only with good alert state
   machine coverage, but no meaningfully new information for 2.5x the cost.
6. **sonnet/baseline** — Solid coverage including the AST Walk/Visitor (unique
   detail), but least precise on internal struct fields and evaluation paths
   compared to others.

## Efficiency

opus/mcp-only is the standout: it delivers the second-best answer at the
**lowest cost ($0.67)** and **fastest runtime (72s)** — 10x cheaper than
opus/baseline and nearly half the cost of sonnet/baseline, while producing a
higher-quality answer than both. sonnet/mcp-full is the worst value proposition
at $3.41 for quality comparable to sonnet/mcp-only at $1.35.

## Verdict

**Winner: opus/mcp-only**
