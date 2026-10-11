package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandIsolationAndTransport(t *testing.T) {
	bin := t.TempDir()
	for _, name := range []string{"claude", "codex"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, provider := range []string{"claude", "codex"} {
		for _, transport := range []string{"cli", "sdk"} {
			for _, withLumen := range []bool{false, true} {
				t.Run(provider+"/"+transport+"/"+map[bool]string{true: "lumen", false: "baseline"}[withLumen], func(t *testing.T) {
					servers := map[string]Server{}
					if withLumen {
						servers["plugin_lumen_lumen"] = Server{Command: "/tmp/lumen path", Args: []string{"stdio"}, Env: map[string]string{"LUMEN_RETRIEVAL": "bm25"}}
					}
					cfg := Config{Provider: provider, Transport: transport, Auth: "environment", RepoRoot: t.TempDir(), Model: "test-model"}
					cmd, cleanup, err := Command(context.Background(), Request{Config: cfg, CWD: t.TempDir(), Prompt: "fix", Servers: servers})
					if err != nil {
						t.Fatal(err)
					}
					defer cleanup()
					args := strings.Join(cmd.Args, " ")
					if strings.Contains(args, "dangerously") {
						t.Fatalf("unexpected permission bypass: %s", args)
					}
					if transport == "cli" && provider == "codex" {
						for _, want := range []string{"exec --json", "--ignore-user-config", "--sandbox workspace-write"} {
							if !strings.Contains(args, want) {
								t.Errorf("missing %s: %s", want, args)
							}
						}
						if !withLumen && !strings.Contains(args, "mcp_servers={}") {
							t.Fatal("baseline not empty")
						}
						if withLumen && !strings.Contains(args, "enabled_tools=") {
							t.Fatal("missing MCP tool allowlist")
						}
					}
					if transport == "cli" && provider == "claude" {
						if !strings.Contains(args, "--strict-mcp-config") || !strings.Contains(args, "--setting-sources ") {
							t.Fatal("unisolated Claude")
						}
					}
					if transport == "sdk" && filepath.Base(cmd.Args[0]) != "node" {
						t.Fatal("SDK did not use node bridge")
					}
				})
			}
		}
	}
}

func TestSubscriptionEnvironment(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sentinel")
	t.Setenv("OPENAI_API_KEY", "sentinel")
	t.Setenv("CODEX_API_KEY", "sentinel")
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CODEX_HOME", "/tmp/existing-login")
	env := strings.Join(Environment("subscription"), "\n")
	if strings.Contains(env, "sentinel") || strings.Contains(env, "CLAUDECODE=") {
		t.Fatal("API credentials or nested-session marker leaked")
	}
	if !strings.Contains(env, "CODEX_HOME=/tmp/existing-login") {
		t.Fatal("lost existing login location")
	}
	if !strings.Contains(strings.Join(Environment("environment"), "\n"), "ANTHROPIC_API_KEY=sentinel") {
		t.Fatal("explicit environment auth ignored")
	}
}

func TestSubscriptionLoginGuard(t *testing.T) {
	for _, tc := range []struct {
		provider, status string
		valid            bool
	}{
		{"claude", `{"loggedIn":true,"authMethod":"claude.ai"}`, true},
		{"claude", `{"loggedIn":true,"authMethod":"api_key"}`, false},
		{"claude", `{"loggedIn":false}`, false},
		{"codex", "Logged in using ChatGPT", true},
		{"codex", "Logged in using an API key", false},
	} {
		bin := filepath.Join(t.TempDir(), tc.provider)
		if err := os.WriteFile(bin, []byte("#!/bin/sh\ncat <<'STATUS'\n"+tc.status+"\nSTATUS\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := checkSubscription(context.Background(), tc.provider, bin); (err == nil) != tc.valid {
			t.Fatalf("%s status %s: %v", tc.provider, tc.status, err)
		}
	}
}

func TestReadOnlyJudgeCommand(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd, cleanup, err := Command(context.Background(), Request{Config: Config{Provider: "codex", Transport: "cli", Auth: "environment"}, CWD: t.TempDir(), ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--skip-git-repo-check") || !strings.Contains(args, "--sandbox read-only") {
		t.Fatalf("judge requires no repo and read-only sandbox: %s", args)
	}
}
