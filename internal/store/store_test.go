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

package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/lumen/internal/chunker"
)

func TestNewStore_CreatesSchema(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	var count int
	err = s.db.QueryRow("SELECT count(*) FROM files").Scan(&count)
	if err != nil {
		t.Fatalf("files table missing: %v", err)
	}
	err = s.db.QueryRow("SELECT count(*) FROM chunks").Scan(&count)
	if err != nil {
		t.Fatalf("chunks table missing: %v", err)
	}
	err = s.db.QueryRow("SELECT count(*) FROM project_meta").Scan(&count)
	if err != nil {
		t.Fatalf("project_meta table missing: %v", err)
	}
}

func TestStore_SetGetMeta(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	if err := s.SetMeta("test_key", "test_value"); err != nil {
		t.Fatal(err)
	}
	val, err := s.GetMeta("test_key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "test_value" {
		t.Fatalf("expected test_value, got %s", val)
	}
}

func TestStore_UpsertAndSearchVectors(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	if err := s.UpsertFile("main.go", "abc123"); err != nil {
		t.Fatal(err)
	}

	chunks := []chunker.Chunk{
		{ID: "chunk1", FilePath: "main.go", Symbol: "Hello", Kind: "function", StartLine: 1, EndLine: 5},
		{ID: "chunk2", FilePath: "main.go", Symbol: "World", Kind: "function", StartLine: 6, EndLine: 10},
	}
	vectors := [][]float32{
		{0.1, 0.2, 0.3, 0.4},
		{0.9, 0.8, 0.7, 0.6},
	}

	if err := s.InsertChunks(chunks, vectors); err != nil {
		t.Fatal(err)
	}

	query := []float32{0.1, 0.2, 0.3, 0.4}
	results, err := s.Search(context.Background(), query, 2, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Symbol != "Hello" {
		t.Fatalf("expected Hello as closest, got %s", results[0].Symbol)
	}
}

func TestStore_DeleteFileChunks(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	if err := s.UpsertFile("main.go", "abc123"); err != nil {
		t.Fatal(err)
	}
	chunks := []chunker.Chunk{
		{ID: "c1", FilePath: "main.go", Symbol: "Hello", Kind: "function", StartLine: 1, EndLine: 5},
	}
	vectors := [][]float32{{0.1, 0.2, 0.3, 0.4}}
	if err := s.InsertChunks(chunks, vectors); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteFileChunks("main.go"); err != nil {
		t.Fatal(err)
	}

	results, err := s.Search(context.Background(), []float32{0.1, 0.2, 0.3, 0.4}, 10, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results after delete, got %d", len(results))
	}
}

func TestStore_GetFileHashes(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	if err := s.UpsertFile("a.go", "hash_a"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFile("b.go", "hash_b"); err != nil {
		t.Fatal(err)
	}

	hashes, err := s.GetFileHashes()
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 2 {
		t.Fatalf("expected 2 file hashes, got %d", len(hashes))
	}
	if hashes["a.go"] != "hash_a" {
		t.Fatalf("expected hash_a, got %s", hashes["a.go"])
	}
}

func TestStore_Stats(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	if err := s.UpsertFile("main.go", "abc123"); err != nil {
		t.Fatal(err)
	}
	chunks := []chunker.Chunk{
		{ID: "c1", FilePath: "main.go", Symbol: "Hello", Kind: "function", StartLine: 1, EndLine: 5},
		{ID: "c2", FilePath: "main.go", Symbol: "World", Kind: "function", StartLine: 6, EndLine: 10},
	}
	vectors := [][]float32{
		{0.1, 0.2, 0.3, 0.4},
		{0.5, 0.6, 0.7, 0.8},
	}
	if err := s.InsertChunks(chunks, vectors); err != nil {
		t.Fatal(err)
	}

	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 {
		t.Fatalf("expected 1 file, got %d", stats.TotalFiles)
	}
	if stats.TotalChunks != 2 {
		t.Fatalf("expected 2 chunks, got %d", stats.TotalChunks)
	}
}

func TestStore_Pragmas(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	var mode int
	if err := s.db.QueryRow("PRAGMA synchronous").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != 1 { // 1 = NORMAL
		t.Fatalf("expected synchronous=NORMAL(1), got %d", mode)
	}
}

func TestStore_ChunkIndexesExist(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	var count int
	err = s.db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND name = 'idx_chunks_file_path'`,
	).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected idx_chunks_file_path to exist, got %d", count)
	}
}

func TestStore_GetMetaBatch(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	_ = s.SetMeta("key1", "val1")
	_ = s.SetMeta("key2", "val2")

	vals, err := s.GetMetaBatch([]string{"key1", "key2", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if vals["key1"] != "val1" {
		t.Fatalf("expected val1, got %s", vals["key1"])
	}
	if vals["key2"] != "val2" {
		t.Fatalf("expected val2, got %s", vals["key2"])
	}
	if _, ok := vals["missing"]; ok {
		t.Fatal("expected missing key to be absent")
	}
}

func TestStore_DimensionMismatchRecreatesTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Create store with 4 dimensions and insert data.
	s1, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := s1.UpsertFile("main.go", "abc"); err != nil {
		t.Fatal(err)
	}
	if err := s1.InsertChunks(
		[]chunker.Chunk{{ID: "c1", FilePath: "main.go", Symbol: "Foo", Kind: "function", StartLine: 1, EndLine: 5}},
		[][]float32{{0.1, 0.2, 0.3, 0.4}},
	); err != nil {
		t.Fatal(err)
	}
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen with different dimensions — should reset data.
	s2, err := New(dbPath, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s2.Close() }()

	// Old data should be gone.
	stats, err := s2.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 0 {
		t.Fatalf("expected 0 files after dimension change, got %d", stats.TotalFiles)
	}
	if stats.TotalChunks != 0 {
		t.Fatalf("expected 0 chunks after dimension change, got %d", stats.TotalChunks)
	}

	// Should be able to insert with new dimensions.
	if err := s2.UpsertFile("main.go", "def"); err != nil {
		t.Fatal(err)
	}
	if err := s2.InsertChunks(
		[]chunker.Chunk{{ID: "c2", FilePath: "main.go", Symbol: "Bar", Kind: "function", StartLine: 1, EndLine: 5}},
		[][]float32{{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}},
	); err != nil {
		t.Fatal(err)
	}

	results, err := s2.Search(context.Background(), []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}, 10, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Symbol != "Bar" {
		t.Fatalf("expected 1 result with symbol Bar, got %v", results)
	}
}

func TestStore_SearchWithPathPrefix(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	// Insert files in two different directories.
	if err := s.UpsertFile("src/main.go", "hash1"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFile("tests/foo_test.go", "hash2"); err != nil {
		t.Fatal(err)
	}

	chunks := []chunker.Chunk{
		{ID: "c1", FilePath: "src/main.go", Symbol: "Main", Kind: "function", StartLine: 1, EndLine: 5},
		{ID: "c2", FilePath: "tests/foo_test.go", Symbol: "TestFoo", Kind: "function", StartLine: 1, EndLine: 5},
	}
	// Use identical vectors so distance doesn't filter anything.
	vec := []float32{0.1, 0.2, 0.3, 0.4}
	vectors := [][]float32{vec, vec}

	if err := s.InsertChunks(chunks, vectors); err != nil {
		t.Fatal(err)
	}

	// Without prefix: both results returned.
	all, err := s.Search(context.Background(), vec, 10, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 results without prefix, got %d", len(all))
	}

	// With prefix "src": only src/main.go result.
	srcResults, err := s.Search(context.Background(), vec, 10, 0, "src")
	if err != nil {
		t.Fatal(err)
	}
	if len(srcResults) != 1 {
		t.Fatalf("expected 1 result with prefix 'src', got %d", len(srcResults))
	}
	if srcResults[0].FilePath != "src/main.go" {
		t.Fatalf("expected src/main.go, got %s", srcResults[0].FilePath)
	}

	// With prefix "tests": only tests/foo_test.go result.
	testResults, err := s.Search(context.Background(), vec, 10, 0, "tests")
	if err != nil {
		t.Fatal(err)
	}
	if len(testResults) != 1 {
		t.Fatalf("expected 1 result with prefix 'tests', got %d", len(testResults))
	}
	if testResults[0].FilePath != "tests/foo_test.go" {
		t.Fatalf("expected tests/foo_test.go, got %s", testResults[0].FilePath)
	}
}

func TestResetAndRecreateVecTable_Transactional(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reset_test.db")

	// Create store with 4 dimensions and insert data.
	s1, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := s1.UpsertFile("a.go", "hash_a"); err != nil {
		t.Fatal(err)
	}
	if err := s1.InsertChunks(
		[]chunker.Chunk{{ID: "c1", FilePath: "a.go", Symbol: "Alpha", Kind: "function", StartLine: 1, EndLine: 3}},
		[][]float32{{0.1, 0.2, 0.3, 0.4}},
	); err != nil {
		t.Fatal(err)
	}
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}

	// Trigger dimension reset by opening with different dimensions.
	s2, err := New(dbPath, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s2.Close() }()

	// chunks and files tables must be empty — no orphaned data.
	var chunkCount, fileCount int
	if err := s2.db.QueryRow("SELECT count(*) FROM chunks").Scan(&chunkCount); err != nil {
		t.Fatalf("query chunks: %v", err)
	}
	if err := s2.db.QueryRow("SELECT count(*) FROM files").Scan(&fileCount); err != nil {
		t.Fatalf("query files: %v", err)
	}
	if chunkCount != 0 {
		t.Fatalf("expected 0 chunks after dimension reset, got %d", chunkCount)
	}
	if fileCount != 0 {
		t.Fatalf("expected 0 files after dimension reset, got %d", fileCount)
	}

	// vec_chunks must exist with the new dimension.
	exists, err := checkTableExists(s2.db, "vec_chunks")
	if err != nil {
		t.Fatalf("checkTableExists: %v", err)
	}
	if !exists {
		t.Fatal("expected vec_chunks to exist after dimension reset")
	}

	// Should be able to insert and search with new dimensions.
	if err := s2.UpsertFile("b.go", "hash_b"); err != nil {
		t.Fatal(err)
	}
	if err := s2.InsertChunks(
		[]chunker.Chunk{{ID: "c2", FilePath: "b.go", Symbol: "Beta", Kind: "function", StartLine: 1, EndLine: 3}},
		[][]float32{{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}},
	); err != nil {
		t.Fatal(err)
	}
	results, err := s2.Search(context.Background(), []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}, 10, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Symbol != "Beta" {
		t.Fatalf("expected 1 result with symbol Beta after reset, got %v", results)
	}
}

func TestStore_SearchPathPrefixNoFalsePositives(t *testing.T) {
	s, err := New(":memory:", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	// "internal/store" and "internal/storefront" share a common prefix string —
	// the LIKE pattern must not match the latter when filtering by the former.
	for _, path := range []string{"internal/store/store.go", "internal/storefront/main.go"} {
		if err := s.UpsertFile(path, "hash"); err != nil {
			t.Fatal(err)
		}
	}
	chunks := []chunker.Chunk{
		{ID: "c1", FilePath: "internal/store/store.go", Symbol: "New", Kind: "function", StartLine: 1, EndLine: 5},
		{ID: "c2", FilePath: "internal/storefront/main.go", Symbol: "Handler", Kind: "function", StartLine: 1, EndLine: 5},
	}
	vec := []float32{0.1, 0.2, 0.3, 0.4}
	if err := s.InsertChunks(chunks, [][]float32{vec, vec}); err != nil {
		t.Fatal(err)
	}

	results, err := s.Search(context.Background(), vec, 10, 0, "internal/store")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].FilePath != "internal/store/store.go" {
		t.Fatalf("expected internal/store/store.go, got %s", results[0].FilePath)
	}
}

func TestStore_FileBasedReadConnection(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dsn, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	// Verify read connection is set for file-based databases.
	if s.readDB == nil {
		t.Fatal("expected readDB to be set for file-based store")
	}

	// Write data via write connection.
	if err := s.UpsertFile("main.go", "abc123"); err != nil {
		t.Fatal(err)
	}
	chunks := []chunker.Chunk{
		{ID: "chunk1", FilePath: "main.go", Symbol: "Hello", Kind: "function", StartLine: 1, EndLine: 5},
		{ID: "chunk2", FilePath: "main.go", Symbol: "World", Kind: "function", StartLine: 6, EndLine: 10},
	}
	vectors := [][]float32{
		{0.1, 0.2, 0.3, 0.4},
		{0.9, 0.8, 0.7, 0.6},
	}
	if err := s.InsertChunks(chunks, vectors); err != nil {
		t.Fatal(err)
	}

	// Search uses the read connection — should see committed data via WAL.
	results, err := s.Search(context.Background(), []float32{0.1, 0.2, 0.3, 0.4}, 2, 0, "")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results from read connection, got %d", len(results))
	}

	// Verify that the read connection is query-only (cannot write).
	_, err = s.readDB.Exec("INSERT INTO project_meta (key, value) VALUES ('test', 'val')")
	if err == nil {
		t.Fatal("expected write on read connection to fail (query_only=ON)")
	}
}

func TestReadMetaAt_ReturnsStoredValue(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	s, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetMeta("project_path", "/some/project"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := ReadMetaAt(dbPath, "project_path")
	if err != nil {
		t.Fatalf("ReadMetaAt: %v", err)
	}
	if got["project_path"] != "/some/project" {
		t.Fatalf("ReadMetaAt(project_path) = %q, want %q", got["project_path"], "/some/project")
	}
}

// TestReadMetaAt_MultipleKeys verifies a single read-only open can fetch every
// key `lumen clean` needs to classify an index.
func TestReadMetaAt_MultipleKeys(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	s, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetMeta("project_path", "/some/project"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMeta("last_indexed_at", "2026-01-02T03:04:05Z"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := ReadMetaAt(dbPath, "project_path", "last_accessed_at", "last_indexed_at")
	if err != nil {
		t.Fatalf("ReadMetaAt: %v", err)
	}
	if got["project_path"] != "/some/project" {
		t.Errorf("project_path = %q, want %q", got["project_path"], "/some/project")
	}
	if got["last_indexed_at"] != "2026-01-02T03:04:05Z" {
		t.Errorf("last_indexed_at = %q, want %q", got["last_indexed_at"], "2026-01-02T03:04:05Z")
	}
	if got["last_accessed_at"] == "" {
		t.Error("last_accessed_at should be stamped by New")
	}
}

func TestReadMetaAt_MissingKeyReturnsEmpty(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	s, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := ReadMetaAt(dbPath, "project_path")
	if err != nil {
		t.Fatalf("ReadMetaAt on empty meta: %v", err)
	}
	if _, ok := got["project_path"]; ok {
		t.Fatalf("ReadMetaAt missing key = %q, want absent", got["project_path"])
	}
}

func TestReadMetaAt_MissingFileReturnsError(t *testing.T) {
	_, err := ReadMetaAt(filepath.Join(t.TempDir(), "does-not-exist.db"), "project_path")
	if err == nil {
		t.Fatal("expected error for missing DB file")
	}
}

// TestNew_StampsLastAccessedAt verifies that opening a store records an
// RFC3339 UTC access timestamp, which `lumen clean` uses to decide whether an
// index is still in use.
func TestNew_StampsLastAccessedAt(t *testing.T) {
	before := time.Now().UTC().Truncate(time.Second)
	dbPath := filepath.Join(t.TempDir(), "index.db")
	s, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	got, err := s.GetMeta("last_accessed_at")
	if err != nil {
		t.Fatalf("GetMeta(last_accessed_at): %v", err)
	}
	if !strings.HasSuffix(got, "Z") {
		t.Errorf("last_accessed_at = %q, want a UTC (Z-suffixed) timestamp", got)
	}
	ts, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("parse last_accessed_at %q: %v", got, err)
	}
	if ts.Before(before) {
		t.Errorf("last_accessed_at = %v, want at or after %v", ts, before)
	}
}

// TestNew_RefreshesLastAccessedAt verifies that reopening an existing index
// bumps the recorded access time rather than leaving a stale value behind.
func TestNew_RefreshesLastAccessedAt(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	s, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	stale := "2020-01-01T00:00:00Z"
	if err := s.SetMeta("last_accessed_at", stale); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()

	got, err := reopened.GetMeta("last_accessed_at")
	if err != nil {
		t.Fatalf("GetMeta(last_accessed_at): %v", err)
	}
	if got == stale {
		t.Error("reopening the store should refresh last_accessed_at")
	}
}

// TestNew_AccessStampDoesNotBlockOnBusyDatabase verifies that opening an index
// while another writer holds the SQLite write lock — the normal state during a
// background reindex — still returns promptly. The access stamp is bookkeeping;
// blocking on it for the 120s busy_timeout would stall search.
func TestNew_AccessStampDoesNotBlockOnBusyDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	s, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	// Hold the write lock from an independent connection, as a concurrently
	// running indexer process would.
	blocker, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Close() }()
	tx, err := blocker.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("INSERT INTO project_meta (key, value) VALUES ('blocker', '1')"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	done := make(chan error, 1)
	start := time.Now()
	go func() {
		reopened, err := New(dbPath, 4)
		if reopened != nil {
			_ = reopened.Close()
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("New on a busy database: %v", err)
		}
		if elapsed := time.Since(start); elapsed > 30*time.Second {
			t.Errorf("New took %v on a busy database, want a bounded wait", elapsed)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("New blocked on a busy database; the access stamp must not wait for the write lock")
	}
}

// TestReadMetaAt_DoesNotRefreshLastAccessedAt guards the invariant that the
// read-only metadata scan used by `lumen clean` never counts as an access.
func TestReadMetaAt_DoesNotRefreshLastAccessedAt(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	s, err := New(dbPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	stale := "2020-01-01T00:00:00Z"
	if err := s.SetMeta("last_accessed_at", stale); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadMetaAt(dbPath, "last_accessed_at"); err != nil {
		t.Fatalf("ReadMetaAt: %v", err)
	}

	got, err := ReadMetaAt(dbPath, "last_accessed_at")
	if err != nil {
		t.Fatalf("ReadMetaAt: %v", err)
	}
	if got["last_accessed_at"] != stale {
		t.Errorf("last_accessed_at = %q, want unchanged %q", got["last_accessed_at"], stale)
	}
}
