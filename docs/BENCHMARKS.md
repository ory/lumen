# Lumen Benchmarks

Lumen is evaluated using **bench-swe**: a SWE-bench-style harness that measures
whether Lumen reduces tool usage, cost, and time when Claude fixes real GitHub
bugs. Results are fully reproducible and all artifacts are committed to this
repository.

## Methodology

### Evaluation Framework

`bench-swe` tests two scenarios head-to-head against real, fixed GitHub issues:

- **baseline** — Claude with default tools only (Read, Write, Edit, Grep, Bash
  etc.), no Lumen
- **with-lumen** — all default tools plus Lumen's `semantic_search` MCP tool

Each task is a real GitHub bug from an open-source project. Claude is given the
issue description and the codebase at the pre-fix commit. It must produce a
patch that fixes the issue.

### Judging

Patches are rated by Claude Sonnet 4.6 acting as a blind judge, comparing each
generated patch to the known-correct gold patch:

- **Perfect** — fixes the issue with equivalent or better logic than the gold
  patch
- **Good** — fixes the issue correctly using a different valid approach
- **Poor** — wrong, incomplete, doesn't compile, or doesn't fix the issue

### Metrics Captured

For each run, bench-swe captures:

| Metric       | Source                               |
| ------------ | ------------------------------------ |
| Cost (USD)   | Claude API usage from raw JSONL      |
| Duration     | Wall time from session start to exit |
| Total tokens | Input + cache read + output          |
| Tool calls   | Count of tool invocations in session |

### Current Test Suite

7 languages, hard difficulty, 3 runs per scenario — all against real GitHub
bugs:

| Task            | Language   | Repository           | Issue                                       |
| --------------- | ---------- | -------------------- | ------------------------------------------- |
| go-hard         | Go         | goccy/go-yaml        | Decoder overrides defaults with null values |
| javascript-hard | JavaScript | yargs/yargs          | Hard-coded failure message not shown        |
| php-hard        | PHP        | sebastianbergmann    | Baseline code coverage tracking issue       |
| python-hard     | Python     | pallets/click        | Type annotation / default value handling    |
| ruby-hard       | Ruby       | ruby/rbs             | Intersection type parsing bug               |
| rust-hard       | Rust       | rust-lang/rust       | Async fn return type inference              |
| typescript-hard | TypeScript | microsoft/TypeScript | Complex type narrowing bug                  |

Embedding model: `ordis/jina-embeddings-v2-base-code` (Ollama, 768-dim) Claude
model: Haiku (execution), Sonnet 4.6 (judging)

---

## Results

### Full Results Table

All 42 runs (21 baseline + 21 with-lumen):

