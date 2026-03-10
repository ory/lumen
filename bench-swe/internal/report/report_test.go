package report

import (
	"encoding/json"
	"fmt"
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
		{ID: "task-1", Language: "go"},
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

func TestTotalTokens(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := totalTokens(nil); got != 0 {
			t.Errorf("totalTokens(nil) = %d, want 0", got)
		}
	})
	t.Run("sum", func(t *testing.T) {
		m := &metrics.Metrics{InputTokens: 100, OutputTokens: 50, CacheRead: 200}
		if got := totalTokens(m); got != 350 {
			t.Errorf("totalTokens = %d, want 350", got)
		}
	})
}

// writeToolCallJSONL writes a minimal raw JSONL file with the given number
// of tool_use events (all named "Bash") plus optional lumen search calls.
func writeToolCallJSONL(t *testing.T, path string, bashCalls, lumenCalls int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	idx := 0
	for range bashCalls {
		_, _ = fmt.Fprintf(f, `{"type":"tool_use","id":"tc_%d","name":"Bash","input":{}}%s`, idx, "\n")
		idx++
	}
	for range lumenCalls {
		_, _ = fmt.Fprintf(f, `{"type":"tool_use","id":"tc_%d","name":"mcp__lumen__semantic_search","input":{"query":"test"}}%s`, idx, "\n")
		idx++
	}
}

func TestGenerateOverview_Columns(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "overview.txt")

	m := &metrics.Metrics{
		CostUSD:      0.5000,
		DurationMS:   120000,
		InputTokens:  3000,
		CacheRead:    7000,
		OutputTokens: 1000,
	}
	mData, _ := json.Marshal(m)
	_ = os.WriteFile(filepath.Join(dir, "task-1-baseline-metrics.json"), mData, 0o644)

	j := &judge.JudgeResult{Rating: judge.Good, FilesCorrect: true}
	jData, _ := json.Marshal(j)
	_ = os.WriteFile(filepath.Join(dir, "task-1-baseline-judge.json"), jData, 0o644)

	// Write a raw JSONL with 5 tool calls
	writeToolCallJSONL(t, filepath.Join(dir, "task-1-baseline-raw.jsonl"), 5, 0)

	cfg := &Config{
		ResultsDir:  dir,
		EmbedModel:  "test-embed",
		ClaudeModel: "test-claude",
		Tasks:       []task.Task{{ID: "task-1", Language: "go"}},
		Scenarios:   []runner.Scenario{runner.Baseline},
		OutputPath:  outPath,
	}

	if err := GenerateOverview(cfg); err != nil {
		t.Fatalf("GenerateOverview: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading overview: %v", err)
	}
	content := string(data)

	// Check new columns exist in header
	for _, col := range []string{"Run", "Tool Calls", "Total Tokens"} {
		if !strings.Contains(content, col) {
			t.Errorf("overview missing column header %q", col)
		}
	}

	// Check data: run=1, tokens=11000 (3000+7000+1000), tool calls=5
	if !strings.Contains(content, "11000") {
		t.Errorf("overview missing total tokens value 11000")
	}
	if !strings.Contains(content, "5") {
		t.Errorf("overview missing tool calls value 5")
	}
	// Check run marker
	if !strings.Contains(content, "=== Run:") {
		t.Error("overview missing run marker")
	}
	if !strings.Contains(content, "embed=test-embed") {
		t.Error("overview missing embed model in marker")
	}
}

func TestGenerateOverview_AppendMode(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "overview.txt")

	cfg := &Config{
		ResultsDir:  dir,
		EmbedModel:  "test-embed",
		ClaudeModel: "test-claude",
		Tasks:       []task.Task{{ID: "task-1", Language: "go"}},
		Scenarios:   []runner.Scenario{runner.Baseline},
		OutputPath:  outPath,
	}

	// Write minimal fixtures
	m := &metrics.Metrics{CostUSD: 0.01, DurationMS: 1000}
	mData, _ := json.Marshal(m)
	_ = os.WriteFile(filepath.Join(dir, "task-1-baseline-metrics.json"), mData, 0o644)

	// Call twice — should append, not overwrite
	if err := GenerateOverview(cfg); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := GenerateOverview(cfg); err != nil {
		t.Fatalf("second call: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading overview: %v", err)
	}
	content := string(data)

	count := strings.Count(content, "=== Run:")
	if count != 2 {
		t.Errorf("expected 2 run markers, got %d", count)
	}
}

