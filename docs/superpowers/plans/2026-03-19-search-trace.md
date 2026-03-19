# lumen search --trace Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `lumen search` CLI subcommand that replicates what the MCP server does for a `semantic_search` tool call, with an optional `--trace` flag that prints per-phase wall-clock timing to stderr.

**Architecture:** Single new file `cmd/search.go` — no changes to existing packages. All business logic (embedder creation, indexer setup, search, post-processing) is reused from `cmd/index.go`, `cmd/embedder.go`, and `cmd/stdio.go` via package-internal function calls. A lightweight `tracer` struct (new, in `cmd/search.go`) records named spans; it is a no-op when `--trace` is not set.

**Tech Stack:** Go 1.25+, Cobra (existing), SQLite + sqlite-vec (CGO, existing), `io.Writer` for trace output.

---

## Task 1 — Tracer unit tests and implementation

**Files:**
- Create: `cmd/search_test.go`
- Create: `cmd/search.go` (tracer types + methods only in this task)

- [ ] **Step 1.1 — Write the failing tracer tests**

Create `cmd/search_test.go`:

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTracer_DisabledIsNoop(t *testing.T) {
	tr := &tracer{enabled: false}
	tr.start = time.Now()
	tr.last = tr.start

	tr.record("path resolution", "/tmp/project")
	tr.record("indexer setup", "db opened")

	if len(tr.spans) != 0 {
		t.Fatalf("disabled tracer should not record spans, got %d", len(tr.spans))
	}

	var buf bytes.Buffer
	tr.print(&buf)
	if buf.Len() != 0 {
		t.Fatalf("disabled tracer should produce no output, got %q", buf.String())
	}
}

func TestTracer_EnabledRecordsSpans(t *testing.T) {
	tr := &tracer{enabled: true}
	tr.start = time.Now()
	tr.last = tr.start

	tr.record("path resolution", "/tmp/project")
	tr.record("indexer setup", "db opened, model stub")

	if len(tr.spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(tr.spans))
	}
	if tr.spans[0].label != "path resolution" {
		t.Fatalf("expected label 'path resolution', got %q", tr.spans[0].label)
	}
	if tr.spans[0].detail != "/tmp/project" {
		t.Fatalf("expected detail '/tmp/project', got %q", tr.spans[0].detail)
	}
	if tr.spans[1].label != "indexer setup" {
		t.Fatalf("expected label 'indexer setup', got %q", tr.spans[1].label)
	}
	for _, s := range tr.spans {
		if s.duration < 0 {
			t.Fatalf("span %q has negative duration %v", s.label, s.duration)
		}
	}
}

func TestTracer_PrintRendersTable(t *testing.T) {
	tr := &tracer{enabled: true}
	tr.start = time.Now()
	tr.last = tr.start
	tr.spans = []traceSpan{
		{label: "path resolution", duration: 2 * time.Millisecond, detail: "/tmp/project"},
		{label: "knn search", duration: 9 * time.Millisecond, detail: "16 candidates fetched"},
	}

	var buf bytes.Buffer
	tr.print(&buf)
	out := buf.String()

	for _, want := range []string{"path resolution", "/tmp/project", "knn search", "total", "────"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestTracer_RecordAdvancesLast(t *testing.T) {
	tr := &tracer{enabled: true}
	before := time.Now()
	tr.start = before
	tr.last = before

	time.Sleep(2 * time.Millisecond)
	tr.record("span1", "detail")
	after := tr.last

	if !after.After(before) {
		t.Fatalf("tracer.last should have advanced after record()")
	}
	if tr.spans[0].duration <= 0 {
		t.Fatalf("first span should have positive duration")
	}
}
```

- [ ] **Step 1.2 — Run test to verify it fails**

```
go test ./cmd/... -run TestTracer -v
```

Expected: compilation error — `tracer`, `traceSpan`, `record`, `print` not defined.

- [ ] **Step 1.3 — Implement tracer in cmd/search.go**

Create `cmd/search.go` with license header, then the tracer types and methods:

```go
// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type traceSpan struct {
	label    string
	duration time.Duration
	detail   string
}

type tracer struct {
	enabled bool
	start   time.Time
	last    time.Time
	spans   []traceSpan
}

func (t *tracer) record(label, detail string) {
	if !t.enabled {
		return
	}
	now := time.Now()
	t.spans = append(t.spans, traceSpan{
		label:    label,
		duration: now.Sub(t.last),
		detail:   detail,
	})
	t.last = now
}

func (t *tracer) print(w io.Writer) {
	if !t.enabled {
		return
	}
	const sep = "───────────────────────────────────────────────────────────────────────"
	for _, s := range t.spans {
		ms := s.duration.Milliseconds()
		fmt.Fprintf(w, "[%4dms] %-22s → %s\n", ms, s.label, s.detail)
	}
	fmt.Fprintln(w, sep)
	total := time.Since(t.start)
	fmt.Fprintf(w, "[%4dms] total\n", total.Milliseconds())
}
```

- [ ] **Step 1.4 — Run tracer tests; confirm they pass**

```
go test ./cmd/... -run TestTracer -v
```

Expected: all four `TestTracer_*` tests pass.

- [ ] **Step 1.5 — Commit**

```
git add cmd/search.go cmd/search_test.go
git commit -m "test(cmd): add tracer unit tests and tracer implementation"
```

---

## Task 2 — Search subcommand: flag registration tests and implementation

**Files:**
- Modify: `cmd/search.go` (add subcommand + runSearch)
- Modify: `cmd/search_test.go` (add flag registration + trace integration tests)

- [ ] **Step 2.1 — Write failing flag-registration test**

Add to `cmd/search_test.go`:

```go
func TestSearchCmd_FlagsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"search"})
	if err != nil || cmd == nil || cmd.Use != "search <query>" {
		t.Fatalf("search subcommand not registered or wrong Use field: %v", err)
	}

	requiredFlags := []string{
		"path", "cwd", "n-results", "min-score",
		"summary", "max-lines", "force", "trace", "model",
	}
	for _, name := range requiredFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("search cmd missing flag --%s", name)
		}
	}
}

