## Content Quality

**Ranking: opus/mcp-full > opus/mcp-only > sonnet/mcp-full > opus/baseline >
sonnet/mcp-only > sonnet/baseline**

**1. opus/mcp-full** — The most polished and well-structured answer. Correctly
covers all four requested areas (engine, functions, AST, rules) with accurate
file/line references (`engine.go:124-128`, `engine.go:1134-1152`,
`functions.go:2152+`, `alerting.go:380`, etc.). The execution flow is presented
clearly with the right level of detail — it explains _why_ things happen (e.g.,
panic-based error propagation, the `StepInvariantExpr` optimization for
`@ start()`/`@ end()`), not just _what_. Correctly notes that the engine
explicitly disclaims handling alert/recording rules. The `QueryEngine`
interface, `Query` interface, `EvalStmt`, and `FunctionCall` type are all
accurately reproduced. The answer benefits from having both semantic search and
traditional tools to cross-reference findings.

**2. opus/mcp-only** — Nearly as comprehensive as opus/mcp-full. Accurately
identifies all key interfaces (`QueryEngine`, `Query`, `Expr`), the `evaluator`
struct, the `FunctionCalls` registry, and both rule types with correct
signatures. The AST node dispatch table is thorough. One minor issue: the
`MatrixSelector` entry says "Handled within `Call` processing for range vector
functions" which is partially misleading — `matrixSelector` is also called
directly in `eval()` for instant evaluation contexts. Line references are
precise. The `EngineOpts` mention adds useful context not found in some other
answers. Slightly less polished prose than mcp-full but substantively
equivalent.

**3. sonnet/mcp-full** — Covers all four areas competently with good code
snippets. The `eval()` dispatch section is presented as inline Go code with
comments rather than a table, which makes it feel more grounded in the actual
source. Correctly shows the `rangeEval` signature and explains the matrix-arg
vs. non-matrix-arg function dispatch paths. The alert state machine diagram
(`inactive → Pending → Firing`) is a nice touch. File/line references are
present but slightly less precise than the opus answers (e.g., `functions.go:39`
for the FunctionCall type vs. the correct `functions.go:60`). The summary flow
diagram at the end is clean and accurate.

**4. opus/baseline** — Impressive given it had no MCP tools. Covers all four
areas with correct structural understanding. The function registration section
accurately notes that `label_replace` and `label_join` are `nil` in the map and
handled specially. The `rangeEval` signature is correctly reproduced. The rules
section correctly explains the `QueryFunc` closure pattern. However, some line
references may be approximate since they came from baseline knowledge rather
than actual file reads. The answer is slightly more compact than the
MCP-assisted versions but doesn't sacrifice accuracy — a testament to the
model's training data including Prometheus source.

**5. sonnet/mcp-only** — Thorough and well-organized with accurate content.
Covers all areas including the `QueryEngine` interface (which sonnet/baseline
missed). The AST node type table with line references (`ast.go:269-279`) is
helpful. However, the "Rule scheduling" section acknowledges uncertainty ("not
shown in indexed chunks, but architecturally...") which is honest but reveals
incomplete exploration. The `ChildrenIter` mention and the `EvalStmt` struct are
good additions. The summary call chain diagram is accurate and useful.

**6. sonnet/baseline** — The weakest of the six, though still competent. Covers
the core areas but misses the `QueryEngine` and `Query` interfaces entirely,
jumping straight to `Engine.exec()`. The `evaluator` struct snippet is
simplified with `// ...` elisions. The function registry section is accurate.
The rules section correctly identifies `QueryFunc` and the state transitions.
Line references are present but some feel approximate (e.g., `engine.go:345` for
the Engine struct). The data flow diagram at the end is a good summary but the
answer overall feels thinner than the others, particularly on AST node types and
the `rangeEval` mechanism.

## Efficiency Analysis

| Scenario        | Duration | Total Tokens (In+Cache+Out) | Cost  | Quality Rank |
| --------------- | -------- | --------------------------- | ----- | ------------ |
| sonnet/baseline | 133.3s   | 63,738                      | $1.13 | 6th          |
| sonnet/mcp-only | 85.0s    | 198,684                     | $1.09 | 5th          |
| sonnet/mcp-full | 75.3s    | 247,216                     | $0.94 | 3rd          |
| opus/baseline   | 225.7s   | 64,525                      | $6.14 | 4th          |
| opus/mcp-only   | 75.5s    | 121,883                     | $0.69 | 2nd          |
| opus/mcp-full   | 89.0s    | 321,620                     | $1.32 | 1st          |

**Most efficient: opus/mcp-only** — Best cost ($0.69), fastest runtime tied with
sonnet/mcp-full, and second-best quality. Opus with semantic search alone found
the right code quickly without the token overhead of traditional tools. This is
a remarkable result: the most expensive model produced the cheapest run because
MCP search eliminated wasteful exploration.

**Best quality-to-cost: opus/mcp-only** — At $0.69 it's the cheapest run across
all six scenarios while delivering the second-best answer. opus/mcp-full is
marginally better in quality but costs nearly 2x more ($1.32).

**Surprising findings:**

- **opus/baseline is the most expensive by far** ($6.14) — 9x the cost of
  opus/mcp-only for a worse answer. The 225.7s runtime suggests extensive
  context-window-based reasoning to compensate for lack of tools.
- **sonnet/baseline is slower than sonnet/mcp runs** despite using fewer tokens
  — the 133.3s duration vs. 75-85s suggests the baseline approach required more
  sequential reasoning turns.
- **Cache reads dramatically help sonnet** — sonnet/mcp-full had 84K cached
  tokens and was both fastest (75.3s) and cheapest ($0.94) among sonnet runs,
  while producing the best sonnet answer.
- **MCP-only opus used fewer input tokens than MCP-only sonnet** (117K vs 194K)
  — opus found relevant code more efficiently with semantic search, needing
  fewer search iterations.

**Recommendation:** For this type of deep codebase question, **opus/mcp-only**
is the clear winner on cost-efficiency. If absolute quality is paramount and
cost is secondary, opus/mcp-full edges ahead. The baseline approaches are
strictly dominated — they cost more (especially opus) and produce worse results
for this kind of cross-cutting architectural question.
