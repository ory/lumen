# Benchmark Analysis Report

**Run:** `swe-20260308-232522-ollama-jina-embeddings-v2-base-code`
**Date:** 2026-03-09
**Embedding model:** `ordis/jina-embeddings-v2-base-code`
**Claude model:** Sonnet (medium effort)
**Tasks:** 5 active (go-easy, go-medium, go-hard, php-easy, php-medium), 1 skipped (php-hard)
**Runs per scenario:** 5

---

## 8.1 Executive Summary

### Search Usage Rate: 4% (1 of 25 with-lumen runs used semantic search)

The single most critical finding in this benchmark is that **Claude almost never
invokes `mcp__lumen__semantic_search`**. Out of 25 with-lumen runs across 5
tasks, only 1 run (go-hard-with-lumen-run3) called semantic search. That single
call returned an empty result, and the run produced no patch (rated Poor).

Because search was almost never used, this benchmark **cannot evaluate chunker
quality or search ranking** in any meaningful way. The primary problem is
upstream of the chunker: Claude is not being sufficiently prompted or compelled
to use the semantic search tool.

### Overall Ratings (5 runs per task, median used for aggregation)

| Task | Lang | Expected Files | Gold Functions | baseline (P/G/Poor) | with-lumen (P/G/Poor) |
|------|------|----------------|----------------|---------------------|----------------------|
| go-easy | go | middleware.go | getAllMethodsForRoute | 0/2/3 | 0/3/2 |
| go-medium | go | regexp.go, old_test.go | routeRegexpGroup.setMatch | 0/5/0 | 5/0/0 |
| go-hard | go | internal/states/state_equal.go | State.RootOutputValuesEqual | 3/0/2 | 4/0/1 |
| php-easy | php | .../HasAttributes.php | setClassCastableAttribute | 5/0/0 | 5/0/0 |
| php-medium | php | .../FreshCommand.php | handle, repositoryExists | 0/5/0 | 0/5/0 |

**Aggregate (5 tasks x 5 runs = 25 runs per scenario):**

| Scenario | Perfect | Good | Poor |
|----------|---------|------|------|
| baseline | 8 | 12 | 5 |
| with-lumen | 14 | 8 | 3 |

With-lumen shows a modest improvement (+6 Perfect, -4 Good, -2 Poor), but this
improvement is likely due to noise/variance rather than Lumen search, since
Lumen search was used in only 1 of 25 runs.

### Top 3 Most Impactful Findings

1. **F-1: Claude does not use semantic search** -- The SessionStart hook and
   PreToolUse hook are insufficient to drive search adoption. This completely
   undermines the benchmark's ability to evaluate search quality.

2. **F-2: Empty search result on the one search call** -- When Claude did call
   semantic search (go-hard-with-lumen-run3), the result was empty. The search
   returned no chunks, and the run failed.

3. **F-3: Issue descriptions are too specific** -- The benchmark tasks include
   exact file paths and function names in the issue body, making Grep/Read the
   obvious tool choice. Claude can solve these tasks without search.

### Recommended Priority Order

1. Fix Claude's search adoption rate (hooks, system prompt, tool positioning)
2. Diagnose why the go-hard search returned empty results
3. Add tasks where the issue description is vague enough to require search
4. Re-run benchmarks after fixing adoption rate to get meaningful chunker data

---

## 8.2 Detailed Findings

#### Finding F-1: Claude does not use Lumen semantic search

