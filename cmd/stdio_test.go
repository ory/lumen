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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"flag"

	"github.com/ory/lumen/internal/config"
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
		parentDBPath := config.DBPathForProject("/project", model)
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
		parentDBPath := config.DBPathForProject("/project", model)
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

func TestIndexerCache_FindEffectiveRoot_GitBoundary(t *testing.T) {
	// Structure: ancestor/ (has an index DB) → repo/ (git root) → subdir/
	// findEffectiveRoot must not walk above the git repo root to adopt the
	// ancestor index.
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	const model = "test-model"

	// Create directory layout.
	ancestor := filepath.Join(tmpDir, "ancestor")
	repo := filepath.Join(ancestor, "repo")
	subdir := filepath.Join(repo, "subdir")
	for _, d := range []string{ancestor, repo, subdir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Initialise a git repository at repo/.
	cmd := exec.Command("git", "init", repo)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	// Create a fake index DB for the ancestor directory (above the git root).
	ancestorDBPath := config.DBPathForProject(ancestor, model)
	if err := os.MkdirAll(filepath.Dir(ancestorDBPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ancestorDBPath, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	ic := &indexerCache{
		cache: make(map[string]cacheEntry),
		model: model,
	}
	root := ic.findEffectiveRoot(subdir)
	if root != subdir {
		t.Fatalf("expected findEffectiveRoot to stop at git boundary and return %s, got %s", subdir, root)
	}
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

	parentDir := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(parentDir, "src")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("no existing DB at preferredRoot falls back to projectPath", func(t *testing.T) {
		ic := &indexerCache{
			embedder: &stubEmbedder{},
			model:    "stub",
			cfg:      config.Config{MaxChunkTokens: 512},
		}
		// No DB exists at parentDir yet — should fall through to findEffectiveRoot(subDir)
		// which returns subDir (no parent index found either).
		_, root, err := ic.getOrCreate(subDir, parentDir)
		if err != nil {
			t.Fatalf("getOrCreate: %v", err)
		}
		if root != subDir {
			t.Fatalf("expected effectiveRoot=%s (fallback), got %s", subDir, root)
		}
	})

	t.Run("existing DB at preferredRoot is adopted", func(t *testing.T) {
		ic := &indexerCache{
			embedder: &stubEmbedder{},
			model:    "stub",
			cfg:      config.Config{MaxChunkTokens: 512},
		}
		// Pre-create the DB file at parentDir so the preferred root is adopted.
		dbPath := config.DBPathForProject(parentDir, "stub")
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
			t.Fatal(err)
		}

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
	})
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

func TestEnsureIndexed_FreshnessTTL(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	ic := &indexerCache{
		embedder: &stubEmbedder{},
		model:    "stub",
		cfg:      config.Config{MaxChunkTokens: 512},
	}

	projectDir := filepath.Join(tmpDir, "proj")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	idx, effectiveRoot, err := ic.getOrCreate(projectDir, "")
	if err != nil {
		t.Fatalf("getOrCreate: %v", err)
	}

	input := SemanticSearchInput{
		Path:     projectDir,
		Cwd:      projectDir,
		Query:    "test",
		NResults: 8,
	}

	// First call: no TTL entry yet — runs EnsureFresh and records lastCheckedAt.
	_, err = ic.ensureIndexed(context.Background(), idx, input, effectiveRoot, nil)
	if err != nil {
		t.Fatalf("first ensureIndexed: %v", err)
	}

	ic.mu.RLock()
	entry := ic.cache[projectDir]
	ic.mu.RUnlock()
	if entry.lastCheckedAt.IsZero() {
		t.Fatal("lastCheckedAt should be set after first ensureIndexed")
	}

	// Second call within TTL: recentlyChecked should be true, skipping the walk.
	if !ic.recentlyChecked(projectDir) {
		t.Fatal("expected recentlyChecked=true immediately after ensureIndexed")
	}

	out, err := ic.ensureIndexed(context.Background(), idx, input, effectiveRoot, nil)
	if err != nil {
		t.Fatalf("second ensureIndexed: %v", err)
	}
	// Should be a no-op: not reindexed, no files counted.
	if out.Reindexed {
		t.Fatal("second call should not reindex within TTL")
	}
}
