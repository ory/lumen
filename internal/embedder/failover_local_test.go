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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/models"
)

// fakeLocal stands in for the in-process embedder in failover tests.
type fakeLocal struct {
	embedErr error
	queries  []string
	closed   atomic.Bool
}

func (f *fakeLocal) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if f.embedErr != nil {
		return nil, f.embedErr
	}
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = []float32{1, 0, 0}
	}
	return out, nil
}

func (f *fakeLocal) EmbedQuery(_ context.Context, q string) ([]float32, error) {
	f.queries = append(f.queries, q)
	return []float32{0, 1, 0}, nil
}

func (f *fakeLocal) Dimensions() int   { return 3584 }
func (f *fakeLocal) ModelName() string { return "fake" }
func (f *fakeLocal) Close() error      { f.closed.Store(true); return nil }

// localConfig writes a config whose first server is the local backend using
// modelPath, optionally followed by an Ollama fallback.
func localConfig(t *testing.T, modelPath, fallbackHost string) (*config.ConfigService, string) {
	t.Helper()
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_MODEL_PATH"} {
		t.Setenv(k, "")
	}
	y := fmt.Sprintf("servers:\n  - backend: local\n    model: %s\n    model_path: %s\n", models.DefaultLocalModel, modelPath)
	if fallbackHost != "" {
		y += fmt.Sprintf("  - backend: ollama\n    host: %s\n    model: test\n    dims: 3\n", fallbackHost)
	}
	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(y), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	return cfg, cfgFile
}

func ggufFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "model.gguf")
	if err := os.WriteFile(p, []byte("GGUF fake model"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func withFakeLocal(fe *FailoverEmbedder, fake *fakeLocal) *atomic.Int32 {
	var created atomic.Int32
	fe.newLocal = func(config.ServerConfig, string, int, *slog.Logger) (Embedder, error) {
		created.Add(1)
		return fake, nil
	}
	return &created
}

func TestFailover_LocalServerNeedsNoEmbeddingServer(t *testing.T) {
	path := ggufFile(t)
	cfg, _ := localConfig(t, path, "")
	fe := NewFailoverEmbedder(cfg)
	fake := &fakeLocal{}
	withFakeLocal(fe, fake)

	srv := cfg.Servers()[0]
	if got, want := fe.ModelName(), LocalModelIdentity(srv); got != want {
		t.Errorf("ModelName = %q, want %q", got, want)
	}
	if !strings.HasPrefix(fe.ModelName(), models.DefaultLocalModel+"@local-file-") {
		t.Errorf("ModelName = %q does not identify the user-supplied file", fe.ModelName())
	}
	if got := fe.Dimensions(); got != 3584 {
		t.Errorf("Dimensions = %d, want 3584", got)
	}
	if _, err := fe.Embed(context.Background(), []string{"code"}); err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if fe.ActiveServerIndex() != 0 {
		t.Errorf("active = %d, want local server 0", fe.ActiveServerIndex())
	}
}

func TestFailover_LocalInferenceErrorDoesNotSwitchEmbeddingSpace(t *testing.T) {
	up := newTestOllamaServer(t, true, 200)
	defer up.Close()
	cfg, _ := localConfig(t, ggufFile(t), up.URL)
	fe := NewFailoverEmbedder(cfg)
	withFakeLocal(fe, &fakeLocal{embedErr: errors.New("native decode failed")})

	_, err := fe.Embed(context.Background(), []string{"code"})
	if err == nil || !strings.Contains(err.Error(), "native decode failed") {
		t.Fatalf("Embed error = %v, want the local inference error", err)
	}
	if fe.ActiveServerIndex() != 0 {
		t.Errorf("active = %d, want 0: local failures must not fail over", fe.ActiveServerIndex())
	}
}

func TestFailover_MissingLocalModelFileIsUnhealthy(t *testing.T) {
	up := newTestOllamaServer(t, true, 200)
	defer up.Close()
	cfg, _ := localConfig(t, filepath.Join(t.TempDir(), "missing.gguf"), up.URL)
	fe := NewFailoverEmbedder(cfg)
	created := withFakeLocal(fe, &fakeLocal{})

	if _, err := fe.Embed(context.Background(), []string{"code"}); err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if fe.ActiveServerIndex() != 1 {
		t.Errorf("active = %d, want fallback 1", fe.ActiveServerIndex())
	}
	if created.Load() != 0 {
		t.Error("local embedder created for an unhealthy server")
	}
	if err := ProbeServer(context.Background(), cfg.Servers()[0]); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("ProbeServer = %v, want missing model_path error", err)
	}
}

func TestFailover_EmbedQueryUsesActiveQueryEncoding(t *testing.T) {
	cfg, _ := localConfig(t, ggufFile(t), "")
	fe := NewFailoverEmbedder(cfg)
	fake := &fakeLocal{}
	withFakeLocal(fe, fake)

	v, err := EmbedQuery(context.Background(), fe, "find parser")
	if err != nil {
		t.Fatal(err)
	}
	if v[1] != 1 || len(fake.queries) != 1 || fake.queries[0] != "find parser" {
		t.Errorf("query not routed to EmbedQuery: vec=%v queries=%v", v, fake.queries)
	}

	// Remote backends have no query encoding and use plain Embed.
	up := newTestOllamaServer(t, true, 200)
	defer up.Close()
	remote := NewFailoverEmbedder(testConfigService(t, config.ServerConfig{Backend: "ollama", Host: up.URL, Model: "test", Dims: 3}))
	if v, err := EmbedQuery(context.Background(), remote, "find parser"); err != nil || len(v) != 3 {
		t.Errorf("remote EmbedQuery = %v, %v", v, err)
	}
}

func TestFailover_CloseAndReloadReleaseLocalModel(t *testing.T) {
	path := ggufFile(t)
	cfg, cfgFile := localConfig(t, path, "")
	fe := NewFailoverEmbedder(cfg)
	first := &fakeLocal{}
	withFakeLocal(fe, first)
	if _, err := fe.Embed(context.Background(), []string{"code"}); err != nil {
		t.Fatal(err)
	}
	if err := fe.Close(); err != nil {
		t.Fatal(err)
	}
	if !first.closed.Load() {
		t.Error("Close did not release the local embedder")
	}

	// A model_path change re-initializes and releases the previous model.
	second := &fakeLocal{}
	withFakeLocal(fe, second)
	if _, err := fe.Embed(context.Background(), []string{"code"}); err != nil {
		t.Fatal(err)
	}
	other := ggufFile(t)
	if err := os.WriteFile(cfgFile, []byte(fmt.Sprintf("servers:\n  - backend: local\n    model: %s\n    model_path: %s\n", models.DefaultLocalModel, other)), 0o644); err != nil {
		t.Fatal(err)
	}
	reloaded, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	fe.cfg = reloaded
	withFakeLocal(fe, &fakeLocal{})
	if _, err := fe.Embed(context.Background(), []string{"code"}); err != nil {
		t.Fatal(err)
	}
	if !second.closed.Load() {
		t.Error("server change did not release the previous local embedder")
	}
}

func TestFailover_RemoteFailureDoesNotSwitchToLocalEmbeddingSpace(t *testing.T) {
	remote := newTestOllamaServer(t, true, 500)
	defer remote.Close()
	path := ggufFile(t)
	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_MODEL_PATH"} {
		t.Setenv(k, "")
	}
	y := fmt.Sprintf("servers:\n  - backend: ollama\n    host: %s\n    model: test\n    dims: 3\n  - backend: local\n    model: %s\n    model_path: %s\n", remote.URL, models.DefaultLocalModel, path)
	if err := os.WriteFile(cfgFile, []byte(y), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	fe := NewFailoverEmbedder(cfg)
	withFakeLocal(fe, &fakeLocal{})

	model := fe.ModelName()
	if model != "test" {
		t.Fatalf("ModelName = %q, want remote model", model)
	}
	pinned := Pin(fe, model)
	if vecs, err := pinned.Embed(context.Background(), []string{"code"}); !errors.Is(err, ErrProfileChanged) {
		t.Fatalf("pinned Embed = %d vectors, %v; want ErrProfileChanged", len(vecs), err)
	}
	if _, err := EmbedQuery(context.Background(), pinned, "query"); !errors.Is(err, ErrProfileChanged) {
		t.Fatalf("pinned EmbedQuery error = %v, want ErrProfileChanged", err)
	}

	// Unpinned calls never cross into the local embedding space either.
	fe2 := NewFailoverEmbedder(cfg)
	withFakeLocal(fe2, &fakeLocal{})
	if vecs, err := fe2.Embed(context.Background(), []string{"code"}); !errors.Is(err, ErrProfileChanged) {
		t.Fatalf("Embed = %d vectors, %v; want ErrProfileChanged", len(vecs), err)
	}
	// New operations see the new profile and may use it.
	if got, want := fe2.ModelName(), LocalModelIdentity(cfg.Servers()[1]); got != want {
		t.Errorf("ModelName after failover = %q, want %q", got, want)
	}
	if _, err := Pin(fe2, fe2.ModelName()).Embed(context.Background(), []string{"code"}); err != nil {
		t.Errorf("Embed pinned to the new profile: %v", err)
	}
}

func TestFailover_PinnedOperationRejectsReloadedProfile(t *testing.T) {
	cfg, cfgFile := localConfig(t, ggufFile(t), "")
	fe := NewFailoverEmbedder(cfg)
	withFakeLocal(fe, &fakeLocal{})

	model := fe.ModelName()
	op := Pin(fe, model)
	if op.ModelName() != model {
		t.Errorf("pinned ModelName = %q, want %q", op.ModelName(), model)
	}
	if _, err := op.Embed(context.Background(), []string{"batch 1"}); err != nil {
		t.Fatalf("batch 1: %v", err)
	}

	// The model changes between two batches of the same operation.
	other := filepath.Join(t.TempDir(), "other.gguf")
	if err := os.WriteFile(other, []byte("GGUF other weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgFile, []byte(fmt.Sprintf("servers:\n  - backend: local\n    model: %s\n    model_path: %s\n", models.DefaultLocalModel, other)), 0o644); err != nil {
		t.Fatal(err)
	}
	reloaded, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	fe.mu.Lock()
	fe.cfg = reloaded
	fe.mu.Unlock()

	if _, err := op.Embed(context.Background(), []string{"batch 2"}); !errors.Is(err, ErrProfileChanged) {
		t.Fatalf("batch 2 error = %v, want ErrProfileChanged", err)
	}
	if fe.ModelName() == model {
		t.Error("ModelName still reports the previous profile")
	}
	if _, err := Pin(fe, fe.ModelName()).Embed(context.Background(), []string{"code"}); err != nil {
		t.Errorf("new operation: %v", err)
	}
}

func TestFailover_ReplacedModelFileReloadsBackend(t *testing.T) {
	path := ggufFile(t)
	cfg, _ := localConfig(t, path, "")
	fe := NewFailoverEmbedder(cfg)
	first := &fakeLocal{}
	created := withFakeLocal(fe, first)

	model := fe.ModelName()
	op := Pin(fe, model)
	if _, err := op.Embed(context.Background(), []string{"code"}); err != nil {
		t.Fatal(err)
	}

	tmp := path + ".new"
	if err := os.WriteFile(tmp, []byte("GGUF replaced weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}

	if _, err := op.Embed(context.Background(), []string{"code"}); !errors.Is(err, ErrProfileChanged) {
		t.Fatalf("Embed pinned to the replaced file: %v, want ErrProfileChanged", err)
	}
	if got := fe.ModelName(); got == model || got != LocalModelIdentity(cfg.Servers()[0]) {
		t.Errorf("ModelName = %q after replacement (was %q)", got, model)
	}
	if _, err := Pin(fe, fe.ModelName()).Embed(context.Background(), []string{"code"}); err != nil {
		t.Fatal(err)
	}
	if !first.closed.Load() || created.Load() != 2 {
		t.Errorf("replaced model not reloaded: closed=%v created=%d", first.closed.Load(), created.Load())
	}
}

// writeServers writes a config with a failing remote primary followed by the
// given second server entry.
func writeServers(t *testing.T, cfgFile, primaryHost, second string) {
	t.Helper()
	y := fmt.Sprintf("servers:\n  - backend: ollama\n    host: %s\n    model: test\n    dims: 3584\n%s", primaryHost, second)
	if err := os.WriteFile(cfgFile, []byte(y), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fullDimLocal returns local-sized vectors, as a real local backend would.
type fullDimLocal struct{ fakeLocal }

func (l *fullDimLocal) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = make([]float32, 3584)
		out[i][0] = 1
	}
	return out, nil
}

func TestFailover_ReloadWhileProbingFallbackDoesNotReturnOtherProfile(t *testing.T) {
	primary := newTestOllamaServer(t, true, 500)
	defer primary.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(entered); <-release })
		_, _ = w.Write([]byte(`{"models":[{"name":"test"}]}`))
	}))
	defer fallback.Close()

	_, cfgFile := localConfig(t, ggufFile(t), "")
	writeServers(t, cfgFile, primary.URL, fmt.Sprintf("  - backend: ollama\n    host: %s\n    model: test\n    dims: 3584\n", fallback.URL))
	cfg, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Watch(); err != nil {
		t.Fatal(err)
	}
	defer cfg.Stop()
	fe := NewFailoverEmbedder(cfg)
	var created atomic.Int32
	fe.newLocal = func(config.ServerConfig, string, int, *slog.Logger) (Embedder, error) {
		created.Add(1)
		return &fullDimLocal{}, nil
	}

	op := Pin(fe, fe.ModelName())
	result := make(chan error, 1)
	go func() {
		_, err := op.Embed(context.Background(), []string{"code"})
		result <- err
	}()

	// The fallback is reconfigured as a local server while it is probed.
	<-entered
	writeServers(t, cfgFile, primary.URL, fmt.Sprintf("  - backend: local\n    model: %s\n    model_path: %s\n", models.DefaultLocalModel, ggufFile(t)))
	deadline := time.Now().Add(5 * time.Second)
	for cfg.Servers()[1].Backend != config.BackendLocal {
		if time.Now().After(deadline) {
			t.Fatal("configuration watcher did not reload")
		}
		time.Sleep(time.Millisecond)
	}
	close(release)

	if err := <-result; !errors.Is(err, ErrProfileChanged) {
		t.Fatalf("Embed pinned to %s = %v, want ErrProfileChanged (local backends created: %d)", op.ModelName(), err, created.Load())
	}
}

