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

package embedder

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
)

func newTestOllamaServer(t *testing.T, healthy bool, embedStatus int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/tags":
			if healthy {
				_, _ = fmt.Fprint(w, `{"models":[
					{"name":"test"},{"name":"a"},{"name":"b"},
					{"name":"test-a"},{"name":"test-b"},{"name":"test-ollama"},
					{"name":"alive"},{"name":"dead"},
					{"name":"big-model"},{"name":"small-model"}
				]}`)
			} else {
				w.WriteHeader(503)
			}
		case r.Method == "POST" && r.URL.Path == "/api/embed":
			w.WriteHeader(embedStatus)
			if embedStatus == 200 {
				_, _ = fmt.Fprint(w, `{"embeddings":[[0.1,0.2,0.3]]}`)
			} else {
				_, _ = fmt.Fprintf(w, `{"error":"status %d"}`, embedStatus)
			}
		default:
			w.WriteHeader(404)
		}
	}))
}

func testConfigService(t *testing.T, servers ...config.ServerConfig) *config.ConfigService {
	t.Helper()
	// Clear env vars that would override the test config servers.
	t.Setenv("LUMEN_BACKEND", "")
	t.Setenv("LUMEN_EMBED_MODEL", "")
	t.Setenv("LUMEN_EMBED_DIMS", "")
	t.Setenv("LUMEN_EMBED_CTX", "")
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("LM_STUDIO_HOST", "")

	y := "servers:\n"
	for _, s := range servers {
		y += fmt.Sprintf("  - backend: %s\n    host: %s\n    model: %s\n    dims: %d\n",
			s.Backend, s.Host, s.Model, s.Dims)
	}
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(y), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	svc, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	return svc
}

func TestFailover_FirstHealthy(t *testing.T) {
	down := newTestOllamaServer(t, false, 200)
	defer down.Close()
	up1 := newTestOllamaServer(t, true, 200)
	defer up1.Close()
	up2 := newTestOllamaServer(t, true, 200)
	defer up2.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: down.URL, Model: "test", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: up1.URL, Model: "test", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: up2.URL, Model: "test", Dims: 3},
	)
	fe := NewFailoverEmbedder(cfg)
	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if fe.ActiveServerIndex() != 1 {
		t.Errorf("active = %d, want 1", fe.ActiveServerIndex())
	}
}

func TestFailover_OnEmbedError(t *testing.T) {
	srv1 := newTestOllamaServer(t, true, 500)
	defer srv1.Close()
	srv2 := newTestOllamaServer(t, true, 200)
	defer srv2.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: srv1.URL, Model: "test", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: srv2.URL, Model: "test", Dims: 3},
	)
	fe := NewFailoverEmbedder(cfg)
	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if fe.ActiveServerIndex() != 1 {
		t.Errorf("active = %d, want 1", fe.ActiveServerIndex())
	}
}

func TestFailover_4xxNoFailover(t *testing.T) {
	srv1 := newTestOllamaServer(t, true, 400)
	defer srv1.Close()
	srv2 := newTestOllamaServer(t, true, 200)
	defer srv2.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: srv1.URL, Model: "test", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: srv2.URL, Model: "test", Dims: 3},
	)
	fe := NewFailoverEmbedder(cfg)
	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if fe.ActiveServerIndex() != 0 {
		t.Errorf("active = %d, want 0 (4xx should not failover)", fe.ActiveServerIndex())
	}
}

func TestFailover_AllExhausted(t *testing.T) {
	d1 := newTestOllamaServer(t, false, 200)
	defer d1.Close()
	d2 := newTestOllamaServer(t, false, 200)
	defer d2.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: d1.URL, Model: "test", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: d2.URL, Model: "test", Dims: 3},
	)
	fe := NewFailoverEmbedder(cfg)
	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error when all down")
	}
}