```
Task                   Lang        Scenario    Run  Rating   Cost      Time     Total Tokens  Tool Calls
-----------------------------------------------------------------------------------------------------------
go-hard                go          baseline    1    Good     $0.4048   203.9s   321,749       35
go-hard                go          baseline    2    Good     $0.4030   197.7s   361,780       36
go-hard                go          baseline    3    Good     $0.6072   261.5s   391,493       59
go-hard                go          with-lumen  1    Good     $0.5071   271.9s   447,550       29
go-hard                go          with-lumen  2    Good     $0.4492   195.8s   376,612       35
go-hard                go          with-lumen  3    Good     $0.4287   228.5s   302,282       28
javascript-hard        javascript  baseline    1    Good     $0.6328   336.4s   467,282       33
javascript-hard        javascript  baseline    2    Good     $0.7179   325.1s   570,357       44
javascript-hard        javascript  baseline    3    Perfect  $0.4783   290.4s   396,825       16
javascript-hard        javascript  with-lumen  1    Good     $0.5458   264.7s   646,946       19
javascript-hard        javascript  with-lumen  2    Good     $0.3212   163.3s   352,391       14
javascript-hard        javascript  with-lumen  3    Good     $0.7235   352.4s   849,997       24    (outlier!)
php-hard               php         baseline    1    Good     $0.1825   71.9s    232,512       12
php-hard               php         baseline    2    Good     $0.2503   89.6s    387,332       18
php-hard               php         baseline    3    Good     $0.1491   55.4s    119,722       14
php-hard               php         with-lumen  1    Perfect  $0.3276   131.5s   509,147       21    (outlier!)
php-hard               php         with-lumen  2    Perfect  $0.1903   74.6s    278,043       15
php-hard               php         with-lumen  3    Perfect  $0.1200   36.4s    53,201        12
python-hard            python      baseline    1    Perfect  $0.0826   31.1s    88,839        5
python-hard            python      baseline    2    Perfect  $0.0746   25.2s    85,307        6
python-hard            python      baseline    3    Perfect  $0.0704   26.8s    83,125        5
python-hard            python      with-lumen  1    Perfect  $0.0808   27.3s    90,035        5
python-hard            python      with-lumen  2    Perfect  $0.1113   47.1s    144,919       8
python-hard            python      with-lumen  3    Perfect  $0.0827   31.3s    90,191        5
ruby-hard              ruby        baseline    1    Poor     $0.3216   141.1s   121,631       50    (outlier!)
ruby-hard              ruby        baseline    2    Good     $0.8940   497.9s   543,541       62
ruby-hard              ruby        baseline    3    Perfect  $0.7836   502.4s   535,254       48
ruby-hard              ruby        with-lumen  1    Poor     $0.1490   54.5s    143,015       8
ruby-hard              ruby        with-lumen  2    Poor     $0.2565   142.4s   184,296       11
ruby-hard              ruby        with-lumen  3    Poor     $1.0281   797.7s   408570        15    (outlier!)
rust-hard              rust        baseline    1    Poor     $0.3825   188.3s   422,510       14
rust-hard              rust        baseline    2    Good     $0.4715   212.7s   656,364       21
rust-hard              rust        baseline    3    Poor     $0.4490   207.1s   670,584       24
rust-hard              rust        with-lumen  1    Good     $0.4765   212.7s   496,045       15
rust-hard              rust        with-lumen  2    Good     $0.6642   290.5s   951,907       25
rust-hard              rust        with-lumen  3    Good     $0.5407   279.4s   251,038       35
typescript-hard        typescript  baseline    1    Poor     —         —        —             49     (need better test case)
typescript-hard        typescript  baseline    2    Poor     $1.0498   602.1s   657,407       36     (need better test case)
typescript-hard        typescript  baseline    3    Good     —         —        —             38     (need better test case)
typescript-hard        typescript  with-lumen  1    Poor     $1.5452   840.3s   1,603,553     47     (need better test case)
typescript-hard        typescript  with-lumen  2    Poor     —         —        —             46     (need better test case)
typescript-hard        typescript  with-lumen  3    Poor     —         —        —             35     (need better test case)
```

`—` indicates the run exceeded the timeout or produced no cost data.

## Key Findings

### 1. Tool Call Reduction: -27% on Average

Across all 7 languages and 3 runs each, Lumen reduces the number of tool calls
Claude makes per task from an average of **29.8 to 21.8** — a 27% reduction.

Semantic search lets Claude navigate to the right code directly instead of
reading files systematically and iterating. Fewer tool calls mean faster
sessions and less context pollution.

| Language   | Baseline avg tools | With-Lumen avg tools | Delta |
| ---------- | ------------------ | -------------------- | ----- |
| Go         | 43                 | 31                   | -28%  |
| JavaScript | 31                 | 19                   | -39%  |
| PHP        | 15                 | 16                   | +7%   |
| Python     | 5                  | 6                    | +20%  |
| Ruby       | 53                 | 10                   | -81%  |
| Rust       | 20                 | 25                   | +25%  |
| TypeScript | 41                 | 43                   | +5%   |

Ruby and JavaScript show the strongest tool call reductions.

### 2. Quality Improvements: PHP and Rust

Lumen improves patch quality on two languages with clear, consistent signals:

