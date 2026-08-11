// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/chunker"
	"github.com/ory/lumen/internal/store"
)

func writeSharedBatchFixture(t *testing.T, projectDir string, files int) {
	t.Helper()
	for i := 0; i < files; i++ {
		name := fmt.Sprintf("file_%d.go", i)
		writeGoFile(t, projectDir, name, fmt.Sprintf("package demo\n\nfunc File%d() {}\n", i))
	}
}

func TestSharedIndexBatchesEmbeddingsAcrossFiles(t *testing.T) {
	projectDir := t.TempDir()
	writeSharedBatchFixture(t, projectDir, 5)
	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexerForProject(filepath.Join(t.TempDir(), "index.db"), emb, 512, "int8", projectDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	stats, err := idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.IndexedFiles != 5 {
		t.Fatalf("IndexedFiles = %d, want 5", stats.IndexedFiles)
	}
	if emb.callCount != 1 {
		t.Fatalf("Embed calls = %d, want 1 cross-file batch", emb.callCount)
	}
}

type failSecondBatchEmbedder struct {
	calls int
}

func (e *failSecondBatchEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.calls++
	if e.calls == 2 {
		return nil, errors.New("injected batch failure")
	}
	vectors := make([][]float32, len(texts))
	for i := range vectors {
		vectors[i] = []float32{1, 0, 0, 0}
	}
	return vectors, nil
}

func (*failSecondBatchEmbedder) Dimensions() int   { return 4 }
func (*failSecondBatchEmbedder) ModelName() string { return "test-model" }

func TestSharedIndexBatchFailureDoesNotCommitRootHash(t *testing.T) {
	projectDir := t.TempDir()
	writeSharedBatchFixture(t, projectDir, 5)
	emb := &failSecondBatchEmbedder{}
	idx, err := NewIndexerForProject(filepath.Join(t.TempDir(), "index.db"), emb, 512, "int8", projectDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()
	idx.embedBatchSize = 2

	if _, err := idx.Index(context.Background(), projectDir, false, nil); err == nil {
		t.Fatal("expected injected embedding failure")
	}
	if _, err := idx.store.GetMeta("root_hash"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("root_hash error = %v, want sql.ErrNoRows", err)
	}
	hashes, err := idx.store.GetFileHashes()
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 2 {
		t.Fatalf("durable revisions = %d, want first flushed batch of 2", len(hashes))
	}
}

type fixedChunker struct {
	chunks []chunker.Chunk
}

func (c fixedChunker) Chunk(string, []byte) ([]chunker.Chunk, error) {
	return append([]chunker.Chunk(nil), c.chunks...), nil
}

type callbackEmbedder struct {
	calls    int
	callback func() error
}

func (e *callbackEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.calls++
	if e.callback != nil {
		callback := e.callback
		e.callback = nil
		if err := callback(); err != nil {
			return nil, err
		}
	}
	vectors := make([][]float32, len(texts))
	for i := range vectors {
		vectors[i] = []float32{1, 0, 0, 0}
	}
	return vectors, nil
}

func (*callbackEmbedder) Dimensions() int   { return 4 }
func (*callbackEmbedder) ModelName() string { return "test-model" }

func TestSharedIndexRetriesWhenPreviouslySharedVectorVanishes(t *testing.T) {
	projectA, projectB := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(projectA, "main.go"), []byte("package demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chunkA := chunker.Chunk{ID: "a", FilePath: "main.go", Symbol: "A", Kind: "function", StartLine: 1, EndLine: 10, Content: strings.Repeat("shared input ", 80)}
	chunkB := chunker.Chunk{ID: "b", FilePath: "main.go", Symbol: "B", Kind: "function", StartLine: 11, EndLine: 20, Content: strings.Repeat("new input ", 80)}
	dbPath := filepath.Join(t.TempDir(), "index.db")
	emb := &callbackEmbedder{}
	idx, err := NewIndexerForProject(dbPath, emb, 512, "int8", projectA)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()
	idx.chunker = fixedChunker{chunks: []chunker.Chunk{chunkA, chunkB}}
	if _, err := idx.store.StoreFileRevision("main.go", "old", []chunker.Chunk{chunkA}, map[int][]float32{0: {1, 0, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	keeper, err := store.NewCollection(dbPath, 4, "int8", projectB)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = keeper.Close() }()
	if attached, err := keeper.AttachExistingFileRevision("main.go", "old"); err != nil || !attached {
		t.Fatalf("attach keeper: %v, %v", attached, err)
	}
	if err := idx.store.DeleteFileChunks("main.go"); err != nil {
		t.Fatal(err)
	}
	emb.callback = func() error { return keeper.DeleteFileChunks("main.go") }

	stats, err := idx.Index(context.Background(), projectA, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if emb.calls != 2 {
		t.Fatalf("Embed calls = %d, want initial missing batch plus full-file retry", emb.calls)
	}
	if stats.ChunksCreated != 2 {
		t.Fatalf("ChunksCreated = %d, want 2", stats.ChunksCreated)
	}
}
