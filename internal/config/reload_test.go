package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func pollUntil(deadline time.Duration, check func() bool) bool {
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		if check() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func TestReload_FileChange(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "LUMEN_MAX_CHUNK_TOKENS"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte("max_chunk_tokens: 512\nservers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: ordis/jina-embeddings-v2-base-code\n"), 0644)

	svc, _ := NewConfigService(cfgFile)
	if err := svc.Watch(); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	defer svc.Stop()

	_ = os.WriteFile(cfgFile, []byte("max_chunk_tokens: 2048\nservers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: ordis/jina-embeddings-v2-base-code\n"), 0644)

	if !pollUntil(2*time.Second, func() bool { return svc.MaxChunkTokens() == 2048 }) {
		t.Errorf("MaxChunkTokens() = %d, want 2048 after reload", svc.MaxChunkTokens())
	}
}

func TestReload_EnvStillWins(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte("max_chunk_tokens: 512\nservers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: ordis/jina-embeddings-v2-base-code\n"), 0644)
	t.Setenv("LUMEN_MAX_CHUNK_TOKENS", "9999")
	svc, _ := NewConfigService(cfgFile)
	if err := svc.Watch(); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	defer svc.Stop()

	_ = os.WriteFile(cfgFile, []byte("max_chunk_tokens: 1024\nservers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: ordis/jina-embeddings-v2-base-code\n"), 0644)

	// Wait for reload to process, then verify env still wins
	time.Sleep(500 * time.Millisecond)
	if got := svc.MaxChunkTokens(); got != 9999 {
		t.Errorf("MaxChunkTokens() = %d, want 9999 (env wins)", got)
	}
}

func TestReload_InvalidRetainsPrevious(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "LUMEN_MAX_CHUNK_TOKENS"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte("max_chunk_tokens: 512\nservers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: ordis/jina-embeddings-v2-base-code\n"), 0644)
	svc, _ := NewConfigService(cfgFile)
	if err := svc.Watch(); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	defer svc.Stop()

	_ = os.WriteFile(cfgFile, []byte("servers: []"), 0644)
	time.Sleep(500 * time.Millisecond)

	if got := svc.MaxChunkTokens(); got != 512 {
		t.Errorf("MaxChunkTokens() = %d, want 512 (retain previous on invalid config)", got)
	}
}

func TestReload_ServerListChange(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "LUMEN_MAX_CHUNK_TOKENS"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte("servers:\n  - backend: ollama\n    host: http://a:11434\n    model: ordis/jina-embeddings-v2-base-code\n  - backend: ollama\n    host: http://b:11434\n    model: ordis/jina-embeddings-v2-base-code\n"), 0644)
	svc, _ := NewConfigService(cfgFile)
	if err := svc.Watch(); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	defer svc.Stop()

	_ = os.WriteFile(cfgFile, []byte("servers:\n  - backend: ollama\n    host: http://a:11434\n    model: ordis/jina-embeddings-v2-base-code\n  - backend: ollama\n    host: http://b:11434\n    model: ordis/jina-embeddings-v2-base-code\n  - backend: ollama\n    host: http://c:11434\n    model: ordis/jina-embeddings-v2-base-code\n"), 0644)

	if !pollUntil(2*time.Second, func() bool { return len(svc.Servers()) == 3 }) {
		t.Errorf("Servers() len = %d, want 3 after reload", len(svc.Servers()))
	}
}

func TestReload_ConcurrentReads(t *testing.T) {
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "LUMEN_MAX_CHUNK_TOKENS"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(cfgFile, []byte("max_chunk_tokens: 512\nservers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: ordis/jina-embeddings-v2-base-code\n"), 0644)
	svc, _ := NewConfigService(cfgFile)
	if err := svc.Watch(); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	defer svc.Stop()

	done := make(chan struct{})
	for range 100 {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = svc.MaxChunkTokens()
					_ = svc.Servers()
					_ = svc.ServerDims(0)
				}
			}
		}()
	}

	for i := range 10 {
		_ = os.WriteFile(cfgFile, fmt.Appendf(nil, "max_chunk_tokens: %d\nservers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: ordis/jina-embeddings-v2-base-code\n", 512+i), 0644)
		time.Sleep(100 * time.Millisecond)
	}
	close(done)
}

func TestReloadPreservesRetrievalAndServerSelection(t *testing.T) {
	for _, key := range []string{"LUMEN_RETRIEVAL", "LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "OLLAMA_HOST", "LM_STUDIO_HOST"} {
		t.Setenv(key, "")
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := "servers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: all-minilm\n  - backend: lmstudio\n    host: http://localhost:1234\n    model: selected-model\n    dims: 4\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewConfigService(path, WithRetrieval("semantic"), WithServerSelection("selected-model", "lmstudio"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("retrieval: bm25\nlog_level: debug\n"+body), 0644); err != nil {
		t.Fatal(err)
	}
	cfg.reload()
	if cfg.Retrieval() != "semantic" || len(cfg.Servers()) != 1 || cfg.Servers()[0].Model != "selected-model" || cfg.Servers()[0].Backend != "lmstudio" || cfg.LogLevel() != "debug" {
		t.Fatalf("lost startup selection: %s %+v", cfg.Retrieval(), cfg.Servers())
	}
	bm, err := NewConfigService(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("retrieval: semantic\nservers: []\nlog_level: warn\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bm.reload()
	if bm.Retrieval() != "bm25" || bm.LogLevel() != "warn" {
		t.Fatal("running BM25 switched retrieval on reload")
	}
}
