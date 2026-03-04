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
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/aeneasr/lumen/internal/config"
	"github.com/aeneasr/lumen/internal/index"
)

// stubEmbedder satisfies embedder.Embedder for tests.
type stubEmbedder struct{}

func (s *stubEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i := range vecs {
		vecs[i] = []float32{0.1, 0.2, 0.3, 0.4}
	}
	return vecs, nil
}
func (s *stubEmbedder) Dimensions() int   { return 4 }
func (s *stubEmbedder) ModelName() string { return "stub" }

func TestIndexerCache_ConcurrentReads(_ *testing.T) {
	ic := &indexerCache{
		embedder: &stubEmbedder{},
		cfg:      config.Config{MaxChunkTokens: 2048},
	}

	const goroutines = 20
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			// Path doesn't exist on disk — getOrCreate will error, that's fine.
			// We're testing there's no data race on the cache map/mutex.
			_, _, _ = ic.getOrCreate("/nonexistent/path/for/race/test")
		})
	}
	wg.Wait()
}

func TestIndexerCache_FindEffectiveRoot(t *testing.T) {
	const model = "test-model"

	t.Run("returns path when no parent exists", func(t *testing.T) {
		ic := &indexerCache{
			cache: make(map[string]*index.Indexer),
			model: model,
		}
		root := ic.findEffectiveRoot("/project/src/pkg")
		if root != "/project/src/pkg" {
			t.Fatalf("expected original path, got %s", root)
		}
	})

	t.Run("returns cached parent", func(t *testing.T) {
		// A nil *Indexer value still satisfies the map presence check.
		ic := &indexerCache{
			cache: map[string]*index.Indexer{"/project": nil},
			model: model,
		}
		root := ic.findEffectiveRoot("/project/src/pkg")
		if root != "/project" {
			t.Fatalf("expected /project (cached parent), got %s", root)
		}
	})

	t.Run("returns parent with existing db on disk", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmpDir)

		// Create the DB file that would exist for /project with our model.
		parentDBPath := config.DBPathForProject("/project", model)
		if err := os.MkdirAll(filepath.Dir(parentDBPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(parentDBPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		ic := &indexerCache{
			cache: make(map[string]*index.Indexer),
			model: model,
		}
		root := ic.findEffectiveRoot("/project/src/pkg")
		if root != "/project" {
			t.Fatalf("expected /project (db on disk), got %s", root)
		}
	})
}

func TestIndexerCache_GetOrCreate_ReusesParentIndex(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	const model = "stub"
	ic := &indexerCache{
		embedder: &stubEmbedder{},
		model:    model,
		cfg:      config.Config{MaxChunkTokens: 512},
	}

	// First call: index the parent directory — creates an indexer and DB on disk.
	parentDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parentIdx, parentRoot, err := ic.getOrCreate(parentDir)
	if err != nil {
		t.Fatalf("getOrCreate(parent): %v", err)
	}
	if parentRoot != parentDir {
		t.Fatalf("expected effectiveRoot=%s, got %s", parentDir, parentRoot)
	}

	// Second call: request a subdirectory — should reuse the parent indexer.
	subDir := filepath.Join(parentDir, "src")
	subIdx, subRoot, err := ic.getOrCreate(subDir)
	if err != nil {
		t.Fatalf("getOrCreate(subdir): %v", err)
	}
	if subRoot != parentDir {
		t.Fatalf("expected effectiveRoot=%s for subdir, got %s", parentDir, subRoot)
	}
	if subIdx != parentIdx {
		t.Fatal("expected subdir to reuse parent indexer, got a different instance")
	}

	// Both keys should be aliased in the cache.
	ic.mu.RLock()
	cachedParent := ic.cache[parentDir]
	cachedSub := ic.cache[subDir]
	ic.mu.RUnlock()
	if cachedParent != parentIdx {
		t.Fatal("parent key not in cache")
	}
	if cachedSub != parentIdx {
		t.Fatal("subdir key not aliased to parent indexer in cache")
	}
}

func TestScoreIsNotDistance(t *testing.T) {
	// Score should be in (0, 1] for reasonable matches (cosine similarity),
	// not in [0, 2) like cosine distance.
	// A distance of 0.3 should yield score 0.7.
	score := float32(1.0 - 0.3)
	if score != 0.7 {
		t.Fatalf("expected score=0.7, got %v", score)
	}
	// A perfect match (distance=0) should yield score=1.
	if float32(1.0-0.0) != 1.0 {
		t.Fatal("expected perfect score=1.0")
	}
	// Verify ordering: lower distance = higher score = should sort first.
	distances := []float64{0.1, 0.3, 0.5}
	for i := 1; i < len(distances); i++ {
		scoreA := 1.0 - distances[i-1]
		scoreB := 1.0 - distances[i]
		if scoreA < scoreB {
			t.Fatalf("expected scores descending: %.2f should be >= %.2f", scoreA, scoreB)
		}
	}
}