- **Category**: QUERY_MISMATCH (misclassified -- this is a tool adoption problem)
- **Task(s)**: All 5 tasks, all 25 with-lumen runs
- **Search query**: N/A -- no search was performed in 24/25 runs
- **Expected**: Claude should use semantic_search as first tool for code discovery
- **Actual**: Claude used Read, Grep, Glob, Bash, Edit directly, never calling search
- **Root cause**: The SessionStart hook injects an `EXTREMELY_IMPORTANT` directive
  telling Claude to use semantic search first, but Claude ignores it. Several
  factors contribute:
  1. The issue descriptions in the benchmark tasks are highly specific -- they
     name exact functions (e.g., `setClassCastableAttribute`, `getAllMethodsForRoute`,
     `repositoryExists()`), making Grep the rational tool choice.
  2. The `--effort medium` flag may cause Claude to take shortcuts and skip the
     hook directives.
  3. The PreToolUse hook only fires for patterns that `looksLikeNaturalLanguage()`
     (>40 chars, multi-word, mostly alphabetic). Most Grep patterns in these
     tasks are exact function names like `setClassCastableAttribute` which are
     single-word and short, so the hook approves them silently.
  4. The `--dangerously-skip-permissions` flag may affect how deferred tools
     (ToolSearch for MCP tools) are loaded. In 24/25 runs, Claude never called
     `ToolSearch` to load `mcp__lumen__semantic_search`.
- **Recommendation**:
  1. Make the SessionStart hook more aggressive -- include an explicit instruction
     to call `ToolSearch` to load the semantic search tool at the start of every
     session.
  2. Lower the `looksLikeNaturalLanguage` threshold from 40 to 20 characters, or
     add a heuristic that fires when the Grep pattern looks like a function/class
     name (CamelCase, snake_case).
  3. Consider a `PostToolUse` hook that reminds Claude about semantic search
     after the first Grep/Read call.
  4. Design benchmark tasks where the issue description does NOT name the exact
     file or function -- force the agent to discover it.
- **Language expert assessment**: general-purpose
- **Impact**: **CRITICAL** -- Without search adoption, no chunker improvements
  can be measured. This is the blocking issue.

#### Finding F-2: Empty search result for go-hard task

- **Category**: MISS (search returned no results)
- **Task(s)**: go-hard (go-hard-with-lumen-run3)
- **Search query**: `refresh-only plan detect output value changes applyable`
- **Expected**: Should return `internal/states/state_equal.go` with
  `State.RootOutputValuesEqual` function (the gold patch fixes a typo where
  `s2.RootOutputValues` should be `s.RootOutputValues`)
- **Actual**: Empty result -- no chunks returned at all
- **Root cause**: Multiple possible causes:
  1. The search path was `/private/var/folders/.../bench-swe-1740176424/repo`
     which is the terraform repo. The index may have failed to build or the
     embedding server may not have been ready.
  2. The stderr log for this run is empty (0 bytes), so there's no diagnostic
     information about indexing failures.
  3. The query `refresh-only plan detect output value changes applyable` is
     semantically reasonable but uses domain terminology (terraform refresh-only
     plan) that might not match the code's vocabulary (`RootOutputValuesEqual`,
     `State.Equal`). The gold function is about equality comparison, not
     "detecting changes" per se.
  4. The `min_score` default of 0.5 may have filtered out low-scoring but
     relevant results. The jina-embeddings model may score terraform domain
     queries poorly against generic Go equality-check code.
- **Recommendation**:
  1. Add diagnostic logging to the search result (e.g., log when 0 results are
     returned, include the number of indexed chunks and the top score even if
     below threshold).
  2. Lower default `min_score` from 0.5 to 0.3 or add a fallback that returns
     top-K results regardless of score when the initial query returns empty.
  3. Add index health checks before search (verify the index exists and has
     chunks).
- **Language expert assessment**: needs-investigation (could be indexing failure
  rather than chunker/embedding issue)
- **Impact**: **HIGH** -- Empty results erode trust in the tool and cause Claude
  to abandon it in future turns.

#### Finding F-3: Benchmark task design biases against semantic search

