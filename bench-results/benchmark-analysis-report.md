# Benchmark Analysis Report

Generated: 2026-03-11 | Embedding Model: `ordis/jina-embeddings-v2-base-code` | Agent Model: `sonnet`

## 8.1 Executive Summary

### Important Correction

The specified results directory `swe-20260311-104032-ollama-jina-embeddings-v2-base-code` does not exist on disk. It was likely a Ruby benchmark run that failed or was cleaned up. This analysis covers all 6 available benchmark runs instead.

**Contrary to the premise that "with-lumen is slower and more expensive," the data shows the opposite.** With-lumen is cheaper and faster in 5 out of 6 tasks. The single exception is C++ where cost is 20% higher but wall time is 14% lower.

### Tasks Analyzed: 6

| Task | Lang | Expected Files | Gold Functions | baseline | with-lumen |
|------|------|---------------|---------------|----------|------------|
| go-hard | Go | decode.go, decode_test.go | `createDecodedNewValue` | Good (files=false) | Good (files=true) |
| javascript-hard | JS | Tokenizer.ts, rules.ts, +2 test files | `list`, `blockquoteBeginRegex` | Perfect | Perfect |
| python-hard | Python | core.py, test_defaults.py, CHANGES.rst | `get_help_record` (line 2800) | Perfect | Perfect |
| cpp-hard | C++ | CMakeLists.txt, fmt-c.h, fmt-c.cc, test_c.c (all new) | N/A (new files) | Good (files=false) | Good (files=false) |
| php-hard | PHP | JsonFormatter.php, JsonFormatterTest.php | `normalizeValue` (Stringable branch) | Good | Good |
| typescript-hard | TS | _parser.ts, parser.test.ts | `parseRawArgs` | Good (files=false) | Good (files=false) |

### Aggregate Performance

| Metric | baseline | with-lumen | Delta |
|--------|----------|------------|-------|
| Total Cost | $2.5829 | $2.5134 | -2.7% (cheaper) |
| Avg Cost | $0.4305 | $0.4189 | -2.7% (cheaper) |
| Avg Time | 184.6s | 139.8s | -24.3% (faster) |
| Avg Output Tokens | 9,235 | 5,849 | -36.6% (fewer) |
| Total Tool Calls | 129 | 148 | +14.7% (more) |
| Perfect ratings | 2/6 | 2/6 | Same |
| Good ratings | 4/6 | 4/6 | Same |
| files_correct=true | 3/6 | 4/6 | +1 (Go improved) |

### Per-Task Cost/Time Comparison

| Task | baseline Cost | with-lumen Cost | Cost Delta | baseline Time | with-lumen Time | Time Delta |
|------|-------------|----------------|-----------|--------------|----------------|-----------|
| go-hard | $0.44 | $0.42 | -4.4% | 199s | 189s | -5.4% |
| javascript-hard | $0.48 | $0.32 | **-32.7%** | 255s | 119s | **-53.2%** |
| python-hard | $0.12 | $0.10 | -19.5% | 43s | 31s | -28.8% |
| cpp-hard | $1.17 | **$1.40** | **+19.6%** | 474s | 410s | -13.6% |
| php-hard | $0.19 | $0.14 | -26.9% | 52s | 34s | -34.0% |
| typescript-hard | $0.19 | $0.14 | -27.1% | 84s | 56s | -33.3% |

### Why C++ is the Only Regression

The C++ task is the sole case where with-lumen costs more ($1.40 vs $1.17). Three factors combine:

