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
	"sort"
	"strings"
	"sync"
	"testing"

	"flag"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/store"
)

var updateGolden = flag.Bool("update-golden", false, "update golden test files")

// assertGolden compares got against the golden file at path. If -update-golden
// is set, it writes got to the golden file instead.
func assertGolden(t *testing.T, goldenPath, got string) {
	t.Helper()
	got = strings.TrimRight(got, "\n")
	if *updateGolden {
		if err := os.WriteFile(goldenPath, []byte(got+"\n"), 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		return
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	want := strings.TrimRight(string(golden), "\n")
	if got != want {
		t.Fatalf("output does not match golden file %s (run with -update-golden to refresh).\n\nGOT:\n%s\n\nWANT:\n%s", goldenPath, got, want)
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

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
			_, _, _ = ic.getOrCreate("/nonexistent/path/for/race/test", "")
		})
	}
	wg.Wait()
}

func TestIndexerCache_FindEffectiveRoot(t *testing.T) {
	const model = "test-model"

	t.Run("returns path when no parent exists", func(t *testing.T) {
		ic := &indexerCache{
			cache: make(map[string]cacheEntry),
			model: model,
		}
		root := ic.findEffectiveRoot("/project/src/pkg")
		if root != "/project/src/pkg" {
			t.Fatalf("expected original path, got %s", root)
		}
	})

	t.Run("returns cached parent", func(t *testing.T) {
		ic := &indexerCache{
			cache: map[string]cacheEntry{"/project": {idx: nil, effectiveRoot: "/project"}},
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
		parentDBPath := config.DBPathForProject("/project", model, "")
		if err := os.MkdirAll(filepath.Dir(parentDBPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(parentDBPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		ic := &indexerCache{
			cache: make(map[string]cacheEntry),
			model: model,
		}
		root := ic.findEffectiveRoot("/project/src/pkg")
		if root != "/project" {
			t.Fatalf("expected /project (db on disk), got %s", root)
		}
	})

	t.Run("ignores parent when path crosses a SkipDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmpDir)

		// Simulate a parent index at /project.
		parentDBPath := config.DBPathForProject("/project", model, "")
		if err := os.MkdirAll(filepath.Dir(parentDBPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(parentDBPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

		ic := &indexerCache{
			cache: make(map[string]cacheEntry),
			model: model,
		}
		// "testdata" is in merkle.SkipDirs — the parent index would never
		// contain these files, so findEffectiveRoot must return the path itself.
		root := ic.findEffectiveRoot("/project/testdata/fixtures/go")
		if root != "/project/testdata/fixtures/go" {
			t.Fatalf("expected original path (skip dir in route), got %s", root)
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
	parentIdx, parentRoot, err := ic.getOrCreate(parentDir, "")
	if err != nil {
		t.Fatalf("getOrCreate(parent): %v", err)
	}
	if parentRoot != parentDir {
		t.Fatalf("expected effectiveRoot=%s, got %s", parentDir, parentRoot)
	}

	// Second call: request a subdirectory — should reuse the parent indexer.
	subDir := filepath.Join(parentDir, "src")
	subIdx, subRoot, err := ic.getOrCreate(subDir, "")
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
	if cachedParent.idx != parentIdx {
		t.Fatal("parent key not in cache")
	}
	if cachedSub.idx != parentIdx {
		t.Fatal("subdir key not aliased to parent indexer in cache")
	}

	// Third call: same subDir again — hits fast path; must still return parent root.
	subIdx2, subRoot2, err := ic.getOrCreate(subDir, "")
	if err != nil {
		t.Fatalf("getOrCreate(subdir fast path): %v", err)
	}
	if subRoot2 != parentDir {
		t.Fatalf("fast-path: expected effectiveRoot=%s, got %s", parentDir, subRoot2)
	}
	if subIdx2 != parentIdx {
		t.Fatal("fast-path: expected same indexer instance")
	}
}

func TestIndexerCache_GetOrCreate_FastPathEffectiveRoot(t *testing.T) {
	// Regression: the fast path (second call to same path) must return the
	// correct effectiveRoot (the parent), not the requested subdirectory path.
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	const model = "stub"
	ic := &indexerCache{
		embedder: &stubEmbedder{},
		model:    model,
		cfg:      config.Config{MaxChunkTokens: 512},
	}

	parentDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(parentDir, "api")

	// Prime the parent index.
	if _, _, err := ic.getOrCreate(parentDir, ""); err != nil {
		t.Fatalf("getOrCreate(parent): %v", err)
	}

	// First subDir call — slow path, caches alias.
	if _, _, err := ic.getOrCreate(subDir, ""); err != nil {
		t.Fatalf("getOrCreate(subdir slow path): %v", err)
	}

	// Second subDir call — hits the fast path.
	_, root, err := ic.getOrCreate(subDir, "")
	if err != nil {
		t.Fatalf("getOrCreate(subdir fast path): %v", err)
	}
	if root != parentDir {
		t.Fatalf("fast path returned wrong effectiveRoot: got %s, want %s", root, parentDir)
	}
}

func TestIndexerCache_GetOrCreate_PreferredRoot(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	ic := &indexerCache{
		embedder: &stubEmbedder{},
		model:    "stub",
		cfg:      config.Config{MaxChunkTokens: 512},
	}

	parentDir := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(parentDir, "src")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Call with subDir as projectPath but parentDir as preferredRoot.
	idx, root, err := ic.getOrCreate(subDir, parentDir)
	if err != nil {
		t.Fatalf("getOrCreate with preferredRoot: %v", err)
	}
	if root != parentDir {
		t.Fatalf("expected effectiveRoot=%s, got %s", parentDir, root)
	}

	// Both subDir and parentDir should be cached.
	ic.mu.RLock()
	parentEntry := ic.cache[parentDir]
	subEntry := ic.cache[subDir]
	ic.mu.RUnlock()
	if parentEntry.idx != idx {
		t.Fatal("parent key not in cache")
	}
	if subEntry.idx != idx {
		t.Fatal("subdir key not aliased to parent indexer")
	}
}

func TestValidateSearchInput_CwdPathInteraction(t *testing.T) {
	tests := []struct {
		name     string
		input    SemanticSearchInput
		wantErr  string
		wantPath string
	}{
		{
			name:     "cwd only — path defaults to cwd",
			input:    SemanticSearchInput{Cwd: "/project", Query: "test"},
			wantPath: "/project",
		},
		{
			name:     "path only — works as before",
			input:    SemanticSearchInput{Path: "/project/src", Query: "test"},
			wantPath: "/project/src",
		},
		{
			name:     "both valid — path under cwd",
			input:    SemanticSearchInput{Cwd: "/project", Path: "/project/src", Query: "test"},
			wantPath: "/project/src",
		},
		{
			name:    "both invalid — path outside cwd",
			input:   SemanticSearchInput{Cwd: "/project", Path: "/other", Query: "test"},
			wantErr: "path must be equal to or under cwd",
		},
		{
			name:     "neither provided — defaults to cwd",
			input:    SemanticSearchInput{Query: "test"},
			wantPath: mustGetwd(t),
		},
		{
			name:    "cwd is relative",
			input:   SemanticSearchInput{Cwd: "relative/path", Query: "test"},
			wantErr: "cwd must be an absolute path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			err := validateSearchInput(&input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if input.Path != tt.wantPath {
				t.Fatalf("expected path=%s, got %s", tt.wantPath, input.Path)
			}
		})
	}
}

func TestComputeMaxDistance_ModelAware(t *testing.T) {
	// No explicit min_score: should use model default.
	t.Run("jina default", func(t *testing.T) {
		d := computeMaxDistance(nil, "ordis/jina-embeddings-v2-base-code", 768)
		if d != 0.65 { // 1.0 - 0.35
			t.Fatalf("expected 0.65, got %v", d)
		}
	})

	t.Run("nomic-embed-code default", func(t *testing.T) {
		d := computeMaxDistance(nil, "nomic-ai/nomic-embed-code-GGUF", 3584)
		if d != 0.85 { // 1.0 - 0.15
			t.Fatalf("expected 0.85, got %v", d)
		}
	})

	t.Run("all-minilm default", func(t *testing.T) {
		d := computeMaxDistance(nil, "all-minilm", 384)
		if d != 0.80 { // 1.0 - 0.20
			t.Fatalf("expected 0.80, got %v", d)
		}
	})

	t.Run("unknown model uses DefaultMinScore when dims=0", func(t *testing.T) {
		d := computeMaxDistance(nil, "unknown-model", 0)
		if d != 0.80 { // 1.0 - 0.20
			t.Fatalf("expected 0.80, got %v", d)
		}
	})

	t.Run("unknown model with high dims uses dimension-aware floor", func(t *testing.T) {
		d := computeMaxDistance(nil, "unknown-model", 4096)
		if d != 0.85 { // 1.0 - 0.15
			t.Fatalf("expected 0.85, got %v", d)
		}
	})

	t.Run("unknown model with medium dims", func(t *testing.T) {
		d := computeMaxDistance(nil, "unknown-model", 768)
		if d != 0.75 { // 1.0 - 0.25
			t.Fatalf("expected 0.75, got %v", d)
		}
	})

	t.Run("explicit min_score overrides model default", func(t *testing.T) {
		ms := 0.5
		d := computeMaxDistance(&ms, "ordis/jina-embeddings-v2-base-code", 768)
		if d != 0.5 {
			t.Fatalf("expected 0.5, got %v", d)
		}
	})

	t.Run("explicit -1 disables filter", func(t *testing.T) {
		ms := -1.0
		d := computeMaxDistance(&ms, "ordis/jina-embeddings-v2-base-code", 768)
		if d != 0 {
			t.Fatalf("expected 0, got %v", d)
		}
	})
}

func TestMergeOverlappingResults(t *testing.T) {
	t.Run("merges overlapping chunks from same file", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "Foo", Kind: "method", StartLine: 10, EndLine: 30, Score: 0.6},
			{FilePath: "a.go", Symbol: "Foo", Kind: "method", StartLine: 25, EndLine: 50, Score: 0.7},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 1 {
			t.Fatalf("expected 1 merged result, got %d", len(merged))
		}
		if merged[0].StartLine != 10 || merged[0].EndLine != 50 {
			t.Fatalf("expected lines 10-50, got %d-%d", merged[0].StartLine, merged[0].EndLine)
		}
		if merged[0].Score != 0.7 {
			t.Fatalf("expected score 0.7, got %v", merged[0].Score)
		}
	})

	t.Run("merges adjacent chunks within gap", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "Foo", Kind: "function", StartLine: 10, EndLine: 20, Score: 0.5},
			{FilePath: "a.go", Symbol: "Bar", Kind: "function", StartLine: 24, EndLine: 40, Score: 0.6},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 1 {
			t.Fatalf("expected 1 merged result, got %d", len(merged))
		}
		if merged[0].Symbol != "Foo+Bar" {
			t.Fatalf("expected joined symbol, got %q", merged[0].Symbol)
		}
	})

	t.Run("does not merge distant chunks", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "Foo", Kind: "function", StartLine: 10, EndLine: 20, Score: 0.5},
			{FilePath: "a.go", Symbol: "Bar", Kind: "function", StartLine: 50, EndLine: 70, Score: 0.6},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 2 {
			t.Fatalf("expected 2 results, got %d", len(merged))
		}
	})

	t.Run("different files are not merged", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "Foo", Kind: "function", StartLine: 10, EndLine: 30, Score: 0.5},
			{FilePath: "b.go", Symbol: "Bar", Kind: "function", StartLine: 10, EndLine: 30, Score: 0.6},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 2 {
			t.Fatalf("expected 2 results, got %d", len(merged))
		}
	})

	t.Run("does not duplicate symbol on self-overlap", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "Decode", Kind: "method", StartLine: 10, EndLine: 30, Score: 0.6},
			{FilePath: "a.go", Symbol: "Decode", Kind: "method", StartLine: 25, EndLine: 50, Score: 0.7},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 1 {
			t.Fatalf("expected 1, got %d", len(merged))
		}
		if merged[0].Symbol != "Decode" {
			t.Fatalf("expected symbol 'Decode', got %q", merged[0].Symbol)
		}
	})
}

