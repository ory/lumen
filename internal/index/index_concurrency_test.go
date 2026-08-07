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

package index

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test infrastructure: blockingEmbedder
// ---------------------------------------------------------------------------

// blockingEmbedder is an embedder that blocks on Embed() until Unblock() is
// called. It signals via the started channel when the first Embed call enters,
// which — because Embed is only called inside indexWithTree under idx.mu.Lock —
// proves the write lock is held.
type blockingEmbedder struct {
	dims      int
	model     string
	started   chan struct{} // closed on first Embed() call
	blockCh   chan struct{} // close to unblock all Embed() calls
	startOnce sync.Once
	unblkOnce sync.Once
}

func newBlockingEmbedder(dims int) *blockingEmbedder {
	return &blockingEmbedder{
		dims:    dims,
		model:   "blocking-test",
		started: make(chan struct{}),
		blockCh: make(chan struct{}),
	}
}

func (e *blockingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
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

func (e *blockingEmbedder) Dimensions() int   { return e.dims }
func (e *blockingEmbedder) ModelName() string { return e.model }
func (e *blockingEmbedder) Unblock()          { e.unblkOnce.Do(func() { close(e.blockCh) }) }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// indexInitialData creates an Indexer backed by a file-based DB, indexes the
// given project directory with a fast (non-blocking) embedder, closes the
// indexer, and returns the DB path so callers can re-open with a different
// embedder.
func indexInitialData(t *testing.T, projectDir string, dims int) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	fastEmb := &mockEmbedder{dims: dims, model: "fast-test"}
	idx, err := NewIndexer(dbPath, fastEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatalf("initial index: %v", err)
	}
	if stats.IndexedFiles == 0 {
		t.Fatal("initial index produced 0 files — test setup is broken")
	}
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}
	return dbPath
}

// ---------------------------------------------------------------------------
// RED tests — these demonstrate the blocking bug and must FAIL before the fix.
// ---------------------------------------------------------------------------

// TestSearch_DoesNotBlockDuringIndexing is the core regression test.
//
// It pre-populates an index, then starts a re-index with a blocking embedder
// (which holds idx.mu.Lock via Index → indexWithTree → flushBatch → Embed).
// A concurrent Search() call must return stale results within 3 seconds.
//
// Before the fix this test FAILS because Search acquires idx.mu.RLock which
// contends with the write lock held by the background Index goroutine.
func TestSearch_DoesNotBlockDuringIndexing(t *testing.T) {
	const dims = 4

	// 1. Seed the project with one file and index it.
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "existing.go", `package main

// Existing is here so we have searchable data.
func Existing() {}
`)
	dbPath := indexInitialData(t, projectDir, dims)

	// 2. Re-open with a blocking embedder and add a new file to trigger re-index.
	blockEmb := newBlockingEmbedder(dims)
	idx, err := NewIndexer(dbPath, blockEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		blockEmb.Unblock()
		_ = idx.Close()
	}()

	writeGoFile(t, projectDir, "new.go", `package main

// NewFunc triggers a re-index because merkle hash changes.
func NewFunc() {}
`)

	// 3. Start indexing in background — will acquire write lock, then block on Embed.
	indexDone := make(chan struct{})
	go func() {
		defer close(indexDone)
		_, _ = idx.Index(context.Background(), projectDir, false, nil)
	}()

	// 4. Wait for Embed to be called → write lock is definitely held.
	select {
	case <-blockEmb.started:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for background Index to call Embed")
	}

	// 5. Search must return within 3 seconds with stale results.
	type searchResult struct {
		count int
		err   error
	}
	searchDone := make(chan searchResult, 1)
	go func() {
		results, err := idx.Search(context.Background(), projectDir, []float32{0.1, 0.1, 0.1, 0.1}, 10, 0, "")
		searchDone <- searchResult{count: len(results), err: err}
	}()

	select {
	case sr := <-searchDone:
		if sr.err != nil {
			t.Fatalf("Search returned error: %v", sr.err)
		}
		// We should get stale results from the initial index.
		if sr.count == 0 {
			t.Fatal("expected stale results from initial index, got 0")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Search() blocked for >3 s while Index() held the write lock.\n" +
			"Bug: idx.mu.RLock() in Search contends with idx.mu.Lock() in Index.")
	}

	// Cleanup: unblock and wait.
	blockEmb.Unblock()
	<-indexDone
}

