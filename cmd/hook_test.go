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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/store"
)

// writeHookTestDB creates a minimal SQLite DB stamped with last_indexed_at,
// using the configured embedding model's dimensions so store.New doesn't reset
// the schema when generateSessionContextInternal opens it.
func writeHookTestDB(t *testing.T, dbPath string, lastIndexedAt time.Time) {
	t.Helper()
	cfg, err := config.NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	dims := cfg.ServerDims(0)
	s, err := store.New(dbPath, dims)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer func() { _ = s.Close() }()
	if err := s.SetMeta("last_indexed_at", lastIndexedAt.UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
}

// TestMain detects when the cmd test binary is invoked as a background
// indexer subprocess (via spawnBackgroundIndexer → os.Executable()) and exits
// immediately instead of running the full test suite.  Without this guard,
// TestSpawnBackgroundIndexer_DoesNotPanic would create a fork-bomb: the
// spawned test binary runs all tests, which spawns more binaries, etc.
func TestMain(m *testing.M) {
	// spawnBackgroundIndexer calls: exec.Command(exe, "index", projectPath)
	// where exe == os.Executable() == this test binary.
	// Detect that pattern and exit cleanly so no tests run in the subprocess.
	if len(os.Args) > 1 && os.Args[1] == "index" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestGenerateSessionContext_NoIndex(t *testing.T) {
	// Use the internal version with a no-op bgIndexer to avoid spawning the
	// test binary as a background process (which would trigger a fork bomb:
	// the spawned binary runs all tests, which spawn more binaries, etc.)
	content := generateSessionContextInternal("/nonexistent/path",
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

func TestGenerateSessionContextForCursor_NoIndex(t *testing.T) {
	content := generateSessionContextInternalWithDirective(
		sessionStartDirective(hookHostCursor, "lumen"),
		"/nonexistent/path",
		func(_, _ string) string { return "" },
		func(_ string) {},
	)
	if strings.Contains(content, "mcp__lumen__semantic_search") {
		t.Error("cursor content should not use Claude-style MCP tool ids")
	}
	if !strings.Contains(content, "Lumen semantic_search tool") {
		t.Error("cursor content should reference the generic Lumen semantic_search tool")
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
				return
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
		return
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
				return
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
		generateSessionContextInternal("/my/worktree",
			func(_, _ string) string { return "/some/donor.db" },
			func(cwd string) { bgCwd = cwd },
		)
		if bgCwd != "/my/worktree" {
			t.Fatalf("expected bgIndexer called with /my/worktree, got %q", bgCwd)
		}
	})

	t.Run("without donor", func(t *testing.T) {
		var bgCwd string
		generateSessionContextInternal("/my/worktree",
			func(_, _ string) string { return "" },
			func(cwd string) { bgCwd = cwd },
		)
		if bgCwd != "/my/worktree" {
			t.Fatalf("expected bgIndexer called even without donor, got %q", bgCwd)
		}
	})
}

func TestGenerateSessionContextInternal_NoSpawnWhenFresh(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := config.NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	emb := newEmbedder(cfg)
	dbPath := config.DBPathForProject("/myproject", emb.ModelName())
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeHookTestDB(t, dbPath, time.Now().Add(-30*time.Second))

	called := false
	generateSessionContextInternal("/myproject",
		func(_, _ string) string { return "" },
		func(_ string) { called = true },
	)
	if called {
		t.Fatal("bgIndexer must not be called when index was recently updated")
	}
}

func TestGenerateSessionContextInternal_SpawnsWhenStale(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	cfg, err := config.NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	emb := newEmbedder(cfg)
	dbPath := config.DBPathForProject("/myproject", emb.ModelName())
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeHookTestDB(t, dbPath, time.Now().Add(-10*time.Minute))

	called := false
	generateSessionContextInternal("/myproject",
		func(_, _ string) string { return "" },
		func(_ string) { called = true },
	)
	if !called {
		t.Fatal("bgIndexer must be called when index is stale")
	}
}

func TestGenerateSessionContextInternal_MessageWithDonor(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	result := generateSessionContextInternal("/my/worktree",
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

	result := generateSessionContextInternal("/my/worktree",
		func(_, _ string) string { return "" },
		func(_ string) {},
	)
	if !strings.Contains(result, "background") {
		t.Errorf("expected 'background' in context when no donor, got: %s", result)
	}
}

func TestGenerateSessionContextInternal_NormalizesToGitRoot(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create a git repo with a subdirectory.
	repoDir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repoDir, "sub", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	// Create a DB for the git root so the hook finds it.
	cfgSvc, cfgErr := config.NewConfigService("")
	if cfgErr != nil {
		t.Fatalf("NewConfigService: %v", cfgErr)
	}
	embSvc := newEmbedder(cfgSvc)
	// Resolve symlinks to match what git rev-parse --show-toplevel returns.
	resolvedRepo, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	dbPath := config.DBPathForProject(resolvedRepo, embSvc.ModelName())
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeHookTestDB(t, dbPath, time.Now().Add(-30*time.Second))

	// Pass a subdirectory as cwd — the hook should normalize to the git root
	// and find the existing DB.
	subDir := filepath.Join(resolvedRepo, "sub", "deep")
	result := generateSessionContextInternal(subDir,
		func(_, _ string) string { return "" },
		func(_ string) {},
	)

	// The result should contain "index ready" because the DB exists at the git root.
	if !strings.Contains(result, "index ready") {
		t.Errorf("expected hook to normalize cwd to git root and find index, got: %s", result)
	}
}

func TestGenerateSessionContextInternal_NonGitUsesParentIndex(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create a non-git directory hierarchy.
	parentDir := filepath.Join(tmpDir, "parent")
	deepDir := filepath.Join(parentDir, "child", "deep")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Resolve symlinks to match what filepath.Abs would produce on macOS.
	resolvedParent, err := filepath.EvalSymlinks(parentDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a DB for the parent directory.
	cfgSvc, cfgErr := config.NewConfigService("")
	if cfgErr != nil {
		t.Fatalf("NewConfigService: %v", cfgErr)
	}
	embSvc := newEmbedder(cfgSvc)
	dbPath := config.DBPathForProject(resolvedParent, embSvc.ModelName())
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeHookTestDB(t, dbPath, time.Now().Add(-30*time.Second))

	// Pass a deep subdirectory as cwd — the hook should walk up and find
	// the parent's index.
	resolvedDeep, err := filepath.EvalSymlinks(deepDir)
	if err != nil {
		t.Fatal(err)
	}
	result := generateSessionContextInternal(resolvedDeep,
		func(_, _ string) string { return "" },
		func(_ string) {},
	)

	if !strings.Contains(result, "index ready") {
		t.Errorf("expected hook to walk up to parent index and find it, got: %s", result)
	}
}

func TestHookOutputJSON(t *testing.T) {
	// Use the internal version with a no-op bgIndexer — same fork-bomb reason
	// as in TestGenerateSessionContext_NoIndex.
	content := generateSessionContextInternal("/nonexistent/path",
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

func TestSessionStartOutputCursorJSON(t *testing.T) {
	content := generateSessionContextInternalWithDirective(
		sessionStartDirective(hookHostCursor, "lumen"),
		"/nonexistent/path",
		func(_, _ string) string { return "" },
		func(_ string) {},
	)

	data, err := json.Marshal(sessionStartOutput(hookHostCursor, content))
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, exists := parsed["hookSpecificOutput"]; exists {
		t.Fatal("cursor output should not use Claude hookSpecificOutput payloads")
	}
	ctx, ok := parsed["additional_context"].(string)
	if !ok || !strings.Contains(ctx, "Lumen semantic_search tool") {
		t.Error("additional_context should contain the generic Lumen semantic_search tool reference")
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