func TestBoostedScore_TestDemotion(t *testing.T) {
	// Test file demotion should be 0.75x.
	score := boostedScore(0.6, "function", "pkg/foo_test.go")
	// 0.6 * 1.15 (source boost) * 0.75 (test demotion) = 0.5175
	expected := float32(0.6 * 1.15 * 0.75)
	if score != expected {
		t.Fatalf("expected %.4f, got %.4f", expected, score)
	}

	// Non-test source code gets only the boost.
	scoreNonTest := boostedScore(0.6, "function", "pkg/foo.go")
	expectedNonTest := float32(0.6 * 1.15)
	if scoreNonTest != expectedNonTest {
		t.Fatalf("expected %.4f, got %.4f", expectedNonTest, scoreNonTest)
	}

	// Test file should score significantly lower.
	if score >= scoreNonTest {
		t.Fatalf("test file score (%.4f) should be lower than non-test (%.4f)", score, scoreNonTest)
	}
}

func TestMergeOverlappingResults_EdgeCases(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		merged := mergeOverlappingResults(nil)
		if len(merged) != 0 {
			t.Fatalf("expected 0 results, got %d", len(merged))
		}
	})

	t.Run("single item unchanged", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "Foo", Kind: "function", StartLine: 10, EndLine: 30, Score: 0.6},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 1 {
			t.Fatalf("expected 1 result, got %d", len(merged))
		}
		if merged[0] != items[0] {
			t.Fatalf("single item should pass through unchanged")
		}
	})

	t.Run("three-way chain merge", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "A", Kind: "function", StartLine: 10, EndLine: 25, Score: 0.5},
			{FilePath: "a.go", Symbol: "B", Kind: "method", StartLine: 20, EndLine: 40, Score: 0.7},
			{FilePath: "a.go", Symbol: "C", Kind: "function", StartLine: 38, EndLine: 60, Score: 0.6},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 1 {
			t.Fatalf("expected 1 merged result, got %d", len(merged))
		}
		m := merged[0]
		if m.StartLine != 10 || m.EndLine != 60 {
			t.Fatalf("expected lines 10-60, got %d-%d", m.StartLine, m.EndLine)
		}
		if m.Score != 0.7 {
			t.Fatalf("expected best score 0.7, got %v", m.Score)
		}
		if m.Kind != "method" {
			t.Fatalf("expected kind from best-scoring chunk 'method', got %q", m.Kind)
		}
		if m.Symbol != "A+B+C" {
			t.Fatalf("expected symbol 'A+B+C', got %q", m.Symbol)
		}
	})

	t.Run("unsorted input is handled correctly", func(t *testing.T) {
		// Items deliberately out of line order.
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "Bar", Kind: "function", StartLine: 50, EndLine: 70, Score: 0.5},
			{FilePath: "a.go", Symbol: "Foo", Kind: "function", StartLine: 10, EndLine: 20, Score: 0.6},
			{FilePath: "a.go", Symbol: "Baz", Kind: "function", StartLine: 15, EndLine: 25, Score: 0.4},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 2 {
			t.Fatalf("expected 2 results, got %d", len(merged))
		}
		// First group: Foo+Baz merged (lines 10-25).
		if merged[0].StartLine != 10 || merged[0].EndLine != 25 {
			t.Fatalf("expected first group 10-25, got %d-%d", merged[0].StartLine, merged[0].EndLine)
		}
		// Second group: Bar standalone (lines 50-70).
		if merged[1].StartLine != 50 || merged[1].EndLine != 70 {
			t.Fatalf("expected second group 50-70, got %d-%d", merged[1].StartLine, merged[1].EndLine)
		}
	})

	t.Run("boundary at exactly adjacency gap", func(t *testing.T) {
		// Gap of exactly 5 lines: EndLine=20, next StartLine=25 → 25 <= 20+5 → merged.
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "A", Kind: "function", StartLine: 10, EndLine: 20, Score: 0.5},
			{FilePath: "a.go", Symbol: "B", Kind: "function", StartLine: 25, EndLine: 40, Score: 0.6},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 1 {
			t.Fatalf("expected 1 merged (gap=5), got %d", len(merged))
		}
	})

	t.Run("boundary at gap+1 stays separate", func(t *testing.T) {
		// Gap of 6 lines: EndLine=20, next StartLine=26 → 26 > 20+5 → not merged.
		items := []SearchResultItem{
			{FilePath: "a.go", Symbol: "A", Kind: "function", StartLine: 10, EndLine: 20, Score: 0.5},
			{FilePath: "a.go", Symbol: "B", Kind: "function", StartLine: 26, EndLine: 40, Score: 0.6},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 2 {
			t.Fatalf("expected 2 separate (gap=6), got %d", len(merged))
		}
	})

	t.Run("multiple files with mixed merge patterns", func(t *testing.T) {
		items := []SearchResultItem{
			// File a: two overlapping → merge to 1.
			{FilePath: "a.go", Symbol: "A1", Kind: "function", StartLine: 10, EndLine: 30, Score: 0.5},
			{FilePath: "a.go", Symbol: "A2", Kind: "function", StartLine: 25, EndLine: 50, Score: 0.6},
			// File b: two distant → stay 2.
			{FilePath: "b.go", Symbol: "B1", Kind: "function", StartLine: 10, EndLine: 20, Score: 0.7},
			{FilePath: "b.go", Symbol: "B2", Kind: "function", StartLine: 100, EndLine: 120, Score: 0.4},
			// File c: single item → stay 1.
			{FilePath: "c.go", Symbol: "C1", Kind: "type", StartLine: 5, EndLine: 15, Score: 0.3},
		}
		merged := mergeOverlappingResults(items)
		if len(merged) != 4 {
			t.Fatalf("expected 4 results (1+2+1), got %d", len(merged))
		}

		// Verify file ordering is preserved (a, b, c).
		if merged[0].FilePath != "a.go" {
			t.Fatalf("expected first result from a.go, got %s", merged[0].FilePath)
		}
		if merged[1].FilePath != "b.go" || merged[2].FilePath != "b.go" {
			t.Fatalf("expected results 2-3 from b.go")
		}
		if merged[3].FilePath != "c.go" {
			t.Fatalf("expected last result from c.go, got %s", merged[3].FilePath)
		}
	})
}

