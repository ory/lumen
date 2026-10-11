package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ory/lumen/internal/models"
)

func TestDefaults_NoFileNoEnv(t *testing.T) {
	// Clear env vars that would override defaults to test clean-slate behaviour.
	for _, key := range []string{
		"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX",
		"OLLAMA_HOST", "LM_STUDIO_HOST",
		"LUMEN_MAX_CHUNK_TOKENS", "LUMEN_FRESHNESS_TTL", "LUMEN_REINDEX_TIMEOUT", "LUMEN_LOG_LEVEL",
	} {
		t.Setenv(key, "")
	}
	svc, err := NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if got := svc.MaxChunkTokens(); got != 512 {
		t.Errorf("MaxChunkTokens() = %d, want 512", got)
	}
	if got := svc.FreshnessTTL(); got != 60*time.Second {
		t.Errorf("FreshnessTTL() = %v, want 60s", got)
	}
	if got := svc.ReindexTimeout(); got != 0 {
		t.Errorf("ReindexTimeout() = %v, want 0", got)
	}
	if got := svc.LogLevel(); got != "info" {
		t.Errorf("LogLevel() = %q, want %q", got, "info")
	}
	servers := svc.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	s := servers[0]
	if s.Backend != "ollama" {
		t.Errorf("Backend = %q, want %q", s.Backend, "ollama")
	}
	if s.Host != "http://localhost:11434" {
		t.Errorf("Host = %q, want %q", s.Host, "http://localhost:11434")
	}
	if s.Model != "ordis/jina-embeddings-v2-base-code" {
		t.Errorf("Model = %q, want %q", s.Model, "ordis/jina-embeddings-v2-base-code")
	}
}

func TestYAMLOverridesDefaults(t *testing.T) {
	// Clear env vars that might interfere
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "LUMEN_MAX_CHUNK_TOKENS"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte(`
max_chunk_tokens: 1024
log_level: debug
servers:
  - backend: lmstudio
    host: http://myhost:5555
    model: nomic-ai/nomic-embed-code-GGUF
  - backend: ollama
    host: http://other:11434
    model: ordis/jina-embeddings-v2-base-code
`), 0644)

	svc, err := NewConfigService(cfgFile)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if got := svc.MaxChunkTokens(); got != 1024 {
		t.Errorf("MaxChunkTokens() = %d, want 1024", got)
	}
	if got := svc.LogLevel(); got != "debug" {
		t.Errorf("LogLevel() = %q, want %q", got, "debug")
	}
	servers := svc.Servers()
	if len(servers) != 2 {
		t.Fatalf("Servers() len = %d, want 2", len(servers))
	}
	if servers[0].Backend != "lmstudio" {
		t.Errorf("server[0].Backend = %q, want %q", servers[0].Backend, "lmstudio")
	}
}

func TestEnvOverridesYAML(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte(`max_chunk_tokens: 1024
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
`), 0644)
	t.Setenv("LUMEN_MAX_CHUNK_TOKENS", "2048")
	svc, _ := NewConfigService(cfgFile)
	if got := svc.MaxChunkTokens(); got != 2048 {
		t.Errorf("MaxChunkTokens() = %d, want 2048", got)
	}
}

func TestEnvServerMapping_Ollama(t *testing.T) {
	for _, k := range []string{"LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "LM_STUDIO_HOST"} {
		t.Setenv(k, "")
	}
	t.Setenv("LUMEN_BACKEND", "ollama")
	t.Setenv("OLLAMA_HOST", "http://custom:9999")
	t.Setenv("LUMEN_EMBED_MODEL", "ordis/jina-embeddings-v2-base-code")
	svc, err := NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	s := svc.Servers()[0]
	if s.Backend != "ollama" {
		t.Errorf("Backend = %q, want ollama", s.Backend)
	}
	if s.Host != "http://custom:9999" {
		t.Errorf("Host = %q, want http://custom:9999", s.Host)
	}
	if s.Model != "ordis/jina-embeddings-v2-base-code" {
		t.Errorf("Model = %q, want ordis/jina-embeddings-v2-base-code", s.Model)
	}
}

