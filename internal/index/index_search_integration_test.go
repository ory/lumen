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

package index_test

import (
	"context"
	"testing"

	"github.com/ory/lumen/internal/index"
)

// distinctEmbedder returns a unique vector for each unique input text so that
// different chunk symbols produce distinct embeddings, making KNN results
// predictable without needing a real embedding backend.
type distinctEmbedder struct {
	dims  int
	seen  map[string]int
	count int
}

func newDistinctEmbedder(dims int) *distinctEmbedder {
	return &distinctEmbedder{dims: dims, seen: make(map[string]int)}
}

func (d *distinctEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i, text := range texts {
		if _, ok := d.seen[text]; !ok {
			d.count++
			d.seen[text] = d.count
		}
		v := make([]float32, d.dims)
		// Place the discriminating value at index (id-1) % dims so each text
		// has a unique non-zero position.
		pos := (d.seen[text] - 1) % d.dims
		v[pos] = 1.0
		vecs[i] = v
	}
	return vecs, nil
}

func (d *distinctEmbedder) Dimensions() int   { return d.dims }
func (d *distinctEmbedder) ModelName() string { return "distinct-stub" }

func TestSearchChunkSummaries_ReturnsBothChunks(t *testing.T) {
	dir := t.TempDir()
	makeGoFile(t, dir, "auth.go", `package auth

func ValidateToken(token string) bool {
	return token != ""
}

func RevokeToken(token string) {
	// revoke the token
	_ = token
}
`)

	emb := newDistinctEmbedder(8)
	sum := &stubSummarizer{}
	idx, err := index.NewIndexer(":memory:", emb, 512, 8, sum, emb)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}

	// Use an arbitrary query vector; with distinct embeddings any query will
	// return results as long as the distance threshold is wide enough.
	queryVec := make([]float32, 8)
	queryVec[0] = 1.0

	results, err := idx.SearchChunkSummaries(queryVec, 10, 2.0, "")
	if err != nil {
		t.Fatalf("SearchChunkSummaries: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected chunk summary results, got none")
	}

	// Both ValidateToken and RevokeToken should appear (they both exceed 2 lines).
	symbols := make(map[string]bool)
	for _, r := range results {
		symbols[r.Symbol] = true
	}
	if !symbols["ValidateToken"] && !symbols["RevokeToken"] {
		t.Errorf("expected ValidateToken or RevokeToken in chunk summary results; got %v", symbols)
	}
}

func TestSearchFileSummaries_ReturnsFile(t *testing.T) {
	dir := t.TempDir()
	makeGoFile(t, dir, "auth.go", `package auth

func ValidateToken(token string) bool {
	return token != ""
}

func RevokeToken(token string) {
	// revoke the token
	_ = token
}
`)

	emb := newDistinctEmbedder(8)
	sum := &stubSummarizer{}
	idx, err := index.NewIndexer(":memory:", emb, 512, 8, sum, emb)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}

	queryVec := make([]float32, 8)
	queryVec[0] = 1.0

	results, err := idx.SearchFileSummaries(queryVec, 10, 2.0)
	if err != nil {
		t.Fatalf("SearchFileSummaries: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected file summary results, got none")
	}

	found := false
	for _, r := range results {
		if r.FilePath == "auth.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected auth.go in file summary results; got %v", results)
	}
}

func TestSearchChunkSummaries_PathPrefixFilters(t *testing.T) {
	dir := t.TempDir()
	makeGoFile(t, dir, "auth.go", `package auth

func ValidateToken(token string) bool {
	return token != ""
}

func RevokeToken(token string) {
	// revoke the token
	_ = token
}
`)
	makeGoFile(t, dir, "main.go", `package main

func Run() {
	// start application
	_ = 1
	_ = 2
}
`)

	emb := newDistinctEmbedder(8)
	sum := &stubSummarizer{}
	idx, err := index.NewIndexer(":memory:", emb, 512, 8, sum, emb)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}

	queryVec := make([]float32, 8)
	queryVec[0] = 1.0

	// Search with pathPrefix="auth" — only auth.go chunks should appear.
	results, err := idx.SearchChunkSummaries(queryVec, 10, 2.0, "auth")
	if err != nil {
		t.Fatalf("SearchChunkSummaries with pathPrefix: %v", err)
	}

	for _, r := range results {
		if r.FilePath != "auth.go" {
			t.Errorf("pathPrefix=auth: unexpected result from %s", r.FilePath)
		}
	}
}
