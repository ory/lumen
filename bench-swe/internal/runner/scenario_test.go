package runner

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAllScenarios(t *testing.T) {
	scenarios := AllScenarios()
	if len(scenarios) != 2 {
		t.Fatalf("got %d scenarios, want 2", len(scenarios))
	}
	if scenarios[0] != Baseline {
		t.Errorf("scenarios[0] = %q, want %q", scenarios[0], Baseline)
	}
	if scenarios[1] != WithLumen {
		t.Errorf("scenarios[1] = %q, want %q", scenarios[1], WithLumen)
	}
}

func TestParseScenarios(t *testing.T) {
	tests := []struct {
		input   string
		want    []Scenario
		wantErr bool
	}{
		{input: "", want: []Scenario{Baseline, WithLumen}},
		{input: "all", want: []Scenario{Baseline, WithLumen}},
		{input: "baseline", want: []Scenario{Baseline}},
		{input: "with-lumen", want: []Scenario{WithLumen}},
		{input: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run("input="+tt.input, func(t *testing.T) {
			got, err := ParseScenarios(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d scenarios, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("scenarios[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWriteMCPConfig(t *testing.T) {
	tests := []struct {
		name      string
		scenario  Scenario
		wantLumen bool
	}{
		{name: "baseline has empty mcpServers", scenario: Baseline, wantLumen: false},
		{name: "with-lumen has lumen server", scenario: WithLumen, wantLumen: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup, err := WriteMCPConfig(tt.scenario, "/usr/local/bin/lumen", "ollama", "test-model")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer cleanup()

			// Read and parse the config file
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading config: %v", err)
			}

			var cfg mcpConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				t.Fatalf("parsing config: %v", err)
			}

			_, hasLumen := cfg.MCPServers["lumen"]
			if hasLumen != tt.wantLumen {
				t.Errorf("has lumen server = %v, want %v", hasLumen, tt.wantLumen)
			}

			if tt.wantLumen {
				srv := cfg.MCPServers["lumen"]
				if srv.Command != "/usr/local/bin/lumen" {
					t.Errorf("Command = %q, want /usr/local/bin/lumen", srv.Command)
				}
				if len(srv.Args) != 1 || srv.Args[0] != "stdio" {
					t.Errorf("Args = %v, want [stdio]", srv.Args)
				}
				if srv.Env["LUMEN_BACKEND"] != "ollama" {
					t.Errorf("LUMEN_BACKEND = %q, want ollama", srv.Env["LUMEN_BACKEND"])
				}
				if srv.Env["LUMEN_EMBED_MODEL"] != "test-model" {
					t.Errorf("LUMEN_EMBED_MODEL = %q, want test-model", srv.Env["LUMEN_EMBED_MODEL"])
				}
			}

			// Call cleanup and verify file is removed
			cleanup()
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Error("cleanup did not remove file")
			}
		})
	}
}

func TestWriteMCPConfig_UnknownScenario(t *testing.T) {
	_, _, err := WriteMCPConfig(Scenario("unknown"), "/bin/lumen", "ollama", "model")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown scenario") {
		t.Errorf("error %q does not contain %q", err.Error(), "unknown scenario")
	}
}

func TestClaudeArgs(t *testing.T) {
	repoRoot := "/tmp/test-repo"

	tests := []struct {
		scenario Scenario
		wantNil  bool
		wantArgs []string
	}{
		{
			scenario: Baseline,
			wantNil:  true,
		},
		{
			scenario: WithLumen,
			wantArgs: []string{"--plugin-dir", repoRoot},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.scenario), func(t *testing.T) {
			got := ClaudeArgs(tt.scenario, repoRoot)
			if tt.wantNil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("got %d args, want %d", len(got), len(tt.wantArgs))
			}
			for i := range got {
				if got[i] != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, got[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestStripPII(t *testing.T) {
	t.Run("HOME path replaced", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/raw.jsonl"
		t.Setenv("HOME", "/Users/testuser")
		t.Setenv("USER", "")
		if err := os.WriteFile(path, []byte("file at /Users/testuser/code/main.go"), 0o644); err != nil {
			t.Fatal(err)
		}
		stripPII(path)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := string(data)
		if strings.Contains(got, "/Users/testuser") {
			t.Errorf("HOME path not replaced: %s", got)
		}
		if !strings.Contains(got, "~/code/main.go") {
			t.Errorf("expected ~ replacement, got: %s", got)
		}
	})

	t.Run("bare USER not replaced without HOME", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/raw.jsonl"
		t.Setenv("HOME", "")
		t.Setenv("USER", "johndoe")
		if err := os.WriteFile(path, []byte("owner: johndoe, path: /home/johndoe"), 0o644); err != nil {
			t.Fatal(err)
		}
		stripPII(path)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := string(data)
		// Bare USER should NOT be replaced — only HOME path replacement covers usernames
		if !strings.Contains(got, "johndoe") {
			t.Errorf("bare USER should not be replaced, got: %s", got)
		}
	})

	t.Run("HOME replaced but bare USER preserved", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/raw.jsonl"
		t.Setenv("HOME", "/home/alice")
		t.Setenv("USER", "alice")
		if err := os.WriteFile(path, []byte("path=/home/alice/repo user=alice"), 0o644); err != nil {
			t.Fatal(err)
		}
		stripPII(path)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got := string(data)
		if strings.Contains(got, "/home/alice") {
			t.Errorf("HOME not replaced: %s", got)
		}
		// After HOME replacement: "path=~/repo user=alice"
		// Bare "alice" is NOT replaced — only the path portion is stripped
		if !strings.Contains(got, "~") {
			t.Errorf("expected ~ in output, got: %s", got)
		}
		if got != "path=~/repo user=alice" {
			t.Errorf("expected bare user preserved, got: %s", got)
		}
	})
}
