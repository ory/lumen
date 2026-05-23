package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfigDocFixture(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return path
}

func clearConfigDocEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"LUMEN_BACKEND",
		"LUMEN_EMBED_MODEL",
		"LUMEN_EMBED_DIMS",
		"LUMEN_EMBED_CTX",
		"OLLAMA_HOST",
		"LM_STUDIO_HOST",
		"LUMEN_MAX_CHUNK_TOKENS",
		"LUMEN_FRESHNESS_TTL",
		"LUMEN_REINDEX_TIMEOUT",
		"LUMEN_LOG_LEVEL",
	} {
		t.Setenv(key, "")
	}
}

func TestDocumentationMinimalOllamaConfig(t *testing.T) {
	clearConfigDocEnv(t)
	path := writeConfigDocFixture(t, `
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
`)

	svc, err := NewConfigService(path)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}

	servers := svc.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Backend != BackendOllama {
		t.Fatalf("Backend = %q, want %q", servers[0].Backend, BackendOllama)
	}
	if got := svc.ServerDims(0); got != 768 {
		t.Fatalf("ServerDims(0) = %d, want 768", got)
	}
	if got := svc.ServerCtxLength(0); got != 8192 {
		t.Fatalf("ServerCtxLength(0) = %d, want 8192", got)
	}
	if got := svc.ServerMinScore(0); got != 0.35 {
		t.Fatalf("ServerMinScore(0) = %f, want 0.35", got)
	}
}

func TestDocumentationMinimalLMStudioConfig(t *testing.T) {
	clearConfigDocEnv(t)
	path := writeConfigDocFixture(t, `
servers:
  - backend: lmstudio
    host: http://localhost:1234
    model: nomic-ai/nomic-embed-code-GGUF
`)

	svc, err := NewConfigService(path)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}

	servers := svc.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Backend != BackendLMStudio {
		t.Fatalf("Backend = %q, want %q", servers[0].Backend, BackendLMStudio)
	}
	if got := svc.ServerDims(0); got != 3584 {
		t.Fatalf("ServerDims(0) = %d, want 3584", got)
	}
	if got := svc.ServerCtxLength(0); got != 8192 {
		t.Fatalf("ServerCtxLength(0) = %d, want 8192", got)
	}
	if got := svc.ServerMinScore(0); got != 0.15 {
		t.Fatalf("ServerMinScore(0) = %f, want 0.15", got)
	}
}

func TestDocumentationFullConfigExample(t *testing.T) {
	clearConfigDocEnv(t)
	path := writeConfigDocFixture(t, `
log_level: info
max_chunk_tokens: 512
freshness_ttl: 60s
reindex_timeout: 0s

servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
    dims: 768
    ctx_length: 8192
    min_score: 0.35
`)

	svc, err := NewConfigService(path)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}

	if got := svc.LogLevel(); got != "info" {
		t.Fatalf("LogLevel() = %q, want info", got)
	}
	if got := svc.MaxChunkTokens(); got != 512 {
		t.Fatalf("MaxChunkTokens() = %d, want 512", got)
	}
	if got := svc.FreshnessTTL(); got != 60*time.Second {
		t.Fatalf("FreshnessTTL() = %v, want 60s", got)
	}
	if got := svc.ReindexTimeout(); got != 0 {
		t.Fatalf("ReindexTimeout() = %v, want 0", got)
	}
}

func TestDocumentationMultiServerFailoverExample(t *testing.T) {
	clearConfigDocEnv(t)
	path := writeConfigDocFixture(t, `
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
  - backend: ollama
    host: http://backup-ollama.example:11434
    model: ordis/jina-embeddings-v2-base-code
  - backend: lmstudio
    host: http://localhost:1234
    model: nomic-ai/nomic-embed-code-GGUF
`)

	svc, err := NewConfigService(path, WithServerSelection("ordis/jina-embeddings-v2-base-code", "ollama"))
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}

	servers := svc.Servers()
	if len(servers) != 2 {
		t.Fatalf("Servers() len = %d, want 2", len(servers))
	}
	if got := servers[0].Host; got != "http://localhost:11434" {
		t.Fatalf("Servers()[0].Host = %q, want http://localhost:11434", got)
	}
	if got := servers[1].Host; got != "http://backup-ollama.example:11434" {
		t.Fatalf("Servers()[1].Host = %q, want http://backup-ollama.example:11434", got)
	}
}

func TestDocumentationCustomModelRequiresDims(t *testing.T) {
	clearConfigDocEnv(t)
	path := writeConfigDocFixture(t, `
servers:
  - backend: ollama
    host: http://localhost:11434
    model: my-custom-embedding-model
    dims: 1024
    ctx_length: 8192
    min_score: 0.20
`)

	svc, err := NewConfigService(path)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if got := svc.ServerDims(0); got != 1024 {
		t.Fatalf("ServerDims(0) = %d, want 1024", got)
	}
}

func TestDocumentationEnvOverridesYAMLFirstServer(t *testing.T) {
	clearConfigDocEnv(t)
	path := writeConfigDocFixture(t, `
max_chunk_tokens: 512
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
`)

	t.Setenv("LUMEN_MAX_CHUNK_TOKENS", "2048")
	t.Setenv("OLLAMA_HOST", "http://ollama.example:11434")

	svc, err := NewConfigService(path)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if got := svc.MaxChunkTokens(); got != 2048 {
		t.Fatalf("MaxChunkTokens() = %d, want 2048", got)
	}
	if got := svc.Servers()[0].Host; got != "http://ollama.example:11434" {
		t.Fatalf("Servers()[0].Host = %q, want http://ollama.example:11434", got)
	}
}

func repoRootFromConfigPackage(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func TestConfigurationDocumentationIsLinkedFromREADME(t *testing.T) {
	root := repoRootFromConfigPackage(t)

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	if !strings.Contains(string(readme), "docs/CONFIGURATION.md") {
		t.Fatalf("README.md must link to docs/CONFIGURATION.md")
	}

	doc, err := os.ReadFile(filepath.Join(root, "docs", "CONFIGURATION.md"))
	if err != nil {
		t.Fatalf("read docs/CONFIGURATION.md: %v", err)
	}
	for _, want := range []string{
		"~/.config/lumen/config.yaml",
		"$XDG_CONFIG_HOME/lumen/config.yaml",
		"servers:",
		"LUMEN_EMBED_MODEL",
		"OLLAMA_HOST",
		"LM_STUDIO_HOST",
	} {
		if !strings.Contains(string(doc), want) {
			t.Fatalf("docs/CONFIGURATION.md missing %q", want)
		}
	}
}
