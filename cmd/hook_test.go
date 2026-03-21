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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/config"
)

func TestGenerateSessionContext_NoIndex(t *testing.T) {
	// Use the internal version with a no-op bgIndexer to avoid spawning the
	// test binary as a background process (which would trigger a fork bomb:
	// the spawned binary runs all tests, which spawn more binaries, etc.)
	content := generateSessionContextInternal("lumen", "/nonexistent/path",
		func(_, _ string) string { return "" },
		func(_ string) {},
	)
	if !strings.Contains(content, "mcp__lumen__semantic_search") {
		t.Error("content should reference the semantic_search tool")
	}
	if strings.Contains(content, "EXTREMELY_IMPORTANT") {
		t.Error("content should not contain EXTREMELY_IMPORTANT directives")
	}
}

func TestEvaluateToolCall_GrepAlwaysSuggests(t *testing.T) {
	cases := []string{
		"error handling middleware",
		"handleSemanticSearch",
		`func\s+\w+Search.*context\.Context`,
		"decodeStruct",
		`structDecodeWithNullValue\|decodeStruct`,
	}
	for _, pattern := range cases {
		t.Run(pattern, func(t *testing.T) {
			input := preToolUseInput{
				ToolName: "Grep",
				Input:    map[string]any{"pattern": pattern},
			}
			result := evaluateToolCall(input, "lumen")
			if result == nil {
				t.Fatal("expected suggestion for Grep, got nil")
			}
			if result.HookSpecificOutput.PermissionDecision != "" {
				t.Errorf("expected no permissionDecision, got %q", result.HookSpecificOutput.PermissionDecision)
			}
			if !strings.Contains(result.HookSpecificOutput.AdditionalContext, "mcp__lumen__semantic_search") {
				t.Error("additionalContext should reference semantic_search tool")
			}
		})
	}
}

func TestEvaluateToolCall_GlobAlwaysSuggests(t *testing.T) {
	input := preToolUseInput{
		ToolName: "Glob",
		Input:    map[string]any{"pattern": "**/*.go"},
	}
	result := evaluateToolCall(input, "lumen")
	if result == nil {
		t.Fatal("expected suggestion for Glob, got nil")
	}
	if result.HookSpecificOutput.PermissionDecision != "" {
		t.Errorf("expected no permissionDecision, got %q", result.HookSpecificOutput.PermissionDecision)
	}
}

func TestEvaluateToolCall_OtherToolSilentAllow(t *testing.T) {
	input := preToolUseInput{
		ToolName: "Read",
		Input:    map[string]any{"path": "/some/file.go"},
	}
	result := evaluateToolCall(input, "lumen")
	if result != nil {
		t.Errorf("expected nil (silent allow) for Read, got suggestion")
	}
}

func TestEvaluateToolCall_BashGrepSuggests(t *testing.T) {
	cases := []string{
		`grep -r "error handling middleware" ./cmd`,
		`grep -n "handleSemanticSearch" .`,
		`grep -n "decodeStruct\|structDecode" ./internal`,
	}
	for _, cmd := range cases {
		t.Run(cmd[:min(40, len(cmd))], func(t *testing.T) {
			input := preToolUseInput{
				ToolName: "Bash",
				Input:    map[string]any{"command": cmd},
			}
			result := evaluateToolCall(input, "lumen")
			if result == nil {
				t.Fatal("expected suggestion for bash grep, got nil")
			}
			if !strings.Contains(result.HookSpecificOutput.AdditionalContext, "mcp__lumen__semantic_search") {
				t.Error("additionalContext should reference semantic_search tool")
			}
		})
	}
}

func TestEvaluateToolCall_BashNonSearchSilentAllow(t *testing.T) {
	cases := []string{
		"go build ./...",
		"go test ./...",
		"git diff HEAD",
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			input := preToolUseInput{
				ToolName: "Bash",
				Input:    map[string]any{"command": cmd},
			}
			result := evaluateToolCall(input, "lumen")
			if result != nil {
				t.Errorf("expected nil for non-search bash command %q, got suggestion", cmd)
			}
		})
	}
}

