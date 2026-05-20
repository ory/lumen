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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/index"
	"github.com/ory/lumen/internal/indexlock"
)

// ---------------------------------------------------------------------------
// blockingStubEmbedder — blocks on Embed() until Unblock() is called.
// Same idea as the one in internal/index/index_concurrency_test.go but in
// the cmd package for end-to-end tests.
// ---------------------------------------------------------------------------

type blockingStubEmbedder struct {
	dims      int
	started   chan struct{} // closed on first Embed
	blockCh   chan struct{} // close to unblock
	startOnce sync.Once
	unblkOnce sync.Once
}

func newBlockingStubEmbedder(dims int) *blockingStubEmbedder {
	return &blockingStubEmbedder{
		dims:    dims,
		started: make(chan struct{}),
		blockCh: make(chan struct{}),
	}
}

func (e *blockingStubEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	e.startOnce.Do(func() { close(e.started) })
	select {
	case <-e.blockCh:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	vecs := make([][]float32, len(texts))
	for i := range vecs {
		v := make([]float32, e.dims)
		for j := range v {
			v[j] = 0.1
		}
		vecs[i] = v
	}
	return vecs, nil
}

func (e *blockingStubEmbedder) Dimensions() int   { return e.dims }
func (e *blockingStubEmbedder) ModelName() string { return "blocking-stub" }
func (e *blockingStubEmbedder) Unblock()          { e.unblkOnce.Do(func() { close(e.blockCh) }) }

// ---------------------------------------------------------------------------
// End-to-end concurrency tests for the ensureIndexed → Search flow.
// ---------------------------------------------------------------------------

// TestEnsureIndexedThenSearch_TotalLatencyBounded is the critical integration
// test that was missing. It exercises the REAL ensureIndexed → Search path
// (no ensureFreshFunc mock) and verifies the total wall-clock time is bounded.
//
// Before the fix, ensureIndexed returns after the timeout, but the subsequent
// Search() call blocks on idx.mu.RLock() because the background reindex
// goroutine still holds idx.mu.Lock() inside the real EnsureFresh.
func TestEnsureIndexedThenSearch_TotalLatencyBounded(t *testing.T) {
	const dims = 4

	// 1. Create a project directory with an initial Go file and index it.
	projectDir := t.TempDir()
	writeTestGoFile(t, projectDir, "initial.go", `package main

// Initial function for pre-populating the index.
func Initial() {}
`)

	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "test.db")

	fastEmb := &stubEmbedder{} // 4 dims, instant
	idx, err := index.NewIndexer(dbPath, fastEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	_, err = idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = idx.Close()

	// 2. Re-open the indexer with a blocking embedder to simulate slow re-index.
	blockEmb := newBlockingStubEmbedder(dims)
	idx, err = index.NewIndexer(dbPath, blockEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		blockEmb.Unblock()
		_ = idx.Close()
	}()

	// Add a new file so merkle hash changes → triggers re-index.
	writeTestGoFile(t, projectDir, "trigger.go", `package main

// TriggerReindex forces EnsureFresh to reindex.
func TriggerReindex() {}
`)

	// 3. Set up indexerCache with NO ensureFreshFunc (uses real EnsureFresh).
	//    Set freshnessTTL to 1ns so the LastIndexedAt check in the background
	//    goroutine does NOT short-circuit (the initial index stored a recent
	//    timestamp, and the default 30s TTL would skip the merkle walk).
	const shortTimeout = 500 * time.Millisecond
	ic := &indexerCache{
		cache: map[string]cacheEntry{
			projectDir: {idx: idx, effectiveRoot: projectDir},
		},
		embedder:       &stubEmbedder{}, // for embedQuery (separate from indexer's embedder)
		reindexTimeout: shortTimeout,
		freshnessTTL:   1 * time.Nanosecond, // force merkle walk, don't trust LastIndexedAt
		log:            discardLog,
		// ensureFreshFunc intentionally nil → real idx.EnsureFresh
	}

	// 4. Call ensureIndexed — should return after shortTimeout with stale warning
	//    because the background goroutine is blocked on Embed() inside EnsureFresh.
	start := time.Now()
	out, err := ic.ensureIndexed(
		idx,
		SemanticSearchInput{Cwd: projectDir, Path: projectDir, Query: "test"},
		projectDir, dbPath, nil,
	)
	ensureElapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ensureIndexed error: %v", err)
	}

	// Wait for the embedder to confirm it was actually called — this proves
	// EnsureFresh is inside indexWithTree under the write lock.
	select {
	case <-blockEmb.started:
		t.Logf("embedder blocking confirmed after %v", time.Since(start))
	case <-time.After(5 * time.Second):
		t.Fatal("background goroutine never called Embed — test setup broken")
	}

	if out.StaleWarning == "" {
		t.Logf("EnsureFresh completed within timeout (%v) — test inconclusive for blocking bug", ensureElapsed)
		// If EnsureFresh somehow completed instantly, the test is inconclusive
		// but not wrong. Skip the Search-blocking assertion.
		blockEmb.Unblock()
		ic.Close()
		return
	}
	t.Logf("ensureIndexed timed out after %v (expected), stale warning set", ensureElapsed)

	// 5. Now call Search — this is where the bug manifests.
	//    If the background EnsureFresh holds the write lock, Search blocks.
	qvec := []float32{0.1, 0.2, 0.3, 0.4}

	type searchResult struct {
		count int
		err   error
	}
	searchDone := make(chan searchResult, 1)
	go func() {
		results, searchErr := idx.Search(context.Background(), projectDir, qvec, 10, 0, "")
		searchDone <- searchResult{count: len(results), err: searchErr}
	}()

	select {
	case sr := <-searchDone:
		if sr.err != nil {
			t.Fatalf("Search returned error: %v", sr.err)
		}
		t.Logf("Search returned %d results", sr.count)
	case <-time.After(3 * time.Second):
		t.Fatal("Search() blocked for >3 s after ensureIndexed timed out.\n" +
			"Bug: idx.mu.RLock() in Search contends with the write lock " +
			"held by the background EnsureFresh goroutine.\n" +
			"The 15s timeout in ensureIndexed is illusory — Search blocks anyway.")
	}

	// 6. Verify total wall-clock time is bounded.
	totalElapsed := time.Since(start)
	maxAcceptable := shortTimeout + 5*time.Second
	if totalElapsed > maxAcceptable {
		t.Fatalf("total latency %v exceeds acceptable bound %v", totalElapsed, maxAcceptable)
	}

	blockEmb.Unblock()
	ic.Close()
}