func TestFailover_ReloadedUninitializedFallbackChangesProfile(t *testing.T) {
	primary := newTestOllamaServer(t, true, 500)
	defer primary.Close()
	fallback := newTestOllamaServer(t, true, 200)
	defer fallback.Close()

	_, cfgFile := localConfig(t, ggufFile(t), "")
	writeServers(t, cfgFile, primary.URL, fmt.Sprintf("  - backend: ollama\n    host: %s\n    model: test\n    dims: 3584\n", fallback.URL))
	cfg, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	fe := NewFailoverEmbedder(cfg)
	op := Pin(fe, fe.ModelName())

	// Reconfigure the fallback, which has not been instantiated yet.
	writeServers(t, cfgFile, primary.URL, fmt.Sprintf("  - backend: ollama\n    host: %s\n    model: a\n    dims: 3584\n", fallback.URL))
	reloaded, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	fe.mu.Lock()
	fe.cfg = reloaded
	fe.mu.Unlock()

	if _, err := op.Embed(context.Background(), []string{"code"}); !errors.Is(err, ErrProfileChanged) {
		t.Fatalf("Embed pinned to %s after fallback switched to model a = %v, want ErrProfileChanged", op.ModelName(), err)
	}
}

func TestFailover_ModelReplacedDuringBackendConstructionIsRejected(t *testing.T) {
	path := ggufFile(t)
	cfg, cfgFile := localConfig(t, path, "")
	fe := NewFailoverEmbedder(cfg)
	withFakeLocal(fe, &fakeLocal{})
	model := fe.ModelName()
	op := Pin(fe, model)

	// A configuration edit that keeps the profile re-initializes the backend;
	// the weights are replaced after the profile is computed but while the
	// backend is constructed.
	if err := os.WriteFile(cfgFile, []byte(fmt.Sprintf("servers:\n  - backend: local\n    model: %s\n    model_path: %s\n    min_score: 0.25\n", models.DefaultLocalModel, path)), 0o644); err != nil {
		t.Fatal(err)
	}
	reloaded, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	var constructed []*fullDimLocal
	fe.mu.Lock()
	fe.cfg = reloaded
	fe.newLocal = func(config.ServerConfig, string, int, *slog.Logger) (Embedder, error) {
		if len(constructed) == 0 {
			tmp := path + ".new"
			if err := os.WriteFile(tmp, []byte("GGUF replaced weights"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(tmp, path); err != nil {
				t.Fatal(err)
			}
		}
		constructed = append(constructed, &fullDimLocal{})
		return constructed[len(constructed)-1], nil
	}
	fe.mu.Unlock()

	if vecs, err := op.Embed(context.Background(), []string{"batch 2"}); !errors.Is(err, ErrProfileChanged) {
		t.Fatalf("pinned Embed = %d vectors, %v; want ErrProfileChanged", len(vecs), err)
	}
	if len(constructed) == 0 || !constructed[0].closed.Load() {
		t.Error("backend constructed while the weights were replaced was not released")
	}

	// A new operation uses the replacement's profile.
	if got := fe.ModelName(); got == model || got != LocalModelIdentity(reloaded.Servers()[0]) {
		t.Errorf("ModelName = %q after replacement (was %q)", got, model)
	}
	if _, err := Pin(fe, fe.ModelName()).Embed(context.Background(), []string{"code"}); err != nil {
		t.Errorf("new operation: %v", err)
	}
}