- **Category**: QUERY_MISMATCH (systemic)
- **Task(s)**: All tasks
- **Search query**: N/A
- **Expected**: Tasks should require code discovery
- **Actual**: All 5 task issue descriptions contain specific enough information
  for Claude to locate the relevant code without search:
  - go-easy: mentions `CORSMethodMiddleware`, `getAllMethodsForRoute`
  - go-medium: mentions `mux.Vars`, host matching, `regexp.go` (implied)
  - go-hard: mentions `terraform plan -refresh-only`, output values
  - php-easy: names `setClassCastableAttribute` and the exact file path
  - php-medium: names `repositoryExists()`, `FreshCommand.php`, `migrate:fresh`
- **Root cause**: SWE-bench tasks are designed with detailed issue descriptions.
  Real-world usage of Lumen would involve vaguer queries like "users report CORS
  headers are wrong" or "migration command crashes on new databases".
- **Recommendation**: Add a second tier of benchmark tasks with intentionally
  vague issue descriptions that strip out function/file names. Example: instead
  of "setClassCastableAttribute uses array_merge which reindexes keys", use
  "custom casts on Eloquent models lose their integer keys after setting".
- **Language expert assessment**: general-purpose
- **Impact**: **HIGH** -- The benchmark cannot measure search quality improvements
  until tasks require search.

#### Finding F-4: ToolSearch deferred loading barrier

- **Category**: QUERY_MISMATCH (infrastructure)
- **Task(s)**: All with-lumen runs
- **Search query**: N/A
- **Expected**: Claude loads and uses `mcp__lumen__semantic_search`
- **Actual**: In 24/25 runs, Claude never calls `ToolSearch` to load the MCP
  tool. In run go-hard-with-lumen-run3 (the one that used search), the first
  tool call was `ToolSearch` with query `select:mcp__lumen__semantic_search`.
- **Root cause**: The deferred tool loading mechanism requires Claude to
  proactively call `ToolSearch` before it can use MCP tools. The SessionStart
  hook tells Claude to use semantic search, but does not explicitly instruct it
  to first call `ToolSearch` to load the tool. Claude may not realize the tool
  needs loading.
- **Recommendation**: Update the SessionStart hook to explicitly instruct:
  "Before using semantic search, you MUST first call `ToolSearch` with query
  `select:mcp__lumen__semantic_search` to load the tool."
- **Language expert assessment**: general-purpose
- **Impact**: **CRITICAL** -- This is likely the primary technical cause of the
  0% usage rate. If Claude does not know it needs to load the tool first, it
  cannot use it.

#### Finding F-5: PreToolUse hook threshold too conservative

- **Category**: QUERY_MISMATCH
- **Task(s)**: All with-lumen runs
- **Search query**: N/A
- **Expected**: PreToolUse hook should suggest semantic search for code discovery
  Grep patterns
- **Actual**: The hook's `looksLikeNaturalLanguage` function requires >40
  characters, multi-word, and >70% alphabetic. Typical Grep patterns in these
  runs are exact identifiers like `getAllMethodsForRoute`, `setClassCastableAttribute`,
  `CORSMethodMiddleware` -- all single-word, so the hook approves them.
- **Root cause**: `looksLikeNaturalLanguage` in `cmd/hook.go:182-207` uses
  `strings.Contains(pattern, " ")` as first check, rejecting any single-word
  pattern. But searching for a function name by its identifier is precisely
  when semantic search could help (finding the definition, not just mentions).
- **Recommendation**: Add a `looksLikeFunctionSearch` heuristic that detects
  CamelCase or snake_case identifiers and suggests semantic search for "finding
  where this is defined". Or use a broader trigger: any Grep pattern that does
  not contain regex metacharacters and is longer than 10 characters.
- **Language expert assessment**: general-purpose
- **Impact**: **MEDIUM** -- Would help in cases where Claude does search for
  identifiers via Grep, but does not address the ToolSearch loading barrier.

---

## 8.3 Conversation Flow Analysis

### Tool Call Counts (averaged across 5 runs per scenario)

