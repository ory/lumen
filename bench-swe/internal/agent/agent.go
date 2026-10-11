// Package agent runs the real coding agents through their CLI or official SDK.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Provider  string `json:"provider"`
	Transport string `json:"transport"`
	Model     string `json:"model"`
	Auth      string `json:"auth"`
	RepoRoot  string `json:"-"`
}

type Server struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type Request struct {
	Config
	CWD            string            `json:"cwd"`
	Prompt         string            `json:"prompt"`
	Servers        map[string]Server `json:"servers"`
	PluginDir      string            `json:"pluginDir,omitempty"`
	Executable     string            `json:"executable"`
	CodexOverrides []string          `json:"codexOverrides,omitempty"`
	ReadOnly       bool              `json:"readOnly,omitempty"`
}

func (c Config) Validate() error {
	if c.Provider != "claude" && c.Provider != "codex" {
		return fmt.Errorf("unknown agent %q (valid: claude, codex)", c.Provider)
	}
	if c.Transport != "cli" && c.Transport != "sdk" {
		return fmt.Errorf("unknown runner %q (valid: cli, sdk)", c.Transport)
	}
	if c.Auth != "subscription" && c.Auth != "environment" {
		return fmt.Errorf("unknown auth %q (valid: subscription, environment)", c.Auth)
	}
	return nil
}

// Environment preserves the clients' existing login stores. No credentials are
// copied. Subscription mode removes API/provider overrides to avoid API billing.
func Environment(auth string) []string {
	var env []string
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		if key == "CLAUDECODE" {
			continue
		}
		if auth == "subscription" {
			switch key {
			case "ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_BASE_URL", "OPENAI_API_KEY", "OPENAI_BASE_URL", "CODEX_API_KEY", "CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY", "CLAUDE_CODE_USE_ANTHROPIC_AWS":
				continue
			}
		}
		env = append(env, e)
	}
	return append(env, "CLAUDE_CODE_DISABLE_AUTO_MEMORY=1")
}