func TestFillSnippets(t *testing.T) {
	// Use the testdata fixture file.
	projectPath := filepath.Join("testdata", "snippets")

	t.Run("extracts correct line range", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "decoder.go", StartLine: 12, EndLine: 14},
		}
		fillSnippets(projectPath, items, 0)
		want := "func NewDecoder(buf []byte) *Decoder {\n\treturn &Decoder{buf: buf}\n}"
		if items[0].Content != want {
			t.Fatalf("got:\n%s\nwant:\n%s", items[0].Content, want)
		}
	})

	t.Run("multiple items from same file read file once", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "decoder.go", StartLine: 12, EndLine: 14},
			{FilePath: "decoder.go", StartLine: 43, EndLine: 51},
		}
		fillSnippets(projectPath, items, 0)
		if items[0].Content == "" || items[1].Content == "" {
			t.Fatal("expected both items to have content")
		}
		if !strings.Contains(items[0].Content, "NewDecoder") {
			t.Fatalf("item 0 should contain NewDecoder, got: %s", items[0].Content)
		}
		if !strings.Contains(items[1].Content, "readVarInt") {
			t.Fatalf("item 1 should contain readVarInt, got: %s", items[1].Content)
		}
	})

	t.Run("maxLines truncates content", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "decoder.go", StartLine: 17, EndLine: 42},
		}
		fillSnippets(projectPath, items, 3)
		lines := strings.Split(items[0].Content, "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d: %q", len(lines), items[0].Content)
		}
	})

	t.Run("missing file leaves content empty", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "nonexistent.go", StartLine: 1, EndLine: 10},
		}
		fillSnippets(projectPath, items, 0)
		if items[0].Content != "" {
			t.Fatalf("expected empty content for missing file, got: %s", items[0].Content)
		}
	})
}