func TestEnvServerMapping_LMStudio(t *testing.T) {
	for _, k := range []string{"LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "OLLAMA_HOST"} {
		t.Setenv(k, "")
	}
	t.Setenv("LUMEN_BACKEND", "lmstudio")
	t.Setenv("LM_STUDIO_HOST", "http://lms:2222")
	t.Setenv("LUMEN_EMBED_MODEL", "nomic-ai/nomic-embed-code-GGUF")
	svc, err := NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if got := svc.Servers()[0].Host; got != "http://lms:2222" {
		t.Errorf("Host = %q, want http://lms:2222", got)
	}
}

func TestHostConflict_BothSet(t *testing.T) {
	for _, k := range []string{"LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	t.Setenv("LUMEN_BACKEND", "lmstudio")
	t.Setenv("OLLAMA_HOST", "http://ollama:1111")
	t.Setenv("LM_STUDIO_HOST", "http://lms:2222")
	t.Setenv("LUMEN_EMBED_MODEL", "nomic-ai/nomic-embed-code-GGUF")
	svc, err := NewConfigService("")
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if got := svc.Servers()[0].Host; got != "http://lms:2222" {
		t.Errorf("Host = %q, want http://lms:2222 (lmstudio should win)", got)
	}
}

func TestEnvBackendOnly_ResetsServerDefaults(t *testing.T) {
	for _, k := range []string{"LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte(`
servers:
  - backend: ollama
    host: http://custom-ollama:11434
    model: all-minilm
`), 0644)

	// Backend override without explicit host/model must not keep stale
	// ollama host/model from YAML.
	t.Setenv("LUMEN_BACKEND", "lmstudio")

	svc, err := NewConfigService(cfgFile)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	s := svc.Servers()[0]
	if s.Backend != BackendLMStudio {
		t.Errorf("Backend = %q, want %q", s.Backend, BackendLMStudio)
	}
	if s.Host != "http://localhost:1234" {
		t.Errorf("Host = %q, want %q", s.Host, "http://localhost:1234")
	}
	if s.Model != models.DefaultLMStudioModel {
		t.Errorf("Model = %q, want %q", s.Model, models.DefaultLMStudioModel)
	}
}

func TestDimsFallback_KnownModel(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte(`
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
`), 0644)
	svc, _ := NewConfigService(cfgFile)
	if got := svc.ServerDims(0); got != 768 {
		t.Errorf("ServerDims(0) = %d, want 768", got)
	}
	if got := svc.ServerMinScore(0); got != 0.35 {
		t.Errorf("ServerMinScore(0) = %f, want 0.35", got)
	}
}

func TestDimsExplicit_OverridesKnown(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte(`
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
    dims: 1024
    min_score: 0.5
`), 0644)
	svc, _ := NewConfigService(cfgFile)
	if got := svc.ServerDims(0); got != 1024 {
		t.Errorf("ServerDims(0) = %d, want 1024", got)
	}
	if got := svc.ServerMinScore(0); got != 0.5 {
		t.Errorf("ServerMinScore(0) = %f, want 0.5", got)
	}
}

func TestDimsUnresolvable_Error(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte(`
servers:
  - backend: ollama
    host: http://localhost:11434
    model: totally-unknown-model
`), 0644)
	_, err := NewConfigService(cfgFile)
	if err == nil {
		t.Fatal("expected error for unresolvable dims")
	}
}

func TestServersForModel_Filters(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte(`
servers:
  - backend: ollama
    host: http://a:11434
    model: ordis/jina-embeddings-v2-base-code
  - backend: lmstudio
    host: http://b:1234
    model: nomic-ai/nomic-embed-code-GGUF
  - backend: ollama
    host: http://c:11434
    model: ordis/jina-embeddings-v2-base-code
`), 0644)
	svc, _ := NewConfigService(cfgFile)
	indices, err := svc.ServersForModel("ordis/jina-embeddings-v2-base-code")
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != 2 || indices[0] != 0 || indices[1] != 2 {
		t.Errorf("got %v, want [0, 2]", indices)
	}
}

func TestServersForModel_NoMatch(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	svc, _ := NewConfigService("")
	_, err := svc.ServersForModel("nonexistent-model")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidation_MissingBackend(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(f, []byte(`servers: [{host: "http://x:1", model: m}]`), 0644)
	if _, err := NewConfigService(f); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidation_UnknownBackend(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(f, []byte(`servers: [{backend: foo, host: "http://x:1", model: m}]`), 0644)
	if _, err := NewConfigService(f); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidation_EmptyServerList(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(f, []byte(`servers: []`), 0644)
	if _, err := NewConfigService(f); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidation_InvalidHost(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(f, []byte(`servers: [{backend: ollama, host: not-a-url, model: ordis/jina-embeddings-v2-base-code}]`), 0644)
	if _, err := NewConfigService(f); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidation_MissingModel(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(f, []byte(`servers: [{backend: ollama, host: "http://x:1"}]`), 0644)
	if _, err := NewConfigService(f); err == nil {
		t.Fatal("expected error")
	}
}

func TestStop_Idempotent(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(f, []byte(`
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
`), 0644)
	svc, err := NewConfigService(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Watch(); err != nil {
		t.Fatal(err)
	}
	// Calling Stop() twice must not panic.
	svc.Stop()
	svc.Stop()
}

func TestModelAlias_Resolution(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	// Use an alias name that maps to nomic-ai/nomic-embed-code-GGUF
	_ = os.WriteFile(f, []byte(`
servers:
  - backend: lmstudio
    host: http://localhost:1234
    model: text-embedding-nomic-embed-code
`), 0644)
	svc, err := NewConfigService(f)
	if err != nil {
		t.Fatalf("NewConfigService should resolve alias: %v", err)
	}
	// Dims should resolve via the alias → canonical model → KnownModels
	if got := svc.ServerDims(0); got != 3584 {
		t.Errorf("ServerDims(0) = %d, want 3584 (via alias resolution)", got)
	}
	if got := svc.ServerCtxLength(0); got != 8192 {
		t.Errorf("ServerCtxLength(0) = %d, want 8192 (via alias resolution)", got)
	}
	if got := svc.ServerMinScore(0); got != 0.15 {
		t.Errorf("ServerMinScore(0) = %f, want 0.15 (via alias resolution)", got)
	}
}

func TestWithModelOverride(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	svc, err := NewConfigService("", WithModelOverride("all-minilm"))
	if err != nil {
		t.Fatal(err)
	}
	servers := svc.Servers()
	if servers[0].Model != "all-minilm" {
		t.Errorf("Model = %q, want %q", servers[0].Model, "all-minilm")
	}
	if got := svc.ServerDims(0); got != 384 {
		t.Errorf("ServerDims(0) = %d, want 384", got)
	}
}

// threeServerYAML is a fixture resembling the real user config: an Ollama
// server and two LM Studio servers with distinct model names.
const threeServerYAML = `
servers:
  - backend: lmstudio
    host: http://remote:1234
    model: text-embedding-nomic-embed-code
    dims: 3584
  - backend: lmstudio
    host: http://remote:1234
    model: text-embedding-jina-embeddings-v2-base-code
    dims: 768
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
    dims: 768
`

func writeThreeServerYAML(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(f, []byte(threeServerYAML), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return f
}

func clearServerEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
}

func TestWithServerSelection_ByModel_UnambiguousYAMLEntry(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	cfg := writeThreeServerYAML(t)
	// ordis/jina-embeddings-v2-base-code appears on exactly one server — the
	// Ollama entry. Filtering by model alone should pick it.
	svc, err := NewConfigService(cfg, WithServerSelection("ordis/jina-embeddings-v2-base-code", ""))
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	servers := svc.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Backend != BackendOllama {
		t.Errorf("Backend = %q, want %q", servers[0].Backend, BackendOllama)
	}
	if servers[0].Host != "http://localhost:11434" {
		t.Errorf("Host = %q, want http://localhost:11434", servers[0].Host)
	}
}

func TestWithServerSelection_ByModel_LMStudioName(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	cfg := writeThreeServerYAML(t)
	// The LM-Studio-specific name is not in KnownModels, but it's in the YAML
	// — filtering by it should pick that LM Studio server.
	svc, err := NewConfigService(cfg, WithServerSelection("text-embedding-jina-embeddings-v2-base-code", ""))
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	servers := svc.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Backend != BackendLMStudio {
		t.Errorf("Backend = %q, want %q", servers[0].Backend, BackendLMStudio)
	}
	if servers[0].Model != "text-embedding-jina-embeddings-v2-base-code" {
		t.Errorf("Model = %q, want text-embedding-jina-embeddings-v2-base-code", servers[0].Model)
	}
}

func TestWithServerSelection_ByModelAndBackend(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	// Same model on two backends — backend flag must disambiguate.
	_ = os.WriteFile(f, []byte(`
servers:
  - backend: ollama
    host: http://a:11434
    model: duplicate-model
    dims: 768
  - backend: lmstudio
    host: http://b:1234
    model: duplicate-model
    dims: 768
`), 0644)
	svc, err := NewConfigService(f, WithServerSelection("duplicate-model", BackendLMStudio))
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	servers := svc.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Backend != BackendLMStudio {
		t.Errorf("Backend = %q, want %q", servers[0].Backend, BackendLMStudio)
	}
}

func TestWithServerSelection_ByBackendOnly(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	cfg := writeThreeServerYAML(t)
	svc, err := NewConfigService(cfg, WithServerSelection("", BackendOllama))
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	servers := svc.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Backend != BackendOllama {
		t.Errorf("Backend = %q, want %q", servers[0].Backend, BackendOllama)
	}
}

func TestWithServerSelection_AliasResolution(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	// YAML stores the canonical name; user passes the alias. Alias resolution
	// must make them match.
	_ = os.WriteFile(f, []byte(`
servers:
  - backend: lmstudio
    host: http://localhost:1234
    model: nomic-ai/nomic-embed-code-GGUF
`), 0644)
	svc, err := NewConfigService(f, WithServerSelection("text-embedding-nomic-embed-code", ""))
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if len(svc.Servers()) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(svc.Servers()))
	}
}

func TestWithServerSelection_NoMatchReturnsError(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	cfg := writeThreeServerYAML(t)
	_, err := NewConfigService(cfg, WithServerSelection("no-such-model", ""))
	if err == nil {
		t.Fatal("expected error for no matching server")
	}
}

func TestWithServerSelection_BackendOnlyNoMatch(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(f, []byte(`
servers:
  - backend: ollama
    host: http://a:11434
    model: ordis/jina-embeddings-v2-base-code
`), 0644)
	_, err := NewConfigService(f, WithServerSelection("", BackendLMStudio))
	if err == nil {
		t.Fatal("expected error when backend filter matches nothing")
	}
}

func TestWithServerSelection_PreservesOrder(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	// Three Ollama servers, all with the same model — filtering should keep
	// all three in their original YAML order so failover still works.
	_ = os.WriteFile(f, []byte(`
servers:
  - backend: ollama
    host: http://a:11434
    model: ordis/jina-embeddings-v2-base-code
  - backend: lmstudio
    host: http://b:1234
    model: nomic-ai/nomic-embed-code-GGUF
  - backend: ollama
    host: http://c:11434
    model: ordis/jina-embeddings-v2-base-code
  - backend: ollama
    host: http://d:11434
    model: ordis/jina-embeddings-v2-base-code
`), 0644)
	svc, err := NewConfigService(f, WithServerSelection("ordis/jina-embeddings-v2-base-code", BackendOllama))
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	servers := svc.Servers()
	if len(servers) != 3 {
		t.Fatalf("Servers() len = %d, want 3", len(servers))
	}
	wantHosts := []string{"http://a:11434", "http://c:11434", "http://d:11434"}
	for i, s := range servers {
		if s.Host != wantHosts[i] {
			t.Errorf("Servers()[%d].Host = %q, want %q", i, s.Host, wantHosts[i])
		}
	}
}

func TestWithServerSelection_EmptyArgsNoop(t *testing.T) {
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	clearServerEnv(t)
	cfg := writeThreeServerYAML(t)
	// Empty model and empty backend → no filter applied.
	svc, err := NewConfigService(cfg, WithServerSelection("", ""))
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if got := len(svc.Servers()); got != 3 {
		t.Errorf("Servers() len = %d, want 3 (no filter should be applied)", got)
	}
}
