package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/agent"
	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
)

func (cfg *Config) AgentConfig() agent.Config {
	c := agent.Config{Provider: cfg.Agent, Transport: cfg.Transport, Auth: cfg.Auth, Model: cfg.Model, RepoRoot: cfg.RepoRoot}
	if c.Provider == "" {
		c.Provider = "claude"
	}
	if c.Transport == "" {
		c.Transport = "cli"
	}
	if c.Auth == "" {
		c.Auth = "subscription"
	}
	if c.Provider == "claude" && c.Model == "" {
		c.Model = cfg.ClaudeModel
	}
	return c
}

// ExecuteAgent is shared by real benchmark runs and live CLI probes. Logs and
// failure metrics survive timeouts and client errors, even without a final event.
func ExecuteAgent(ctx context.Context, cfg *Config, repoDir string, s Scenario, prompt, slug string, timeout time.Duration, taskEnv []string) (*metrics.Metrics, error) {
	c := cfg.AgentConfig()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	mcp, err := scenarioConfig(s, cfg.LumenBinary, cfg.Retrieval, cfg.Backend, cfg.EmbedModel)
	if err != nil {
		return nil, err
	}
	req := agent.Request{Config: c, CWD: repoDir, Prompt: prompt, Servers: mcp.MCPServers}
	if s == WithLumen && c.Provider == "claude" {
		req.PluginDir = cfg.RepoRoot
	}
	if s == WithLumen && c.Provider == "codex" {
		// Codex has no Claude SessionStart hook; supply the same search guidance.
		req.Prompt = "Use the Lumen semantic_search MCP tool to locate relevant code before editing. It supports natural-language questions and exact symbols.\n\n" + prompt
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd, cleanup, err := agent.Command(runCtx, req)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	cmd.Env = append(cmd.Env, lumenEnv(cfg)...)
	cmd.Env = append(cmd.Env, taskEnv...)
	rawPath := filepath.Join(cfg.ResultsDir, slug+"-raw.jsonl")
	raw, err := os.Create(rawPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = raw.Close() }()
	stderrPath := filepath.Join(cfg.ResultsDir, slug+"-stderr.log")
	stderr, err := os.Create(stderrPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stderr.Close() }()
	cmd.Stdout, cmd.Stderr = raw, stderr
	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start).Milliseconds()
	_ = raw.Close()
	m, extractErr := metrics.ExtractFromJSONL(rawPath)
	if m == nil {
		m = &metrics.Metrics{CostUnknown: true}
	}
	m.Agent, m.Runner, m.Model = c.Provider, c.Transport, c.Model
	if m.DurationMS == 0 {
		m.DurationMS = duration
	}
	err = errors.Join(runCtx.Err(), runErr, extractErr)
	if err != nil {
		m.Error = err.Error()
	}
	saveErr := m.SaveToFile(filepath.Join(cfg.ResultsDir, slug+"-metrics.json"))
	stripPII(rawPath)
	stripPII(stderrPath)
	if err != nil {
		return m, fmt.Errorf("%s/%s failed (see %s): %w", c.Provider, c.Transport, stderrPath, errors.Join(err, saveErr))
	}
	return m, saveErr
}