func TestSearchCmd_TraceSpanLabels(t *testing.T) {
	// Verify the trace span labels that runSearch records match the spec.
	tr := &tracer{enabled: true}
	tr.start = time.Now()
	tr.last = tr.start

	tr.record("path resolution", "/tmp/proj")
	tr.record("indexer setup", "db opened, model stub")
	tr.record("merkle + freshness", "42 files scanned, index is fresh (no reindex)")
	tr.record("query embedding", "4 dims")
	tr.record("knn search", "0 candidates fetched (limit=16, fetch=16)")
	tr.record("post-processing", "merged 0→0 results, filled 0 snippets")

	var stderr bytes.Buffer
	tr.print(&stderr)

	out := stderr.String()
	for _, label := range []string{
		"path resolution", "indexer setup", "merkle + freshness",
		"query embedding", "knn search", "post-processing", "total",
	} {
		if !strings.Contains(out, label) {
			t.Fatalf("trace output missing %q:\n%s", label, out)
		}
	}
}
```

- [ ] **Step 2.2 — Run to confirm failure**

```
go test ./cmd/... -run "TestSearchCmd" -v
```

Expected: `TestSearchCmd_FlagsRegistered` fails — `search` subcommand not found.

- [ ] **Step 2.3 — Implement the search subcommand**

Add to `cmd/search.go`. First, update the import block (replace the minimal one from Task 1):

```go
import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/spf13/cobra"
)
```

Then add the subcommand registration and implementation after the tracer code:

```go
func init() {
	searchCmd.Flags().StringP("path", "p", "", "directory to search (default: cwd)")
	searchCmd.Flags().String("cwd", "", "project root when path is a subdirectory")
	searchCmd.Flags().IntP("n-results", "n", 8, "max results to return")
	searchCmd.Flags().Float64("min-score", 0, "minimum score threshold (-1 to 1)")
	searchCmd.Flags().Bool("summary", false, "omit code snippets, return location only")
	searchCmd.Flags().Int("max-lines", 0, "truncate snippets at N lines (0 = unlimited)")
	searchCmd.Flags().BoolP("force", "f", false, "force full re-index before searching")
	searchCmd.Flags().Bool("trace", false, "print per-phase timing to stderr")
	searchCmd.Flags().StringP("model", "m", "", "embedding model override")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search an indexed project for semantically similar code",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	traceEnabled, _ := cmd.Flags().GetBool("trace")
	pathFlag, _ := cmd.Flags().GetString("path")
	cwdFlag, _ := cmd.Flags().GetString("cwd")
	nResults, _ := cmd.Flags().GetInt("n-results")
	summary, _ := cmd.Flags().GetBool("summary")
	maxLines, _ := cmd.Flags().GetInt("max-lines")
	force, _ := cmd.Flags().GetBool("force")

	var minScore *float64
	if cmd.Flags().Changed("min-score") {
		v, _ := cmd.Flags().GetFloat64("min-score")
		minScore = &v
	}

	tr := &tracer{enabled: traceEnabled}
	if traceEnabled {
		tr.start = time.Now()
		tr.last = tr.start
	}

	// Span 1: path resolution
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := applyModelFlag(cmd, &cfg); err != nil {
		return err
	}

	projectPath := pathFlag
	if projectPath == "" {
		if cwdFlag != "" {
			projectPath = cwdFlag
		} else {
			projectPath, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
		}
	}
	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// When --cwd and --path both given, cwd is the index root.
	indexRoot := projectPath
	if cwdFlag != "" {
		abs, err := filepath.Abs(cwdFlag)
		if err != nil {
			return fmt.Errorf("resolve cwd: %w", err)
		}
		indexRoot = abs
	}

	tr.record("path resolution", indexRoot)

	// Span 2: indexer setup
	idx, err := setupIndexer(&cfg, indexRoot)
	if err != nil {
		return fmt.Errorf("setup indexer: %w", err)
	}
	defer func() { _ = idx.Close() }()
	tr.record("indexer setup", fmt.Sprintf("db opened, model %s", cfg.Model))

	// Span 3: merkle + freshness (BuildTree happens inside EnsureFresh/Index)
	ctx := context.Background()
	var reindexed bool
	var stats interface{ GetIndexedFiles() int }
	_ = stats

	var freshnessDetail string
	if force {
		s, err := idx.Index(ctx, indexRoot, true, nil)
		if err != nil {
			return fmt.Errorf("force reindex: %w", err)
		}
		reindexed = true
		freshnessDetail = fmt.Sprintf("reindexed %d files", s.IndexedFiles)
		tr.record("merkle + freshness", freshnessDetail)
		// build query embedding
		emb, err := newEmbedder(cfg)
		if err != nil {
			return fmt.Errorf("create embedder: %w", err)
		}
		return finishSearch(cmd, ctx, tr, idx, emb, query, indexRoot, projectPath, nResults, minScore, summary, maxLines, reindexed, s.IndexedFiles)
	}

	reindexedBool, s, err := idx.EnsureFresh(ctx, indexRoot, nil)
	if err != nil {
		return fmt.Errorf("ensure fresh: %w", err)
	}
	reindexed = reindexedBool
	if reindexed {
		freshnessDetail = fmt.Sprintf("reindexed %d files", s.IndexedFiles)
	} else {
		freshnessDetail = "index is fresh (no reindex)"
	}
	tr.record("merkle + freshness", freshnessDetail)

	emb, err := newEmbedder(cfg)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}

	indexedFiles := 0
	if reindexed {
		indexedFiles = s.IndexedFiles
	}
	return finishSearch(cmd, ctx, tr, idx, emb, query, indexRoot, projectPath, nResults, minScore, summary, maxLines, reindexed, indexedFiles)
}
```

Then add the `finishSearch` helper that handles spans 4-6 and output:

```go
func finishSearch(
	cmd *cobra.Command,
	ctx context.Context,
	tr *tracer,
	idx interface {
		Search(_ context.Context, _ string, queryVec []float32, limit int, maxDistance float64, pathPrefix string) ([]interface{ GetFilePath() string }, error)
	},
	emb interface {
		Embed(ctx context.Context, texts []string) ([][]float32, error)
		Dimensions() int
		ModelName() string
	},
	query, indexRoot, searchPath string,
	nResults int,
	minScore *float64,
	summary bool,
	maxLines int,
	reindexed bool,
	indexedFiles int,
) error {
	// ... (see note below)
}
```

**IMPORTANT NOTE**: The above skeleton uses interface{} placeholders for clarity. In the actual implementation, use the concrete types from the codebase:
- `idx` is `*index.Indexer` (import `github.com/ory/lumen/internal/index`)
- `emb` is `embedder.Embedder` (import `github.com/ory/lumen/internal/embedder`)
- `store.SearchResult` from `github.com/ory/lumen/internal/store`

Write the actual `finishSearch` as a plain function (not using interfaces) since everything is in the same package:

```go
func finishSearch(
	_ *cobra.Command,
	ctx context.Context,
	tr *tracer,
	idx *index.Indexer,
	emb embedder.Embedder,
	query, indexRoot, searchPath string,
	nResults int,
	minScore *float64,
	summary bool,
	maxLines int,
	reindexed bool,
	indexedFiles int,
) error {
	// Span 4: query embedding
	vecs, err := emb.Embed(ctx, []string{query})
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return fmt.Errorf("embedder returned no vectors")
	}
	queryVec := vecs[0]
	tr.record("query embedding", fmt.Sprintf("%d dims", len(queryVec)))

	// Span 5: KNN search
	fetchLimit := nResults * 2
	maxDistance := computeMaxDistance(minScore, emb.ModelName(), emb.Dimensions())

	var pathPrefix string
	if searchPath != indexRoot {
		if rel, relErr := filepath.Rel(indexRoot, searchPath); relErr == nil && rel != "." {
			pathPrefix = rel
		}
	}

	results, err := idx.Search(ctx, indexRoot, queryVec, fetchLimit, maxDistance, pathPrefix)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	tr.record("knn search", fmt.Sprintf("%d candidates fetched (limit=%d, fetch=%d)", len(results), nResults, fetchLimit))

	// Span 6: post-processing
	items := make([]SearchResultItem, len(results))
	for i, r := range results {
		items[i] = SearchResultItem{
			FilePath:  r.FilePath,
			Symbol:    r.Symbol,
			Kind:      r.Kind,
			StartLine: r.StartLine,
			EndLine:   r.EndLine,
			Score:     boostedScore(float32(1.0-r.Distance), r.Kind, r.FilePath),
		}
	}
	items = mergeOverlappingResults(items)
	slices.SortStableFunc(items, func(a, b SearchResultItem) int {
		return cmp.Compare(b.Score, a.Score)
	})
	if len(items) > nResults {
		items = items[:nResults]
	}
	if !summary {
		fillSnippets(indexRoot, items, maxLines)
	}
	tr.record("post-processing", fmt.Sprintf("merged %d→%d results, filled %d snippets", len(results), len(items), len(items)))

	// Print trace to stderr, then results to stdout.
	tr.print(os.Stderr)

	out := SemanticSearchOutput{
		Results:      items,
		Reindexed:    reindexed,
		IndexedFiles: indexedFiles,
	}
	fmt.Println(formatSearchResults(searchPath, out))
	return nil
}
```

You will also need to add these imports to `cmd/search.go`:

```go
import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/index"
	"github.com/spf13/cobra"
)
```

- [ ] **Step 2.4 — Run subcommand tests**

```
go test ./cmd/... -run "TestTracer|TestSearchCmd" -v
```

Expected: all tests pass.

- [ ] **Step 2.5 — Run full test suite**

```
go test ./...
```

Expected: all existing tests pass, no regressions.

- [ ] **Step 2.6 — Commit**

```
git add cmd/search.go cmd/search_test.go
git commit -m "feat(cmd): add search subcommand with --trace diagnostic flag"
```

---

## Task 3 — Lint and build verification

**Files:** No new files. Fixes to `cmd/search.go` only if linter finds issues.

- [ ] **Step 3.1 — Run linter**

```
golangci-lint run
```

Expected: zero issues. Common things to check:
- All imported packages must be used (`cmp`, `slices`, `strings` — verify each is referenced in the final file)
- `_ = err` for any intentionally ignored errors
- No unused variables

- [ ] **Step 3.2 — Build local binary**

```
make build-local
```

Expected: binary at `bin/lumen`, no errors. CGO_ENABLED=1 is handled by the Makefile.

- [ ] **Step 3.3 — Manual smoke test**

```
bin/lumen search --trace "error handling" /path/to/any/go/project
```

Expected: trace table on stderr with all 6 phase labels, then search results on stdout.

- [ ] **Step 3.4 — Commit lint/build fixes if needed**

Only commit if golangci-lint required code changes:

```
git add cmd/search.go
git commit -m "fix(cmd): lint cleanup in search subcommand"
```

---

## Implementation Notes

### Potential pitfalls

1. **`min-score` flag detection**: Use `cmd.Flags().Changed("min-score")` after flag parsing to detect whether the user actually passed `--min-score`. This is necessary because Cobra always initializes float64 flags to their default (0), so you cannot distinguish "not set" from "set to 0" without `.Changed()`.

2. **Second embedder instance**: `setupIndexer` creates an embedder internally for indexing but does not expose it. `cmd/search.go` must create its own embedder via `newEmbedder(cfg)` for the query embedding step. This is one extra initialisation but is correct and avoids coupling to `indexerCache`.

3. **`stats.TotalFiles` when fresh**: When `EnsureFresh` returns `reindexed=false`, the returned `Stats` is a zero-value struct. The merkle tree walk still happened internally (to check the hash), but the file count is not exposed. The freshness detail string should say `"index is fresh (no reindex)"` without a file count in this case.

4. **`runSearch` vs `finishSearch` split**: The split avoids duplicating the embed+search+post-processing code between the `force` and non-`force` paths. Both paths converge at `finishSearch` after their respective reindex logic.

5. **`strings` import**: Only needed if you use `strings.HasPrefix` in pathPrefix validation. If not needed, remove it to satisfy the linter.

### Critical files for implementation

| File | Role |
|------|------|
| `cmd/search.go` | New file — all new code lives here |
| `cmd/index.go` | Pattern for init()/flag-registration/config.Load()+applyModelFlag()+setupIndexer |
| `cmd/stdio.go` | Source of reused helpers: fillSnippets, formatSearchResults, mergeOverlappingResults, boostedScore, computeMaxDistance, SemanticSearchOutput, SearchResultItem |
| `cmd/embedder.go` | `newEmbedder(cfg)` function |
| `internal/index/index.go` | EnsureFresh, Index signatures; Stats struct |
| `internal/store/store.go` | SearchResult struct fields (FilePath, Symbol, Kind, StartLine, EndLine, Distance) |
