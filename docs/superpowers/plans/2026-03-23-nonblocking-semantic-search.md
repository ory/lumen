# Non-blocking semantic_search Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `semantic_search` return stale results with a warning after 15s instead of blocking indefinitely on reindexing.

**Architecture:** Replace the synchronous `EnsureFresh()` call in `ensureIndexed()` with a goroutine guarded by a 15s timeout. The goroutine acquires the flock, reindexes, and calls `touchChecked()` on success. A `sync.WaitGroup` on `indexerCache` ensures graceful shutdown.

**Tech Stack:** Go, `sync.WaitGroup`, `indexlock` (flock), `context.Background()`

**Spec:** `docs/superpowers/specs/2026-03-23-nonblocking-semantic-search-design.md`

---

## Chunk 1: Core implementation

### Task 1: Add `sync.WaitGroup` and `StaleWarning` field

**Files:**
- Modify: `cmd/stdio.go:78-84` (SemanticSearchOutput struct)
- Modify: `cmd/stdio.go:132-142` (indexerCache struct)

- [ ] **Step 1: Add `StaleWarning` to `SemanticSearchOutput`**

In `cmd/stdio.go`, add the field after `SeedWarning`:

```go
type SemanticSearchOutput struct {
	Results      []SearchResultItem `json:"results"`
	Reindexed    bool               `json:"reindexed"`
	IndexedFiles int                `json:"indexed_files,omitempty"`
	FilteredHint string             `json:"filtered_hint,omitempty"`
	SeedWarning  string             `json:"seed_warning,omitempty"`
	StaleWarning string             `json:"stale_warning,omitempty"`
}
```

- [ ] **Step 2: Add `wg` field to `indexerCache`**

In `cmd/stdio.go`, add a `sync.WaitGroup` to `indexerCache`:

```go
type indexerCache struct {
	mu            sync.RWMutex
	cache         map[string]cacheEntry
	embedder      embedder.Embedder
	model         string
	cfg           config.Config
	freshnessTTL  time.Duration
	findDonorFunc func(string, string) string
	seedFunc      func(string, string) (bool, error)
	log           *slog.Logger
	wg            sync.WaitGroup // tracks background reindex goroutines
}
```

- [ ] **Step 3: Add constant for reindex timeout**

Add near `defaultFreshnessTTL` (line 122):

```go
const reindexTimeout = 15 * time.Second
const backgroundReindexMaxDuration = 10 * time.Minute
```

- [ ] **Step 4: Compile check**

Run: `go build ./...`
Expected: PASS (no behavior change yet)

- [ ] **Step 5: Commit**

```bash
git add cmd/stdio.go
git commit -m "refactor(cmd): add StaleWarning field and WaitGroup to indexerCache"
```

---

### Task 2: Update `Close()` to wait for background goroutines

**Files:**
- Modify: `cmd/stdio.go:153-165` (Close method)

- [ ] **Step 1: Write the failing test**

In `cmd/stdio_test.go`, add a test that verifies `Close()` waits for a background goroutine tracked by `wg`:

```go
func TestIndexerCache_CloseWaitsForBackground(t *testing.T) {
	ic := &indexerCache{
		cache: make(map[string]cacheEntry),
	}

	done := make(chan struct{})
	ic.wg.Add(1)
	go func() {
		defer ic.wg.Done()
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	ic.Close()

	select {
	case <-done:
		// goroutine finished before Close returned — correct
	default:
		t.Fatal("Close() returned before background goroutine finished")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd -run TestIndexerCache_CloseWaitsForBackground -count=1`
Expected: FAIL — `Close()` does not call `wg.Wait()` yet

- [ ] **Step 3: Update `Close()` to wait**

Replace the `Close` method (lines 153-165):

```go
// Close waits for any background reindex goroutines to finish, then
// closes all cached indexers. Call on MCP server shutdown.
func (ic *indexerCache) Close() {
	ic.wg.Wait()
	ic.mu.Lock()
	defer ic.mu.Unlock()
	seen := make(map[*index.Indexer]bool)
	for _, entry := range ic.cache {
		if !seen[entry.idx] {
			seen[entry.idx] = true
			_ = entry.idx.Close()
		}
	}
	ic.cache = nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd -run TestIndexerCache_CloseWaitsForBackground -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/stdio.go cmd/stdio_test.go
git commit -m "feat(cmd): Close() waits for background reindex goroutines"
```

