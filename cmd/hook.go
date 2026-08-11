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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/git"
	"github.com/ory/lumen/internal/store"
)

// backgroundIndexStaleness is how old last_indexed_at must be before
// SessionStart spawns a background indexer. This prevents every new terminal
// from triggering a full merkle walk when the index was just updated.
const backgroundIndexStaleness = 5 * time.Minute

const (
	hookHostClaude = "claude"
	hookHostCursor = "cursor"
)

// NOTE: Hooks are now declared in hooks/hooks.json (plugin system).
// The hook subcommands remain as the execution targets for those declarations.

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookSessionStartCmd)
	hookCmd.AddCommand(hookPreToolUseCmd)

	hookSessionStartCmd.Flags().StringVar(&hookSessionStartHost, "host", hookHostClaude, "Hook host output format")
	hookSessionStartCmd.Flags().StringVar(&hookSessionStartPluginName, "plugin-name", "", "Claude Code plugin name")
	hookPreToolUseCmd.Flags().StringVar(&hookPreToolUsePluginName, "plugin-name", "", "Claude Code plugin name")
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook handlers for AI coding agent integration",
}

var (
	hookSessionStartHost       string
	hookSessionStartPluginName string
	hookPreToolUsePluginName   string
)

var hookSessionStartCmd = &cobra.Command{
	Use:   "session-start [mcp-name]",
	Short: "Output SessionStart hook JSON for Claude Code or Cursor",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runHookSessionStart,
}

// hookOutput is the JSON structure Claude Code expects from a synchronous hook.
type hookOutput struct {
	HookSpecificOutput hookSpecificOutput `json:"hookSpecificOutput"`
}

type hookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

type cursorHookOutput struct {
	AdditionalContext string `json:"additional_context"`
}

// sessionStartInput is the subset of SessionStart hook input used across hosts.
type sessionStartInput struct {
	CWD string `json:"cwd"`
}

func runHookSessionStart(_ *cobra.Command, args []string) error {
	mcpName := filepath.Base(os.Args[0])
	if len(args) > 0 {
		mcpName = args[0]
	}

	host, err := normalizeHookHost(hookSessionStartHost)
	if err != nil {
		return err
	}

	var input sessionStartInput
	_ = json.NewDecoder(os.Stdin).Decode(&input)

	cwd := input.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	content := generateSessionContextForHost(host, mcpName, hookSessionStartPluginName, cwd)
	out := sessionStartOutput(host, content)

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}

func generateSessionContextForHost(host, mcpName, pluginName, cwd string) string {
	return generateSessionContextInternalWithDirective(sessionStartDirective(host, mcpName, pluginName), cwd, config.FindDonorIndex, spawnBackgroundIndexer)
}

func sessionStartDirective(host, mcpName, pluginName string) string {
	if host == hookHostCursor {
		return "Use the Lumen semantic_search tool first for any code discovery task — before Grep, Bash, or Read."
	}
	toolRef := mcpToolReference(pluginName, mcpName, "semantic_search")
	return "Load and call " + toolRef + " first for any code discovery task — before Grep, Bash, or Read."
}

func mcpToolReference(pluginName, mcpName, toolName string) string {
	if pluginName != "" {
		mcpName = "plugin_" + pluginName + "_" + mcpName
	}
	return "mcp__" + mcpName + "__" + toolName
}