// TestSearch_ReturnsStaleDataDuringReindex verifies that a search during an
// active re-index returns the previously-indexed data (stale but correct),
// not an empty result set.
//
// The key insight: during incremental re-index, Index() calls
// DeleteFileChunks(relPath) for changed files BEFORE calling Embed(). So
// chunks for the changed file are gone by the time the embedder blocks.
// However, unchanged files are untouched. We verify that search returns
// results from the unchanged file while the changed file is being re-indexed.
func TestSearch_ReturnsStaleDataDuringReindex(t *testing.T) {
	const dims = 4

	projectDir := t.TempDir()

	// stable.go is indexed initially and never modified, so its chunks
	// survive the incremental re-index triggered by changes to greet.go.
	writeGoFile(t, projectDir, "stable.go", `package main

// Stable is an unchanged function that survives re-indexing.
func Stable() {}
`)
	writeGoFile(t, projectDir, "greet.go", `package main

// Greet says hello.
func Greet(name string) {}
`)
	dbPath := indexInitialData(t, projectDir, dims)

	blockEmb := newBlockingEmbedder(dims)
	idx, err := NewIndexer(dbPath, blockEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		blockEmb.Unblock()
		_ = idx.Close()
	}()

	// Modify greet.go so the merkle hash changes and triggers a re-index.
	// stable.go is untouched — its chunks remain in the DB.
	writeGoFile(t, projectDir, "greet.go", `package main

// Greet says hello.
func Greet(name string) {}

// Farewell says goodbye.
func Farewell(name string) {}
`)

	indexDone := make(chan struct{})
	go func() {
		defer close(indexDone)
		_, _ = idx.Index(context.Background(), projectDir, false, nil)
	}()

	select {
	case <-blockEmb.started:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for Embed")
	}

	type searchResult struct {
		files map[string]bool
		err   error
	}
	searchDone := make(chan searchResult, 1)
	go func() {
		results, err := idx.Search(context.Background(), projectDir, []float32{0.1, 0.1, 0.1, 0.1}, 10, 0, "")
		files := make(map[string]bool)
		for _, r := range results {
			files[r.FilePath] = true
		}
		searchDone <- searchResult{files: files, err: err}
	}()

	select {
	case sr := <-searchDone:
		if sr.err != nil {
			t.Fatalf("Search error: %v", sr.err)
		}
		// stable.go was not modified, so its chunks survive the incremental
		// re-index and should appear in stale results.
		if !sr.files["stable.go"] {
			t.Fatalf("expected stale results to include stable.go, got files: %v", sr.files)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Search blocked for >3 s during re-index")
	}

	blockEmb.Unblock()
	<-indexDone
}

// TestParallelSearches_CompleteDuringIndexing verifies that N concurrent
// search queries all complete within a bounded time while indexing is in
// progress. Before the fix, they all pile up behind the write lock.
func TestParallelSearches_CompleteDuringIndexing(t *testing.T) {
	const dims = 4
	const parallelSearches = 5

	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "calc.go", `package main

// Add returns a + b.
func Add(a, b int) int { return a + b }
`)
	dbPath := indexInitialData(t, projectDir, dims)

	blockEmb := newBlockingEmbedder(dims)
	idx, err := NewIndexer(dbPath, blockEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		blockEmb.Unblock()
		_ = idx.Close()
	}()

	writeGoFile(t, projectDir, "sub.go", `package main

// Sub returns a - b.
func Sub(a, b int) int { return a - b }
`)

	indexDone := make(chan struct{})
	go func() {
		defer close(indexDone)
		_, _ = idx.Index(context.Background(), projectDir, false, nil)
	}()

	select {
	case <-blockEmb.started:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for Embed")
	}

	// Launch N parallel searches. Each runs in a separate goroutine;
	// we monitor completion with a per-search channel and a 3s deadline.
	// NOTE: Before the fix, Search ignores context and blocks on RLock,
	// so we cannot use wg.Go (which waits for return). Instead we use
	// a time-bounded select pattern and unblock the embedder to let any
	// stuck goroutines drain after the test assertion.
	type searchOutcome struct {
		idx int
		err error
	}
	outcomes := make(chan searchOutcome, parallelSearches)
	for i := range parallelSearches {
		go func() {
			_, searchErr := idx.Search(context.Background(), projectDir, []float32{0.1, 0.1, 0.1, 0.1}, 10, 0, "")
			outcomes <- searchOutcome{idx: i, err: searchErr}
		}()
	}

	// Collect results with a 3s deadline.
	deadline := time.After(3 * time.Second)
	completed := 0
	var failures []string
	for completed < parallelSearches {
		select {
		case out := <-outcomes:
			completed++
			if out.err != nil {
				failures = append(failures, fmt.Sprintf("search %d error: %v", out.idx, out.err))
			}
		case <-deadline:
			failures = append(failures, fmt.Sprintf("%d of %d searches blocked >3 s", parallelSearches-completed, parallelSearches))
			goto done
		}
	}
done:
	if len(failures) > 0 {
		t.Fatalf("parallel searches failed/blocked during indexing:\n%v", failures)
	}

	blockEmb.Unblock()
	<-indexDone
}