func TestFailover_DimensionsReflectActive(t *testing.T) {
	down := newTestOllamaServer(t, false, 200)
	defer down.Close()
	up := newTestOllamaServer(t, true, 200)
	defer up.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: down.URL, Model: "a", Dims: 768},
		config.ServerConfig{Backend: "ollama", Host: up.URL, Model: "b", Dims: 1024},
	)
	fe := NewFailoverEmbedder(cfg)
	// Dimensions() eagerly inits and picks the healthy server (b, 1024 dims).
	if got := fe.Dimensions(); got != 1024 {
		t.Errorf("before embed: Dimensions() = %d, want 1024 (healthy server)", got)
	}
	_, _ = fe.Embed(context.Background(), []string{"hello"})
	if got := fe.Dimensions(); got != 1024 {
		t.Errorf("after embed: Dimensions() = %d, want 1024", got)
	}
	if got := fe.ModelName(); got != "b" {
		t.Errorf("ModelName() = %q, want %q", got, "b")
	}
}

func TestFailover_SingleServer(t *testing.T) {
	srv := newTestOllamaServer(t, true, 200)
	defer srv.Close()
	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: srv.URL, Model: "test", Dims: 3},
	)
	fe := NewFailoverEmbedder(cfg)
	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if fe.ActiveServerIndex() != 0 {
		t.Errorf("active = %d, want 0", fe.ActiveServerIndex())
	}
}

func TestFailover_LazyInit(t *testing.T) {
	srv1 := newTestOllamaServer(t, true, 200)
	defer srv1.Close()
	srv2 := newTestOllamaServer(t, true, 200)
	defer srv2.Close()
	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: srv1.URL, Model: "test", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: srv2.URL, Model: "test", Dims: 3},
	)
	fe := NewFailoverEmbedder(cfg)
	_, _ = fe.Embed(context.Background(), []string{"hello"})
	fe.mu.Lock()
	if fe.servers[0].emb == nil {
		t.Error("server[0].emb should be initialized")
	}
	if fe.servers[1].emb != nil {
		t.Error("server[1].emb should be nil (lazy)")
	}
	fe.mu.Unlock()
}

func TestFailover_ReloadPicksUpNewServers(t *testing.T) {
	// Clear env vars that would override the test config servers.
	t.Setenv("LUMEN_BACKEND", "")
	t.Setenv("LUMEN_EMBED_MODEL", "")
	t.Setenv("LUMEN_EMBED_DIMS", "")
	t.Setenv("LUMEN_EMBED_CTX", "")
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("LM_STUDIO_HOST", "")

	down := newTestOllamaServer(t, false, 200)
	defer down.Close()
	up := newTestOllamaServer(t, true, 200)
	defer up.Close()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(fmt.Sprintf("servers:\n  - backend: ollama\n    host: %s\n    model: test\n    dims: 3\n", down.URL)), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	if err := cfg.Watch(); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	defer cfg.Stop()

	fe := NewFailoverEmbedder(cfg)
	_, err = fe.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error with only down server")
	}

	// Hot reload adds the healthy server
	if err := os.WriteFile(cfgFile, []byte(fmt.Sprintf("servers:\n  - backend: ollama\n    host: %s\n    model: test\n    dims: 3\n  - backend: ollama\n    host: %s\n    model: test\n    dims: 3\n", down.URL, up.URL)), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// serversChanged() now detects the config reload automatically
	_, err = fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed after reload: %v", err)
	}
}

