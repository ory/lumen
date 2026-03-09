package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aeneasr/lumen/bench-swe/internal/judge"
	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
	"github.com/aeneasr/lumen/bench-swe/internal/runner"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
)

func TestFindResult(t *testing.T) {
	results := []taskResult{
		{task: task.Task{ID: "t1"}, scenario: runner.Baseline},
		{task: task.Task{ID: "t1"}, scenario: runner.WithLumen},
		{task: task.Task{ID: "t2"}, scenario: runner.Baseline},
	}

	t.Run("match", func(t *testing.T) {
		r := findResult(results, "t1", runner.WithLumen)
		if r == nil {
			t.Fatal("expected result, got nil")
		}
		if r.task.ID != "t1" || r.scenario != runner.WithLumen {
			t.Errorf("got task=%q scenario=%q", r.task.ID, r.scenario)
		}
	})

	t.Run("no match by ID", func(t *testing.T) {
		r := findResult(results, "t99", runner.Baseline)
		if r != nil {
			t.Errorf("expected nil, got %+v", r)
		}
	})

	t.Run("no match by scenario", func(t *testing.T) {
		r := findResult(results, "t2", runner.WithLumen)
		if r != nil {
			t.Errorf("expected nil, got %+v", r)
		}
	})
}

func TestUniqueLanguages(t *testing.T) {
	t.Run("mixed", func(t *testing.T) {
		tasks := []task.Task{
			{Language: "go"},
			{Language: "python"},
			{Language: "go"},
			{Language: "rust"},
		}
		got := uniqueLanguages(tasks)
		if len(got) != 3 {
			t.Fatalf("got %d languages, want 3", len(got))
		}
		// Order should be: go, python, rust (insertion order)
		want := []string{"go", "python", "rust"}
		for i, w := range want {
			if got[i] != w {
				t.Errorf("got[%d] = %q, want %q", i, got[i], w)
			}
		}
	})

	t.Run("all same", func(t *testing.T) {
		tasks := []task.Task{
			{Language: "go"},
			{Language: "go"},
		}
		got := uniqueLanguages(tasks)
		if len(got) != 1 {
			t.Fatalf("got %d languages, want 1", len(got))
		}
	})

	t.Run("empty", func(t *testing.T) {
		got := uniqueLanguages(nil)
		if len(got) != 0 {
			t.Fatalf("got %d languages, want 0", len(got))
		}
	})
}

func TestRatingRank(t *testing.T) {
	tests := []struct {
		rating judge.Rating
		want   int
	}{
		{judge.Perfect, 3},
		{judge.Good, 2},
		{judge.Poor, 1},
		{judge.Rating("Unknown"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.rating), func(t *testing.T) {
			got := ratingRank(tt.rating)
			if got != tt.want {
				t.Errorf("ratingRank(%q) = %d, want %d", tt.rating, got, tt.want)
			}
		})
	}
}

func TestGenerateSummary(t *testing.T) {
	dir := t.TempDir()

	tasks := []task.Task{
		{ID: "task-1", Language: "go", Difficulty: "easy"},
	}
	scenarios := []runner.Scenario{runner.Baseline, runner.WithLumen}

	// Write fixture metrics
	m := &metrics.Metrics{
		CostUSD:      0.0042,
		DurationMS:   15000,
		InputTokens:  5000,
		CacheRead:    1000,
		CacheCreated: 200,
		OutputTokens: 800,
	}
	mData, _ := json.Marshal(m)
	_ = os.WriteFile(filepath.Join(dir, "task-1-baseline-metrics.json"), mData, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "task-1-with-lumen-metrics.json"), mData, 0o644)

	// Write fixture judge results
	j := &judge.JudgeResult{
		Rating:          judge.Perfect,
		FilesCorrect:    true,
		LogicEquivalent: true,
	}
	jData, _ := json.Marshal(j)
	_ = os.WriteFile(filepath.Join(dir, "task-1-baseline-judge.json"), jData, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "task-1-with-lumen-judge.json"), jData, 0o644)

	cfg := &Config{
		ResultsDir:  dir,
		EmbedModel:  "test-model",
		ClaudeModel: "test-claude",
		Tasks:       tasks,
		Scenarios:   scenarios,
	}

	results := loadResults(cfg)
	if err := generateSummary(cfg, results); err != nil {
		t.Fatalf("generateSummary: %v", err)
	}

	// Verify output file
	data, err := os.ReadFile(filepath.Join(dir, "summary-report.md"))
	if err != nil {
		t.Fatalf("reading summary: %v", err)
	}
	content := string(data)

	checks := []string{
		"SWE-Bench Summary",
		"test-model",
		"test-claude",
		"task-1",
		"Perfect",
		"$0.0042",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("summary does not contain %q", check)
		}
	}
}

func TestGenerate_EmptyResults(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		ResultsDir:  dir,
		EmbedModel:  "test-model",
		ClaudeModel: "test-claude",
		Tasks:       []task.Task{{ID: "task-1", Language: "go", Difficulty: "easy"}},
		Scenarios:   []runner.Scenario{runner.Baseline},
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Summary report should exist with placeholder dashes
	data, err := os.ReadFile(filepath.Join(dir, "summary-report.md"))
	if err != nil {
		t.Fatalf("reading summary: %v", err)
	}
	content := string(data)
	// The em-dash character used as placeholder
	if !strings.Contains(content, "\u2014") {
		t.Error("summary should contain em-dash placeholder for missing data")
	}

	// Detail report should also exist
	if _, err := os.Stat(filepath.Join(dir, "detail-report.md")); err != nil {
		t.Errorf("detail report not created: %v", err)
	}
}
