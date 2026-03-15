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
	"os"
	"path/filepath"
	"testing"

	"github.com/ory/lumen/internal/index"
	"github.com/ory/lumen/internal/summarizer"
)

type stubEmbedder struct{ dims int }

func (s *stubEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i := range vecs {
		v := make([]float32, s.dims)
		v[0] = 1
		vecs[i] = v
	}
	return vecs, nil
}
func (s *stubEmbedder) Dimensions() int   { return s.dims }
func (s *stubEmbedder) ModelName() string { return "stub" }

type stubSummarizer struct {
	chunkCalls int
	fileCalls  int
}

func (s *stubSummarizer) SummarizeChunk(_ context.Context, chunk summarizer.ChunkInfo) (string, error) {
	s.chunkCalls++
	return "summary of " + chunk.Symbol, nil
}

func (s *stubSummarizer) SummarizeFile(_ context.Context, _ []string) (string, error) {
	s.fileCalls++
	return "file summary", nil
}

func makeGoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIndexer_SummaryPass_ChunkFilterMinLines(t *testing.T) {
	dir := t.TempDir()
	makeGoFile(t, dir, "main.go", `package main

func Short() {}

func Long() {
	_ = 1
	_ = 2
	_ = 3
}
`)

	emb := &stubEmbedder{dims: 4}
	sum := &stubSummarizer{}
	idx, err := index.NewIndexer(":memory:", emb, 512, 4, sum, emb)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}

	if sum.chunkCalls == 0 {
		t.Fatal("expected at least one chunk summary call for Long()")
	}
}

func TestIndexer_SummaryPass_FileSummaryGeneratedFromChunks(t *testing.T) {
	dir := t.TempDir()
	makeGoFile(t, dir, "auth.go", `package auth

func ValidateToken(token string) bool {
	return token != ""
}

func RevokeToken(token string) {
	// revoke
}
`)

	emb := &stubEmbedder{dims: 4}
	sum := &stubSummarizer{}
	idx, err := index.NewIndexer(":memory:", emb, 512, 4, sum, emb)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}

	if sum.fileCalls == 0 {
		t.Fatal("expected file summary to be generated")
	}
}

func TestIndexer_NoSummarizer_NoPanics(t *testing.T) {
	dir := t.TempDir()
	makeGoFile(t, dir, "main.go", `package main

func Hello() {
	println("hello")
}
`)

	emb := &stubEmbedder{dims: 4}
	idx, err := index.NewIndexer(":memory:", emb, 512, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}
}
