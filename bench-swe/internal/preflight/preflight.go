package preflight

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/runner"
)

type Config struct {
	RepoRoot    string
	LumenBinary string
	Backend     string
	EmbedModel  string
	OllamaHost  string
}

func Validate(ctx context.Context, cfg *Config) error {
	fmt.Println("Pre-flight checks...")

	if err := buildLumen(ctx, cfg); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	if err := checkOllama(cfg); err != nil {
		return fmt.Errorf("ollama: %w", err)
	}

	if err := checkClaude(ctx); err != nil {
		return fmt.Errorf("claude: %w", err)
	}

	if err := probeScenarios(ctx, cfg); err != nil {
		return fmt.Errorf("scenario probes: %w", err)
	}

	fmt.Println("Pre-flight checks passed.")
	return nil
}

func buildLumen(ctx context.Context, cfg *Config) error {
	fmt.Print("  Building lumen... ")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", cfg.LumenBinary, ".")
	cmd.Dir = cfg.RepoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	fmt.Println("ok")
	return nil
}

func checkOllama(cfg *Config) error {
	host := cfg.OllamaHost
	if host == "" {
		host = "http://localhost:11434"
	}

	fmt.Printf("  Checking Ollama at %s... ", host)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(host + "/api/tags")
	if err != nil {
		return fmt.Errorf("cannot reach Ollama at %s: %w", host, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ollama returned %d", resp.StatusCode)
	}
	fmt.Println("ok")
	return nil
}

func checkClaude(ctx context.Context) error {
	fmt.Print("  Checking claude CLI... ")
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}
	fmt.Println("ok")
	return nil
}

func probeScenarios(ctx context.Context, cfg *Config) error {
	probes := []struct {
		name string
		fn   func(context.Context, *Config) error
	}{
		{"baseline", probeBaseline},
		{"with-lumen", probeWithLumen},
		{"with-lumen hook", probeHookFiringWithLumen},
	}

	for _, p := range probes {
		if err := p.fn(ctx, cfg); err != nil {
			return fmt.Errorf("%s probe: %w", p.name, err)
		}
		fmt.Printf("  Probing %s scenario... ok\n", p.name)
	}
	return nil
}

func probeBaseline(ctx context.Context, cfg *Config) error {
	mcpPath, cleanup, err := runner.WriteMCPConfig(runner.Baseline, cfg.LumenBinary, cfg.Backend, cfg.EmbedModel)
	if err != nil {
		return err
	}
	defer cleanup()

	// Ask Claude to actually invoke the tool and check for a tool_use event in
	// the stream-json output. A plain text-listing prompt is unreliable: the
	// SessionStart hook injects "Call mcp__lumen__semantic_search" into the
	// system context, so Claude mentions the name in text even when the MCP
	// server is absent from the config. An actual tool_use call only appears in
	// stream-json when Claude can genuinely invoke the tool.
	const prompt = "Call mcp__lumen__semantic_search with query 'test'."
	extraArgs := []string{"--output-format", "stream-json"}
	out, err := runClaudeProbe(ctx, mcpPath, extraArgs, prompt)
	if err != nil {
		return err
	}

	if hasToolUseEvent(out, "mcp__lumen__semantic_search") {
		return fmt.Errorf("baseline should NOT have semantic_search, but tool was successfully invoked")
	}
	return nil
}

// hasToolUseEvent returns true when the stream-json output contains a tool_use
// event for the named tool. Each line in stream-json is a complete JSON event;
// a tool_use call appears as an assistant message event containing both
// "tool_use" and the tool name on the same line.
func hasToolUseEvent(streamJSON, toolName string) bool {
	for _, line := range strings.Split(streamJSON, "\n") {
		if strings.Contains(line, `"tool_use"`) && strings.Contains(line, `"`+toolName+`"`) {
			return true
		}
	}
	return false
}

func probeWithLumen(ctx context.Context, cfg *Config) error {
	mcpPath, cleanup, err := runner.WriteMCPConfig(runner.WithLumen, cfg.LumenBinary, cfg.Backend, cfg.EmbedModel)
	if err != nil {
		return err
	}
	defer cleanup()

	const prompt = "List all tools available to you, including MCP tools. If you have no MCP tools, say NONE."
	out, err := runClaudeProbe(ctx, mcpPath, runner.ClaudeArgs(runner.WithLumen, cfg.RepoRoot), prompt)
	if err != nil {
		return err
	}

	if !strings.Contains(strings.ToLower(out), "semantic_search") {
		return fmt.Errorf("with-lumen should have semantic_search, but probe output does not mention it")
	}
	return nil
}

// probeHookFiringWithLumen verifies that the PreToolUse hook intercepts a
// natural-language Grep call and redirects Claude to mcp__lumen__semantic_search
// in with-lumen (where Grep/Glob are available alongside lumen).
func probeHookFiringWithLumen(ctx context.Context, cfg *Config) error {
	mcpPath, cleanup, err := runner.WriteMCPConfig(runner.WithLumen, cfg.LumenBinary, cfg.Backend, cfg.EmbedModel)
	if err != nil {
		return err
	}
	defer cleanup()

	extraArgs := append(runner.ClaudeArgs(runner.WithLumen, cfg.RepoRoot),
		"--output-format", "stream-json",
	)
	// Pattern is >40 chars, spaces, no regex metacharacters — matches
	// looksLikeNaturalLanguage so the PreToolUse hook returns "suggest"
	// naming mcp__lumen__semantic_search. Claude should call it instead.
	const prompt = "Use the Grep tool with pattern " +
		"'find all places where database connection errors are handled with retry and exponential backoff logic'. " +
		"Report what you find."
	out, err := runClaudeProbe(ctx, mcpPath, extraArgs, prompt)
	if err != nil {
		return err
	}

	if !strings.Contains(out, "mcp__lumen__semantic_search") {
		return fmt.Errorf("with-lumen hook probe: PreToolUse hook did not redirect to mcp__lumen__semantic_search")
	}
	return nil
}

func runClaudeProbe(ctx context.Context, mcpConfigPath string, extraArgs []string, prompt string) (string, error) {
	args := []string{
		"--print",
		"--model", "haiku",
		"--effort", "low",
		"--strict-mcp-config",
		"--mcp-config", mcpConfigPath,
	}
	args = append(args, extraArgs...)
	args = append(args, "--", prompt)

	ctx, cancel := context.WithTimeout(ctx, 900*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Env = cleanEnvForClaude()
	// Run probes from a neutral temp dir so Claude does not auto-discover
	// the .claude-plugin/ in the repo root, which would inject lumen's
	// semantic_search into every scenario (including baseline).
	// with-lumen probes are unaffected because they pass --plugin-dir explicitly.
	cmd.Dir = os.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude probe failed: %w\n%s", err, out)
	}
	return string(out), nil
}

// cleanEnvForClaude returns os.Environ() without CLAUDECODE so that
// claude can be spawned from inside a Claude Code session.
func cleanEnvForClaude() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			env = append(env, e)
		}
	}
	return env
}