func TestPreToolUseOutputJSON(t *testing.T) {
	result := evaluateToolCall(preToolUseInput{
		ToolName: "Grep",
		Input:    map[string]any{"pattern": "error handling middleware"},
	}, "lumen")
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	hso, ok := parsed["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("missing hookSpecificOutput key")
	}
	if _, exists := hso["permissionDecision"]; exists {
		t.Errorf("expected no permissionDecision field, got %v", hso["permissionDecision"])
	}
	if hso["hookEventName"] != "PreToolUse" {
		t.Errorf("expected hookEventName=PreToolUse, got %v", hso["hookEventName"])
	}
}

func TestGenerateSessionContextInternal_SpawnsWhenNoDB(t *testing.T) {
	// No DB exists → bgIndexer must be called regardless of donor presence.
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	t.Run("with donor", func(t *testing.T) {
		var bgCwd string
		generateSessionContextInternal("lumen", "/my/worktree",
			func(_, _ string) string { return "/some/donor.db" },
			func(cwd string) { bgCwd = cwd },
		)
		if bgCwd != "/my/worktree" {
			t.Fatalf("expected bgIndexer called with /my/worktree, got %q", bgCwd)
		}
	})

	t.Run("without donor", func(t *testing.T) {
		var bgCwd string
		generateSessionContextInternal("lumen", "/my/worktree",
			func(_, _ string) string { return "" },
			func(cwd string) { bgCwd = cwd },
		)
		if bgCwd != "/my/worktree" {
			t.Fatalf("expected bgIndexer called even without donor, got %q", bgCwd)
		}
	})
}

func TestGenerateSessionContextInternal_NoSpawnWhenDBExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Use the same model the function will load so the DB path matches.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	dbPath := config.DBPathForProject("/myproject", cfg.Model)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	called := false
	generateSessionContextInternal("lumen", "/myproject",
		func(_, _ string) string { return "/some/donor.db" },
		func(_ string) { called = true },
	)
	if called {
		t.Fatal("bgIndexer must not be called when an index already exists")
	}
}

func TestGenerateSessionContextInternal_MessageWithDonor(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	result := generateSessionContextInternal("lumen", "/my/worktree",
		func(_, _ string) string { return "/some/donor.db" },
		func(_ string) {},
	)
	if !strings.Contains(result, "background") {
		t.Errorf("expected 'background' in context when donor found, got: %s", result)
	}
}

func TestGenerateSessionContextInternal_MessageWithoutDonor(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	result := generateSessionContextInternal("lumen", "/my/worktree",
		func(_, _ string) string { return "" },
		func(_ string) {},
	)
	if !strings.Contains(result, "background") {
		t.Errorf("expected 'background' in context when no donor, got: %s", result)
	}
}

func TestHookOutputJSON(t *testing.T) {
	// Use the internal version with a no-op bgIndexer — same fork-bomb reason
	// as in TestGenerateSessionContext_NoIndex.
	content := generateSessionContextInternal("lumen", "/nonexistent/path",
		func(_, _ string) string { return "" },
		func(_ string) {},
	)
	out := hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName:     "SessionStart",
			AdditionalContext: content,
		},
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	hso, ok := parsed["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("missing hookSpecificOutput key")
	}
	if hso["hookEventName"] != "SessionStart" {
		t.Errorf("hookEventName = %v, want SessionStart", hso["hookEventName"])
	}
	ctx, ok := hso["additionalContext"].(string)
	if !ok || !strings.Contains(ctx, "mcp__lumen__semantic_search") {
		t.Error("additionalContext should contain tool reference")
	}
}

// TestSpawnBackgroundIndexer_DoesNotPanic verifies that spawnBackgroundIndexer
// does not panic or block on a path that contains no indexable files.
// The spawned process will acquire the lock, find nothing to index, and exit.
func TestSpawnBackgroundIndexer_DoesNotPanic(t *testing.T) {
	// Use a temp directory — no Go files, so the indexer exits quickly.
	spawnBackgroundIndexer(t.TempDir())
	// If we reach here without panic or deadlock, the test passes.
}