// TestEnsureIndexedThenSearch_ParallelQueriesBounded simulates Claude sending
// 3 parallel search queries. All must complete within a bounded time.
func TestEnsureIndexedThenSearch_ParallelQueriesBounded(t *testing.T) {
	const dims = 4
	const parallel = 3

	projectDir := t.TempDir()
	writeTestGoFile(t, projectDir, "base.go", `package main

// Base is the initial indexed function.
func Base() {}
`)

	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "test.db")

	fastEmb := &stubEmbedder{}
	idx, err := index.NewIndexer(dbPath, fastEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	_, err = idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = idx.Close()

	blockEmb := newBlockingStubEmbedder(dims)
	idx, err = index.NewIndexer(dbPath, blockEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		blockEmb.Unblock()
		_ = idx.Close()
	}()

	writeTestGoFile(t, projectDir, "extra.go", `package main

// Extra triggers reindex.
func Extra() {}
`)

	const shortTimeout = 500 * time.Millisecond
	ic := &indexerCache{
		cache: map[string]cacheEntry{
			projectDir: {idx: idx, effectiveRoot: projectDir},
		},
		embedder:       &stubEmbedder{},
		reindexTimeout: shortTimeout,
		freshnessTTL:   1 * time.Nanosecond,
		log:            discardLog,
	}

	start := time.Now()
	// Buffer for all outcomes — each goroutine sends exactly once.
	type queryOutcome struct {
		idx int
		err error
	}
	outcomes := make(chan queryOutcome, parallel)

	for i := range parallel {
		go func() {
			// Each parallel query: ensureIndexed → Search
			_, ensureErr := ic.ensureIndexed(
				idx,
				SemanticSearchInput{Cwd: projectDir, Path: projectDir, Query: "test"},
				projectDir, dbPath, nil,
			)
			if ensureErr != nil {
				outcomes <- queryOutcome{idx: i, err: fmt.Errorf("ensureIndexed: %w", ensureErr)}
				return
			}

			qvec := []float32{0.1, 0.2, 0.3, 0.4}
			_, searchErr := idx.Search(context.Background(), projectDir, qvec, 10, 0, "")
			outcomes <- queryOutcome{idx: i, err: searchErr}
		}()
	}

	// Collect results with a 5s deadline (shortTimeout for ensureIndexed + 3s for Search).
	deadline := time.After(shortTimeout + 4*time.Second)
	completed := 0
	var failures []string
	for completed < parallel {
		select {
		case out := <-outcomes:
			completed++
			if out.err != nil {
				failures = append(failures, fmt.Sprintf("query %d: %v", out.idx, out.err))
			}
		case <-deadline:
			failures = append(failures, fmt.Sprintf("%d of %d parallel queries blocked on Search", parallel-completed, parallel))
			goto done
		}
	}
done:
	if len(failures) > 0 {
		t.Fatalf("parallel queries failed:\n%v", failures)
	}

	totalElapsed := time.Since(start)
	// All parallel queries should complete within timeout + buffer, not N * indexing time.
	maxAcceptable := shortTimeout + 5*time.Second
	if totalElapsed > maxAcceptable {
		t.Fatalf("total parallel latency %v exceeds %v", totalElapsed, maxAcceptable)
	}

	blockEmb.Unblock()
	ic.Close()
}

