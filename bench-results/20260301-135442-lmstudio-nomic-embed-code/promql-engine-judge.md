## Content Quality

**Ranking: opus/mcp-full > opus/mcp-only > sonnet/mcp-only > opus/baseline >
sonnet/mcp-full > sonnet/baseline**

**1. opus/mcp-full** — The most comprehensive and precise answer. It correctly
identifies the Engine struct at `engine.go:345-361`, the evaluator at
`engine.go:1138-1152`, and provides accurate line references throughout.
Uniquely covers `VectorMatching` struct details (`ast.go:309-323`), safety sets
(`AtModifierUnsafeFunctions`, `AnchoredSafeFunctions`, `SmoothedSafeFunctions`),
the `Alert` struct (`alerting.go:84-100`), and a detailed table of AST node
types with line ranges. The evaluator method table (listing `rangeEval`,
`rangeEvalAgg`, `matrixSelector`, `vectorSelectorSingle` with locations) is a
standout. Coverage of the rules layer is thorough, showing both struct
definitions and eval method internals. File/line references are consistently
specific and appear well-grounded in actual code reads.

**2. opus/mcp-only** — Very strong, nearly matching mcp-full in correctness and
structure. Covers the Engine struct, evaluator, AST dispatch, function
registration, and rules thoroughly. Notable for explaining the panic-based error
handling pattern (`evaluator.error()` panics, `evaluator.recover()` catches),
the `StepInvariantExpr` preprocessing optimization (`preprocessExprHelper` at
`engine.go:4246-4341`), and the explicit engine comment about not handling
alert/recording statements. Slightly less detailed on AST node types and missing
some of the struct-level detail that mcp-full provides (e.g., `VectorMatching`,
`Alert` struct). Line references are precise.

**3. sonnet/mcp-only** — Excellent completeness and organization. Correctly
traces the full execution path from `query.Exec` through `execEvalStmt` to
`evaluator.eval`. The explanation of `rangeEval` mechanics (pre-evaluate
sub-expressions into matrices, loop timestamps, gather per-step vectors, call
funcCall) is the clearest of all answers. Correctly identifies special-cased
functions (`label_replace`, `label_join`, `info`) with `nil` entries in the map.
The call chain summary at the end is detailed and accurate. Slightly less
precise on some line numbers compared to opus variants, and the rules section is
thinner (correctly describes `QueryFunc` but infers the type signature rather
than citing a specific location).

**4. opus/baseline** — Solid and accurate despite having no tool access.
Correctly describes all major components: Engine, evaluator, AST nodes, function
registry, binary expression dispatch, and both rule types. The function
registration section is well-done, noting the `EvalNodeHelper` purpose. The
alert state machine description is detailed (including `keepFiringFor` and the
15-minute resolved retention). Line references are present but some appear
interpolated from general knowledge rather than verified reads. Missing some
specifics like the `Engine` struct fields and the preprocessing optimization.

**5. sonnet/mcp-full** — Correct in all major points but surprisingly concise
given it had full tool access. Covers the same ground as others but with less
depth — the AST dispatch table is shorter, the function registration section is
adequate but not exceptional, and the rules section is the briefest of the six
(correctly describes the flow but lacks struct definitions). The
`EvalNodeHelper` mention is good. The "Summary Flow" diagram is clean but
simpler than others. Given the high token/cost usage, the information density is
disappointing.

**6. sonnet/baseline** — Accurate and well-structured but the weakest overall.
Covers all required topics and the end-to-end flow diagram is good. However, it
presents some information less precisely — the `FunctionCall` type signature
location is given as `functions.go:2153-2237` (which is actually the map, not
the type), and the binary expression section uses pseudocode that doesn't
perfectly match the actual code structure. The AST visitor/walk section is a
nice inclusion but less relevant to the evaluation question. Line references are
the least reliable of the six.

---

## Efficiency Analysis

| Scenario        | Duration | Total Input | Cost  | Quality Rank |
| --------------- | -------- | ----------- | ----- | ------------ |
| opus/mcp-only   | 71.9s    | 114.7K      | $0.67 | 2nd          |
| sonnet/baseline | 139.3s   | 62.1K       | $0.82 | 6th          |
| sonnet/mcp-only | 96.2s    | 241.5K      | $1.35 | 3rd          |
| opus/mcp-full   | 161.8s   | 63.0K       | $1.83 | 1st          |
| sonnet/mcp-full | 115.6s   | 955.1K      | $3.41 | 5th          |
| opus/baseline   | 273.2s   | 61.8K       | $7.16 | 4th          |

**Best efficiency: opus/mcp-only** — Fastest wall-clock time (71.9s), lowest
cost ($0.67), and second-best quality. This is the clear winner for
quality-to-cost ratio. It found the right information quickly via semantic
search without needing to read entire files, and Opus's reasoning produced a
thorough, well-organized answer.

**Worst efficiency: opus/baseline** — By far the most expensive ($7.16) and
slowest (273.2s), yet ranked only 4th in quality. Without tools, Opus spent
heavily on reasoning tokens to reconstruct information from training data. The
4x cost premium over opus/mcp-only for inferior output makes this the worst
value proposition.

**Surprising findings:**

1. **sonnet/mcp-full is an anti-pattern.** It consumed 955K total input tokens
   and cost $3.41 — the second most expensive — yet produced the 5th-ranked
   answer. It appears to have read far too many files without effectively
   synthesizing the information. More tool access made Sonnet _less_ efficient
   here.

2. **opus/mcp-only vs opus/mcp-full** — mcp-only was faster (71.9s vs 161.8s)
   and cheaper ($0.67 vs $1.83) but produced slightly less detailed output. The
   mcp-full answer's additional detail (struct definitions, safety sets, AST
   line ranges) may justify the 2.7x cost increase depending on use case.

3. **Cache reads matter.** The baseline and mcp-full runs show ~28K cache read
   tokens, indicating prompt caching. The mcp-only runs show 0 cache reads,
   suggesting different prompt structures. Despite this, opus/mcp-only was still
   cheapest overall.

4. **Sonnet baseline outperformed sonnet/mcp-full on cost-per-quality.** At
   $0.82 for 6th place vs $3.41 for 5th place, the baseline was 4x cheaper for
   comparable quality — suggesting that for Sonnet, heavy file reading actually
   degraded the cost-effectiveness.

**Recommendation:** **opus/mcp-only** is the optimal quality-to-cost tradeoff —
highest quality tier at the lowest cost. For maximum quality regardless of cost,
**opus/mcp-full** at $1.83 is still very reasonable. Avoid the sonnet/mcp-full
configuration, which combines high cost with mediocre output.