func TestConcurrentProjectSelectionKeepsSearchAndStatusScoped(t *testing.T) {
	const dims = 4
	projectA := t.TempDir()
	projectB := t.TempDir()
	writeGoFile(t, projectA, "alpha.go", `package alpha

func Alpha() {}
`)
	writeGoFile(t, projectB, "beta.go", `package beta

func Beta() {}
`)

	idx, err := NewIndexer(filepath.Join(t.TempDir(), "shared.db"), &mockEmbedder{dims: dims, model: "project-race"}, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()
	if _, err := idx.Index(context.Background(), projectA, false, nil); err != nil {
		t.Fatalf("index project A: %v", err)
	}
	if _, err := idx.Index(context.Background(), projectB, false, nil); err != nil {
		t.Fatalf("index project B: %v", err)
	}

	type projectCase struct {
		path string
		file string
	}
	cases := []projectCase{{path: projectA, file: "alpha.go"}, {path: projectB, file: "beta.go"}}
	start := make(chan struct{})
	errs := make(chan error, 40)
	var wg sync.WaitGroup
	for i := range 40 {
		tc := cases[i%len(cases)]
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results, searchErr := idx.Search(context.Background(), tc.path, []float32{0.1, 0.1, 0.1, 0.1}, 5, 0, "")
			if searchErr != nil {
				errs <- fmt.Errorf("search %s: %w", tc.path, searchErr)
				return
			}
			if len(results) == 0 || results[0].FilePath != tc.file {
				errs <- fmt.Errorf("search %s returned wrong membership: %+v", tc.path, results)
				return
			}
			status, statusErr := idx.Status(tc.path)
			if statusErr != nil {
				errs <- fmt.Errorf("status %s: %w", tc.path, statusErr)
				return
			}
			if status.ProjectPath != tc.path || status.IndexedFiles != 1 {
				errs <- fmt.Errorf("status %s returned wrong membership: %+v", tc.path, status)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
}

// TestSearch_RespectsContextCancellation verifies that a blocked Search call
// returns promptly when its context is cancelled.
func TestSearch_RespectsContextCancellation(t *testing.T) {
	const dims = 4

	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "ctx.go", `package main

// DoWork does work.
func DoWork() {}
`)
	dbPath := indexInitialData(t, projectDir, dims)

	blockEmb := newBlockingEmbedder(dims)
	idx, err := NewIndexer(dbPath, blockEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		blockEmb.Unblock()
		_ = idx.Close()
	}()

	writeGoFile(t, projectDir, "ctx2.go", `package main

// MoreWork does more work.
func MoreWork() {}
`)

	indexDone := make(chan struct{})
	go func() {
		defer close(indexDone)
		_, _ = idx.Index(context.Background(), projectDir, false, nil)
	}()

	select {
	case <-blockEmb.started:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for Embed")
	}

	// Search with a 500ms deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	searchDone := make(chan error, 1)
	go func() {
		_, err := idx.Search(ctx, projectDir, []float32{0.1, 0.1, 0.1, 0.1}, 10, 0, "")
		searchDone <- err
	}()

	select {
	case err := <-searchDone:
		elapsed := time.Since(start)
		// We accept either:
		// - context.DeadlineExceeded (if search respects the context), or
		// - nil with results (if search doesn't block at all — the fixed behavior)
		if err != nil && err != context.DeadlineExceeded && ctx.Err() == nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if elapsed > 2*time.Second {
			t.Fatalf("Search took %v despite context cancellation; should have returned within ~500ms", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Search blocked >3 s and ignored context cancellation")
	}

	blockEmb.Unblock()
	<-indexDone
}

// TestSearch_EmptyIndexDuringFirstIndex verifies that Search on a completely
// empty database (first-ever index in progress) returns an empty result set,
// not an error or a hang.
func TestSearch_EmptyIndexDuringFirstIndex(t *testing.T) {
	const dims = 4

	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "first.go", `package main

// First is the first function ever.
func First() {}
`)

	// Don't pre-populate — this IS the first index.
	dbPath := filepath.Join(t.TempDir(), "empty.db")
	blockEmb := newBlockingEmbedder(dims)
	idx, err := NewIndexer(dbPath, blockEmb, 512)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		blockEmb.Unblock()
		_ = idx.Close()
	}()

	indexDone := make(chan struct{})
	go func() {
		defer close(indexDone)
		_, _ = idx.Index(context.Background(), projectDir, false, nil)
	}()

	select {
	case <-blockEmb.started:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for Embed")
	}

	searchDone := make(chan error, 1)
	go func() {
		_, err := idx.Search(context.Background(), projectDir, []float32{0.1, 0.1, 0.1, 0.1}, 10, 0, "")
		// Results from an empty DB before indexing completes are unexpected
		// but acceptable — it means the fix works better than expected.
		searchDone <- err
	}()

	select {
	case err := <-searchDone:
		// nil is fine (empty results), context error is fine too.
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Search on empty index returned unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Search blocked >3 s on empty index during first-ever indexing")
	}

	blockEmb.Unblock()
	<-indexDone
}
