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
	"github.com/ory/lumen/internal/merkle"
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
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook handlers for AI coding agent integration",
}

var hookSessionStartHost string

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

	content := generateSessionContextForHost(host, mcpName, cwd)
	out := sessionStartOutput(host, content)

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}

func generateSessionContextForHost(host, mcpName, cwd string) string {
	return generateSessionContextInternalWithDirective(sessionStartDirective(host, mcpName), cwd, config.FindDonorIndex, spawnBackgroundIndexer)
}

func sessionStartDirective(host, mcpName string) string {
	if host == hookHostCursor {
		return "Use the Lumen semantic_search tool first for any code discovery task — before Grep, Bash, or Read."
	}
	toolRef := "mcp__" + mcpName + "__semantic_search"
	return "Call " + toolRef + " first for any code discovery task — before Grep, Bash, or Read."
}

func generateSessionContextInternalWithDirective(directive, cwd string, findDonor func(string, string) string, bgIndexer func(string)) string {
	cfg, err := config.NewConfigService(config.DefaultConfigPath())
	if err != nil {
		return directive + " No index yet — auto-created on first call."
	}
	emb := newEmbedder(cfg)
	modelName := emb.ModelName()
	dims := cfg.ServerDims(0)
	allowBackgroundIndex := false

	// Normalize cwd to the git repository root so the DB path matches what
	// `lumen index` and the MCP handler use. For non-git directories, reuse an
	// existing ancestor index, but do not invent a new background crawl at a
	// plain parent/root just because a host opened a session there.
	if root, err := git.RepoRoot(cwd); err == nil {
		if unindexable, _ := merkle.IsRootUnindexable(root); !unindexable {
			cwd = root
			allowBackgroundIndex = true
		} else if !hasLumenBoundaryFile(cwd) {
			if ancestor := findAncestorIndex(cwd, modelName); ancestor != "" {
				cwd = ancestor
				allowBackgroundIndex = true
			}
		}
	} else if !hasLumenBoundaryFile(cwd) {
		if ancestor := findAncestorIndex(cwd, modelName); ancestor != "" {
			cwd = ancestor
			allowBackgroundIndex = true
		}
	} else {
		allowBackgroundIndex = true
	}
	if hasLumenBoundaryFile(cwd) {
		allowBackgroundIndex = true
	}
	if unindexable, reason := merkle.IsRootUnindexable(cwd); unindexable {
		return directive + " Index root blocked: " + reason + "."
	}

	dbPath := config.DBPathForProject(cwd, modelName)
	if _, err := os.Stat(dbPath); err != nil {
		if !allowBackgroundIndex {
			return directive + " No index yet — auto-created on first semantic_search call."
		}

		// No index yet — kick off background pre-warming so the first search
		// in this session doesn't pay the full seed + embed cost synchronously.
		bgIndexer(cwd)
		if donorPath := findDonor(cwd, modelName); donorPath != "" {
			return directive + " Sibling worktree index found — indexing in background."
		}
		return directive + " No index yet — indexing in background."
	}

	s, err := store.New(dbPath, dims)
	if err != nil {
		return directive
	}
	defer func() { _ = s.Close() }()

	// Spawn background indexer if the index is stale or has never been
	// successfully completed. This avoids spawning on every session start
	// when the index was recently updated.
	if allowBackgroundIndex {
		if val, metaErr := s.GetMeta("last_indexed_at"); metaErr != nil || val == "" {
			bgIndexer(cwd)
		} else if t, parseErr := time.Parse(time.RFC3339, val); parseErr != nil || time.Since(t) > backgroundIndexStaleness {
			bgIndexer(cwd)
		}
	}

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

// generateSessionContextInternal is the testable core of generateSessionContext.
// findDonor and bgIndexer are injected so tests can verify behaviour without
// spawning real processes or requiring a live git repository.
func generateSessionContextInternal(cwd string, findDonor func(string, string) string, bgIndexer func(string)) string {
	return generateSessionContextInternalWithDirective(sessionStartDirective(hookHostClaude, "lumen"), cwd, findDonor, bgIndexer)
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
