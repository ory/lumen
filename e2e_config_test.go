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

//go:build e2e

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startServerWithEnv launches the MCP server subprocess with a custom
// environment. The caller controls all env vars (XDG_CONFIG_HOME,
// XDG_DATA_HOME, OLLAMA_HOST, etc.).
func startServerWithEnv(t *testing.T, env []string) *mcp.ClientSession {
	t.Helper()

	cmd := exec.Command(serverBinary, "stdio")
	cmd.Env = env

	transport := &mcp.CommandTransport{Command: cmd}
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "e2e-config-test-client",
		Version: "0.1.0",
	}, nil)

	ctx := context.Background()
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// ollamaHostForTest returns the Ollama host from the environment or the default.
func ollamaHostForTest() string {
	if h := os.Getenv("OLLAMA_HOST"); h != "" {
		return h
	}
	return "http://localhost:11434"
}

// baseEnv returns the minimal env vars needed by the subprocess (HOME, PATH).
func baseEnv(dataHome string) []string {
	return []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"XDG_DATA_HOME=" + dataHome,
		"LUMEN_RETRIEVAL=semantic",
	}
}

// writeConfigYAML writes a config.yaml into <configHome>/lumen/config.yaml and
// returns the configHome path.
func writeConfigYAML(t *testing.T, content string) string {
	t.Helper()
	configHome := t.TempDir()
	dir := filepath.Join(configHome, "lumen")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return configHome
}

// newDownServer creates an httptest server that returns 503 on health checks
// and 500 on embed requests (simulating an unhealthy backend).
func newDownServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// --- Happy Path Tests ---

func TestE2E_Config_YAMLDrivesServerSelection(t *testing.T) {
	requireSemantic(t)
	configHome := writeConfigYAML(t, fmt.Sprintf(`
servers:
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
max_chunk_tokens: 100
freshness_ttl: 1s
`, ollamaHostForTest()))

	dataHome := t.TempDir()
	env := append(baseEnv(dataHome),
		"XDG_CONFIG_HOME="+configHome,
	)

	session := startServerWithEnv(t, env)

	out := callSearch(t, session, map[string]any{
		"query": "authentication",
		"path":  sampleProjectPath(t),
	})

	if len(out.Results) == 0 {
		t.Fatal("expected search results from config.yaml-driven server, got none")
	}
	if !out.Reindexed {
		t.Error("expected reindexing on first search")
	}
}

func TestE2E_Config_MultiServerFailover(t *testing.T) {
	requireSemantic(t)
	down := newDownServer(t)

	configHome := writeConfigYAML(t, fmt.Sprintf(`
servers:
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
max_chunk_tokens: 100
freshness_ttl: 1s
`, down.URL, ollamaHostForTest()))

	dataHome := t.TempDir()
	env := append(baseEnv(dataHome),
		"XDG_CONFIG_HOME="+configHome,
	)

	session := startServerWithEnv(t, env)

	out := callSearch(t, session, map[string]any{
		"query": "database connection",
		"path":  sampleProjectPath(t),
	})

	if len(out.Results) == 0 {
		t.Fatal("expected search results after failover to healthy server, got none")
	}
}

