// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package store

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ory/lumen/internal/chunker"
)

func TestSharedReplacementGarbageCollectsDisplacedRevision(t *testing.T) {
	s, err := NewCollection(":memory:", 4, "int8", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	oldChunk := chunker.Chunk{ID: "old", FilePath: "main.go", Symbol: "Old", Kind: "function", StartLine: 1, EndLine: 1, Content: "func Old() {}"}
	newChunk := chunker.Chunk{ID: "new", FilePath: "main.go", Symbol: "New", Kind: "function", StartLine: 1, EndLine: 1, Content: "func New() {}"}
	if _, err := s.StoreFileRevision("main.go", "old", []chunker.Chunk{oldChunk}, map[int][]float32{0: {1, 0, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StoreFileRevision("main.go", "new", []chunker.Chunk{newChunk}, map[int][]float32{0: {0, 1, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	stats, err := s.CollectionStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.UniqueVectors != 1 || stats.ChunkReferences != 1 {
		t.Fatalf("displaced revision was not collected: %+v", stats)
	}
}

func TestMissingChunkInputsBatchesLargeQueries(t *testing.T) {
	s, err := NewCollection(":memory:", 4, "int8", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	chunks := make([]chunker.Chunk, 300)
	vectors := make(map[int][]float32, len(chunks))
	for i := range chunks {
		chunks[i] = chunker.Chunk{ID: strconv.Itoa(i), FilePath: "large.go", Content: "input " + strconv.Itoa(i)}
		vectors[i] = []float32{1, 0, 0, 0}
	}
	if _, err := s.StoreFileRevision("large.go", "large", chunks, vectors); err != nil {
		t.Fatal(err)
	}
	missing, err := s.MissingChunkInputs(chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Fatalf("missing = %v, want none", missing)
	}
	chunks[299].Content = "changed"
	missing, err = s.MissingChunkInputs(chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0] != 299 {
		t.Fatalf("missing = %v, want [299]", missing)
	}
}

func TestInsertSharedChunksRequiresRegisteredRevision(t *testing.T) {
	s, err := NewCollection(":memory:", 4, "int8", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	err = s.InsertChunks([]chunker.Chunk{{ID: "x", FilePath: "missing.go", Content: "x"}}, [][]float32{{1, 0, 0, 0}})
	if err == nil || !strings.Contains(err.Error(), `no file revision registered for "missing.go"; call UpsertFile first`) {
		t.Fatalf("error = %v", err)
	}
}

func TestSharedCollectionReusesRevisionsAndVectors(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	projectA := filepath.Join(t.TempDir(), "worktree-a")
	projectB := filepath.Join(t.TempDir(), "worktree-b")
	s, err := NewCollection(dbPath, 4, "int8", projectA)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	chunks := []chunker.Chunk{
		{ID: "a", FilePath: "main.go", Symbol: "A", Kind: "function", StartLine: 1, EndLine: 2, Content: "func A() {}"},
		{ID: "b", FilePath: "main.go", Symbol: "B", Kind: "function", StartLine: 3, EndLine: 4, Content: "func B() {}"},
	}
	vectors := map[int][]float32{
		0: {1, 0, 0, 0},
		1: {0, 1, 0, 0},
	}
	created, err := s.StoreFileRevision("main.go", "abcd", chunks, vectors)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first revision should be created")
	}

	if err := s.UseProject(projectB); err != nil {
		t.Fatal(err)
	}
	reused, err := s.AttachExistingFileRevision("main.go", "abcd")
	if err != nil {
		t.Fatal(err)
	}
	if !reused {
		t.Fatal("second worktree should reuse the complete revision")
	}
	stats, err := s.CollectionStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.UniqueVectors != 2 || stats.ChunkReferences != 4 || stats.SharedReferences != 2 {
		t.Fatalf("unexpected dedup stats: %+v", stats)
	}

	results, err := s.Search(context.Background(), []float32{1, 0, 0, 0}, 2, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Symbol != "A" {
		t.Fatalf("unexpected project-local search results: %+v", results)
	}

	if err := s.UseProject(projectA); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFileChunks("main.go"); err != nil {
		t.Fatal(err)
	}
	stats, err = s.CollectionStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.UniqueVectors != 2 {
		t.Fatalf("shared vectors removed while another project referenced them: %+v", stats)
	}

	if err := s.UseProject(projectB); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFileChunks("main.go"); err != nil {
		t.Fatal(err)
	}
	stats, err = s.CollectionStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.UniqueVectors != 0 || stats.ChunkReferences != 0 {
		t.Fatalf("last-reference GC left vectors behind: %+v", stats)
	}
}

func TestSharedCollectionFloat32Override(t *testing.T) {
	s, err := NewCollection(":memory:", 3, "float32", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	chunk := chunker.Chunk{ID: "c", FilePath: "x.go", Symbol: "X", Kind: "function", StartLine: 1, EndLine: 1, Content: "func X() {}"}
	if _, err := s.StoreFileRevision("x.go", "01", []chunker.Chunk{chunk}, map[int][]float32{0: {0.1, 0.2, 0.3}}); err != nil {
		t.Fatal(err)
	}
	results, err := s.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 1, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Symbol != "X" {
		t.Fatalf("unexpected float32 results: %+v", results)
	}
}

func TestSharedCollectionValidatesStorageProfile(t *testing.T) {
	if _, err := NewCollection(":memory:", 4, "float16", t.TempDir()); err == nil {
		t.Fatal("expected unsupported vector storage to fail")
	}
	dbPath := filepath.Join(t.TempDir(), "index.db")
	s, err := NewCollection(dbPath, 4, "int8", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name       string
		dimensions int
		storage    string
	}{
		{"dimensions", 5, "int8"},
		{"storage", 4, "float32"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCollection(dbPath, tc.dimensions, tc.storage, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), "collection profile mismatch") {
				t.Fatalf("error = %v, want profile mismatch", err)
			}
		})
	}
}

func TestSharedSearchAdaptivelyExpandsSparseProjectCandidates(t *testing.T) {
	s, err := NewCollection(":memory:", 4, "int8", "/project-a")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	chunks := make([]chunker.Chunk, 80)
	vectors := make(map[int][]float32, len(chunks))
	for i := range chunks {
		suffix := strconv.Itoa(i + 1)
		chunks[i] = chunker.Chunk{
			ID: "a" + suffix, FilePath: "crowded.go", Symbol: "Crowded", Kind: "function",
			StartLine: i + 1, EndLine: i + 1, Content: "crowded input " + suffix,
		}
		vectors[i] = []float32{1, float32(i+1) / 10000, 0, 0}
	}
	if _, err := s.StoreFileRevision("crowded.go", "aa", chunks, vectors); err != nil {
		t.Fatal(err)
	}

	if err := s.UseProject("/project-b"); err != nil {
		t.Fatal(err)
	}
	target := chunker.Chunk{ID: "target", FilePath: "nested/target.go", Symbol: "Target", Kind: "function", StartLine: 1, EndLine: 1, Content: "target input"}
	if _, err := s.StoreFileRevision("nested/target.go", "bb", []chunker.Chunk{target}, map[int][]float32{0: {0, 1, 0, 0}}); err != nil {
		t.Fatal(err)
	}

	// The nearest 80 global vectors belong to project A. Project B's result
	// is only found after candidate doubling exhausts that dense prefix.
	results, err := s.Search(context.Background(), []float32{1, 0, 0, 0}, 1, 0, "nested")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Symbol != "Target" {
		t.Fatalf("adaptive sparse-project search failed: %+v", results)
	}
}

func TestSharedCollectionConcurrentRevisionInsertionIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	projectA, projectB := t.TempDir(), t.TempDir()
	a, err := NewCollection(dbPath, 4, "int8", projectA)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	b, err := NewCollection(dbPath, 4, "int8", projectB)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close() }()
	chunk := chunker.Chunk{ID: "same", FilePath: "same.go", Symbol: "Same", Kind: "function", StartLine: 1, EndLine: 1, Content: "func Same() {}"}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, collection := range []*Store{a, b} {
		wg.Add(1)
		go func(s *Store) {
			defer wg.Done()
			<-start
			_, err := s.StoreFileRevision("same.go", "cc", []chunker.Chunk{chunk}, map[int][]float32{0: {1, 0, 0, 0}})
			errs <- err
		}(collection)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	stats, err := a.CollectionStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.UniqueVectors != 1 || stats.ChunkReferences != 2 {
		t.Fatalf("concurrent insertion was not idempotent: %+v", stats)
	}
}

func TestSharedCollectionConcurrentFirstOpenIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	const openers = 8
	start := make(chan struct{})
	stores := make(chan *Store, openers)
	errs := make(chan error, openers)
	var wg sync.WaitGroup
	for range openers {
		project := t.TempDir()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			s, err := NewCollection(dbPath, 4, "int8", project)
			if err != nil {
				errs <- err
				return
			}
			stores <- s
		}()
	}
	close(start)
	wg.Wait()
	close(stores)
	close(errs)
	for s := range stores {
		if !s.IsShared() {
			t.Error("concurrent open returned a legacy store")
		}
		if err := s.Close(); err != nil {
			t.Errorf("close concurrent store: %v", err)
		}
	}
	for err := range errs {
		t.Errorf("concurrent first open: %v", err)
	}
}

func TestSharedRefreshMapsMissingVectorKeyToSentinel(t *testing.T) {
	s, err := NewCollection(":memory:", 4, "int8", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	chunk := chunker.Chunk{ID: "gone", FilePath: "gone.go", Symbol: "Gone", Kind: "function", StartLine: 1, EndLine: 1, Content: "func Gone() {}"}
	if _, err := s.StoreFileRevision("gone.go", "aa", []chunker.Chunk{chunk}, map[int][]float32{0: {1, 0, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("PRAGMA foreign_keys=OFF"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("DELETE FROM vector_keys"); err != nil {
		t.Fatal(err)
	}
	_, err = s.StoreFileRevision("gone.go", "aa", []chunker.Chunk{chunk}, map[int][]float32{0: {1, 0, 0, 0}})
	if !errors.Is(err, ErrVectorVanished) {
		t.Fatalf("refresh error = %v, want ErrVectorVanished", err)
	}
}

func TestSharedCleanupRemovesOnlyStaleMemberships(t *testing.T) {
	projectA, projectB := t.TempDir(), t.TempDir()
	s, err := NewCollection(":memory:", 4, "int8", projectA)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	chunk := chunker.Chunk{ID: "shared", FilePath: "shared.go", Symbol: "Shared", Kind: "function", StartLine: 1, EndLine: 1, Content: "func Shared() {}"}
	if _, err := s.StoreFileRevision("shared.go", "dd", []chunker.Chunk{chunk}, map[int][]float32{0: {1, 0, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	if err := s.UseProject(projectB); err != nil {
		t.Fatal(err)
	}
	if attached, err := s.AttachExistingFileRevision("shared.go", "dd"); err != nil || !attached {
		t.Fatalf("attach second project: attached=%v err=%v", attached, err)
	}
	old := time.Now().Add(-60 * 24 * time.Hour).UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`UPDATE projects SET last_accessed_at = ? WHERE path = ?`, old, projectA); err != nil {
		t.Fatal(err)
	}
	cleanup, err := s.CleanupStaleProjects(time.Now().Add(-30 * 24 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if cleanup.ProjectsRemoved != 1 || cleanup.VectorsRemoved != 0 {
		t.Fatalf("unexpected cleanup stats: %+v", cleanup)
	}
	stats, err := s.CollectionStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.UniqueVectors != 1 || stats.ChunkReferences != 1 {
		t.Fatalf("active project's shared data was not preserved: %+v", stats)
	}
}

func TestInt8RecallAt8AgainstFloat32(t *testing.T) {
	const (
		dimensions  = 64
		vectorCount = 200
		queryCount  = 20
		limit       = 8
	)
	project := t.TempDir()
	floatStore, err := NewCollection(":memory:", dimensions, "float32", project)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = floatStore.Close() }()
	int8Store, err := NewCollection(":memory:", dimensions, "int8", project)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = int8Store.Close() }()

	rng := rand.New(rand.NewSource(42)) //nolint:gosec // deterministic recall fixture
	chunks := make([]chunker.Chunk, vectorCount)
	vectors := make(map[int][]float32, vectorCount)
	for i := range vectorCount {
		suffix := strconv.Itoa(i)
		chunks[i] = chunker.Chunk{ID: suffix, FilePath: "vectors.go", Symbol: "V" + suffix, Kind: "function", StartLine: i + 1, EndLine: i + 1, Content: "vector " + suffix}
		vector := make([]float32, dimensions)
		for j := range vector {
			vector[j] = float32(rng.NormFloat64())
		}
		vectors[i] = vector
	}
	for _, s := range []*Store{floatStore, int8Store} {
		if _, err := s.StoreFileRevision("vectors.go", "ee", chunks, vectors); err != nil {
			t.Fatal(err)
		}
	}

	matches := 0
	for range queryCount {
		query := make([]float32, dimensions)
		for j := range query {
			query[j] = float32(rng.NormFloat64())
		}
		want, err := floatStore.Search(context.Background(), query, limit, 0, "")
		if err != nil {
			t.Fatal(err)
		}
		got, err := int8Store.Search(context.Background(), query, limit, 0, "")
		if err != nil {
			t.Fatal(err)
		}
		wantSymbols := make(map[string]struct{}, len(want))
		for _, result := range want {
			wantSymbols[result.Symbol] = struct{}{}
		}
		for _, result := range got {
			if _, ok := wantSymbols[result.Symbol]; ok {
				matches++
			}
		}
	}
	recall := float64(matches) / float64(queryCount*limit)
	if recall < 0.95 {
		t.Fatalf("int8 recall@8 = %.3f, want >= 0.95", recall)
	}
}

func TestSharedInt8StorageAtMostTwentyPercentOfSeparateFloat32(t *testing.T) {
	if testing.Short() {
		t.Skip("storage size fixture writes three multi-megabyte databases")
	}
	const (
		dimensions = 768
		chunkCount = 1000
	)
	base := t.TempDir()
	projectA, projectB := t.TempDir(), t.TempDir()
	chunks := make([]chunker.Chunk, chunkCount)
	vectorMap := make(map[int][]float32, chunkCount)
	vectorSlice := make([][]float32, chunkCount)
	rng := rand.New(rand.NewSource(7)) //nolint:gosec // deterministic size fixture
	for i := range chunkCount {
		suffix := strconv.Itoa(i)
		chunks[i] = chunker.Chunk{ID: suffix, FilePath: "bulk.go", Symbol: "Bulk" + suffix, Kind: "function", StartLine: i + 1, EndLine: i + 1, Content: "bulk " + suffix}
		vector := make([]float32, dimensions)
		for j := range vector {
			vector[j] = float32(rng.NormFloat64())
		}
		vectorMap[i] = vector
		vectorSlice[i] = vector
	}

	sharedPath := filepath.Join(base, "shared.db")
	shared, err := NewCollection(sharedPath, dimensions, "int8", projectA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := shared.StoreFileRevision("bulk.go", "ff", chunks, vectorMap); err != nil {
		t.Fatal(err)
	}
	if err := shared.UseProject(projectB); err != nil {
		t.Fatal(err)
	}
	if attached, err := shared.AttachExistingFileRevision("bulk.go", "ff"); err != nil || !attached {
		t.Fatalf("attach shared revision: attached=%v err=%v", attached, err)
	}
	if err := shared.Close(); err != nil {
		t.Fatal(err)
	}

	var separateBytes int64
	for i := range 2 {
		path := filepath.Join(base, "float-"+strconv.Itoa(i)+".db")
		legacy, err := New(path, dimensions)
		if err != nil {
			t.Fatal(err)
		}
		if err := legacy.UpsertFile("bulk.go", "ff"); err != nil {
			t.Fatal(err)
		}
		if err := legacy.InsertChunks(chunks, vectorSlice); err != nil {
			t.Fatal(err)
		}
		if err := legacy.Close(); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		separateBytes += info.Size()
	}
	sharedInfo, err := os.Stat(sharedPath)
	if err != nil {
		t.Fatal(err)
	}
	ratio := float64(sharedInfo.Size()) / float64(separateBytes)
	if ratio > 0.20 {
		t.Fatalf("shared int8 size ratio = %.3f (%d/%d), want <= 0.20", ratio, sharedInfo.Size(), separateBytes)
	}
}
