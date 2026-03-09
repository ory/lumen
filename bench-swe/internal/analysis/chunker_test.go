package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPatchFiles(t *testing.T) {
	tests := []struct {
		name  string
		patch string
		want  []string
	}{
		{
			name:  "standard diff",
			patch: "--- a/handler.go\n+++ b/handler.go\n@@ -1 +1 @@\n-old\n+new\n",
			want:  []string{"handler.go"},
		},
		{
			name:  "multiple files",
			patch: "--- a/file1.go\n+++ b/file1.go\n@@ -1 +1 @@\n-old\n+new\n--- a/pkg/file2.go\n+++ b/pkg/file2.go\n@@ -1 +1 @@\n-old\n+new\n",
			want:  []string{"file1.go", "pkg/file2.go"},
		},
		{
			name:  "no +++ lines",
			patch: "some random text\nno diff here\n",
			want:  nil,
		},
		{
			name:  "empty",
			patch: "",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPatchFiles(tt.patch)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d files, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("files[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractResultFiles(t *testing.T) {
	tests := []struct {
		name    string
		results string
		want    []string
	}{
		{
			name:    "single filename",
			results: `<result:file filename="handler.go">code here</result:file>`,
			want:    []string{"handler.go"},
		},
		{
			name:    "multiple filenames",
			results: "<result:file filename=\"handler.go\">code</result:file>\n<result:file filename=\"server.go\">code</result:file>",
			want:    []string{"handler.go", "server.go"},
		},
		{
			name:    "no filenames",
			results: "plain text with no xml tags",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResultFiles(tt.results)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d files, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("files[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"go", "Go"},
		{"", ""},
		{"Go", "Go"},
	}

	for _, tt := range tests {
		t.Run("input="+tt.input, func(t *testing.T) {
			got := capitalizeFirst(tt.input)
			if got != tt.want {
				t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractSearchCalls(t *testing.T) {
	t.Run("valid tool_use and tool_result", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		content := `{"type":"tool_use","id":"tu1","name":"mcp__lumen__semantic_search","input":{"query":"error handling","cwd":"/repo"}}
{"type":"tool_result","tool_use_id":"tu1","content":"<result:file filename=\"handler.go\">code here</result:file>"}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		calls, err := extractSearchCalls(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(calls) != 1 {
			t.Fatalf("got %d calls, want 1", len(calls))
		}
		if calls[0].Query != "error handling" {
			t.Errorf("Query = %q, want %q", calls[0].Query, "error handling")
		}
		if calls[0].CWD != "/repo" {
			t.Errorf("CWD = %q, want %q", calls[0].CWD, "/repo")
		}
		if !strings.Contains(calls[0].Results, "handler.go") {
			t.Errorf("Results does not contain handler.go: %q", calls[0].Results)
		}
	})

	t.Run("no semantic_search calls", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		content := `{"type":"tool_use","id":"tu1","name":"Read","input":{"path":"/file.go"}}
{"type":"tool_result","tool_use_id":"tu1","content":"file contents"}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		calls, err := extractSearchCalls(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(calls) != 0 {
			t.Errorf("got %d calls, want 0", len(calls))
		}
	})

	t.Run("nested Claude CLI format", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		content := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu1","name":"mcp__lumen__semantic_search","input":{"query":"CORS middleware","path":"/repo"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":[{"type":"text","text":"<result:file filename=\"middleware.go\">code</result:file>"}]}]}}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		calls, err := extractSearchCalls(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(calls) != 1 {
			t.Fatalf("got %d calls, want 1", len(calls))
		}
		if calls[0].Query != "CORS middleware" {
			t.Errorf("Query = %q, want %q", calls[0].Query, "CORS middleware")
		}
		if !strings.Contains(calls[0].Results, "middleware.go") {
			t.Errorf("Results does not contain middleware.go: %q", calls[0].Results)
		}
	})

	t.Run("list content format", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		content := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu1","name":"mcp__lumen__semantic_search","input":{"query":"handler","path":"/repo"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":[{"type":"text","text":"<result:file filename=\"a.go\">part1</result:file>"},{"type":"text","text":"<result:file filename=\"b.go\">part2</result:file>"}]}]}}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		calls, err := extractSearchCalls(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(calls) != 1 {
			t.Fatalf("got %d calls, want 1", len(calls))
		}
		if !strings.Contains(calls[0].Results, "a.go") || !strings.Contains(calls[0].Results, "b.go") {
			t.Errorf("Results missing expected files: %q", calls[0].Results)
		}
	})

	t.Run("mixed tools filters to search only", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		content := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu1","name":"Bash","input":{"command":"ls"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":"file list"}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu2","name":"mcp__lumen__semantic_search","input":{"query":"test","path":"/repo"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu2","content":[{"type":"text","text":"<result:file filename=\"test.go\">code</result:file>"}]}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu3","name":"Read","input":{"file_path":"/repo/test.go"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu3","content":"package main"}]}}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		calls, err := extractSearchCalls(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(calls) != 1 {
			t.Fatalf("got %d calls, want 1 (only semantic_search)", len(calls))
		}
		if calls[0].Query != "test" {
			t.Errorf("Query = %q, want %q", calls[0].Query, "test")
		}
	})

	t.Run("path field used as CWD fallback", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "raw.jsonl")
		content := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu1","name":"mcp__lumen__semantic_search","input":{"query":"q","path":"/my/repo"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":"result"}]}}`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		calls, err := extractSearchCalls(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(calls) != 1 {
			t.Fatalf("got %d calls, want 1", len(calls))
		}
		if calls[0].CWD != "/my/repo" {
			t.Errorf("CWD = %q, want %q", calls[0].CWD, "/my/repo")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := extractSearchCalls("/nonexistent/raw.jsonl")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