func newTestLMStudioServer(t *testing.T, healthy bool, embedStatus int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/models":
			if healthy {
				w.WriteHeader(200)
				_, _ = fmt.Fprint(w, `{"data":[{"id":"test-lm"}]}`)
			} else {
				w.WriteHeader(503)
			}
		case r.Method == "POST" && r.URL.Path == "/v1/embeddings":
			w.WriteHeader(embedStatus)
			if embedStatus == 200 {
				_, _ = fmt.Fprint(w, `{"data":[{"embedding":[0.1,0.2,0.3]}]}`)
			} else {
				_, _ = fmt.Fprintf(w, `{"error":"status %d"}`, embedStatus)
			}
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestFailover_PrimaryDown_FallsBackWithLogging(t *testing.T) {
	// Simulate real scenario: remote LM Studio is down, local Ollama is up.
	lmDown := newTestLMStudioServer(t, false, 200)
	defer lmDown.Close()
	ollamaUp := newTestOllamaServer(t, true, 200)
	defer ollamaUp.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "lmstudio", Host: lmDown.URL, Model: "test-lm", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: ollamaUp.URL, Model: "test-ollama", Dims: 3},
	)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	fe := NewFailoverEmbedder(cfg)
	fe.SetLogger(logger)

	result, err := fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed should succeed via fallback, got: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty embeddings")
	}
	if fe.ActiveServerIndex() != 1 {
		t.Errorf("active = %d, want 1 (should have fallen back to ollama)", fe.ActiveServerIndex())
	}

	logs := buf.String()
	if !strings.Contains(logs, "health probe failed") {
		t.Error("expected log entry for failed health probe on server 0")
	}
	if !strings.Contains(logs, "selected embedding server") {
		t.Error("expected log entry for server selection")
	}
	if !strings.Contains(logs, "test-ollama") {
		t.Error("expected selected server log to mention the ollama model name")
	}
}

func TestFailover_PrimaryUp_SelectsPrimary(t *testing.T) {
	// Both servers are healthy — should select primary (server 0).
	lmUp := newTestLMStudioServer(t, true, 200)
	defer lmUp.Close()
	ollamaUp := newTestOllamaServer(t, true, 200)
	defer ollamaUp.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "lmstudio", Host: lmUp.URL, Model: "test-lm", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: ollamaUp.URL, Model: "test-ollama", Dims: 3},
	)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	fe := NewFailoverEmbedder(cfg)
	fe.SetLogger(logger)

	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if fe.ActiveServerIndex() != 0 {
		t.Errorf("active = %d, want 0 (primary should be selected)", fe.ActiveServerIndex())
	}

	logs := buf.String()
	if !strings.Contains(logs, "selected embedding server") {
		t.Error("expected log entry for server selection")
	}
	if !strings.Contains(logs, "test-lm") {
		t.Error("expected selected server log to mention the lmstudio model name")
	}
	if strings.Contains(logs, "health probe failed") {
		t.Error("no health probe warnings expected when primary is up")
	}
}

func TestFailover_AllDown_LogsWarning(t *testing.T) {
	lmDown := newTestLMStudioServer(t, false, 200)
	defer lmDown.Close()
	ollamaDown := newTestOllamaServer(t, false, 200)
	defer ollamaDown.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "lmstudio", Host: lmDown.URL, Model: "test-lm", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: ollamaDown.URL, Model: "test-ollama", Dims: 3},
	)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	fe := NewFailoverEmbedder(cfg)
	fe.SetLogger(logger)

	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error when all servers are down")
	}

	logs := buf.String()
	if !strings.Contains(logs, "no healthy embedding server found") {
		t.Error("expected 'no healthy embedding server found' warning")
	}
	// Should have two health probe warnings (one per server)
	if count := strings.Count(logs, "health probe failed"); count != 2 {
		t.Errorf("expected 2 health probe warnings, got %d", count)
	}
}

func TestFailover_LMStudioWithoutConfiguredModelFallsBack(t *testing.T) {
	lmEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/models" {
			_, _ = fmt.Fprint(w, `{"data":[]}`)
			return
		}
		t.Fatalf("LM Studio embedding endpoint must not be called when no model is loaded: %s %s", r.Method, r.URL.Path)
	}))
	defer lmEmpty.Close()
	ollamaUp := newTestOllamaServer(t, true, http.StatusOK)
	defer ollamaUp.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "lmstudio", Host: lmEmpty.URL, Model: "test-lm", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: ollamaUp.URL, Model: "test-ollama", Dims: 3},
	)
	fe := NewFailoverEmbedder(cfg)
	if _, err := fe.Embed(context.Background(), []string{"hello"}); err != nil {
		t.Fatalf("Embed should succeed through Ollama fallback: %v", err)
	}
	if got := fe.ActiveServerIndex(); got != 1 {
		t.Fatalf("ActiveServerIndex = %d, want 1", got)
	}
}