| Task | Scenario | Avg Tool Calls | Search Calls | Avg Grep | Avg Read | Found Gold? | Median Rating |
|------|----------|---------------|--------------|----------|----------|-------------|---------------|
| go-easy | baseline | 22.4 | 0 | 10.2 | 4.0 | Yes (all runs) | Poor |
| go-easy | with-lumen | 11.4 | 0 | 3.8 | 3.2 | Yes (all runs) | Good |
| go-medium | baseline | 3.8 | 0 | 1.2 | 1.6 | Yes (all runs) | Good |
| go-medium | with-lumen | 3.0 | 0 | 0.0 | 2.0 | Yes (all runs) | Perfect |
| go-hard | baseline | 13.8 | 0 | 4.8 | 3.2 | 3/5 runs | Perfect |
| go-hard | with-lumen | 2.6 | 0.2 | 0.0 | 1.6 | 4/5 runs | Perfect |
| php-easy | baseline | 3.6 | 0 | 0.4 | 1.6 | Yes (all runs) | Perfect |
| php-easy | with-lumen | 3.4 | 0 | 0.2 | 1.4 | Yes (all runs) | Perfect |
| php-medium | baseline | 5.2 | 0 | 1.6 | 1.2 | Yes (all runs) | Good |
| php-medium | with-lumen | 4.6 | 0 | 0.6 | 1.4 | Yes (all runs) | Good |

### Key Observations

1. **With-lumen runs use fewer tools overall** -- Despite not using semantic
   search, with-lumen runs consistently use fewer Grep calls. This may be an
   artifact of the SessionStart hook's general instruction to "stop and ask if
   you know the exact literal string" before Grepping, which causes Claude to
   skip Grep and go directly to Read/Edit when it has enough information from
   the issue description.

2. **go-easy is the hardest task** -- Both scenarios struggle with go-easy
   (CORS middleware bug). The issue requires understanding that `route.Match()`
   should replace the matcher-iteration pattern AND that `ErrMethodMismatch`
   must be handled. Claude frequently uses the simpler `route.regexp.path.Match`
   which is incorrect.

3. **go-medium shows with-lumen advantage** -- All 5 with-lumen runs got
   Perfect while all 5 baseline runs got Good. But this is NOT due to search
   (0 search calls). The with-lumen runs used fewer tools (3.0 vs 3.8 avg),
   suggesting the hook instructions caused Claude to be more focused and
   direct.

4. **Compensation patterns** -- In baseline runs, Claude's typical flow is:
   Grep (find function) -> Read (read file) -> Edit (make change) -> Bash
   (run tests). In with-lumen runs, the flow is similar but shorter:
   Read (direct file access) -> Edit -> Bash. Claude skips the Grep discovery
   step because the issue descriptions are specific enough.

5. **php tasks are solved identically** -- Both php-easy and php-medium show
   no difference between scenarios. The tasks are straightforward enough that
   Claude solves them in 3-5 tool calls regardless of Lumen availability.

---

## 8.4 Priority Matrix

| Priority | Finding | Category | Impact | Effort | Languages |
|----------|---------|----------|--------|--------|-----------|
| P0 | F-4: ToolSearch barrier | QUERY_MISMATCH | CRITICAL | LOW | All |
| P0 | F-1: Claude ignores search | QUERY_MISMATCH | CRITICAL | MEDIUM | All |
| P1 | F-2: Empty search results | MISS | HIGH | MEDIUM | Go |
| P1 | F-3: Task design bias | QUERY_MISMATCH | HIGH | HIGH | All |
| P2 | F-5: PreToolUse threshold | QUERY_MISMATCH | MEDIUM | LOW | All |

### Recommended Action Plan

**Phase 1 (Immediate -- unblocks all measurement):**
- Update SessionStart hook to include explicit `ToolSearch` instruction
- Test with 1 task to verify >80% search adoption rate before re-running full
  benchmark

**Phase 2 (After search adoption is fixed):**
- Investigate empty search results (add diagnostic logging, lower min_score)
- Re-run benchmark to get actual chunker/search quality data
- Analyze new results to identify real MISSING_NODE, WRONG_BOUNDARY, etc. issues