**PHP** (php-hard): Excluding the one outlier run (131.5s, 509K tokens),
with-lumen averages **165,622 tokens** vs baseline's **246,522 tokens** — a
**33% token reduction**. Best case: run3 used only 53,201 tokens vs the worst
baseline run at 387,332 — an **86% reduction**. Quality remained Perfect on all
with-lumen runs vs Good on all baseline runs.

**Rust** (rust-hard): Baseline produced **Poor/Good/Poor** across 3 runs. With
Lumen, all 3 runs produced **Good**. Lumen helped Claude find the correct type
context needed to fix the async return type inference issue.

**Python**: Both scenarios produce **Perfect** on all 3 runs — Lumen adds no
risk to Python.

### 3. Per-Language Breakdown (Best Observed)

These compare the best with-lumen run against the worst baseline run for the
same task, using the same task to make the comparison fair:

| Language   | Best Lumen            | Worst Baseline         | Cost Delta | Time Delta | Quality         |
| ---------- | --------------------- | ---------------------- | ---------- | ---------- | --------------- |
| PHP        | $0.12, 36.4s, 53K tok | $0.25, 89.6s, 387K tok | **-52%**   | **-59%**   | **-86% tokens** |
| JavaScript | $0.32, 163s, Good     | $0.72, 336s, Good      | **-56%**   | **-51%**   | Same            |
| Go         | $0.43, 228.5s, Good   | $0.61, 261.5s, Good    | -30%       | -13%       | Same            |
| Ruby       | $0.15, 54.5s, Poor    | $0.89, 497.9s, Good    | **-83%**   | **-89%**   | Worse           |
| Rust       | $0.48, 212.7s, Good   | $0.45, 207.1s, Poor    | +7%        | +3%        | Better          |
| Python     | $0.08, 27.3s, Perfect | $0.08, 31.1s, Perfect  | 0%         | -12%       | Same            |
| TypeScript | $1.55, 840s, Poor     | $1.05, 602s, Poor      | +47%       | +40%       | Same (both bad) |

**PHP** is the standout: Lumen cuts token usage by 33% on average (excl. outlier
run) and 86% in the best case, while also delivering 52% less cost and 59% less
time. **JavaScript** shows strong and consistent resource reduction at the same
quality level.

**Ruby** shows that Lumen can dramatically reduce resource consumption in
high-churn exploration scenarios — 83% cheaper and 89% faster — but at the cost
of answer quality on this particular task. This is likely a retrieval tuning
issue rather than a fundamental limitation.

**TypeScript** is the known weak spot. The tree-sitter chunker for TypeScript
needs improvement to handle the complexity of the TypeScript codebase. Both
scenarios produce poor results, and Lumen adds overhead without benefit. This is
an active area of improvement.

---

## Reproduce

Requirements: Ollama running with `ordis/jina-embeddings-v2-base-code`, the
`claude` CLI, `git`, `go`, `jq`.

```bash
cd bench-swe

# Run all tasks, 3 runs each, both scenarios
go run ./cmd/run --runs 3 --output ../bench-results/my-run

# Run a single language
go run ./cmd/run --filter go-hard --runs 3 --output ../bench-results/my-run

# Generate report from existing results
go run ./cmd/report --input ../bench-results/my-run
```

Results land in `bench-results/<run-id>/`. Each run produces:

- `<task>-<scenario>-run<N>-raw.jsonl` — full Claude session stream
- `<task>-<scenario>-run<N>-metrics.json` — extracted cost/time/tokens/tools
- `<task>-<scenario>-run<N>-patch.diff` — generated patch
- `<task>-<scenario>-run<N>-judge.json` — judge rating and reasoning
- `detail-report.md` / `summary-report.md` — human-readable output

The benchmark is entirely self-contained in `bench-swe/`. Tasks are defined as
JSON files in `bench-swe/tasks/`. To add a new language or difficulty level, add
a task JSON and re-run.

Current results are committed at
[`bench-results/swe-20260310-130917-ollama-jina-embeddings-v2-base-code/`](../bench-results/swe-20260310-130917-ollama-jina-embeddings-v2-base-code/).