func TestE2E_Config_HotReloadServerFailover(t *testing.T) {
	requireSemantic(t)
	// Start with only an unhealthy server — search should fail.
	down := newDownServer(t)

	configHome := writeConfigYAML(t, fmt.Sprintf(`
servers:
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
max_chunk_tokens: 100
freshness_ttl: 1s
`, down.URL))

	dataHome := t.TempDir()
	env := append(baseEnv(dataHome),
		"XDG_CONFIG_HOME="+configHome,
	)

	session := startServerWithEnv(t, env)

	// Search should fail — only server is unhealthy
	result1 := callSearchRaw(t, session, map[string]any{
		"query": "authentication",
		"path":  sampleProjectPath(t),
	})
	if !result1.IsError {
		t.Fatal("expected error when only server is unhealthy")
	}

	// Hot reload: add a healthy server
	cfgFile := filepath.Join(configHome, "lumen", "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(fmt.Sprintf(`
servers:
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
max_chunk_tokens: 100
freshness_ttl: 1s
`, down.URL, ollamaHostForTest())), 0644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	// Wait for fsnotify to pick up the change
	time.Sleep(1 * time.Second)

	// Search should now succeed after failover to the newly added healthy server
	out := callSearch(t, session, map[string]any{
		"query": "authentication",
		"path":  sampleProjectPath(t),
	})
	if len(out.Results) == 0 {
		t.Fatal("expected search results after hot-reloading a healthy server, got none")
	}
}

// --- Edge Case Tests ---

func TestE2E_Config_EnvVarsOverrideYAML(t *testing.T) {
	requireSemantic(t)
	// config.yaml references a model that doesn't exist
	configHome := writeConfigYAML(t, fmt.Sprintf(`
servers:
  - backend: ollama
    host: %s
    model: nonexistent-model-name
    dims: 384
max_chunk_tokens: 100
freshness_ttl: 1s
`, ollamaHostForTest()))

	dataHome := t.TempDir()
	env := append(baseEnv(dataHome),
		"XDG_CONFIG_HOME="+configHome,
		// Env var overrides the bad model
		"LUMEN_EMBED_MODEL=all-minilm",
	)

	session := startServerWithEnv(t, env)

	out := callSearch(t, session, map[string]any{
		"query": "authentication",
		"path":  sampleProjectPath(t),
	})

	if len(out.Results) == 0 {
		t.Fatal("expected search results with env var overriding config.yaml model, got none")
	}
}

func TestE2E_Config_InvalidReloadPreservesPrevious(t *testing.T) {
	requireSemantic(t)
	configHome := writeConfigYAML(t, fmt.Sprintf(`
servers:
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
max_chunk_tokens: 100
freshness_ttl: 1s
`, ollamaHostForTest()))

	dataHome := t.TempDir()
	env := append(baseEnv(dataHome),
		"XDG_CONFIG_HOME="+configHome,
	)

	session := startServerWithEnv(t, env)

	// First search succeeds
	out1 := callSearch(t, session, map[string]any{
		"query": "authentication",
		"path":  sampleProjectPath(t),
	})
	if len(out1.Results) == 0 {
		t.Fatal("expected search results before invalid reload")
	}

	// Reload to invalid config (empty servers list)
	cfgFile := filepath.Join(configHome, "lumen", "config.yaml")
	if err := os.WriteFile(cfgFile, []byte("servers: []\n"), 0644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}
	time.Sleep(1 * time.Second)

	// Search should still work with the previous valid config
	out2 := callSearch(t, session, map[string]any{
		"query": "authentication",
		"path":  sampleProjectPath(t),
	})
	if len(out2.Results) == 0 {
		t.Fatal("expected search results after invalid config reload (should retain previous)")
	}
}

func TestE2E_Config_NoConfigFileEnvVarsOnly(t *testing.T) {
	requireSemantic(t)
	// Don't create a config.yaml — set XDG_CONFIG_HOME to an empty temp dir
	configHome := t.TempDir()
	dataHome := t.TempDir()
	env := append(baseEnv(dataHome),
		"XDG_CONFIG_HOME="+configHome,
		"OLLAMA_HOST="+ollamaHostForTest(),
		"LUMEN_EMBED_MODEL=all-minilm",
		"LUMEN_MAX_CHUNK_TOKENS=100",
		"LUMEN_FRESHNESS_TTL=1s",
	)

	session := startServerWithEnv(t, env)

	out := callSearch(t, session, map[string]any{
		"query": "authentication",
		"path":  sampleProjectPath(t),
	})

	if len(out.Results) == 0 {
		t.Fatal("expected search results with env-vars-only config (no config.yaml)")
	}
}

// --- Unhappy Path Tests ---

func TestE2E_Config_AllServersUnhealthy(t *testing.T) {
	down1 := newDownServer(t)
	down2 := newDownServer(t)

	configHome := writeConfigYAML(t, fmt.Sprintf(`
servers:
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
  - backend: ollama
    host: %s
    model: all-minilm
    dims: 384
max_chunk_tokens: 100
freshness_ttl: 1s
`, down1.URL, down2.URL))

	dataHome := t.TempDir()
	env := append(baseEnv(dataHome),
		"XDG_CONFIG_HOME="+configHome,
	)

	session := startServerWithEnv(t, env)

	result := callSearchRaw(t, session, map[string]any{
		"query": "authentication",
		"path":  sampleProjectPath(t),
	})

	if !result.IsError {
		t.Fatal("expected error when all servers are unhealthy")
	}

	// Verify we get an error message, not a crash
	text := getTextContent(t, result)
	if text == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestE2E_Config_UnknownBackendRejectsStartup(t *testing.T) {
	configHome := writeConfigYAML(t, `
servers:
  - backend: foobar
    host: http://localhost:9999
    model: test-model
    dims: 384
`)

	dataHome := t.TempDir()
	env := append(baseEnv(dataHome),
		"XDG_CONFIG_HOME="+configHome,
	)

	cmd := exec.Command(serverBinary, "stdio")
	cmd.Env = env

	transport := &mcp.CommandTransport{Command: cmd}
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "e2e-config-test-client",
		Version: "0.1.0",
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Connect(ctx, transport, nil)
	if err == nil {
		t.Fatal("expected server to reject startup with unknown backend, but connection succeeded")
	}
}