**Phase 3 (Task improvement):**
- Add "vague description" variants of existing tasks
- Add tasks for languages beyond Go and PHP
- Add tasks that target code patterns known to be difficult for chunkers
  (anonymous functions, middleware chains, decorator patterns)

---

## 8.5 Self-Review Notes

1. **JSONL parsing**: The `bench-swe extract` tool correctly handles the JSONL
   format and finds tool calls. The 0 search calls finding is real, not a
   parsing artifact. Verified by cross-referencing: files with significant token
   usage (e.g., go-easy-with-lumen-run1 at $1.07) show 24 tool calls but 0
   search calls -- Claude is doing extensive work, just not with Lumen.

2. **Missing Phase 4 analysis**: Because search was used only once (with empty
   results), there is insufficient data to diagnose chunk quality issues
   (MISSING_NODE, WRONG_BOUNDARY, etc.). The chunker code review in Phase 4/6
   identified the following patterns for future reference:
   - Go chunker (`goast.go`) captures FuncDecl and GenDecl, which covers the
     gold patch functions (getAllMethodsForRoute, RootOutputValuesEqual,
     setMatch). No MISSING_NODE issues expected for these tasks.
   - PHP chunker captures `method_declaration` and `class_declaration`, which
     covers `setClassCastableAttribute` and `handle`/`repositoryExists`. PHP
     trait methods (`HasAttributes` is a trait) are captured via
     `trait_declaration` and `method_declaration` -- no gaps expected.

3. **Issue categories**: The categories defined in the agent spec are
   comprehensive for chunker/search issues, but this benchmark's dominant
   finding (Claude not using the tool at all) does not fit neatly into any
   category. Consider adding a `TOOL_ADOPTION` or `INTEGRATION_FAILURE`
   category for cases where the tool is available but never invoked.

4. **Phase ordering**: The current phase ordering is sound. However, Phase 6
   (language expert subagent) and Phase 7 (self-review subagent) add limited
   value when the primary finding is a tool adoption problem rather than a
   chunker quality problem. These phases would become essential after fixing
   the adoption issue and re-running.

5. **Gold patches not on disk**: The task JSON files reference
   `gold_patch_file` paths (e.g., `patches/go-easy.patch`) that do not exist in
   the `bench-swe/tasks/` directory. This is not a problem for the analysis since
   the actual patches are captured in the results directory, but the agent
   definition should note this. The gold patches may be generated at runtime from
   `base_commit` and `fix_commit` diffs.

6. **Run count**: This benchmark has 5 runs per scenario (50 total runs for 5
   tasks x 2 scenarios). The summary-report.md shows aggregated ratings using a
   "Good" label that corresponds to the median or mode across runs, marked with
   a dagger (indicating aggregation). The detail-report.md has per-run
   breakdowns.

---

## Appendix A: Per-Run Detail

### go-easy (gorilla/mux CORS middleware)

**Gold patch:** Replace matcher-iteration + `routeRegexp.Match()` with
`route.Match()` + `ErrMethodMismatch` handling in `getAllMethodsForRoute()`
in `middleware.go`.

| Run | baseline Rating | with-lumen Rating | baseline Tools | with-lumen Tools | Search |
|-----|----------------|-------------------|----------------|------------------|--------|
| 1 | Poor | Poor | 24 | 24 | 0 |
| 2 | Good | Poor | 6 | 17 | 0 |
| 3 | Poor | Good | 34 | 7 | 0 |
| 4 | Poor | Good | 35 | 3 | 0 |
| 5 | Good | Good | 13 | 6 | 0 |

### go-medium (gorilla/mux host variable extraction)

**Gold patch:** Strip port from host when `wildcardHostPort` is true in
`routeRegexpGroup.setMatch()` in `regexp.go`.