---

### Task 3: Implement non-blocking `ensureIndexed()`

**Files:**
- Modify: `cmd/stdio.go:568-651` (ensureIndexed method)

- [ ] **Step 1: Replace synchronous `EnsureFresh` with timeout-guarded goroutine**

Replace lines 615-650 (from the `logger.Debug("freshness TTL expired...")` through the end of `ensureIndexed`) with:

```go
	ic.logger().Debug("freshness TTL expired or first check, building merkle tree",
		"cwd", input.Cwd,
		"effective_root", projectDir,
	)

	// Run EnsureFresh in a goroutine with a 15s timeout. If reindexing
	// takes longer, return stale results with a warning while the
	// goroutine continues in the background.
	type freshResult struct {
		reindexed bool
		stats     index.Stats
		err       error
	}
	done := make(chan freshResult, 1) // buffered: goroutine must never block on send

	bgCtx, bgCancel := context.WithTimeout(context.Background(), backgroundReindexMaxDuration)

	lockPath := indexlock.LockPathForDB(dbPath)
	ic.wg.Add(1)
	go func() {
		defer ic.wg.Done()
		defer bgCancel()

		lk, lockErr := indexlock.TryAcquire(lockPath)
		if lockErr != nil {
			ic.logger().Warn("background reindex: failed to acquire lock", "project", projectDir, "err", lockErr)
			done <- freshResult{}
			return
		}
		if lk == nil {
			// Another process grabbed the lock between our IsHeld check and now.
			ic.logger().Debug("background reindex: lock held by another process, skipping", "project", projectDir)
			done <- freshResult{}
			return
		}
		defer lk.Release()

		reindexed, stats, err := idx.EnsureFresh(bgCtx, projectDir, nil) // nil progress: request ctx may be gone
		if err != nil {
			ic.logger().Warn("background reindex failed", "project", projectDir, "err", err)
		} else {
			ic.touchChecked(projectDir)
		}
		done <- freshResult{reindexed: reindexed, stats: stats, err: err}
	}()

	timer := time.NewTimer(reindexTimeout)
	defer timer.Stop()

	select {
	case result := <-done:
		bgCancel() // release context resources early
		if result.err != nil {
			return out, fmt.Errorf("ensure fresh: %w", result.err)
		}
		elapsed := time.Since(start)
		if !result.reindexed {
			ic.logger().Debug("index fresh, caching result",
				"cwd", input.Cwd,
				"effective_root", projectDir,
				"elapsed_ms", elapsed.Milliseconds(),
			)
		} else {
			ic.logger().Info("reindex triggered",
				"cwd", input.Cwd,
				"search_path", input.Path,
				"effective_root", projectDir,
				"total_project_files", result.stats.TotalFiles,
				"files_indexed", result.stats.IndexedFiles,
				"chunks_created", result.stats.ChunksCreated,
				"files_changed", result.stats.FilesChanged,
				"elapsed_ms", elapsed.Milliseconds(),
			)
		}
		out.Reindexed = result.reindexed
		if result.reindexed {
			out.IndexedFiles = result.stats.IndexedFiles
		}
		return out, nil

	case <-timer.C:
		ic.logger().Info("reindex timeout, returning stale results",
			"project", projectDir,
			"timeout", reindexTimeout,
		)
		out.StaleWarning = "Index is being updated in the background. Results may be incomplete or outdated. A follow-up search in ~30s will return fresh results."
		return out, nil
	}
```

- [ ] **Step 2: Compile check**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 3: Run existing tests**

Run: `go test ./cmd -count=1`
Expected: PASS (existing behavior preserved for fast paths)

- [ ] **Step 4: Commit**

```bash
git add cmd/stdio.go
git commit -m "feat(cmd): non-blocking ensureIndexed with 15s timeout and background reindex"
```

---

### Task 4: Update `formatSearchResults` to render `StaleWarning`

**Files:**
- Modify: `cmd/stdio.go:1001+` (formatSearchResults function)

