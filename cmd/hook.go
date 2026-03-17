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

	"github.com/spf13/cobra"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/store"
)

// NOTE: Hooks are now declared in hooks/hooks.json (plugin system).
// The hook subcommands remain as the execution targets for those declarations.

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookSessionStartCmd)
	hookCmd.AddCommand(hookPreToolUseCmd)
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook handlers for AI coding agent integration",
}

var hookSessionStartCmd = &cobra.Command{
	Use:   "session-start [mcp-name]",
	Short: "Output SessionStart hook JSON for Claude Code",
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

// sessionStartInput is the JSON structure Claude Code sends to SessionStart hooks.
type sessionStartInput struct {
	CWD string `json:"cwd"`
}

func runHookSessionStart(_ *cobra.Command, args []string) error {
	mcpName := filepath.Base(os.Args[0])
	if len(args) > 0 {
		mcpName = args[0]
	}

	var input sessionStartInput
	_ = json.NewDecoder(os.Stdin).Decode(&input)

	cwd := input.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	content := generateSessionContext(mcpName, cwd)

	out := hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName:     "SessionStart",
			AdditionalContext: content,
		},
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}

// generateSessionContext builds concise context for the SessionStart hook.
// If an index exists, it includes stats and top symbols to create a natural
// data dependency — Claude sees useful data and wants more from the tool.
func generateSessionContext(mcpName, cwd string) string {
	toolRef := "mcp__" + mcpName + "__semantic_search"
	directive := "Call " + toolRef + " first for any code discovery task — before Grep, Bash, or Read."

	cfg, err := config.Load()
	if err != nil {
		return directive + " No index yet — auto-created on first call."
	}

	dbPath := config.DBPathForProject(cwd, cfg.Model)
	if _, err := os.Stat(dbPath); err != nil {
		if donorPath := config.FindDonorIndex(cwd, cfg.Model); donorPath != "" {
			return directive + " Sibling worktree index found — fast incremental re-index on first search."
		}
		return directive + " No index yet — auto-created on first call."
	}

	s, err := store.New(dbPath, cfg.Dims)
	if err != nil {
		return directive
	}
	defer func() { _ = s.Close() }()

	stats, err := s.Stats()
	if err != nil {
		return directive
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

	result := evaluateToolCall(input, mcpName)
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
func evaluateToolCall(input preToolUseInput, mcpName string) *hookOutput {
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

	toolRef := "mcp__" + mcpName + "__semantic_search"
	return &hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName: "PreToolUse",
			AdditionalContext: fmt.Sprintf(
				"Use %s instead of Grep/Glob/find/rg for significantly faster and better search results to reduce context window use and give better quality results.",
				toolRef,
			),
		},
	}
}