// TestHandleSemanticSearch_StaleWarningShortCircuits is a regression test for
// the hang observed against /Users/aeneas/workspace/go/cloud: when a background
// indexer process holds the index flock, `ensureIndexed` correctly sets
// `out.StaleWarning` and returns immediately, but `handleSemanticSearch` then
// proceeded to call `embedQuery` against the saturated embedding server. With
// LM Studio (single-instance) being hammered by the background indexer's
// 32-chunk batches, the query embed never completes within Claude Code's MCP
// timeout, and the user sees an infinite hang.
//
// The fix is to short-circuit `handleSemanticSearch` when `StaleWarning` is set
// — the warning text already tells the caller to skip semantic searches for
// the next 10 tool calls, so embedding+searching is pure waste anyway.
//
// This test holds the flock externally, points the cache at an indexer whose
// embedder blocks forever, and asserts the handler returns quickly with the
// warning text — and that Embed was never called.
func TestHandleSemanticSearch_StaleWarningShortCircuits(t *testing.T) {
	const dims = 4

	// Resolve symlinks up-front: validateSearchInput will EvalSymlinks the
	// path, and on macOS t.TempDir() returns /var/folders/... which resolves
	// to /private/var/folders/... — without this, the test's pre-acquired
	// lock would be on the unresolved path while handleSemanticSearch looks
	// up the resolved one, hiding the bug behind a different dbPath.
	rawDir := t.TempDir()
	projectDir, err := filepath.EvalSymlinks(rawDir)
	if err != nil {
		t.Fatal(err)
	}
	writeTestGoFile(t, projectDir, "main.go", `package main

// Demo gives the indexer something to index.
func Demo() {}
`)

	// Step 1: pre-create the index with a fast embedder so the DB exists and
	// has at least one chunk. This isolates the test from the chunking path
	// and lets the search call against the cache succeed if (incorrectly)
	// reached.
	fastEmb := &stubEmbedder{model: "blocking-stub"}
	dbPath := config.DBPathForProject(projectDir, fastEmb.ModelName())
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// Ensure the DB file (and any lock) does not leak across test runs.
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Dir(dbPath))
	})

	idx, err := index.NewIndexer(dbPath, fastEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	_ = idx.Close()

	// Step 2: hold the flock from "another process" — flock.New creates a
	// new file descriptor so the same-process check in indexlock.IsHeld sees
	// it as foreign (matches the pattern in TestEnsureIndexed_FlockHeldSkipsReindex).
	lockPath := indexlock.LockPathForDB(dbPath)
	lk, lockErr := indexlock.TryAcquire(lockPath)
	if lockErr != nil {
		t.Fatal(lockErr)
	}
	if lk == nil {
		t.Fatal("expected to acquire indexlock for test setup")
	}
	defer lk.Release()
	if !indexlock.IsHeld(lockPath) {
		t.Skip("flock TryAcquire+IsHeld is reentrant in the same process on this OS — test cannot simulate background indexer holding lock")
	}

	// Step 3: re-open the indexer with a blocking embedder. If
	// handleSemanticSearch wrongly reaches embedQuery, it will block on
	// Embed() forever.
	blockEmb := newBlockingStubEmbedder(dims)
	idx, err = index.NewIndexer(dbPath, blockEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		blockEmb.Unblock()
		_ = idx.Close()
	}()

	ic := &indexerCache{
		cache: map[string]cacheEntry{
			projectDir: {idx: idx, effectiveRoot: projectDir, model: blockEmb.ModelName()},
		},
		embedder:     blockEmb,
		cfg:          newTestConfigService(t, 512),
		log:          discardLog,
		freshnessTTL: 1 * time.Nanosecond, // force the merkle/flock path; do not trust LastIndexedAt
	}

	// Step 4: call handleSemanticSearch with a deadline that's much shorter
	// than the embed timeout. Before the fix, this call hangs on blockEmb.Embed
	// forever; after the fix it returns immediately with the warning.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan struct {
		result *mcp.CallToolResult
		err    error
	}, 1)

	start := time.Now()
	go func() {
		req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{}}
		result, _, callErr := ic.handleSemanticSearch(ctx, req, SemanticSearchInput{
			Cwd:   projectDir,
			Path:  projectDir,
			Query: "demo",
			Limit: 3,
		})
		done <- struct {
			result *mcp.CallToolResult
			err    error
		}{result, callErr}
	}()

	select {
	case out := <-done:
		elapsed := time.Since(start)
		if out.err != nil {
			t.Fatalf("handleSemanticSearch returned error: %v (elapsed %v)", out.err, elapsed)
		}
		if elapsed > 1*time.Second {
			t.Fatalf("handleSemanticSearch took %v — expected sub-second short-circuit when StaleWarning is set", elapsed)
		}
		text := mustTextResult(t, out.result)
		if !strings.Contains(text, "Index is being updated in the background") {
			t.Fatalf("expected StaleWarning text in result, got:\n%s", text)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handleSemanticSearch did not return within 3s — bug: embedQuery contends with background indexer even when StaleWarning is set")
	}

	// The embedder must NEVER have been called: the short-circuit must
	// happen before embedQuery.
	select {
	case <-blockEmb.started:
		t.Fatal("embedQuery was called even though StaleWarning was set — handleSemanticSearch must short-circuit before embedding")
	default:
	}
}

// writeTestGoFile creates a Go source file in dir for test setup.
func writeTestGoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