func TestFailover_PrimaryFailsDuringEmbed_LogsFailover(t *testing.T) {
	// Primary is healthy but returns 500 on embed; should failover with logging.
	srv1 := newTestOllamaServer(t, true, 500)
	defer srv1.Close()
	srv2 := newTestOllamaServer(t, true, 200)
	defer srv2.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: srv1.URL, Model: "test-a", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: srv2.URL, Model: "test-b", Dims: 3},
	)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	fe := NewFailoverEmbedder(cfg)
	fe.SetLogger(logger)

	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed should succeed via failover, got: %v", err)
	}
	if fe.ActiveServerIndex() != 1 {
		t.Errorf("active = %d, want 1", fe.ActiveServerIndex())
	}

	logs := buf.String()
	if !strings.Contains(logs, "embedding server failed, trying next") {
		t.Error("expected failover log entry")
	}
}

func TestFailover_Unreachable_LogsConnectionError(t *testing.T) {
	// Server 0 is unreachable (closed immediately), server 1 is healthy.
	unreachable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	unreachable.Close() // close immediately to make it unreachable

	up := newTestOllamaServer(t, true, 200)
	defer up.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: unreachable.URL, Model: "dead", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: up.URL, Model: "alive", Dims: 3},
	)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	fe := NewFailoverEmbedder(cfg)
	fe.SetLogger(logger)

	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed should succeed via fallback, got: %v", err)
	}
	if fe.ActiveServerIndex() != 1 {
		t.Errorf("active = %d, want 1", fe.ActiveServerIndex())
	}

	logs := buf.String()
	if !strings.Contains(logs, "health probe failed") {
		t.Error("expected 'health probe failed' with connection error for unreachable server")
	}
	if !strings.Contains(logs, "selected embedding server") {
		t.Error("expected server selection log")
	}
}

// newTogglableOllamaServer creates an Ollama test server whose health and embed
// responses can be toggled at runtime via atomic booleans.
func newTogglableOllamaServer(t *testing.T, healthy *atomic.Bool, embedOK *atomic.Bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/tags":
			if healthy.Load() {
				_, _ = fmt.Fprint(w, `{"models":[{"name":"test-a"},{"name":"test-b"}]}`)
			} else {
				w.WriteHeader(503)
			}
		case r.Method == "POST" && r.URL.Path == "/api/embed":
			if embedOK.Load() {
				w.WriteHeader(200)
				_, _ = fmt.Fprint(w, `{"embeddings":[[0.1,0.2,0.3]]}`)
			} else {
				w.WriteHeader(500)
				_, _ = fmt.Fprint(w, `{"error":"server error"}`)
			}
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestFailover_ReprobesAfterCooldown(t *testing.T) {
	// Both servers start unhealthy (health probe returns 503).
	healthy1 := &atomic.Bool{}
	healthy2 := &atomic.Bool{}
	embed1 := &atomic.Bool{}
	embed2 := &atomic.Bool{}
	embed1.Store(true)
	embed2.Store(true)

	srv1 := newTogglableOllamaServer(t, healthy1, embed1)
	defer srv1.Close()
	srv2 := newTogglableOllamaServer(t, healthy2, embed2)
	defer srv2.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: srv1.URL, Model: "test-a", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: srv2.URL, Model: "test-b", Dims: 3},
	)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	fe := NewFailoverEmbedder(cfg)
	fe.SetLogger(logger)

	// Step 1: All servers down — expect error.
	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err == nil || !strings.Contains(err.Error(), "unhealthy") {
		t.Fatalf("expected 'unhealthy' error, got: %v", err)
	}

	// Step 2: Make server 0 healthy.
	healthy1.Store(true)

	// Step 3: Embed immediately — should still fail (cooldown hasn't elapsed).
	_, err = fe.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error before cooldown elapsed, but Embed succeeded")
	}

	// Step 4: Expire the cooldown by backdating lastProbeTime.
	fe.mu.Lock()
	fe.lastProbeTime = time.Now().Add(-reprobeInterval - time.Second)
	fe.mu.Unlock()

	// Step 5: Now Embed should re-probe and succeed.
	_, err = fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("expected success after cooldown, got: %v", err)
	}

	// Verify re-probe was logged.
	logs := buf.String()
	if !strings.Contains(logs, "re-probing embedding servers after cooldown") {
		t.Error("expected 're-probing' log entry")
	}
}