| Run | baseline Rating | with-lumen Rating | baseline Tools | with-lumen Tools | Search |
|-----|----------------|-------------------|----------------|------------------|--------|
| 1 | Good | Perfect | 4 | 3 | 0 |
| 2 | Good | Perfect | 3 | 3 | 0 |
| 3 | Good | Perfect | 3 | 3 | 0 |
| 4 | Good | Perfect | 6 | 3 | 0 |
| 5 | Good | Perfect | 3 | 3 | 0 |

### go-hard (terraform refresh-only output values)

**Gold patch:** Fix typo `s2.RootOutputValues` -> `s.RootOutputValues` in
`State.RootOutputValuesEqual()` in `internal/states/state_equal.go`.

| Run | baseline Rating | with-lumen Rating | baseline Tools | with-lumen Tools | Search |
|-----|----------------|-------------------|----------------|------------------|--------|
| 1 | Poor | Perfect | 34 | 3 | 0 |
| 2 | Poor | Perfect | 29 | 3 | 0 |
| 3 | Perfect | Poor | 2 | 3 | 1 (empty) |
| 4 | Perfect | Perfect | 2 | 2 | 0 |
| 5 | Perfect | Perfect | 2 | 2 | 0 |

### php-easy (Laravel array_merge key reindexing)

**Gold patch:** Replace `array_merge` with `array_replace` in
`setClassCastableAttribute()` in `HasAttributes.php`.

| Run | baseline Rating | with-lumen Rating | baseline Tools | with-lumen Tools | Search |
|-----|----------------|-------------------|----------------|------------------|--------|
| 1 | Perfect | Perfect | 3 | 3 | 0 |
| 2 | Perfect | Perfect | 4 | 4 | 0 |
| 3 | Perfect | Perfect | 4 | 3 | 0 |
| 4 | Perfect | Perfect | 4 | 3 | 0 |
| 5 | Perfect | Perfect | 3 | 4 | 0 |

### php-medium (Laravel migrate:fresh missing database)

**Gold patch:** Wrap `repositoryExists()` in try/catch for `QueryException` in
`FreshCommand.php`.

| Run | baseline Rating | with-lumen Rating | baseline Tools | with-lumen Tools | Search |
|-----|----------------|-------------------|----------------|------------------|--------|
| 1 | Good | Good | 8 | 5 | 0 |
| 2 | Good | Good | 4 | 5 | 0 |
| 3 | Good | Good | 7 | 4 | 0 |
| 4 | Good | Good | 3 | 4 | 0 |
| 5 | Good | Good | 4 | 5 | 0 |

---

## Appendix B: Chunker Code Assessment (for future reference)

These observations will become actionable once the search adoption problem is
resolved:

### Go Chunker (goast.go)
- Captures `FuncDecl` (functions, methods with receiver) and `GenDecl`
  (types, consts, vars)
- Method symbols use `ReceiverType.MethodName` format (e.g., `State.Equal`)
- Doc comments are included via `declRange` helper
- **Potential gap**: Does not capture `init()` functions with special
  significance, anonymous function literals, or closure variables

### PHP Chunker (tree-sitter)
- Captures `function_definition`, `class_declaration`, `method_declaration`,
  `trait_declaration`, `interface_declaration`
- **Potential gap**: Does not capture anonymous classes, closures, or
  middleware-style function chains common in Laravel
- **Potential gap**: For traits like `HasAttributes`, individual methods are
  captured but may not carry the trait name as context in the symbol name
  (symbol would be `setClassCastableAttribute` not
  `HasAttributes::setClassCastableAttribute`)

### Split Logic (index/split.go)
- Chunks exceeding `LUMEN_MAX_CHUNK_TOKENS` (default 512) are split at line
  boundaries with overlap
- **Potential issue for large PHP methods**: `HasAttributes` trait methods in
  Laravel can be 100+ lines. If `setClassCastableAttribute` is split, the
  relevant `array_merge` call at line 1119 may end up in a chunk with a generic
  symbol name like `setClassCastableAttribute (part 2)`