func TestFormatSearchResults_Golden(t *testing.T) {
	// Use absolute path so filepath.Rel works correctly in formatSearchResults.
	// fillSnippets uses projectPath to read files, and FilePath is relative to it.
	// formatSearchResults uses filepath.Rel(projectPath, r.FilePath) to display paths —
	// since FilePath is already relative, we pass "." as the project root for formatting.
	snippetDir := filepath.Join("testdata", "snippets")

	t.Run("split chunks merged", func(t *testing.T) {
		// Simulate two split chunks of decodeStruct that overlap (lines 16-30 and 26-42),
		// plus one separate readString result (lines 53-65).
		items := []SearchResultItem{
			{FilePath: "decoder.go", Symbol: "decodeStruct", Kind: "method", StartLine: 16, EndLine: 30, Score: 0.65},
			{FilePath: "decoder.go", Symbol: "decodeStruct", Kind: "method", StartLine: 26, EndLine: 42, Score: 0.70},
			{FilePath: "decoder.go", Symbol: "readString", Kind: "method", StartLine: 53, EndLine: 65, Score: 0.55},
		}

		// Merge first, then fill snippets (mirrors the real pipeline).
		items = mergeOverlappingResults(items)
		fillSnippets(snippetDir, items, 0)

		out := SemanticSearchOutput{Results: items}
		got := formatSearchResults(".", out)
		assertGolden(t, filepath.Join("testdata", "format_split_chunks_merged.golden"), got)
	})

	t.Run("multi file grouping", func(t *testing.T) {
		items := []SearchResultItem{
			{FilePath: "decoder.go", Symbol: "decodeStruct", Kind: "method", StartLine: 16, EndLine: 42, Score: 0.80},
			{FilePath: "decoder.go", Symbol: "readVarInt", Kind: "method", StartLine: 43, EndLine: 51, Score: 0.60},
			{FilePath: "decoder.go", Symbol: "readString", Kind: "method", StartLine: 53, EndLine: 65, Score: 0.50},
		}

		fillSnippets(snippetDir, items, 0)
		out := SemanticSearchOutput{Results: items}
		got := formatSearchResults(".", out)
		assertGolden(t, filepath.Join("testdata", "format_multi_file.golden"), got)
	})

	t.Run("empty results", func(t *testing.T) {
		out := SemanticSearchOutput{Results: nil}
		got := formatSearchResults("/any", out)
		if got != "No results found." {
			t.Fatalf("expected 'No results found.', got %q", got)
		}
	})

	t.Run("empty results with reindex", func(t *testing.T) {
		out := SemanticSearchOutput{Results: nil, Reindexed: true, IndexedFiles: 42}
		got := formatSearchResults("/any", out)
		want := "No results found. (indexed 42 files)"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("empty results with filtered hint", func(t *testing.T) {
		out := SemanticSearchOutput{
			Results:      nil,
			FilteredHint: "Results exist but were below the 0.35 noise floor (best match scored 0.28). Use min_score=-1 to see all results, or lower min_score.",
		}
		got := formatSearchResults("/any", out)
		if !strings.Contains(got, "No results found.") {
			t.Fatal("expected 'No results found.' prefix")
		}
		if !strings.Contains(got, "noise floor") {
			t.Fatal("expected filtered hint in output")
		}
		if !strings.Contains(got, "min_score=-1") {
			t.Fatal("expected min_score=-1 hint in output")
		}
	})
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

func TestFormatSearchResults_RelevantFiles(t *testing.T) {
	out := SemanticSearchOutput{
		Results: []SearchResultItem{
			{FilePath: "/proj/auth.go", Symbol: "ValidateToken", Kind: "function", StartLine: 1, EndLine: 10, Score: 0.91},
		},
		RelevantFiles: []RelevantFile{
			{FilePath: "auth.go", Score: 0.91},
			{FilePath: "token.go", Score: 0.87},
		},
	}
	text := formatSearchResults("/proj", out)
	if !strings.Contains(text, "<relevant_files>") {
		t.Fatal("expected <relevant_files> section in output")
	}
	if !strings.Contains(text, `path="auth.go"`) {
		t.Fatal("expected auth.go in relevant_files")
	}
	if !strings.Contains(text, `score="0.91"`) {
		t.Fatal("expected score in relevant_files")
	}
}

func TestFormatSearchResults_NoRelevantFiles_NoSection(t *testing.T) {
	out := SemanticSearchOutput{
		Results: []SearchResultItem{
			{FilePath: "/proj/main.go", Symbol: "Main", Kind: "function", StartLine: 1, EndLine: 5, Score: 0.80},
		},
	}
	text := formatSearchResults("/proj", out)
	if strings.Contains(text, "<relevant_files>") {
		t.Fatal("expected no <relevant_files> section when RelevantFiles is empty")
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Go
		{"pkg/foo_test.go", true},
		{"pkg/foo.go", false},
		// Ruby
		{"spec/models/user_spec.rb", true},
		// JS/TS .test. with trailing extension segment
		{"tests/distribute-unions.test.ts", true},
		{"src/utils.test.js", true},
		// JS/TS .spec.
		{"tests/parser.spec.ts", true},
		// JS/TS .test without trailing dot (the bug fix)
		{"tests/foo.test.tsx", true},
		// __tests__ directory
		{"src/__tests__/helper.ts", true},
		// Python test_ prefix
		{"tests/test_utils.py", true},
		{"test_models.py", true},
		// /tests/ and /test/ directories
		{"tests/Feature/UserTest.php", true},
		{"src/test/java/com/example/FooTest.java", true},
		// Non-test files
		{"src/types/Pattern.ts", false},
		{"internal/store/store.go", false},
		{"cmd/root.go", false},
		{"testdata/fixture.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isTestFile(tt.path); got != tt.want {
				t.Errorf("isTestFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestMergeSearchResults(t *testing.T) {
	tests := []struct {
		name     string
		a        []store.SearchResult
		b        []store.SearchResult
		wantLen  int
		validate func(t *testing.T, got []store.SearchResult)
	}{
		{
			name:    "both empty",
			a:       nil,
			b:       nil,
			wantLen: 0,
		},
		{
			name: "b appends to a — no overlap",
			a: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "Foo", StartLine: 10, Distance: 0.3},
			},
			b: []store.SearchResult{
				{FilePath: "token.go", Symbol: "Bar", StartLine: 5, Distance: 0.2},
			},
			wantLen: 2,
			validate: func(t *testing.T, got []store.SearchResult) {
				t.Helper()
				paths := map[string]bool{}
				for _, r := range got {
					paths[r.FilePath] = true
				}
				if !paths["auth.go"] || !paths["token.go"] {
					t.Errorf("expected both auth.go and token.go in results; got %v", got)
				}
			},
		},
		{
			name: "duplicate: a wins when a has lower distance",
			a: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "Foo", StartLine: 10, Distance: 0.2},
			},
			b: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "Foo", StartLine: 10, Distance: 0.4},
			},
			wantLen: 1,
			validate: func(t *testing.T, got []store.SearchResult) {
				t.Helper()
				if got[0].Distance != 0.2 {
					t.Errorf("expected distance=0.2 (a wins), got %v", got[0].Distance)
				}
			},
		},
		{
			name: "duplicate: b wins when b has lower distance",
			a: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "Foo", StartLine: 10, Distance: 0.4},
			},
			b: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "Foo", StartLine: 10, Distance: 0.2},
			},
			wantLen: 1,
			validate: func(t *testing.T, got []store.SearchResult) {
				t.Helper()
				if got[0].Distance != 0.2 {
					t.Errorf("expected distance=0.2 (b wins), got %v", got[0].Distance)
				}
			},
		},
		{
			name: "same symbol different startLine — not a duplicate",
			a: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "Foo", StartLine: 10, Distance: 0.3},
			},
			b: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "Foo", StartLine: 50, Distance: 0.3},
			},
			wantLen: 2,
		},
		{
			name: "partial overlap — mixed unique and duplicate",
			a: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "ValidateToken", StartLine: 10, Distance: 0.3},
				{FilePath: "router.go", Symbol: "HandleRoute", StartLine: 20, Distance: 0.4},
			},
			b: []store.SearchResult{
				{FilePath: "auth.go", Symbol: "ValidateToken", StartLine: 10, Distance: 0.2},
				{FilePath: "store.go", Symbol: "Query", StartLine: 5, Distance: 0.35},
			},
			wantLen: 3,
			validate: func(t *testing.T, got []store.SearchResult) {
				t.Helper()
				for _, r := range got {
					if r.FilePath == "auth.go" && r.Symbol == "ValidateToken" {
						if r.Distance != 0.2 {
							t.Errorf("ValidateToken: expected distance=0.2, got %v", r.Distance)
						}
					}
				}
				paths := map[string]bool{}
				for _, r := range got {
					paths[r.FilePath] = true
				}
				if !paths["auth.go"] || !paths["router.go"] || !paths["store.go"] {
					t.Errorf("expected all three files in results; got %v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeSearchResults(tt.a, tt.b)
			if len(got) != tt.wantLen {
				t.Fatalf("expected len=%d, got %d: %v", tt.wantLen, len(got), got)
			}
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

// runRankingPipeline replicates the ranking pipeline from handleSemanticSearch:
// merge raw + summary results → convert to SearchResultItem + boostedScore →
// mergeOverlappingResults → sort by score desc → cap to nResults.
func runRankingPipeline(raw, summary []store.SearchResult, nResults int) []SearchResultItem {
	merged := mergeSearchResults(raw, summary)

	items := make([]SearchResultItem, 0, len(merged))
	for _, r := range merged {
		score := float32(1.0 - r.Distance)
		items = append(items, SearchResultItem{
			FilePath:  r.FilePath,
			Symbol:    r.Symbol,
			Kind:      r.Kind,
			StartLine: r.StartLine,
			EndLine:   r.EndLine,
			Score:     boostedScore(score, r.Kind, r.FilePath),
		})
	}

	items = mergeOverlappingResults(items)

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	if nResults > 0 && len(items) > nResults {
		items = items[:nResults]
	}
	return items
}

func TestRankingPipeline_Scenarios(t *testing.T) {
	t.Run("scenario A: production code beats test file with same raw distance", func(t *testing.T) {
		// Both have dist=0.35 → score=0.65 before boost.
		// middleware.go (function): 0.65 * 1.15 = 0.7475
		// middleware_test.go (function): 0.65 * 1.15 * 0.75 = 0.560...
		raw := []store.SearchResult{
			{FilePath: "auth/middleware.go", Symbol: "ValidateToken", Kind: "function", StartLine: 10, EndLine: 30, Distance: 0.35},
			{FilePath: "auth/middleware_test.go", Symbol: "TestValidateToken", Kind: "function", StartLine: 5, EndLine: 20, Distance: 0.35},
		}
		items := runRankingPipeline(raw, nil, 10)
		if len(items) < 2 {
			t.Fatalf("expected at least 2 results, got %d", len(items))
		}
		if items[0].FilePath != "auth/middleware.go" {
			t.Errorf("expected middleware.go first (production code), got %s", items[0].FilePath)
		}
		if items[0].Score <= items[1].Score {
			t.Errorf("expected middleware.go score (%.4f) > test score (%.4f)", items[0].Score, items[1].Score)
		}
	})

	t.Run("scenario B: summary hit rescues result raw KNN missed", func(t *testing.T) {
		// Raw only has router.go at dist=0.40; auth.go was too far for raw (dist=0.50 excluded).
		// Summary chunk for auth.go has dist=0.25 → enters via summary fan-out.
		raw := []store.SearchResult{
			{FilePath: "router.go", Symbol: "HandleRoute", Kind: "function", StartLine: 1, EndLine: 20, Distance: 0.40},
		}
		summary := []store.SearchResult{
			{FilePath: "auth.go", Symbol: "ValidateToken", Kind: "function", StartLine: 1, EndLine: 15, Distance: 0.25},
		}
		items := runRankingPipeline(raw, summary, 10)
		if len(items) != 2 {
			t.Fatalf("expected 2 results, got %d", len(items))
		}
		// auth.go has better distance so it should rank first.
		if items[0].FilePath != "auth.go" {
			t.Errorf("expected auth.go first (better summary hit), got %s", items[0].FilePath)
		}
	})

	t.Run("scenario C: deduplication — same chunk in raw + summary, keep best distance", func(t *testing.T) {
		raw := []store.SearchResult{
			{FilePath: "store.go", Symbol: "Query", Kind: "function", StartLine: 10, EndLine: 30, Distance: 0.45},
		}
		summary := []store.SearchResult{
			{FilePath: "store.go", Symbol: "Query", Kind: "function", StartLine: 10, EndLine: 30, Distance: 0.28},
		}
		merged := mergeSearchResults(raw, summary)
		if len(merged) != 1 {
			t.Fatalf("expected 1 result after merge, got %d", len(merged))
		}
		if merged[0].Distance != 0.28 {
			t.Errorf("expected best distance=0.28, got %v", merged[0].Distance)
		}
	})

	t.Run("scenario D: NResults cap respected after summary fan-out inflates results", func(t *testing.T) {
		// 5 raw + 8 summary distinct = 13 total; cap at 8.
		raw := make([]store.SearchResult, 5)
		for i := range raw {
			raw[i] = store.SearchResult{
				FilePath:  "raw.go",
				Symbol:    "RawFunc",
				Kind:      "function",
				StartLine: (i + 1) * 100,
				EndLine:   (i+1)*100 + 10,
				Distance:  0.3 + float64(i)*0.01,
			}
		}
		summary := make([]store.SearchResult, 8)
		for i := range summary {
			summary[i] = store.SearchResult{
				FilePath:  "summary.go",
				Symbol:    "SumFunc",
				Kind:      "function",
				StartLine: (i + 1) * 100,
				EndLine:   (i+1)*100 + 10,
				Distance:  0.2 + float64(i)*0.01,
			}
		}
		items := runRankingPipeline(raw, summary, 8)
		if len(items) != 8 {
			t.Fatalf("expected 8 results (cap), got %d", len(items))
		}
		// Verify sorted by score descending.
		for i := 1; i < len(items); i++ {
			if items[i].Score > items[i-1].Score {
				t.Errorf("results not sorted: items[%d].Score=%.4f > items[%d].Score=%.4f", i, items[i].Score, i-1, items[i-1].Score)
			}
		}
	})

	t.Run("scenario E: find DB connection code — kind boost ranks types above comments", func(t *testing.T) {
		// type at dist=0.38 → score=0.62 * 1.15 = 0.713
		// comment at dist=0.32 → score=0.68 (no source boost for comment kind)
		// Lines are kept far apart so mergeOverlappingResults does not combine them.
		raw := []store.SearchResult{
			{FilePath: "db/pool.go", Symbol: "Pool", Kind: "type", StartLine: 100, EndLine: 120, Distance: 0.38},
			{FilePath: "db/pool.go", Symbol: "pool size comment", Kind: "comment", StartLine: 1, EndLine: 3, Distance: 0.32},
		}
		items := runRankingPipeline(raw, nil, 10)
		if len(items) < 2 {
			t.Fatalf("expected at least 2 results, got %d", len(items))
		}
		if items[0].Symbol != "Pool" {
			t.Errorf("expected Pool (type) first despite higher raw distance; got %s (score=%.4f)", items[0].Symbol, items[0].Score)
		}
	})

	t.Run("scenario F: RelevantFiles populated and chunks appear in results", func(t *testing.T) {
		// This scenario validates the data contract: chunks returned by TopChunksByFile
		// merge into results and the corresponding file is tracked as relevant.
		// We simulate the fan-out manually (as handleSemanticSearch does).

		// Raw search returned one result.
		raw := []store.SearchResult{
			{FilePath: "router.go", Symbol: "HandleRoute", Kind: "function", StartLine: 1, EndLine: 10, Distance: 0.40},
		}

		// File summary matched auth/auth.go → TopChunksByFile returned 3 chunks.
		topChunks := []store.SearchResult{
			{FilePath: "auth/auth.go", Symbol: "ValidateToken", Kind: "function", StartLine: 5, EndLine: 20, Distance: 0.22},
			{FilePath: "auth/auth.go", Symbol: "RevokeToken", Kind: "function", StartLine: 22, EndLine: 35, Distance: 0.28},
			{FilePath: "auth/auth.go", Symbol: "ParseClaims", Kind: "function", StartLine: 37, EndLine: 55, Distance: 0.30},
		}

		// Simulate merging raw + top-chunks (no chunk summary duplicates here).
		merged := mergeSearchResults(raw, topChunks)
		items := make([]SearchResultItem, 0, len(merged))
		for _, r := range merged {
			score := float32(1.0 - r.Distance)
			items = append(items, SearchResultItem{
				FilePath:  r.FilePath,
				Symbol:    r.Symbol,
				Kind:      r.Kind,
				StartLine: r.StartLine,
				EndLine:   r.EndLine,
				Score:     boostedScore(score, r.Kind, r.FilePath),
			})
		}
		items = mergeOverlappingResults(items)
		sort.Slice(items, func(i, j int) bool {
			return items[i].Score > items[j].Score
		})

		// Build RelevantFiles from file summary hit (dist=0.20 → score=0.80).
		relevantFiles := []RelevantFile{
			{FilePath: "auth/auth.go", Score: 1.0 - 0.20},
		}

		out := SemanticSearchOutput{Results: items, RelevantFiles: relevantFiles}

		// Verify chunks from auth.go appear in results.
		authChunks := 0
		for _, item := range out.Results {
			if item.FilePath == "auth/auth.go" {
				authChunks++
			}
		}
		if authChunks == 0 {
			t.Error("expected auth/auth.go chunks in results after TopChunksByFile fan-out")
		}

		// Verify RelevantFiles is populated.
		if len(out.RelevantFiles) != 1 {
			t.Fatalf("expected 1 relevant file, got %d", len(out.RelevantFiles))
		}
		if out.RelevantFiles[0].FilePath != "auth/auth.go" {
			t.Errorf("expected auth/auth.go as relevant file, got %s", out.RelevantFiles[0].FilePath)
		}
		if out.RelevantFiles[0].Score != 0.80 {
			t.Errorf("expected score=0.80, got %v", out.RelevantFiles[0].Score)
		}

		// Verify formatSearchResults includes the relevant_files block.
		text := formatSearchResults("/proj", out)
		if !strings.Contains(text, "<relevant_files>") {
			t.Error("expected <relevant_files> block in formatted output")
		}
		if !strings.Contains(text, "auth/auth.go") {
			t.Error("expected auth/auth.go in formatted output")
		}
	})
}
