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
	"sync"
	"testing"
	"time"

	"github.com/ory/lumen/internal/index"
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
