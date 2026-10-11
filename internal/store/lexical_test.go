// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ory/lumen/internal/chunker"
)

func lexicalStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenLexicalCollection(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func putLexical(t *testing.T, s *Store, path, hash, symbol, body string) {
	t.Helper()
	_, err := s.StoreFileRevision(path, hash, []chunker.Chunk{{ID: hash, FilePath: path, Symbol: symbol, Content: body, Kind: "function", StartLine: 1, EndLine: 3, Signature: "func " + symbol + "()", References: []string{"reservePayment"}, Imports: []string{"./store"}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLexicalSearchFieldsIdentifiersAndLiteralQuery(t *testing.T) {
	s := lexicalStore(t)
	putLexical(t, s, "auth/token.go", "aa", "refreshAccessToken", "return fetchCredential()")
	putLexical(t, s, "notes.go", "bb", "notes", "refreshAccessToken refreshAccessToken refreshAccessToken")
	for _, query := range []string{"refreshAccessToken", "refresh access token", `refreshAccessToken OR " : *`, "auth/token.go"} {
		results, err := s.SearchLexical(context.Background(), query, 10, "")
		if err != nil {
			t.Fatalf("%q: %v", query, err)
		}
		if len(results) == 0 || results[0].Symbol != "refreshAccessToken" {
			t.Fatalf("%q: %+v", query, results)
		}
		if results[0].ContentHash != "aa" || len(results[0].References) != 1 || len(results[0].Imports) != 1 {
			t.Fatalf("lost metadata: %+v", results[0])
		}
	}
	results, err := s.SearchLexical(context.Background(), `"():*`, 10, "")
	if err != nil || len(results) != 0 {
		t.Fatalf("punctuation: %+v %v", results, err)
	}
	var body any
	if err := s.db.QueryRow(`SELECT body FROM lexical_chunks LIMIT 1`).Scan(&body); err != nil || body != nil {
		t.Fatalf("source text retained: %v %v", body, err)
	}
	var vectors bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE name='vec_vectors')`).Scan(&vectors); err != nil || vectors {
		t.Fatalf("vector table exists: %v %v", vectors, err)
	}
}

func TestLexicalMembershipFilteringAndRevisionGC(t *testing.T) {
	s := lexicalStore(t)
	if err := s.UseProject("/a"); err != nil {
		t.Fatal(err)
	}
	putLexical(t, s, "target/a.go", "aa", "needle", "needle")
	putLexical(t, s, "target_%/a.go", "bb", "needle", "needle")
	for i := range 40 {
		putLexical(t, s, filepath.Join("other", string(rune('A'+i))+".go"), "cc", "needle", "needle needle needle")
	}
	if err := s.UseProject("/b"); err != nil {
		t.Fatal(err)
	}
	attached, err := s.AttachExistingFileRevision("target/a.go", "aa")
	if err != nil || !attached {
		t.Fatalf("attach: %v %v", attached, err)
	}
	results, err := s.SearchLexical(context.Background(), "needle", 1, "target")
	if err != nil || len(results) != 1 || results[0].FilePath != "target/a.go" {
		t.Fatalf("membership/filter before limit: %+v %v", results, err)
	}
	if err := s.UseProject("/a"); err != nil {
		t.Fatal(err)
	}
	results, err = s.SearchLexical(context.Background(), "needle", 1, "target_%")
	if err != nil || len(results) != 1 || results[0].FilePath != "target_%/a.go" {
		t.Fatalf("literal path: %+v %v", results, err)
	}
	putLexical(t, s, "target/a.go", "dd", "newSymbol", "fresh")
	if err := s.UseProject("/b"); err != nil {
		t.Fatal(err)
	}
	results, err = s.SearchLexical(context.Background(), "needle", 10, "")
	if err != nil || len(results) != 1 || results[0].ContentHash != "aa" {
		t.Fatalf("shared revision lost: %+v %v", results, err)
	}
	if err := s.DeleteFileChunks("target/a.go"); err != nil {
		t.Fatal(err)
	}
	var oldRows int
	if err := s.db.QueryRow(`SELECT count(*) FROM file_revisions WHERE content_hash=?`, hashBlob("aa")).Scan(&oldRows); err != nil || oldRows != 0 {
		t.Fatalf("orphan revision: %d %v", oldRows, err)
	}
	if _, err := s.SearchLexical(context.Background(), "needle", 10, ""); err != nil {
		t.Fatal(err)
	}
}

func TestLexicalCleanupAndReopen(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "lexical.db")
	s, err := OpenLexicalCollection(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UseProject(root); err != nil {
		t.Fatal(err)
	}
	putLexical(t, s, "a.go", "aa", "a", "needle")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	stats, shared, err := CleanupCollectionAt(dbPath, time.Now().Add(-time.Hour))
	if err != nil || !shared || stats.ProjectsLeft != 1 {
		t.Fatalf("cleanup: %+v %v %v", stats, shared, err)
	}
	s, err = OpenLexicalCollection(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if err := s.UseProject(root); err != nil {
		t.Fatal(err)
	}
	chunks, err := s.ListProjectChunks("")
	if err != nil || len(chunks) != 1 {
		t.Fatalf("reopened metadata: %+v %v", chunks, err)
	}
	collection, err := s.CollectionStats()
	if err != nil || collection.ProjectChunks != 1 || collection.UniqueVectors != 1 || collection.VectorStorage != "lexical" {
		t.Fatalf("stats: %+v %v", collection, err)
	}
	stats, err = s.CleanupStaleProjects(time.Now().Add(time.Hour))
	if err != nil || stats.VectorsRemoved != 1 {
		t.Fatalf("gc: %+v %v", stats, err)
	}
}

func TestIdentifierText(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"refreshAccessToken", "refreshAccessToken refresh Access Token"},
		{"HTTPServer", "HTTPServer HTTP Server"},
		{"refresh_access_token", "refresh_access_token refresh access token"},
		{"sha256Hash", "sha256Hash sha 256 Hash"},
	} {
		if got := identifierText(test.input); got != test.want {
			t.Errorf("%s: got %q want %q", test.input, got, test.want)
		}
	}
}

func TestLexicalFieldWeightsAndUnsharedRevisionReplacement(t *testing.T) {
	s := lexicalStore(t)
	cases := []chunker.Chunk{
		{ID: "body", FilePath: "a.go", Symbol: "body", Content: "deduplicate neutral neutral neutral", StartLine: 1, EndLine: 1},
		{ID: "comment", FilePath: "b.go", Symbol: "comment", Comments: "deduplicate neutral neutral neutral", StartLine: 1, EndLine: 1},
		{ID: "signature", FilePath: "c.go", Symbol: "signature", Signature: "deduplicate neutral neutral neutral", StartLine: 1, EndLine: 1},
	}
	for i, c := range cases {
		if _, err := s.StoreFileRevision(c.FilePath, string(rune('a'+i)), []chunker.Chunk{c}, nil); err != nil {
			t.Fatal(err)
		}
	}
	results, err := s.SearchLexical(context.Background(), "deduplicate", 3, "")
	if err != nil || len(results) != 3 {
		t.Fatalf("results: %+v %v", results, err)
	}
	if results[0].Symbol != "signature" || results[1].Symbol != "comment" || results[2].Symbol != "body" {
		t.Fatalf("field weighting: %+v", results)
	}
	putLexical(t, s, "a.go", "dd", "replacement", "fresh")
	results, err = s.SearchLexical(context.Background(), "deduplicate", 10, "a.go")
	if err != nil || len(results) != 0 {
		t.Fatalf("stale FTS entry: %+v %v", results, err)
	}
	var entries int
	if err := s.db.QueryRow(`SELECT count(*) FROM lexical_chunks`).Scan(&entries); err != nil || entries != 3 {
		t.Fatalf("stale physical entries: %d %v", entries, err)
	}
}

func TestLookupLexicalSymbolsBoundsAndMembership(t *testing.T) {
	s := lexicalStore(t)
	putLexical(t, s, "allowed/a.go", "aa", "helper", "body")
	putLexical(t, s, "outside/b.go", "bb", "helper", "body")
	putLexical(t, s, "allowed/c.go", "cc", "unrelated", "helper")
	rows, err := s.LookupLexicalSymbols(context.Background(), []string{"helper"}, "allowed")
	if err != nil || len(rows) != 1 || rows[0].FilePath != "allowed/a.go" {
		t.Fatalf("symbol lookup: %+v %v", rows, err)
	}
	if err := s.UseProject("/another"); err != nil {
		t.Fatal(err)
	}
	rows, err = s.LookupLexicalSymbols(context.Background(), []string{"helper"}, "")
	if err != nil || len(rows) != 0 {
		t.Fatalf("membership leak: %+v %v", rows, err)
	}
	chunks := make([]chunker.Chunk, 260)
	for i := range chunks {
		chunks[i] = chunker.Chunk{ID: string(rune('a' + i)), FilePath: "many.go", Symbol: "ambiguous", StartLine: i + 1, EndLine: i + 1}
	}
	if _, err := s.StoreFileRevision("many.go", "dd", chunks, nil); err != nil {
		t.Fatal(err)
	}
	rows, err = s.LookupLexicalSymbols(context.Background(), []string{"ambiguous"}, "")
	if err != nil || len(rows) != 257 {
		t.Fatalf("ambiguity sentinel: %d %v", len(rows), err)
	}
}

func TestLexicalConcurrentRevisionDedup(t *testing.T) {
	db := filepath.Join(t.TempDir(), "index.db")
	a, err := OpenLexicalCollection(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	b, err := OpenLexicalCollection(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close() }()
	if err := a.UseProject(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := b.UseProject(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	chunks := []chunker.Chunk{{ID: "shared", FilePath: "main.go", Symbol: "Original", Kind: "function", Content: "func Original() {}", StartLine: 1, EndLine: 1}}
	done := make(chan error, 2)
	for _, s := range []*Store{a, b} {
		go func(s *Store) { _, err := s.StoreFileRevision("main.go", "aa", chunks, nil); done <- err }(s)
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	stats, err := a.CollectionStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.UniqueVectors != 1 || stats.ChunkReferences != 2 {
		t.Fatalf("concurrent duplicate rows: %+v", stats)
	}
	replacement := []chunker.Chunk{{ID: "changed", FilePath: "main.go", Symbol: "Changed", Kind: "function", Content: "func Changed() {}", StartLine: 1, EndLine: 1}}
	if _, err := a.StoreFileRevision("main.go", "bb", replacement, nil); err != nil {
		t.Fatal(err)
	}
	results, err := b.SearchLexical(context.Background(), "Original", 10, "")
	if err != nil || len(results) != 1 {
		t.Fatalf("sibling lost revision: %+v %v", results, err)
	}
	if err := b.DeleteFileChunks("main.go"); err != nil {
		t.Fatal(err)
	}
	stats, err = a.CollectionStats()
	if err != nil || stats.UniqueVectors != 1 {
		t.Fatalf("GC failed: %+v %v", stats, err)
	}
}

func TestLexicalWritesRollbackMembershipAndFTS(t *testing.T) {
	s := lexicalStore(t)
	if _, err := s.db.Exec(`CREATE TRIGGER fail_membership BEFORE INSERT ON project_files BEGIN SELECT RAISE(ABORT,'test write failure'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := s.StoreFileRevision("main.go", "aa", []chunker.Chunk{{ID: "rollback", FilePath: "main.go", Symbol: "RolledBack", Content: "needle", StartLine: 1, EndLine: 1}}, nil)
	if err == nil {
		t.Fatal("expected membership write failure")
	}
	for _, table := range []string{"file_revisions", "chunk_defs", "lexical_chunks", "project_files"} {
		var n int
		if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("partial %s write: %d %v", table, n, err)
		}
	}
}

// Merkle paths and filepath.Rel filters use the native separator, which is a
// backslash on Windows. Exercise that layout on every platform.
func TestLexicalPathFilterUsesNativeSeparator(t *testing.T) {
	prev := lexicalPathSeparator
	lexicalPathSeparator = `\`
	t.Cleanup(func() { lexicalPathSeparator = prev })
	s := lexicalStore(t)
	putLexical(t, s, `src\main.go`, "aa", "reservePayment", "reserve payment")
	putLexical(t, s, `src\nested\deep.go`, "bb", "reservePayment", "reserve payment nested")
	putLexical(t, s, `src-other\main.go`, "cc", "reservePayment", "reserve payment other")
	want := map[string]bool{`src\main.go`: true, `src\nested\deep.go`: true}
	for _, filter := range []string{`src`, `src\`} {
		check := func(name string, results []SearchResult, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("%s %q: %v", name, filter, err)
			}
			got := map[string]bool{}
			for _, r := range results {
				got[r.FilePath] = true
			}
			if len(got) != len(want) || !got[`src\main.go`] || !got[`src\nested\deep.go`] {
				t.Fatalf("%s %q: got %v, want %v", name, filter, got, want)
			}
		}
		results, err := s.SearchLexical(context.Background(), "reservePayment", 10, filter)
		check("search", results, err)
		results, err = s.LookupLexicalSymbols(context.Background(), []string{"reservePayment"}, filter)
		check("lookup", results, err)
		results, err = s.ListProjectChunks(filter)
		check("list", results, err)
	}
}

// A project reached through a symlink (macOS /var -> /private/var, symlinked
// checkouts) must select the same membership as its real path. Otherwise
// `lumen index` and a later search or status address different projects.
func TestUseProjectCanonicalizesSymlinks(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	s := lexicalStore(t)
	if err := s.UseProject(link); err != nil {
		t.Fatal(err)
	}
	putLexical(t, s, "a.go", "aa", "Checkout", "reserve()")
	if err := s.UseProject(real); err != nil {
		t.Fatal(err)
	}
	results, err := s.SearchLexical(context.Background(), "Checkout", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("real path sees %d results indexed through its symlink, want 1", len(results))
	}
}