- [ ] **Step 1: Write the failing test**

Add to `cmd/stdio_test.go`:

```go
func TestFormatSearchResults_StaleWarning(t *testing.T) {
	out := SemanticSearchOutput{
		Results: []SearchResultItem{
			{FilePath: "/proj/main.go", Symbol: "main", Kind: "function", StartLine: 1, EndLine: 5, Score: 0.9},
		},
		StaleWarning: "Index is being updated in the background.",
	}
	text := formatSearchResults("/proj", out)
	if !strings.Contains(text, "Warning: Index is being updated") {
		t.Fatalf("expected stale warning in output, got:\n%s", text)
	}
}

func TestFormatSearchResults_NoStaleWarning(t *testing.T) {
	out := SemanticSearchOutput{
		Results: []SearchResultItem{
			{FilePath: "/proj/main.go", Symbol: "main", Kind: "function", StartLine: 1, EndLine: 5, Score: 0.9},
		},
	}
	text := formatSearchResults("/proj", out)
	if strings.Contains(text, "Warning:") {
		t.Fatalf("unexpected warning in output, got:\n%s", text)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd -run TestFormatSearchResults_StaleWarning -count=1`
Expected: FAIL — StaleWarning not rendered yet

- [ ] **Step 3: Add StaleWarning rendering**

In `formatSearchResults`, after the `SeedWarning` block (around line 1024-1026), add:

```go
	if out.StaleWarning != "" {
		fmt.Fprintf(&b, "\nWarning: %s", out.StaleWarning)
	}
```

Also add it in the empty-results branch (after line 1010-1011):

```go
	if out.StaleWarning != "" {
		b.WriteString("\nWarning: ")
		b.WriteString(out.StaleWarning)
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd -run TestFormatSearchResults -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/stdio.go cmd/stdio_test.go
git commit -m "feat(cmd): render StaleWarning in semantic_search output"
```

---

## Chunk 2: Testing and verification

### Task 5: Test flock-skip fast path

**Files:**
- Modify: `cmd/stdio_test.go`

- [ ] **Step 1: Write the test**

Verify that when the flock is already held (by session-start or another process),
`ensureIndexed` returns immediately with no `StaleWarning` (existing behavior preserved):

```go
func TestEnsureIndexed_FlockHeldSkipsReindex(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	lockPath := indexlock.LockPathForDB(dbPath)

	// Pre-acquire the lock to simulate a running indexer.
	lk, err := indexlock.TryAcquire(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if lk == nil {
		t.Fatal("expected to acquire lock")
	}
	defer lk.Release()

	ic := &indexerCache{
		cache: make(map[string]cacheEntry),
	}

	idx, idxErr := index.NewIndexer(dbPath, nil)
	if idxErr != nil {
		t.Fatal(idxErr)
	}
	defer idx.Close()

	out, err := ic.ensureIndexed(
		context.Background(),
		idx,
		SemanticSearchInput{Cwd: tmpDir, Path: tmpDir, Query: "test"},
		tmpDir, dbPath, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if out.StaleWarning != "" {
		t.Fatalf("expected no StaleWarning when flock held, got: %s", out.StaleWarning)
	}
}
```

- [ ] **Step 2: Run test**

Run: `go test ./cmd -run TestEnsureIndexed_FlockHeldSkipsReindex -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add cmd/stdio_test.go
git commit -m "test(cmd): verify ensureIndexed skips reindex when flock is held"
```

---

### Task 6: Test timeout path — inject slow EnsureFresh via hook

**Files:**
- Modify: `cmd/stdio.go:132-142` (add test hook to indexerCache)
- Modify: `cmd/stdio_test.go`

To test the timeout path without a real slow embedder, add an optional test hook
to `indexerCache` that wraps `EnsureFresh`. This is the same pattern used for
`findDonorFunc` and `seedFunc`.

- [ ] **Step 1: Add `ensureFreshFunc` hook to `indexerCache`**

```go
type indexerCache struct {
	// ... existing fields ...
	ensureFreshFunc func(ctx context.Context, idx *index.Indexer, projectDir string, progress index.ProgressFunc) (bool, index.Stats, error) // nil uses idx.EnsureFresh
}
```