func TestGenerateOverview_Comparison(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "overview.txt")

	// Baseline: expensive and slow
	blMetrics := &metrics.Metrics{
		CostUSD:      1.0000,
		DurationMS:   300000,
		InputTokens:  10000,
		CacheRead:    40000,
		OutputTokens: 5000,
	}
	blData, _ := json.Marshal(blMetrics)
	_ = os.WriteFile(filepath.Join(dir, "task-1-baseline-metrics.json"), blData, 0o644)

	blJudge := &judge.JudgeResult{Rating: judge.Poor}
	bjData, _ := json.Marshal(blJudge)
	_ = os.WriteFile(filepath.Join(dir, "task-1-baseline-judge.json"), bjData, 0o644)
	writeToolCallJSONL(t, filepath.Join(dir, "task-1-baseline-raw.jsonl"), 40, 0)

	// With-lumen: cheap and fast
	lmMetrics := &metrics.Metrics{
		CostUSD:      0.2000,
		DurationMS:   60000,
		InputTokens:  2000,
		CacheRead:    8000,
		OutputTokens: 1000,
	}
	lmData, _ := json.Marshal(lmMetrics)
	_ = os.WriteFile(filepath.Join(dir, "task-1-with-lumen-metrics.json"), lmData, 0o644)

	lmJudge := &judge.JudgeResult{Rating: judge.Perfect}
	ljData, _ := json.Marshal(lmJudge)
	_ = os.WriteFile(filepath.Join(dir, "task-1-with-lumen-judge.json"), ljData, 0o644)
	writeToolCallJSONL(t, filepath.Join(dir, "task-1-with-lumen-raw.jsonl"), 5, 3)

	cfg := &Config{
		ResultsDir:  dir,
		EmbedModel:  "test-embed",
		ClaudeModel: "test-claude",
		Tasks:       []task.Task{{ID: "task-1", Language: "go"}},
		Scenarios:   []runner.Scenario{runner.Baseline, runner.WithLumen},
		OutputPath:  outPath,
	}

	if err := GenerateOverview(cfg); err != nil {
		t.Fatalf("GenerateOverview: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading overview: %v", err)
	}
	content := string(data)

	// Check comparison section exists
	if !strings.Contains(content, "--- Comparison: worst baseline vs best with-lumen ---") {
		t.Error("missing comparison header")
	}
	if !strings.Contains(content, "cheaper") {
		t.Error("missing cost comparison")
	}
	if !strings.Contains(content, "faster") {
		t.Error("missing time comparison")
	}
	if !strings.Contains(content, "fewer tokens") {
		t.Error("missing tokens comparison")
	}
	if !strings.Contains(content, "fewer tool calls") {
		t.Error("missing tool calls comparison")
	}
	if !strings.Contains(content, "Poor (baseline) vs Perfect (lumen)") {
		t.Error("missing rating comparison")
	}
}

func TestGenerateOverview_SingleScenarioNoComparison(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "overview.txt")

	m := &metrics.Metrics{CostUSD: 0.01, DurationMS: 1000}
	mData, _ := json.Marshal(m)
	_ = os.WriteFile(filepath.Join(dir, "task-1-baseline-metrics.json"), mData, 0o644)

	cfg := &Config{
		ResultsDir:  dir,
		EmbedModel:  "test-embed",
		ClaudeModel: "test-claude",
		Tasks:       []task.Task{{ID: "task-1", Language: "go"}},
		Scenarios:   []runner.Scenario{runner.Baseline},
		OutputPath:  outPath,
	}

	if err := GenerateOverview(cfg); err != nil {
		t.Fatalf("GenerateOverview: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading overview: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "--- Comparison:") {
		t.Error("comparison section should not appear with single scenario")
	}
}

func TestGenerate_EmptyResults(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		ResultsDir:  dir,
		EmbedModel:  "test-model",
		ClaudeModel: "test-claude",
		Tasks:       []task.Task{{ID: "task-1", Language: "go"}},
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
