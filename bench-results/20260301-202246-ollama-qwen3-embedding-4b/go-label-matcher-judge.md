Now I have the ground truth. Here's my evaluation:

## Content Quality

### Ranking: Best to Worst

**1. opus / mcp-only** — Excellent. All code snippets are verbatim correct with accurate line references (21-30, 46-53, 55-70, 72-79). Shows both `NewMatcher` and `MustNewMatcher` in full. Concise summary paragraph at the end ties it together. No errors, no hallucinations, no unnecessary extras.

**2. sonnet / mcp-only** — Very good. Correct code, accurate line references, includes the `matchTypeToStr` mapping and a helpful table of operators. Shows both constructors. Slightly more verbose than needed but all information is accurate. The mention of duplication in `matcher.go` is a reasonable observation (there is a `matcher.go` fixture too). Minor nit: line reference "21-37" for the MatchType section is slightly broad but not wrong.

**3. sonnet / mcp-full** — Correct and well-organized. Accurate line references, shows all key definitions. More concise than mcp-only variant. Only mentions `MustNewMatcher` signature without the body, which is fine for the question asked. Clean formatting.

**4. opus / mcp-full** — Correct, clean table format for the types. Accurate line references. Slightly less complete in that it only describes `NewMatcher` rather than showing the full code, but the question asked to "show" it. Still, the description is accurate and the summary is precise. Compact and efficient.

**5. opus / baseline** — Correct content and line references. Correctly identifies the source as `testdata/fixtures/go/labels_matcher.go`. Adds bonus info about the `Matches` method (line 108) which wasn't asked but is contextually useful. Only shows the function signature for `NewMatcher` rather than the full body, which partially misses the "show the constructor" request.

**6. sonnet / baseline** — Correct but has a minor line reference issue: says `MatchType` starts at line 22 when it's actually line 22 for the type declaration but line 25-30 for the const block. Shows `NewMatcher` starting at line 56 (correct). Claims `Matcher` struct is at lines 47-53 (correct). Mentions `labels_regexp.go:53` for `NewFastRegexMatcher` which adds useful context. The mention of duplication in `matcher.go` is reasonable. Overall solid but the line-56 start for `NewMatcher` is actually line 55 (the comment). Minor inaccuracy.

All six answers are fundamentally correct — no hallucinations of types or incorrect code. The differences are mainly in presentation, completeness of code shown, and precision of line numbers.

## Efficiency Analysis

| Run | Duration | Cost | Quality Rank |
|-----|----------|------|-------------|
| opus / mcp-only | 11.5s | $0.107 | 1st |
| sonnet / mcp-only | 13.6s | $0.115 | 2nd |
| sonnet / mcp-full | 14.4s | $0.277 | 3rd |
| opus / mcp-full | 15.9s | $0.264 | 4th |
| opus / baseline | 16.9s | $0.259 | 5th |
| sonnet / baseline | 49.9s | $0.749 | 6th |

**Key observations:**

- **MCP-only is the clear winner** on efficiency. Both models achieved their best cost and speed in mcp-only mode, with zero cache reads (fresh context) and still came in under $0.12. The semantic search index is highly effective for this type of "find the definition" question.

- **sonnet / baseline is a dramatic outlier** at $0.75 and 50 seconds — 7× more expensive than the best run. The 28K cache-read tokens suggest it explored extensively before finding the answer. This is a case where semantic search completely dominates keyword/file-walking approaches.

- **mcp-full provides no benefit over mcp-only** for this question type. The extra context from CLAUDE.md and tooling roughly doubled the cost (~$0.27 vs ~$0.11) without improving answer quality. The mcp-full variants actually ranked lower than their mcp-only counterparts.

- **opus vs sonnet**: Nearly identical quality and cost in mcp-only mode. opus was slightly faster (11.5s vs 13.6s) and slightly cheaper. In baseline mode, opus was dramatically more efficient than sonnet ($0.26 vs $0.75), suggesting opus is better at directed searching without semantic search assistance.

**Recommendation:** For factual code lookup questions, **mcp-only** with either model offers the best quality-to-cost ratio — top-tier answers at ~$0.11. The opus/mcp-only combination is the overall winner: fastest, cheapest, and highest quality.
