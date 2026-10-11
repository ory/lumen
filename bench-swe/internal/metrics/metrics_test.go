package metrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFromJSONL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
		check   func(t *testing.T, m *Metrics)
	}{
		{
			name:    "valid JSONL with result event",
			content: `{"type":"result","total_cost_usd":0.0042,"duration_ms":15000,"usage":{"input_tokens":5000,"cache_read_input_tokens":1000,"cache_creation_input_tokens":200,"output_tokens":800}}`,
			check: func(t *testing.T, m *Metrics) {
				if m.CostUSD != 0.0042 {
					t.Errorf("CostUSD = %f, want 0.0042", m.CostUSD)
				}
				if m.DurationMS != 15000 {
					t.Errorf("DurationMS = %d, want 15000", m.DurationMS)
				}
				if m.InputTokens != 5000 {
					t.Errorf("InputTokens = %d, want 5000", m.InputTokens)
				}
				if m.CacheRead != 1000 {
					t.Errorf("CacheRead = %d, want 1000", m.CacheRead)
				}
				if m.CacheCreated != 200 {
					t.Errorf("CacheCreated = %d, want 200", m.CacheCreated)
				}
				if m.OutputTokens != 800 {
					t.Errorf("OutputTokens = %d, want 800", m.OutputTokens)
				}
			},
		},
		{
			name: "result event in middle of file",
			content: `{"type":"assistant","text":"hello"}
{"type":"tool_use","name":"edit"}
{"type":"result","total_cost_usd":0.01,"duration_ms":5000,"usage":{"input_tokens":100,"cache_read_input_tokens":0,"cache_creation_input_tokens":0,"output_tokens":50}}
{"type":"done"}`,
			check: func(t *testing.T, m *Metrics) {
				if m.CostUSD != 0.01 {
					t.Errorf("CostUSD = %f, want 0.01", m.CostUSD)
				}
				if m.InputTokens != 100 {
					t.Errorf("InputTokens = %d, want 100", m.InputTokens)
				}
			},
		},
		{
			name:    "no result event",
			content: `{"type":"assistant","text":"hello"}`,
			wantErr: "no result event found",
		},
		{
			name:    "empty file",
			content: "",
			wantErr: "no result event found",
		},
		{
			name:    "malformed JSON on result line",
			content: `{"type":"result", broken json`,
			wantErr: "no result event found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "raw.jsonl")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			m, err := ExtractFromJSONL(path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, m)
		})
	}
}

func TestExtractAnswer(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		content := `{"type":"assistant","text":"working"}
{"type":"result","total_cost_usd":0.01,"duration_ms":1000,"result":"The fix is applied","usage":{"input_tokens":100,"cache_read_input_tokens":0,"cache_creation_input_tokens":0,"output_tokens":50}}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := ExtractAnswer(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "The fix is applied" {
			t.Errorf("got %q, want %q", got, "The fix is applied")
		}
	})

	t.Run("no result event", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		if err := os.WriteFile(path, []byte(`{"type":"assistant","text":"hi"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := ExtractAnswer(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty result string", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		content := `{"type":"result","total_cost_usd":0.01,"duration_ms":1000,"result":"","usage":{"input_tokens":100,"cache_read_input_tokens":0,"cache_creation_input_tokens":0,"output_tokens":50}}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := ExtractAnswer(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

func TestMetrics_SaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")

	original := &Metrics{
		CostUSD:      0.0042,
		DurationMS:   15000,
		InputTokens:  5000,
		CacheRead:    1000,
		CacheCreated: 200,
		OutputTokens: 800,
	}

	if err := original.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if *loaded != *original {
		t.Errorf("loaded metrics %+v != original %+v", loaded, original)
	}
}

func TestLoadFromFile_Errors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := LoadFromFile("/nonexistent/path/metrics.json")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadFromFile(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestCodexEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw.jsonl")
	data := `{"type":"thread.started","thread_id":"123"}
{"type":"item.completed","item":{"type":"agent_message","text":"Fixed."}}
{"type": "turn.completed", "usage":{"input_tokens":1000,"cached_input_tokens":800,"output_tokens":50}}
{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":10}}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := ExtractFromJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	if !m.CostUnknown || m.CostLabel() != "n/a" || m.InputTokens != 280 || m.CacheRead != 820 || m.OutputTokens != 60 || m.TotalTokens() != 1160 {
		t.Fatalf("incorrect Codex normalization: %+v", m)
	}
	answer, err := ExtractAnswer(path)
	if err != nil || answer != "Fixed." {
		t.Fatalf("answer %q: %v", answer, err)
	}
}

func TestFailedCompletion(t *testing.T) {
	for _, content := range []string{
		`{"type":"turn.failed","error":{"message":"limit reached"}}`,
		`{"type":"result","subtype":"error_max_turns","is_error":true,"result":"unfinished"}`,
		`{"type":"turn.completed","usage":{}}
{"type":"turn.failed","error":{"message":"failed later"}}`,
	} {
		path := filepath.Join(t.TempDir(), "raw.jsonl")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ExtractFromJSONL(path); err == nil {
			t.Fatalf("accepted failed run: %s", content)
		}
	}
}
