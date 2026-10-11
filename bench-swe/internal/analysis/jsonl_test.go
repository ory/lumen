package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractToolCalls_NestedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.jsonl")
	content := `{"type":"system","subtype":"init","session_id":"s1"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu1","name":"mcp__lumen__semantic_search","input":{"query":"error handling","path":"/repo"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":[{"type":"text","text":"<result:file filename=\"handler.go\">code here</result:file>"}]}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu2","name":"Read","input":{"file_path":"/repo/handler.go"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu2","content":[{"type":"text","text":"package main\nfunc handler() {}"}]}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu3","name":"mcp__lumen__semantic_search","input":{"query":"middleware","path":"/repo"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu3","content":[{"type":"text","text":"<result:file filename=\"middleware.go\">mw code</result:file>"}]}]}}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	calls, err := ExtractToolCalls(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("got %d calls, want 3", len(calls))
	}

	// First call: semantic_search
	if calls[0].Name != "mcp__lumen__semantic_search" {
		t.Errorf("calls[0].Name = %q, want mcp__lumen__semantic_search", calls[0].Name)
	}
	if calls[0].Input["query"] != "error handling" {
		t.Errorf("calls[0] query = %q, want %q", calls[0].Input["query"], "error handling")
	}
	if calls[0].Result == "" || calls[0].Result == "<no result>" {
		t.Errorf("calls[0].Result is empty, expected content with handler.go")
	}

	// Second call: Read
	if calls[1].Name != "Read" {
		t.Errorf("calls[1].Name = %q, want Read", calls[1].Name)
	}

	// Third call: semantic_search
	if calls[2].Name != "mcp__lumen__semantic_search" {
		t.Errorf("calls[2].Name = %q, want mcp__lumen__semantic_search", calls[2].Name)
	}
	if calls[2].Result == "" {
		t.Error("calls[2].Result is empty, expected content with middleware.go")
	}

	// Check indices
	for i, c := range calls {
		if c.Index != i {
			t.Errorf("calls[%d].Index = %d, want %d", i, c.Index, i)
		}
	}
}

func TestExtractToolCalls_TopLevelFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.jsonl")
	content := `{"type":"tool_use","id":"tu1","name":"mcp__lumen__semantic_search","input":{"query":"test","cwd":"/repo"}}
{"type":"tool_result","tool_use_id":"tu1","content":"plain string result"}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	calls, err := ExtractToolCalls(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}
	if calls[0].Result != "plain string result" {
		t.Errorf("Result = %q, want %q", calls[0].Result, "plain string result")
	}
}

func TestExtractToolCalls_ListContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.jsonl")
	content := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu1","name":"Bash","input":{"command":"ls"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":[{"type":"text","text":"file1.go"},{"type":"text","text":"file2.go"}]}]}}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	calls, err := ExtractToolCalls(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}
	if calls[0].Result != "file1.go\nfile2.go" {
		t.Errorf("Result = %q, want %q", calls[0].Result, "file1.go\nfile2.go")
	}
}

func TestExtractToolCalls_SkipsNonToolEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.jsonl")
	content := `{"type":"system","subtype":"init","hook_id":"h1","session_id":"s1"}
{"type":"system","subtype":"hook","hook_event":"SessionStart","session_id":"s1"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"Let me think..."}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Here is the answer"}]}}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	calls, err := ExtractToolCalls(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("got %d calls, want 0", len(calls))
	}
}

func TestExtractToolCalls_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.jsonl")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	calls, err := ExtractToolCalls(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("got %d calls, want 0", len(calls))
	}
}

func TestExtractToolCalls_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.jsonl")
	content := `not valid json
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu1","name":"Read","input":{"path":"/f"}}]}}
{also broken
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":"ok"}]}}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	calls, err := ExtractToolCalls(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}
	if calls[0].Result != "ok" {
		t.Errorf("Result = %q, want %q", calls[0].Result, "ok")
	}
}

func TestExtractToolCalls_MissingFile(t *testing.T) {
	_, err := ExtractToolCalls("/nonexistent/raw.jsonl")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIsLumenSearchTool(t *testing.T) {
	for name, want := range map[string]bool{
		"mcp__lumen__semantic_search":              true,
		"mcp__plugin_lumen_lumen__semantic_search": true,
		"mcp__lumen__index_status":                 false,
		"Grep":                                     false,
	} {
		if got := IsLumenSearchTool(name); got != want {
			t.Errorf("IsLumenSearchTool(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestCodexToolEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw.jsonl")
	data := `{"type":"item.started","item":{"id":"m1","type":"mcp_tool_call","server":"plugin_lumen_lumen","tool":"semantic_search","status":"in_progress"}}
{"type":"item.updated","item":{"id":"m1","type":"mcp_tool_call","server":"plugin_lumen_lumen","tool":"semantic_search","status":"in_progress"}}
{"type":"item.completed","item":{"id":"m1","type":"mcp_tool_call","server":"plugin_lumen_lumen","tool":"semantic_search","arguments":{"query":"retry","path":"/repo"},"status":"completed","result":{"content":[{"type":"text","text":"budget.go"}]},"error":null}}
{"type":"item.completed","item":{"id":"c1","type":"command_execution","command":"go test ./...","aggregated_output":"ok","status":"completed"}}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	calls, err := ExtractToolCalls(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].Result != "budget.go" || calls[0].SearchPath() != "/repo" || calls[1].Name != "Bash" {
		t.Fatalf("calls: %+v", calls)
	}
	if ok, err := HasLumenSearch(path); err != nil || !ok {
		t.Fatalf("missing successful call: %v", err)
	}
}

func TestSearchRequiresSuccessfulToolResult(t *testing.T) {
	for _, data := range []string{
		`{"type":"assistant","message":{"content":[{"type":"text","text":"mcp__plugin_lumen_lumen__semantic_search"}]}}`,
		`{"type":"tool_use","id":"x","name":"mcp__lumen__semantic_search","input":{}}`,
		`{"type":"tool_use","id":"x","name":"mcp__lumen__semantic_search","input":{}}
{"type":"tool_result","tool_use_id":"x","content":"failed","is_error":true}`,
		`{"type":"item.completed","item":{"id":"x","type":"mcp_tool_call","server":"lumen","tool":"semantic_search","status":"failed","error":{"message":"broken"}}}`,
	} {
		path := filepath.Join(t.TempDir(), "raw.jsonl")
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if ok, err := HasLumenSearch(path); err != nil || ok {
			t.Fatalf("false positive search: %s (%v)", data, err)
		}
	}
}