1. **First search returned an ERROR**: result size overflow (93,112 characters exceeded Claude's tool result limit), wasting the call entirely
2. **Green-field task**: All 4 expected files are new (fmt-c.h, fmt-c.cc, test_c.c, CMakeLists.txt additions). Search can only find existing code, not code that needs to be written.
3. **More cache reads**: with-lumen read 2.3M cached tokens vs 1.1M baseline. The search results added context that expanded Claude's exploration, leading to 81 tool calls vs 67 baseline.

Despite higher cost, with-lumen was still 64 seconds faster for C++.

### Overall Search Hit Rate

Across 17 semantic search calls:

| Classification | Count | Percentage |
|---------------|-------|------------|
| DIRECT_HIT | 7 | 41% |
| FILE_HIT | 3 | 18% |
| MISS | 5 | 29% (all C++ -- expected, gold files don't exist yet) |
| ERROR | 2 | 12% (1 result overflow, 1 invalid parameter) |

### Top 3 Most Impactful Findings

1. **Result size overflow on large codebases (C++)**: `formatSearchResults()` in `cmd/stdio.go` has no output size budget. When search returns chunks from large C++ header files, the XML output exceeds Claude's tool result limit, causing the entire result to be dumped to a file the agent cannot read inline. This wastes the search call entirely.

2. **MCP schema rejects `n_results` parameter (Python)**: Claude naturally tries to pass `n_results` to control result count, but the tool schema rejects it as an unexpected property. This wastes a search call on a validation error.

3. **Test framework call expressions not indexed (TS/JS)**: `describe()`, `it()`, `test()` blocks in JavaScript/TypeScript test files produce no chunks because the tree-sitter queries only capture `function_declaration`, `class_declaration`, etc. Test cases are invisible to search.

---

## 8.2 Detailed Findings

#### Finding F-1: MCP parameter validation rejects n_results

- **Category**: QUERY_MISMATCH (tool interface issue)
- **Task(s)**: python-hard
- **Search query**: `boolean flag default_map show_default help output secondary opts`
- **Expected**: Search results from src/click/core.py
- **Actual**: `MCP error -32602: invalid params: validating "arguments": validating root: unexpected additional properties ["n_results"]`
- **Root cause**: Claude sent `n_results: 5` as a parameter, but the MCP tool schema for `semantic_search` does not declare this property. Claude naturally expects to control result count as most search APIs support it.
- **Recommendation**: Add an optional `n_results` (or `limit`) parameter to the `semantic_search` MCP tool schema in `cmd/stdio.go`. Default to 10, accept values 1-50.
- **Language expert assessment**: general-purpose
- **Impact**: HIGH -- wastes a tool call and forces a retry in any language

#### Finding F-2: Result size overflow causes file dump instead of inline display

- **Category**: MISSING_CONTEXT (output formatting issue)
- **Task(s)**: cpp-hard
- **Search query**: `C API C binding C interface extern C`
- **Expected**: Relevant chunks from include/fmt/base.h, include/fmt/format.h
- **Actual**: `result (93,112 characters) exceeds maximum allowed tokens. Output has been saved to ~/.claude/projects/.../tool-results/...txt`
- **Root cause**: The `formatSearchResults()` function at `/Users/aeneas/workspace/go/agent-index-go/cmd/stdio.go:725` writes all chunk content without any size cap. The fmt codebase has very large header files (base.h is 2700+ lines). When multiple large chunks are returned, the total XML output exceeds Claude's ~25K token tool result limit.
- **Recommendation**: Add a total output size budget to `formatSearchResults()`. When accumulated output exceeds a threshold (e.g., 30K characters), stop adding chunk content and append a truncation notice. Alternatively, truncate individual chunk snippets to a maximum line count (e.g., 50 lines per chunk) and inform the user they can `Read` the file for full content.
- **Language expert assessment**: general-purpose -- affects any codebase with large files (C/C++ headers, generated code, large Python modules)
- **Impact**: HIGH -- completely wastes a search call and negates the benefit of search

#### Finding F-3: Oversized chunks produce combined symbol names

- **Category**: OVERSIZED_SPLIT
- **Task(s)**: go-hard, javascript-hard
- **Search query**: `decodeValue pointer NullType set nil` (Go), `list item tokenization boundary check` (JS)
- **Expected**: Precise function chunks for `createDecodedNewValue` (Go, lines 989-1010) and `list` method (JS, lines 184-528)
- **Actual**: Go returned a 140-line merged chunk `astNodeType+Decoder.decodeValue+Decoder.createDecodableValue` (score 0.75). JS returned a 344-line chunk `_Tokenizer+list` (score 0.83). Both contained the gold code.
- **Root cause**: In `/Users/aeneas/workspace/go/agent-index-go/internal/index/split.go:28`, `splitOversizedChunks` splits at line boundaries when content exceeds `LUMEN_MAX_CHUNK_TOKENS * 4` characters. The Go AST chunker produces entire-function chunks, and `mergeUndersizedChunks` at line 153 concatenates adjacent small chunks with `+` in the symbol name. The JS `list` method is 344 lines -- far exceeding the 512-token budget.
- **Recommendation**: (1) For sub-chunks created by splitting, preserve the original symbol name rather than combining with `+`. (2) Consider inner-block boundary detection for very large methods.
- **Language expert assessment**: general-purpose -- large functions exist in every language
- **Impact**: MEDIUM -- the gold code was found, just with imprecise boundaries

#### Finding F-4: C++ task is inherently unsearchable (new files)

- **Category**: QUERY_MISMATCH (inherent limitation)
- **Task(s)**: cpp-hard
- **Search query**: `C API C binding C interface extern C`, `vformat vprint type-erased format arguments C API`, etc. (5 searches)
- **Expected**: N/A -- gold patch creates 4 entirely new files
- **Actual**: Search returned relevant existing code (vformat, format_arg types, score 0.74-0.86) that provided useful design context
- **Root cause**: Not a bug. Semantic search cannot find code that does not exist yet. The search results actually helped Claude understand the existing API surface before creating new files.
- **Recommendation**: No chunker change needed. This is inherent to green-field tasks.
- **Language expert assessment**: benchmark-specific
- **Impact**: LOW -- search was helpful despite being classified as MISS

#### Finding F-5: Python search missed test file and CHANGES.rst

- **Category**: SCORE_INVERSION / MISSING_CONTEXT
- **Task(s)**: python-hard
- **Search query**: `boolean flag default show_default help output secondary opts`
- **Expected**: src/click/core.py (found, rank 1, score 0.75), tests/test_defaults.py (not found), CHANGES.rst (not indexed)
- **Actual**: core.py was found with the correct `get_help_record` function
- **Root cause**: CHANGES.rst is not in `supportedExtensions` (by design). The test file's existing functions did not semantically match the query well enough.
- **Recommendation**: No change needed -- Claude achieved Perfect rating despite this miss.
- **Language expert assessment**: general-purpose for test file scoring; benchmark-specific for CHANGES.rst
- **Impact**: LOW

#### Finding F-6: Test framework call expressions not indexed

- **Category**: MISSING_NODE
- **Task(s)**: typescript-hard (test/parser.test.ts not found)
- **Search query**: `--no- prefix negation boolean flag parsing`
- **Expected**: Both src/_parser.ts and test/parser.test.ts
- **Actual**: Only _parser.ts returned (score 0.70)
- **Root cause**: The tree-sitter TypeScript queries in `/Users/aeneas/workspace/go/agent-index-go/internal/chunker/languages.go:66-85` capture `function_declaration`, `class_declaration`, `method_definition`, etc. They do not capture `call_expression` patterns like `describe(...)`, `it(...)`, or `test(...)`. Test files using Vitest/Jest/Mocha produce no chunks for individual test cases; the entire file becomes one large fallback chunk.
- **Recommendation**: Add tree-sitter query patterns for common test framework call expressions in JS/TS:
  ```
  (expression_statement (call_expression function: (identifier) @name (#match? @name "^(describe|it|test)$"))) @decl
  ```
  This would create individual chunks for each `describe` and `it` block.
- **Language expert assessment**: general-purpose -- Jest, Vitest, Mocha, Jasmine all use this pattern. Analogous patterns exist for Ruby (RSpec), Python (pytest), and Go (TestXxx).
- **Impact**: MEDIUM -- test files are common bug-fix targets

#### Finding F-7: Embedding model keyword bias causes score inversions

- **Category**: SCORE_INVERSION
- **Task(s)**: go-hard
- **Search query**: `null node handling default struct values`
- **Expected**: decode.go `createDecodedNewValue` function
- **Actual**: ast/ast.go `Null` function (score 0.70) and encode.go `NullNode` type (score 0.70) ranked above decode.go results
- **Root cause**: The `jina-embeddings-v2-base-code` model weights keyword overlap heavily. "null" and "node" in the query match `ast.Null()` and `NullNode` type definitions literally, while the gold function `createDecodedNewValue` only contains "null" in its body (`node.Type() == ast.NullType`), not its name.
- **Recommendation**: This is an embedding model limitation. Consider: (1) Boosting score for chunks whose body matches query keywords even if the symbol name does not. (2) Evaluating alternative code embedding models.
- **Language expert assessment**: general-purpose -- keyword-biased embeddings affect all languages
- **Impact**: LOW -- Claude found the right file on subsequent, more targeted queries

---

## 8.3 Conversation Flow Analysis

### Comparison Table

| Task | Scenario | Total Tools | Search Calls | Grep/Glob/Read | Found Gold File? | Rating | Cost | Time |
|------|----------|-------------|-------------|----------------|-----------------|--------|------|------|
| go-hard | baseline | 21 | 0 | 3 Grep, 8 Read | Yes | Good | $0.44 | 199s |
| go-hard | with-lumen | 30 | 5 | 3 Grep, 10 Read | Yes | Good (files=true) | $0.42 | 189s |
| javascript-hard | baseline | 18 | 0 | 4 Grep, 2 Read | Yes | Perfect | $0.48 | 255s |
| javascript-hard | with-lumen | 16 | 2 | 1 Grep, 2 Read | Yes | Perfect | $0.32 | 119s |
| python-hard | baseline | 7 | 0 | 2 Grep, 1 Read | Yes | Perfect | $0.12 | 43s |
| python-hard | with-lumen | 5 | 2 (1 error) | 1 Grep, 1 Read | Yes | Perfect | $0.10 | 31s |
| cpp-hard | baseline | 67 | 0 | 1 Grep, 25 Read | Partial | Good | $1.17 | 474s |
| cpp-hard | with-lumen | 81 | 5 (1 error) | 3 Grep, 20 Read | Partial | Good | $1.40 | 410s |
| php-hard | baseline | 10 | 0 | 1 Grep, 4 Read | Yes | Good | $0.19 | 52s |
| php-hard | with-lumen | 7 | 2 | 0 Grep, 3 Read | Yes | Good | $0.14 | 34s |
| typescript-hard | baseline | 6 | 0 | 0 Grep, 2 Read | Yes | Good | $0.19 | 84s |
| typescript-hard | with-lumen | 9 | 1 | 0 Grep, 3 Read | Yes | Good | $0.14 | 56s |

### Narrative Analysis

**Lumen provides clear benefits in 5/6 tasks.** In JavaScript, Python, PHP, TypeScript, and Go, with-lumen consistently reduces both cost (4-33% savings) and time (5-53% savings) while maintaining or improving quality.

**JavaScript shows the most dramatic improvement**: $0.32 vs $0.48 cost (-33%) and 119s vs 255s time (-53%). Lumen's first search directly found the `list` method in Tokenizer.ts (score 0.83), allowing Claude to skip the exploratory Grep phase entirely.

**PHP and TypeScript show the cleanest patterns**: With Lumen, Claude goes straight from semantic search to reading the relevant file to editing. Zero Grep calls needed.

**C++ is the only cost regression** ($1.40 vs $1.17) due to the result overflow error and the inherent unsearchability of green-field tasks. However, with-lumen was still 64 seconds faster.

**Compensation patterns are predictable**: When Lumen search results are good (JS, Python, PHP, TS), Claude uses 0-1 Grep calls vs 1-4 in baseline. When Lumen results error out or are irrelevant (C++), Claude falls back to the same Grep/Read patterns as baseline, making Lumen overhead additive.

**Output token savings are dramatic**: with-lumen averages 5,849 output tokens vs 9,235 for baseline (-37%). This means Claude reasons less and acts more decisively when given good search results.

---

## 8.4 Priority Matrix

| Priority | Finding | Category | Impact | Effort | Languages Affected |
|----------|---------|----------|--------|--------|-------------------|
| P1 | F-2: Cap result output size | MISSING_CONTEXT | HIGH | Low (output formatting) | All (esp. C/C++) |
| P1 | F-1: Add n_results parameter | QUERY_MISMATCH | HIGH | Low (schema change) | All |
| P2 | F-6: Index test framework call expressions | MISSING_NODE | MEDIUM | Medium (new tree-sitter queries) | JS, TS, Ruby |
| P3 | F-3: Improve oversized chunk splitting | OVERSIZED_SPLIT | MEDIUM | High (algorithmic) | All |
| P4 | F-7: Embedding model keyword bias | SCORE_INVERSION | LOW | Very High (model change) | All |
| -- | F-4: New file creation tasks | QUERY_MISMATCH | N/A | N/A (inherent) | All |
| -- | F-5: Non-code files not indexed | MISSING_CONTEXT | LOW | N/A (by design) | Python, Ruby |

### Recommended Fix Order

1. **Add output size budget to `formatSearchResults()`** -- prevents the C++ result overflow. Cap total output at ~30K characters, truncate chunks to ~50 lines, append "Read file for full content" hints. This is a small change in `cmd/stdio.go:725-795`.

2. **Add `n_results` parameter to semantic_search schema** -- add an optional `limit` (or `n_results`) integer parameter to the MCP tool definition. Default 10, range 1-50. Small schema change in `cmd/stdio.go`.

3. **Add test framework patterns to tree-sitter queries** -- add `call_expression` patterns for `describe`, `it`, `test` in JS/TS queries in `internal/chunker/languages.go`. This improves test file discoverability across all JS/TS codebases.

---

## 8.5 Self-Review Notes

1. **Missing benchmark directory**: The user-specified directory `swe-20260311-104032-ollama-jina-embeddings-v2-base-code` does not exist. This may have been a Ruby benchmark run that failed or is still in progress. The modified `patches/ruby-hard.patch` and `tasks/ruby/hard.json` in git status suggest Ruby benchmarking was being set up.

2. **User premise is inverted**: The data does not support "with-lumen is slower and more expensive." It is faster and cheaper in 5/6 cases. The user may be extrapolating from the C++ case or from the missing Ruby run.

3. **Tool call count is misleading**: With-lumen uses more total tool calls (148 vs 129) but fewer output tokens (35K vs 55K). The search calls add tool count but reduce reasoning tokens. Net cost is lower.

4. **Go analyzer cross-validation**: The Go `bench-swe analyze` command confirms the file-level hit rates but reports per-directory across all 10 languages (most showing "0 valid runs" since only 1 language ran per directory). The manual analysis is more precise.

5. **Missing issue category**: The agent definition should add a `TOOL_INTERFACE` category for F-1 and F-2 type issues that are not about chunker quality but about how results are delivered to the agent. These are the highest-impact fixes but do not fit neatly into the existing categories.

6. **C++ cache_read anomaly**: With-lumen C++ reads 2.3M cached tokens vs 1.1M baseline. This is the primary cost driver for the C++ regression. The search results (even when they worked) added large amounts of context from fmt header files, expanding Claude's working set.

---

## Appendix A: Raw Search Call Classifications

### Go-hard (5 search calls)

| # | Query | Classification | Gold File Hit? | Gold Symbol Hit? | Score |
|---|-------|---------------|----------------|-----------------|-------|
| 1 | null node handling default struct values | FILE_HIT | ast/ast.go (wrong file), not decode.go | No | 0.70 |
| 2 | decode null value unmarshal | DIRECT_HIT | decode.go rank 2 | decodeValue yes | 0.56 |
| 3 | default value struct field initialization | DIRECT_HIT | decode.go rank 1 | setDefaultValueIfConflicted | 0.61 |
| 4 | null pointer field decode unmarshal test | FILE_HIT | decode_test.go rank 1 | Wrong test functions | 0.52 |
| 5 | decodeValue pointer NullType set nil | DIRECT_HIT | decode.go rank 1 | decodeValue+createDecodableValue | 0.75 |

### JavaScript-hard (2 search calls)

| # | Query | Classification | Gold File Hit? | Gold Symbol Hit? | Score |
|---|-------|---------------|----------------|-----------------|-------|
| 1 | list item tokenization boundary check... | DIRECT_HIT | Tokenizer.ts rank 1 | list method | 0.83 |
| 2 | fencesBeginRegex headingBeginRegex... | DIRECT_HIT | Tokenizer.ts rank 1 | list method | 0.74 |

### Python-hard (2 search calls)

| # | Query | Classification | Gold File Hit? | Gold Symbol Hit? | Score |
|---|-------|---------------|----------------|-----------------|-------|
| 1 | boolean flag default_map... | ERROR | MCP validation error (n_results) | N/A | N/A |
| 2 | boolean flag default show_default... | DIRECT_HIT | core.py rank 1 | get_help_record | 0.75 |

### C++-hard (5 search calls)

| # | Query | Classification | Gold File Hit? | Gold Symbol Hit? | Score |
|---|-------|---------------|----------------|-----------------|-------|
| 1 | C API C binding C interface extern C | ERROR | Result overflow (93K chars) | N/A | N/A |
| 2 | vformat vprint type-erased format... | MISS | Existing files only (expected) | N/A | 0.86 |
| 3 | format_arg basic_format_args... | MISS | Existing files only (expected) | N/A | 0.79 |
| 4 | format_arg type enum int_value... | MISS | Existing files only (expected) | N/A | 0.78 |
| 5 | basic_format_args get args_ desc_ | MISS | Existing files only (expected) | N/A | 0.74 |

### PHP-hard (2 search calls)

| # | Query | Classification | Gold File Hit? | Gold Symbol Hit? | Score |
|---|-------|---------------|----------------|-----------------|-------|
| 1 | NormalizerFormatter normalizer... | FILE_HIT | Parent class (NormalizerFormatter.php) | normalizeValue found | 0.72 |
| 2 | JsonFormatter normalizer handle objects | DIRECT_HIT | JsonFormatter.php rank 1 | JsonFormatter+normalizeRecord | 0.77 |

### TypeScript-hard (1 search call)

| # | Query | Classification | Gold File Hit? | Gold Symbol Hit? | Score |
|---|-------|---------------|----------------|-----------------|-------|
| 1 | --no- prefix negation boolean flag parsing | DIRECT_HIT | _parser.ts rank 1 | parseRawArgs | 0.70 |
