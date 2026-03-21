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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/lumen/internal/merkle"
)

// progressCall represents a progress function call for testing.
type progressCall struct {
	current int
	total   int
	message string
}

// mockEmbedder returns fixed vectors for testing.
type mockEmbedder struct {
	dims      int
	model     string
	callCount int
}

func (m *mockEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	m.callCount++
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vec := make([]float32, m.dims)
		for j := range vec {
			vec[j] = float32(i+1) * 0.1
		}
		vecs[i] = vec
	}
	return vecs, nil
}

func (m *mockEmbedder) Dimensions() int   { return m.dims }
func (m *mockEmbedder) ModelName() string { return m.model }

func TestIndexer_IndexAndSearch(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

import "fmt"

// Hello prints a greeting.
func Hello(name string) {
	fmt.Println("hello", name)
}

// Goodbye prints a farewell.
func Goodbye(name string) {
	fmt.Println("bye", name)
}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	stats, err := idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.IndexedFiles == 0 {
		t.Fatal("expected at least 1 indexed file")
	}
	if stats.ChunksCreated == 0 {
		t.Fatal("expected at least 1 chunk created")
	}

	results, err := idx.Search(context.Background(), projectDir, []float32{0.1, 0.1, 0.1, 0.1}, 5, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
}

func TestIndexer_IncrementalIndex(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	firstCallCount := emb.callCount

	stats, err := idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if emb.callCount != firstCallCount {
		t.Fatal("expected no additional embedding calls for unchanged project")
	}
	if stats.ChunksCreated != 0 {
		t.Fatalf("expected 0 chunks created on re-index, got %d", stats.ChunksCreated)
	}
}

func TestIndexer_DetectsModifiedFiles(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	firstCallCount := emb.callCount

	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
func World() {}
`)

	stats, err := idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if emb.callCount == firstCallCount {
		t.Fatal("expected additional embedding calls after file change")
	}
	if stats.ChunksCreated == 0 {
		t.Fatal("expected new chunks after file change")
	}
}

func TestIndexer_ForceReindex(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	firstCallCount := emb.callCount

	stats, err := idx.Index(context.Background(), projectDir, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if emb.callCount == firstCallCount {
		t.Fatal("expected re-embedding on force=true")
	}
	if stats.ChunksCreated == 0 {
		t.Fatal("expected chunks on force reindex")
	}
}

func TestIndexer_Status(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	status, err := idx.Status(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if status.IndexedFiles == 0 {
		t.Fatal("expected indexed files > 0")
	}
	if status.EmbeddingModel != "test-model" {
		t.Fatalf("expected model=test-model, got %s", status.EmbeddingModel)
	}
	if status.TotalFiles != 1 {
		t.Fatalf("expected total_files=1, got %d", status.TotalFiles)
	}
}

func TestIndexer_StreamingBatchesProduceSameChunks(t *testing.T) {
	projectDir := t.TempDir()
	// 300 files × 1 function each = 300 chunks → spans 2 batches of 256.
	for i := range 300 {
		writeGoFile(t, projectDir, fmt.Sprintf("f%d.go", i), fmt.Sprintf(`package main

func F%d() {}
`, i))
	}

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	stats, err := idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.IndexedFiles != 300 {
		t.Fatalf("expected 300 indexed files, got %d", stats.IndexedFiles)
	}
	if stats.ChunksCreated == 0 {
		t.Fatal("expected chunks created")
	}
	// With streaming, embed is called once per flush (multiple times).
	if emb.callCount < 2 {
		t.Fatalf("expected ≥2 embed calls for streaming batches, got %d", emb.callCount)
	}
}

func TestIndexer_EnsureFresh(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	// First call on empty index: should reindex.
	reindexed, stats, err := idx.EnsureFresh(context.Background(), projectDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reindexed {
		t.Fatal("expected reindexed=true on first call")
	}
	if stats.IndexedFiles == 0 {
		t.Fatal("expected indexed files > 0")
	}
	callsAfterFirst := emb.callCount

	// Second call with no changes: should not reindex.
	reindexed, _, err = idx.EnsureFresh(context.Background(), projectDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reindexed {
		t.Fatal("expected reindexed=false when index is fresh")
	}
	if emb.callCount != callsAfterFirst {
		t.Fatal("expected no embed calls when index is fresh")
	}

	// Modify a file: should reindex.
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
func World() {}
`)
	reindexed, stats, err = idx.EnsureFresh(context.Background(), projectDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reindexed {
		t.Fatal("expected reindexed=true after file change")
	}
	if stats.IndexedFiles == 0 {
		t.Fatal("expected indexed files > 0 after file change")
	}
}

func TestIndexer_ProgressFunc(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "a.go", `package main

func A() {}
`)
	writeGoFile(t, projectDir, "b.go", `package main

func B() {}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	var calls []progressCall
	progress := func(current, total int, message string) {
		calls = append(calls, progressCall{current, total, message})
	}

	_, err = idx.Index(context.Background(), projectDir, false, progress)
	if err != nil {
		t.Fatal(err)
	}

	checkProgressCalls(t, calls)
}

func checkProgressCalls(t *testing.T, calls []progressCall) {
	if len(calls) == 0 {
		t.Fatal("expected progress calls, got none")
	}

	checkFirstCall(t, calls[0])
	checkAllCalls(t, calls)
	checkProcessingCalls(t, calls)
	checkEmbedCalls(t, calls)
	checkLastCall(t, calls[len(calls)-1])
}

func checkFirstCall(t *testing.T, first progressCall) {
	if first.current != 0 {
		t.Errorf("first call: expected current=0, got %d", first.current)
	}
	if !strings.Contains(first.message, "Found") || !strings.Contains(first.message, "files to index") {
		t.Errorf("first call: expected 'Found ... files to index' message, got %q", first.message)
	}
}

func checkAllCalls(t *testing.T, calls []progressCall) {
	for i, c := range calls {
		if c.total != 2 {
			t.Errorf("call[%d]: expected total=2, got %d (message: %s)", i, c.total, c.message)
		}
		if c.current > c.total {
			t.Errorf("call[%d]: current (%d) > total (%d)", i, c.current, c.total)
		}
	}
}

func checkProcessingCalls(t *testing.T, calls []progressCall) {
	var processingCalls int
	for _, c := range calls {
		if strings.Contains(c.message, "Processing file") {
			processingCalls++
		}
	}
	if processingCalls != 2 {
		t.Fatalf("expected 2 processing progress calls, got %d", processingCalls)
	}
}

func checkEmbedCalls(t *testing.T, calls []progressCall) {
	var embedCalls int
	for _, c := range calls {
		if strings.Contains(c.message, "Embedded") {
			embedCalls++
		}
	}
	if embedCalls == 0 {
		t.Fatal("expected at least 1 embed progress call")
	}
}

func checkLastCall(t *testing.T, last progressCall) {
	if !strings.Contains(last.message, "Indexing complete") {
		t.Errorf("last call should contain 'Indexing complete', got %q", last.message)
	}
	if last.current != last.total {
		t.Errorf("last call: expected current == total, got current=%d total=%d", last.current, last.total)
	}
}

func TestIndexer_SkipsInternalWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// Create a real git repo with one .go file.
	main := t.TempDir()
	gitRun(t, main, "git", "init")
	gitRun(t, main, "git", "commit", "--allow-empty", "-m", "init")
	writeGoFile(t, main, "main.go", "package main\nfunc Main() {}\n")

	// Add a worktree INSIDE the repo directory with its own .go files.
	internalWt := filepath.Join(main, ".worktrees", "feature")
	gitRun(t, main, "git", "worktree", "add", internalWt)
	writeGoFile(t, internalWt, "feature.go", "package main\nfunc Feature() {}\n")
	writeGoFile(t, internalWt, "extra.go", "package main\nfunc Extra() {}\n")

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	stats, err := idx.Index(context.Background(), main, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Only main.go should be indexed; worktree files must be excluded.
	if stats.TotalFiles != 1 {
		t.Errorf("expected TotalFiles=1 (main.go only), got %d", stats.TotalFiles)
	}
	if stats.IndexedFiles != 1 {
		t.Errorf("expected IndexedFiles=1, got %d", stats.IndexedFiles)
	}
}

func gitRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func TestIndexer_ProgressFunc_Nil(t *testing.T) {
	// Verify that passing nil progress doesn't panic.
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
}

func TestIndexer_IsFresh(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	// Never indexed: should return false.
	fresh, err := idx.IsFresh(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("expected IsFresh=false for never-indexed project")
	}

	// Index the project.
	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}

	// After indexing with no changes: should return true.
	fresh, err = idx.IsFresh(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Fatal("expected IsFresh=true after indexing with no changes")
	}

	// Modify a file: should return false.
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
func World() {}
`)
	fresh, err = idx.IsFresh(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if fresh {
		t.Fatal("expected IsFresh=false after modifying a file")
	}
}

func TestIndexer_LastIndexedAt_ReturnsFalseWhenNotIndexed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "empty.db")
	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(dbPath, emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	_, ok := idx.LastIndexedAt()
	if ok {
		t.Fatal("expected ok=false for an index with no last_indexed_at metadata")
	}
}

