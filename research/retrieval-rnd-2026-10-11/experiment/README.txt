Reproduction from this saved bundle

The local go.mod keeps these archived standalone programs out of repository-wide Go checks. They are separate executables, not a package to build together; copy them into the frozen checkout as described below.

The original scripts intentionally preserve their exact experiment paths. Copy this directory's experiment-* files and experiment.py into /tmp/lumen-research-20261011 (or adapt all hardcoded paths consistently). Use a separate machine or preserve existing results before rerunning. Extract experiment-source.tar.gz as described below. No production checkout is modified.

# Retrieval mechanism experiment, 2026-10-11

This is a source-aware synthetic mechanism check, not a statistically meaningful held-out evaluation. All 30 cases were declared in experiment.py before any method was evaluated. 10 reuse smoke identities; 20 prospective cases target other sample fixtures. No parameters or queries were chosen from observed retrieval output.

## Reproduce

Required: Python 3, Go 1.25 toolchain and dependencies already present, C compiler, ripgrep. No model services, downloads, or agent calls.

The frozen source lives in experiment-source.tar.gz; its files have per-file SHA256s in experiment-source-manifest.json. The harness and experiment script are also provided separately for review. Extract source to `/tmp/lumen-research-20261011/experiment-checkout`; place `experiment-main.go` as `cmd/experiment/main.go` there. Put `experiment.py` in `/tmp/lumen-research-20261011`.

```
cd /tmp/lumen-research-20261011/experiment-checkout
GOCACHE=/tmp/lumen-research-20261011/experiment-gocache go build -tags fts5 -o /tmp/lumen-research-20261011/experiment-bm25 ./cmd/experiment
python3 /tmp/lumen-research-20261011/experiment.py > /tmp/lumen-research-20261011/experiment-run.log
```

The script copies five smoke files plus all existing sample-project and language fixtures, freezes queries, builds a corpus token lexicon and trigram postings, then creates a fresh actual Lumen lexical index and executes 2 repetitions per unique query (discarding first timing), reusing identical-query results across method labels. rg performs 5 repetitions including process invocation, scanning, parsing, ranking and cap enforcement. Actual Lumen uses SearchLexical with maxChunkTokens512, limit3, no relation expansion. Index timing starts before opening/indexing a fresh DB but source pages are OS-cache warm. Database bytes include active WAL and SHM.

Both return max3 results, each with max1024 content characters and at most4096 UTF-8 bytes total serialized JSON. Timing of output normalization differs: rg includes it; Go timing excludes trivial Python final normalization. Comparisons measure implemented workflows, not a pure algorithm speed ratio. BM25_fuzzy includes separately measured Python expansion cost plus in-process Go query cost. Compilation is excluded.

rg uses case-insensitive fixed-substring OR search over original words and camel/snake components, removes a fixed generic stopword list, and ranks source windows by distinct matched term rarity. It retains the highest-ranked 9-line window per file. It is a strong reproducible lexical baseline, not an adaptive human/agent best-of-many search baseline. Lumen returns symbol-aware chunks; granularity is intentionally part of the system comparison.

The expansion lexicon is corpus-derived ASCII code-shaped tokens (comments also contribute), len>=5. Unknown query tokens generate candidates from OR character trigram postings, retain128 by overlap/frequency, verify Levenshtein distance<=1 (len5–11) or<=2 (len>=12), add up to3, and preserve all original query text. This is approximate and not recall-complete. No manual query-specific synonyms. No fuzzy expansion of an exact-known token. No transposition-as-one edit.

Target precision is against deliberately sparse file+symbol-string labels; it is NOT exhaustive relevance precision. No-answer cases include fictional concepts and unsupported claims containing valid symbols: nonempty retrieval is spurious context cost, not a calibrated answer-level false-positive rate. Payload bytes are measured, tokens estimated only if explicitly divided by four elsewhere.

## Runtime amendment

The initial 12-repetition sweep was stopped before final results because the current Go/SQLite query plan took roughly one second per call on this corpus. Corpus, queries, candidate parameters, scoring and caps stayed unchanged. The completed run uses2 calls per unique query and reports the second as a warm timing; medians in the table aggregate different queries, not repeated timing samples of the same query. A separate3-repeat SQL probe diagnoses the execution plan. No significance, tail-latency or isolated-machine performance claim is justified. Other agents and short diagnostic processes were active on this machine.

## FTS-first diagnostic

After the main experiment, run `python3 /tmp/lumen-research-20261011/experiment-plan-reproduce.py`. It changes one SQL join-order expression in the temporary copied checkout, builds a separate binary, immediately restores the source, reuses the same index and frozen requests, and compares every returned path, line, content and score. The patch is separately reviewable in `experiment-fts-first.patch`. Do not apply this to production based solely on these synthetic cases: filters, worktree memberships, score ties, update correctness, other SQLite versions and high-recall queries need regression coverage.

`experiment-sqlprobe.go` is an additional standalone Go main package: place it in `cmd/sqlprobe/main.go` of the copy, then run `GOCACHE=/tmp/lumen-research-20261011/experiment-gocache go run -tags fts5 ./cmd/sqlprobe /tmp/lumen-research-20261011/experiment-index.db`. Its SQL uses the same selected columns, scoring expression, literal arguments, order, limit and reader PRAGMAs as the production query. It compares original vs explicit FTS-first CROSS JOIN plans for ValidateToken and prints SQLite version, EXPLAIN and3 timings. All-query API equivalence comes from experiment-plan-comparison.json.