func generateSessionContextInternalWithDirective(directive, cwd string, findDonor func(string, string) string, bgIndexer func(string)) string {
	cfg, err := config.NewConfigService(config.DefaultConfigPath())
	if err != nil {
		return directive + " No index yet — auto-created on first call."
	}
	emb := newEmbedder(cfg)
	modelName := emb.ModelName()
	dims := cfg.ServerDims(0)

	// Normalize cwd to the git repository root so the DB path matches what
	// `lumen index` and the MCP handler use. For non-git directories, walk
	// up to reuse an existing ancestor index.
	if root, err := git.RepoRoot(cwd); err == nil {
		cwd = root
	} else if ancestor := findAncestorIndex(cwd, modelName); ancestor != "" {
		cwd = ancestor
	}

	dbPath := configuredDBPath(cfg, cwd, modelName)
	if _, err := os.Stat(dbPath); err != nil {
		// No index yet — kick off background pre-warming so the first search
		// in this session doesn't pay the full seed + embed cost synchronously.
		bgIndexer(cwd)
		if donorPath := findDonor(cwd, modelName); donorPath != "" {
			return directive + " Sibling worktree index found — indexing in background."
		}
		return directive + " No index yet — indexing in background."
	}

	s, err := store.NewCollection(dbPath, dims, cfg.VectorStorage(), cwd)
	if err != nil {
		return directive
	}
	defer func() { _ = s.Close() }()

	stats, err := s.Stats()
	if err != nil {
		return directive
	}

	// An index is only ready after a successful indexing pass produced chunks.
	// A database can exist (and even contain file rows) after embedding failed;
	// announcing that state as ready hides broken model/backend installations.
	lastIndexedAt, metaErr := s.GetMeta("last_indexed_at")
	lastIndexError, _ := s.GetMeta(store.MetaLastIndexError)
	completed := metaErr == nil && lastIndexedAt != ""
	stale := true
	if completed {
		if t, parseErr := time.Parse(time.RFC3339, lastIndexedAt); parseErr == nil {
			stale = time.Since(t) > backgroundIndexStaleness
		}
	}
	if !completed || stale || stats.TotalChunks == 0 || lastIndexError != "" {
		bgIndexer(cwd)
	}
	if lastIndexError != "" {
		return fmt.Sprintf("Lumen index unhealthy: last indexing attempt failed (%d files, %d chunks available) — retrying in background. %s", stats.TotalFiles, stats.TotalChunks, directive)
	}
	if !completed || stats.TotalChunks == 0 {
		return fmt.Sprintf("Lumen index not ready: %d files, %d chunks indexed — indexing in background. %s", stats.TotalFiles, stats.TotalChunks, directive)
	}

	symbols, _ := s.TopSymbols(10)

	var sb strings.Builder
	fmt.Fprintf(&sb, "Lumen index ready: %d files, %d chunks indexed.", stats.TotalFiles, stats.TotalChunks)
	if len(symbols) > 0 {
		sb.WriteString(" Top symbols: ")
		sb.WriteString(strings.Join(symbols, ", "))
		sb.WriteString(".")
	}
	sb.WriteString(" " + directive)
	return sb.String()
}

// generateSessionContextInternal is the testable core of generateSessionContext.
// findDonor and bgIndexer are injected so tests can verify behaviour without
// spawning real processes or requiring a live git repository.
func generateSessionContextInternal(cwd string, findDonor func(string, string) string, bgIndexer func(string)) string {
	return generateSessionContextInternalWithDirective(sessionStartDirective(hookHostClaude, "lumen", ""), cwd, findDonor, bgIndexer)
}

func normalizeHookHost(host string) (string, error) {
	switch strings.ToLower(host) {
	case "", hookHostClaude:
		return hookHostClaude, nil
	case hookHostCursor:
		return hookHostCursor, nil
	default:
		return "", fmt.Errorf("unsupported hook host %q", host)
	}
}

func sessionStartOutput(host, content string) any {
	if host == hookHostCursor {
		return cursorHookOutput{AdditionalContext: content}
	}
	return hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName:     "SessionStart",
			AdditionalContext: content,
		},
	}
}

// --- PreToolUse hook ---

var hookPreToolUseCmd = &cobra.Command{
	Use:   "pre-tool-use [mcp-name]",
	Short: "Intercept Grep calls and suggest semantic search when appropriate",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runHookPreToolUse,
}

// preToolUseInput is the JSON structure Claude Code sends to PreToolUse hooks.
type preToolUseInput struct {
	ToolName string         `json:"tool_name"`
	Input    map[string]any `json:"tool_input"`
}

func runHookPreToolUse(_ *cobra.Command, args []string) error {
	mcpName := filepath.Base(os.Args[0])
	if len(args) > 0 {
		mcpName = args[0]
	}

	var input preToolUseInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		// Can't parse input — allow silently (exit 0, no stdout).
		return nil
	}

	result := evaluateToolCall(input, mcpName, hookPreToolUsePluginName)
	if result == nil {
		// Silent allow — exit 0 with no stdout.
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}

// evaluateToolCall determines whether a tool call should be intercepted
// with a suggestion to use semantic search instead.
// Returns nil for silent allow (no output), or a hookOutput with a suggestion.
func evaluateToolCall(input preToolUseInput, mcpName, pluginName string) *hookOutput {
	switch input.ToolName {
	case "Grep", "Glob":
		// Always suggest semantic search for any file/code search.
	case "Bash":
		cmd, _ := input.Input["command"].(string)
		if !strings.Contains(cmd, "grep") && !strings.Contains(cmd, "rg ") && !strings.Contains(cmd, "find ") {
			return nil
		}
	default:
		return nil
	}

	toolRef := mcpToolReference(pluginName, mcpName, "semantic_search")
	return &hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName: "PreToolUse",
			AdditionalContext: fmt.Sprintf(
				"Load and call %s instead of Grep/Glob/find/rg for significantly faster and better search results to reduce context window use and give better quality results.",
				toolRef,
			),
		},
	}
}