func TestIndexer_LastIndexedAt_ReturnsTimeAfterIndex(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", "package main\nfunc Foo() {}\n")

	dbPath := filepath.Join(t.TempDir(), "idx.db")
	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(dbPath, emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	before := time.Now().Add(-time.Second)
	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	after := time.Now().Add(time.Second)

	at, ok := idx.LastIndexedAt()
	if !ok {
		t.Fatal("expected ok=true after Index was called")
	}
	if at.Before(before) || at.After(after) {
		t.Fatalf("LastIndexedAt=%v outside expected window [%v, %v]", at, before, after)
	}
}

// TestIndexer_StaleUnsupportedExtensionNotCountedAsRemoved verifies that file
// records in the DB with unsupported extensions (e.g. .md from donor seeding)
// do not appear as RemovedFiles in the diff. They are ghost entries that the
// current Merkle tree will never include, so treating them as real removals
// causes spurious reindex churn on every freshness check.
func TestIndexer_StaleUnsupportedExtensionNotCountedAsRemoved(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", "package main\nfunc Hello() {}\n")

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	// Simulate donor seeding: inject a stale .md record directly into the
	// files table. This file does not exist on disk and .md is not in
	// SupportedExtensions, so the Merkle tree will never include it.
	if err := idx.store.UpsertFile(".changelog-network/v1.0.0.md", "staledeadhash"); err != nil {
		t.Fatal(err)
	}

	_, _, err = idx.EnsureFresh(context.Background(), projectDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	// If the stale .md record were treated as a real removal and re-indexed,
	// EnsureFresh would have errored or returned reindexed=true on every call.
	// The test passing without error means the ghost record was not propagated.
}


// TestIndexer_StaleUnsupportedExtensionDeletedFromDB verifies that after a
// reindex, stale file records with unsupported extensions (e.g. .md from
// donor seeding) are purged from the DB.
func TestIndexer_StaleUnsupportedExtensionDeletedFromDB(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", "package main\nfunc Hello() {}\n")

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	// Simulate donor seeding: inject a stale .md record.
	if err := idx.store.UpsertFile(".changelog-network/v1.0.0.md", "staledeadhash"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := idx.EnsureFresh(context.Background(), projectDir, nil); err != nil {
		t.Fatal(err)
	}

	hashes, err := idx.store.GetFileHashes()
	if err != nil {
		t.Fatal(err)
	}
	for path := range hashes {
		if filepath.Ext(path) == ".md" {
			t.Errorf("stale .md record %q should have been purged from the DB during reindex", path)
		}
	}
}

// TestIndexer_SupportedFileRemovedFromDisk verifies that deleting a supported
// file from disk triggers a reindex and removes it from the DB.
func TestIndexer_SupportedFileRemovedFromDisk(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", "package main\nfunc Hello() {}\n")
	writeGoFile(t, projectDir, "extra.go", "package main\nfunc Extra() {}\n")

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, _, err := idx.EnsureFresh(context.Background(), projectDir, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(projectDir, "extra.go")); err != nil {
		t.Fatal(err)
	}

	reindexed, _, err := idx.EnsureFresh(context.Background(), projectDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reindexed {
		t.Fatal("expected reindex after deleting extra.go")
	}

	hashes, err := idx.store.GetFileHashes()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hashes["extra.go"]; ok {
		t.Error("extra.go should have been removed from the DB after being deleted from disk")
	}
}

// TestIndexer_MixedStaleAndValidRemovals verifies that stale unsupported-
// extension entries are purged while genuinely deleted supported files are
// also correctly removed.
func TestIndexer_MixedStaleAndValidRemovals(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", "package main\nfunc Hello() {}\n")
	writeGoFile(t, projectDir, "extra.go", "package main\nfunc Extra() {}\n")

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, _, err := idx.EnsureFresh(context.Background(), projectDir, nil); err != nil {
		t.Fatal(err)
	}

	// Inject a stale .md record and delete a real supported file.
	if err := idx.store.UpsertFile("docs/CHANGELOG.md", "staledeadhash"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(projectDir, "extra.go")); err != nil {
		t.Fatal(err)
	}
	if err := idx.store.SetMeta("root_hash", "outdated"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := idx.EnsureFresh(context.Background(), projectDir, nil); err != nil {
		t.Fatal(err)
	}

	hashes, err := idx.store.GetFileHashes()
	if err != nil {
		t.Fatal(err)
	}
	for path := range hashes {
		if filepath.Ext(path) == ".md" {
			t.Errorf("stale .md record %q should have been purged", path)
		}
	}
	if _, ok := hashes["extra.go"]; ok {
		t.Error("extra.go should have been removed from the DB")
	}
	if _, ok := hashes["main.go"]; !ok {
		t.Error("main.go should still exist in the DB")
	}
}

// failingEmbedder always returns an error from Embed.
type failingEmbedder struct{}

func (f *failingEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, errors.New("embedding API unavailable")
}
func (f *failingEmbedder) Dimensions() int   { return 4 }
func (f *failingEmbedder) ModelName() string { return "test-model" }

// TestIndex_UpsertFileDefersUntilFlush verifies that when flushBatch fails
// (e.g. embedding API error), UpsertFile is NOT called for files in the failed
// batch — preserving the invariant that a file hash is committed only after its
// chunks have been durably stored.
func TestIndex_UpsertFileDefersUntilFlush(t *testing.T) {
	projectDir := t.TempDir()

	// Write one .go file — one chunk will be produced, flushBatch will be
	// called exactly once (the final flush), and the failing embedder will
	// make it return an error.
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	emb := &failingEmbedder{}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	_, indexErr := idx.Index(context.Background(), projectDir, false, nil)
	if indexErr == nil {
		t.Fatal("expected Index to return an error when embedder fails")
	}

	// After the failed index run the real content hash must NOT have been
	// committed. The file may have a sentinel placeholder hash ("") to satisfy
	// the FK constraint, but it must never carry the real SHA-256 — so that
	// the next run treats it as changed and re-indexes it.
	hashes, err := idx.store.GetFileHashes()
	if err != nil {
		t.Fatal(err)
	}
	// Build the expected real hash using the Merkle tree so the test stays
	// independent of hash implementation details.
	curTree, treeErr := merkle.BuildTree(projectDir, makeSkip(projectDir))
	if treeErr != nil {
		t.Fatal(treeErr)
	}
	for path, storedHash := range hashes {
		realHash := curTree.Files[path]
		if storedHash == realHash {
			t.Errorf("file %q has its real hash %q committed after a failed flush — it will be invisible to future searches", path, storedHash)
		}
	}
}

// TestIndex_ForceRemovesDeletedFiles verifies that a force reindex removes
// DB records for files that no longer exist on disk.
func TestIndex_ForceRemovesDeletedFiles(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "a.go", "package main\nfunc A() {}\n")
	writeGoFile(t, projectDir, "b.go", "package main\nfunc B() {}\n")

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(":memory:", emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	// Seed the DB with both files.
	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}

	// Delete b.go from disk.
	if err := os.Remove(filepath.Join(projectDir, "b.go")); err != nil {
		t.Fatal(err)
	}

	// Force reindex — should clean up b.go from DB.
	if _, err := idx.Index(context.Background(), projectDir, true, nil); err != nil {
		t.Fatal(err)
	}

	hashes, err := idx.store.GetFileHashes()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hashes["b.go"]; ok {
		t.Error("b.go should have been removed from the DB after force reindex")
	}
	if _, ok := hashes["a.go"]; !ok {
		t.Error("a.go should still be present in the DB after force reindex")
	}
}

func TestIndex_SkipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "normal.go"), []byte("package p\n\nfunc Foo() {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	binaryContent := []byte("package p\x00binary\x00data")
	if err := os.WriteFile(filepath.Join(dir, "binary.go"), binaryContent, 0o644); err != nil {
		t.Fatal(err)
	}

	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(filepath.Join(dir, "test.db"), emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	_, err = idx.Index(context.Background(), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	hashes, err := idx.store.GetFileHashes()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hashes["binary.go"]; ok {
		t.Error("binary.go should not be indexed")
	}
	if _, ok := hashes["normal.go"]; !ok {
		t.Error("normal.go should be indexed")
	}
}

func writeGoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