// Command prepares a disposable configuration; cleanup must run after the child
// exits. Both transports use the installed CLI and the same explicit MCP setup.
func Command(ctx context.Context, req Request) (*exec.Cmd, func(), error) {
	if err := req.Config.Validate(); err != nil {
		return nil, nil, err
	}
	binary, err := exec.LookPath(req.Provider)
	if err != nil {
		return nil, nil, err
	}
	if req.Auth == "subscription" {
		if err := checkSubscription(ctx, req.Provider, binary); err != nil {
			return nil, nil, err
		}
	}
	req.Executable, err = filepath.Abs(binary)
	if err != nil {
		return nil, nil, err
	}
	dir, err := os.MkdirTemp("", "bench-agent-*")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	fail := func(err error) (*exec.Cmd, func(), error) { cleanup(); return nil, nil, err }
	if req.Servers == nil {
		req.Servers = map[string]Server{}
	}
	req.CodexOverrides = []string{
		"mcp_servers=" + codexServers(req.Servers), "plugins={}",
		`approval_policy="never"`, `model_reasoning_effort="high"`,
		`web_search="disabled"`, "sandbox_workspace_write.network_access=true",
	}
	if req.ReadOnly {
		req.CodexOverrides = append(req.CodexOverrides, "sandbox_workspace_write.network_access=false")
	}
	if req.Auth == "subscription" {
		req.CodexOverrides = append(req.CodexOverrides, `forced_login_method="chatgpt"`)
	}
	var args []string
	var stdin string
	if req.Transport == "sdk" {
		// The TS SDK has no ignore-user-config option. Its executable override keeps
		// auth in the real CODEX_HOME while excluding unrelated user MCP/plugins.
		if req.Provider == "codex" {
			wrapper := filepath.Join(dir, "codex")
			script := "#!/bin/sh\nsubcommand=\"$1\"\nshift\nexec " + shellQuote(req.Executable) + " \"$subcommand\" --ignore-user-config --ignore-rules --ephemeral \"$@\"\n"
			if err := os.WriteFile(wrapper, []byte(script), 0o700); err != nil {
				return fail(err)
			}
			req.Executable = wrapper
		}
		data, err := json.Marshal(req)
		if err != nil {
			return fail(err)
		}
		binary = "node"
		args = []string{filepath.Join(req.RepoRoot, "bench-swe", "sdk", "run.mjs")}
		stdin = string(data)
	} else if req.Provider == "claude" {
		data, err := json.Marshal(map[string]any{"mcpServers": req.Servers})
		if err != nil {
			return fail(err)
		}
		configPath := filepath.Join(dir, "mcp.json")
		if err := os.WriteFile(configPath, data, 0o600); err != nil {
			return fail(err)
		}
		args = []string{"--print", "--output-format", "stream-json", "--verbose", "--effort", "high",
			"--strict-mcp-config", "--mcp-config", configPath, "--setting-sources", "", "--no-session-persistence",
			"--permission-mode", "acceptEdits", "--allowedTools", "Bash,Read,Edit,Write,Glob,Grep,mcp__plugin_lumen_lumen__*"}
		if req.ReadOnly {
			args = append(args, "--tools", "")
		}
		if req.Model != "" {
			args = append(args, "--model", req.Model)
		}
		if req.PluginDir != "" {
			args = append(args, "--plugin-dir", req.PluginDir)
		}
		stdin = req.Prompt
	} else {
		sandbox := "workspace-write"
		if req.ReadOnly {
			sandbox = "read-only"
		}
		args = []string{"exec", "--json", "--ephemeral", "--ignore-user-config", "--ignore-rules", "--sandbox", sandbox, "--cd", req.CWD}
		if req.ReadOnly {
			args = append(args, "--skip-git-repo-check")
		}
		for _, override := range req.CodexOverrides {
			args = append(args, "-c", override)
		}
		if req.Model != "" {
			args = append(args, "--model", req.Model)
		}
		args = append(args, "-")
		stdin = req.Prompt
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	manageProcess(cmd)
	cmd.Dir = req.CWD
	cmd.Env = Environment(req.Auth)
	cmd.Stdin = strings.NewReader(stdin)
	return cmd, cleanup, nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// JSON-quoted strings are also TOML basic strings for CLI paths and env values.
func codexServers(servers map[string]Server) string {
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	var entries []string
	for _, name := range names {
		s := servers[name]
		args := make([]string, 0, len(s.Args))
		for _, a := range s.Args {
			args = append(args, strconv.Quote(a))
		}
		keys := make([]string, 0, len(s.Env))
		for k := range s.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var env []string
		for _, k := range keys {
			env = append(env, strconv.Quote(k)+"="+strconv.Quote(s.Env[k]))
		}
		entries = append(entries, strconv.Quote(name)+"={command="+strconv.Quote(s.Command)+",args=["+strings.Join(args, ",")+"],env={"+strings.Join(env, ",")+"},required=true,enabled_tools=[\"semantic_search\",\"index_status\"]}")
	}
	return "{" + strings.Join(entries, ",") + "}"
}

// Fail before launching a paid turn if the stored login is API-backed. Never
// print the raw status response: it may contain account identifiers.
func checkSubscription(ctx context.Context, provider, binary string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	args := []string{"login", "status"}
	if provider == "claude" {
		args = []string{"auth", "status", "--json"}
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = Environment("subscription")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s subscription login unavailable; sign in with %s first", provider, provider)
	}
	if provider == "codex" {
		if !strings.Contains(string(out), "Logged in using ChatGPT") {
			return fmt.Errorf("codex is not signed in with ChatGPT; use codex login or explicitly select --auth environment")
		}
	} else {
		var status struct {
			LoggedIn   bool   `json:"loggedIn"`
			AuthMethod string `json:"authMethod"`
		}
		if json.Unmarshal(out, &status) != nil || !status.LoggedIn || status.AuthMethod != "claude.ai" {
			return fmt.Errorf("claude is not signed in with a Claude plan; use claude auth login or explicitly select --auth environment")
		}
	}
	return nil
}