- [ ] **Step 2: Use the hook in `ensureIndexed`'s goroutine**

In the goroutine body (Task 3), replace:
```go
reindexed, stats, err := idx.EnsureFresh(bgCtx, projectDir, nil)
```
with:
```go
		ensureFresh := ic.ensureFreshFunc
		if ensureFresh == nil {
			ensureFresh = func(ctx context.Context, idx *index.Indexer, dir string, p index.ProgressFunc) (bool, index.Stats, error) {
				return idx.EnsureFresh(ctx, dir, p)
			}
		}
		reindexed, stats, err := ensureFresh(bgCtx, idx, projectDir, nil)
```

- [ ] **Step 3: Write the timeout test**

```go
func TestEnsureIndexed_TimeoutReturnsStaleWarning(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a real indexer so ensureIndexed has something to work with.
	idx, err := index.NewIndexer(dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	ic := &indexerCache{
		cache: map[string]cacheEntry{
			tmpDir: {idx: idx, effectiveRoot: tmpDir},
		},
		// Simulate a slow EnsureFresh that takes longer than reindexTimeout.
		ensureFreshFunc: func(ctx context.Context, _ *index.Indexer, _ string, _ index.ProgressFunc) (bool, index.Stats, error) {
			select {
			case <-time.After(30 * time.Second):
				return true, index.Stats{IndexedFiles: 100}, nil
			case <-ctx.Done():
				return false, index.Stats{}, ctx.Err()
			}
		},
	}

	start := time.Now()
	out, err := ic.ensureIndexed(
		context.Background(),
		idx,
		SemanticSearchInput{Cwd: tmpDir, Path: tmpDir, Query: "test"},
		tmpDir, dbPath, nil,
	)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out.StaleWarning == "" {
		t.Fatal("expected StaleWarning to be set after timeout")
	}
	if elapsed > 20*time.Second {
		t.Fatalf("ensureIndexed took %v, expected ~15s timeout", elapsed)
	}
	if out.Reindexed {
		t.Fatal("expected Reindexed=false after timeout")
	}

	// Wait for background goroutine to finish (WaitGroup).
	ic.Close()
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd -run TestEnsureIndexed_TimeoutReturnsStaleWarning -count=1 -timeout=60s`
Expected: PASS — returns in ~15s with StaleWarning set

- [ ] **Step 5: Write the fast-path test (completes before timeout)**

```go
func TestEnsureIndexed_FastEnsureFreshNoWarning(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	idx, err := index.NewIndexer(dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	ic := &indexerCache{
		cache: map[string]cacheEntry{
			tmpDir: {idx: idx, effectiveRoot: tmpDir},
		},
		// Simulate a fast EnsureFresh.
		ensureFreshFunc: func(_ context.Context, _ *index.Indexer, _ string, _ index.ProgressFunc) (bool, index.Stats, error) {
			return true, index.Stats{IndexedFiles: 42}, nil
		},
	}

	out, err := ic.ensureIndexed(
		context.Background(),
		idx,
		SemanticSearchInput{Cwd: tmpDir, Path: tmpDir, Query: "test"},
		tmpDir, dbPath, nil,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out.StaleWarning != "" {
		t.Fatalf("unexpected StaleWarning: %s", out.StaleWarning)
	}
	if !out.Reindexed {
		t.Fatal("expected Reindexed=true")
	}
	if out.IndexedFiles != 42 {
		t.Fatalf("expected IndexedFiles=42, got %d", out.IndexedFiles)
	}

	ic.Close()
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./cmd -run TestEnsureIndexed -count=1 -timeout=60s`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/stdio.go cmd/stdio_test.go
git commit -m "test(cmd): add timeout and fast-path tests for non-blocking ensureIndexed"
```

---

### Task 7: Full test suite and lint

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 2: Run linter**

Run: `golangci-lint run`
Expected: PASS with zero issues

- [ ] **Step 3: Run vet**

Run: `go vet ./...`
Expected: PASS (external dependency warnings OK)

- [ ] **Step 4: Final commit if any lint fixes needed**

```bash
git add -A
git commit -m "style: fix lint issues from non-blocking search implementation"
```