func TestFailover_ReprobesAfterFailoverExhaustion(t *testing.T) {
	// Both servers pass health check but return 500 on embed.
	healthy1 := &atomic.Bool{}
	healthy2 := &atomic.Bool{}
	embed1 := &atomic.Bool{}
	embed2 := &atomic.Bool{}
	healthy1.Store(true)
	healthy2.Store(true)
	// embed bools default to false → 500

	srv1 := newTogglableOllamaServer(t, healthy1, embed1)
	defer srv1.Close()
	srv2 := newTogglableOllamaServer(t, healthy2, embed2)
	defer srv2.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: srv1.URL, Model: "test-a", Dims: 3},
		config.ServerConfig{Backend: "ollama", Host: srv2.URL, Model: "test-b", Dims: 3},
	)

	fe := NewFailoverEmbedder(cfg)

	// Step 1: Embed fails — both servers return 500, failover exhausted.
	_, err := fe.Embed(context.Background(), []string{"hello"})
	if err == nil || !strings.Contains(err.Error(), "exhausted") {
		t.Fatalf("expected 'exhausted' error, got: %v", err)
	}

	// Step 2: ActiveServerIndex should be -1 after exhaustion.
	if idx := fe.ActiveServerIndex(); idx != -1 {
		t.Fatalf("expected ActiveServerIndex() == -1 after exhaustion, got %d", idx)
	}

	// Step 3: Fix server 0 to return 200 on embed.
	embed1.Store(true)

	// Step 4: Expire cooldown.
	fe.mu.Lock()
	fe.lastProbeTime = time.Now().Add(-reprobeInterval - time.Second)
	fe.mu.Unlock()

	// Step 5: Embed should re-probe and succeed.
	_, err = fe.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("expected success after cooldown + fix, got: %v", err)
	}
}

func TestFailover_ModelNameTriggersInit(t *testing.T) {
	// Server 0 (LM Studio-like, 3584 dims) is DOWN.
	// Server 1 (Ollama-like, 768 dims) is UP.
	// ModelName() should eagerly init and return server 1's model, not server 0's.
	down := newTestOllamaServer(t, false, 200)
	defer down.Close()
	up := newTestOllamaServer(t, true, 200)
	defer up.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: down.URL, Model: "big-model", Dims: 3584},
		config.ServerConfig{Backend: "ollama", Host: up.URL, Model: "small-model", Dims: 768},
	)

	fe := NewFailoverEmbedder(cfg)

	// Before any Embed() call, ModelName() should return the healthy server's model.
	if got := fe.ModelName(); got != "small-model" {
		t.Errorf("ModelName() = %q, want %q (should eagerly init and pick healthy server)", got, "small-model")
	}
}

func TestFailover_DimensionsTriggersInit(t *testing.T) {
	// Server 0 (3584 dims) is DOWN, server 1 (768 dims) is UP.
	// Dimensions() should eagerly init and return 768, not 3584.
	down := newTestOllamaServer(t, false, 200)
	defer down.Close()
	up := newTestOllamaServer(t, true, 200)
	defer up.Close()

	cfg := testConfigService(t,
		config.ServerConfig{Backend: "ollama", Host: down.URL, Model: "big-model", Dims: 3584},
		config.ServerConfig{Backend: "ollama", Host: up.URL, Model: "small-model", Dims: 768},
	)

	fe := NewFailoverEmbedder(cfg)

	// Before any Embed() call, Dimensions() should return the healthy server's dims.
	if got := fe.Dimensions(); got != 768 {
		t.Errorf("Dimensions() = %d, want 768 (should eagerly init and pick healthy server)", got)
	}
}
