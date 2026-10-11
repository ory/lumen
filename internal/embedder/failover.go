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
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/ory/lumen/internal/config"
)

const healthCheckTimeout = 5 * time.Second
const reprobeInterval = 30 * time.Second

// ErrProfileChanged is returned instead of embeddings when the active
// embedding profile (model identity and dimensions) is not the one the
// operation started with, e.g. after a configuration reload, a replaced local
// model file, or a failover to a server with a different model. Vectors of
// different profiles must not share a collection; retry the operation with
// the current profile.
var ErrProfileChanged = errors.New("embedding profile changed")

// profile identifies an embedding space: the stored model identity (see
// ModelName) and the vector dimensionality.
type profile struct {
	model string
	dims  int
}

// FailoverEmbedder wraps multiple backend embedders with ordered health-check
// failover. It probes servers on first use and fails over on transient errors.
// When the ConfigService's server list changes (e.g. via hot reload), the
// embedder re-initializes on the next Embed call.
type FailoverEmbedder struct {
	cfg           *config.ConfigService
	logger        *slog.Logger
	newLocal      func(srv config.ServerConfig, identity string, dims int, logger *slog.Logger) (Embedder, error)
	mu            sync.Mutex
	servers       []serverEntry
	cachedConfigs []config.ServerConfig // snapshot at last init; servers, profiles, and probes derive from it
	profiles      []profile             // per server, computed at last init
	active        int
	checked       bool
	lastProbeTime time.Time
}

type serverEntry struct {
	emb     Embedder
	healthy bool
}

// NewFailoverEmbedder creates a FailoverEmbedder. Servers are probed lazily on
// first Embed call.
func NewFailoverEmbedder(cfg *config.ConfigService) *FailoverEmbedder {
	return &FailoverEmbedder{cfg: cfg, active: -1, newLocal: func(srv config.ServerConfig, identity string, dims int, logger *slog.Logger) (Embedder, error) {
		return newLocalFor(srv, identity, dims, logger)
	}}
}

// SetLogger attaches a logger to the embedder for structured diagnostic output.
func (f *FailoverEmbedder) SetLogger(l *slog.Logger) {
	f.logger = l
}

// ActiveServerIndex returns the index of the currently active server, or -1
// if no healthy server has been found yet.
func (f *FailoverEmbedder) ActiveServerIndex() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.active
}

// maybeReprobe checks whether servers need (re-)initialization and does so
// if required. Must be called with f.mu held. Returns the current active index.
func (f *FailoverEmbedder) maybeReprobe(ctx context.Context, log bool) int {
	needsInit := !f.checked || f.serversChanged()
	needsReprobe := f.active < 0 && time.Since(f.lastProbeTime) >= reprobeInterval
	if needsInit || needsReprobe {
		if needsReprobe && log && f.logger != nil {
			f.logger.Info("re-probing embedding servers after cooldown")
		}
		f.initServers(ctx)
		f.checked = true
	}
	return f.active
}

// Dimensions returns dims for the active server. On first call it probes
// servers for health to ensure the returned dimensions match the server
// that will actually handle embeddings.
func (f *FailoverEmbedder) Dimensions() int {
	return f.currentProfile().dims
}

// ModelName returns the model name for the active server. On first call it
// probes servers for health to ensure the returned name matches the server
// that will actually handle embeddings. For the local backend the name also
// identifies the model artifact and preprocessing (see LocalModelIdentity).
func (f *FailoverEmbedder) ModelName() string {
	return f.currentProfile().model
}

// currentProfile returns the active server's profile, or the first server's
// when none is healthy.
func (f *FailoverEmbedder) currentProfile() profile {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := max(f.maybeReprobe(context.Background(), false), 0)
	if idx < len(f.profiles) {
		return f.profiles[idx]
	}
	return profile{}
}

// serverProfile computes the profile of a server.
func serverProfile(srv config.ServerConfig) profile {
	p := profile{model: srv.Model, dims: srv.ResolvedDims()}
	if srv.Backend == config.BackendLocal {
		p.model = LocalModelIdentity(srv)
	}
	return p
}

// Pin binds e to the embedding profile named model (as returned by
// e.ModelName()) for one operation, such as indexing into or querying a
// collection opened for that model. Calls through the returned Embedder fail
// with ErrProfileChanged rather than return vectors of another profile.
// Embedders other than FailoverEmbedder are returned unchanged.
func Pin(e Embedder, model string) Embedder {
	f, ok := e.(*FailoverEmbedder)
	if !ok {
		return e
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return &pinned{f: f, want: f.profileFor(model)}
}

// ModelDimensions returns the dimensions of e's embedding profile named model
// (as returned by e.ModelName()), as used by Pin, or 0 if e has no such
// profile.
func ModelDimensions(e Embedder, model string) int {
	f, ok := e.(*FailoverEmbedder)
	if !ok {
		if e.ModelName() == model {
			return e.Dimensions()
		}
		return 0
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.profileFor(model).dims
}

// profileFor returns the profile named model: the active server's when it
// serves that model, since that is the profile ModelName and Dimensions
// report, otherwise the first configured server's with that model. Must be
// called with f.mu held.
func (f *FailoverEmbedder) profileFor(model string) profile {
	active := f.maybeReprobe(context.Background(), false)
	if active >= 0 && active < len(f.profiles) && f.profiles[active].model == model {
		return f.profiles[active]
	}
	for _, p := range f.profiles {
		if p.model == model {
			return p
		}
	}
	return profile{model: model}
}

// pinned is a FailoverEmbedder view bound to one profile.
type pinned struct {
	f    *FailoverEmbedder
	want profile
}

func (p *pinned) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.f.do(ctx, &p.want, embedCall(ctx, texts))
}

func (p *pinned) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return first(p.f.do(ctx, &p.want, queryCall(ctx, query)))
}

func (p *pinned) Dimensions() int   { return p.want.dims }
func (p *pinned) ModelName() string { return p.want.model }

// Embed generates embeddings for the given texts. On first call it probes
// all servers for health and selects the first healthy one. On transient
// errors (5xx, network) it fails over to the next healthy server.
func (f *FailoverEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return f.do(ctx, nil, embedCall(ctx, texts))
}

// EmbedQuery embeds a search query with the active server's query encoding
// (e.g. the local backend's query instruction), failing over like Embed.
func (f *FailoverEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return first(f.do(ctx, nil, queryCall(ctx, query)))
}

func embedCall(ctx context.Context, texts []string) func(Embedder) ([][]float32, error) {
	return func(emb Embedder) ([][]float32, error) { return emb.Embed(ctx, texts) }
}

func queryCall(ctx context.Context, query string) func(Embedder) ([][]float32, error) {
	return func(emb Embedder) ([][]float32, error) {
		v, err := EmbedQuery(ctx, emb, query)
		if err != nil {
			return nil, err
		}
		return [][]float32{v}, nil
	}
}

func first(vecs [][]float32, err error) ([]float32, error) {
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// Close releases backend resources such as a loaded local model. The
// embedder re-initializes lazily if used afterwards.
func (f *FailoverEmbedder) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closeServers()
	f.servers = nil
	f.cachedConfigs = nil
	f.profiles = nil
	f.active = -1
	f.checked = false
	return nil
}

// closeServers closes initialized backend embedders. Must be called with
// f.mu held.
func (f *FailoverEmbedder) closeServers() {
	for _, s := range f.servers {
		if c, ok := s.emb.(io.Closer); ok {
			_ = c.Close()
		}
	}
}

// do runs call on the active server, failing over on transient errors. With
// want set, the active server must have that profile. Without it, a call may
// fail over between remote servers but never into or out of the local
// backend's embedding space.
func (f *FailoverEmbedder) do(ctx context.Context, want *profile, call func(Embedder) ([][]float32, error)) ([][]float32, error) {
	strict := want != nil
	var lastErr error
	for attempt := 0; ; attempt++ {
		// Re-initialize first if the configuration changed (e.g. while a
		// fallback was probed), so the profile checked below is the one of the
		// backend that will run the call.
		f.mu.Lock()
		active := f.maybeReprobe(ctx, attempt == 0)
		if active < 0 || active >= len(f.servers) {
			f.mu.Unlock()
			if attempt == 0 {
				return nil, fmt.Errorf("all embedding servers are unhealthy")
			}
			return nil, fmt.Errorf("all embedding servers exhausted")
		}
		local := f.cachedConfigs[active].Backend == config.BackendLocal
		cur := f.profiles[active]
		if want == nil {
			want = &cur
		} else if cur != *want && (strict || local) {
			f.mu.Unlock()
			return nil, profileChanged(*want, cur, lastErr)
		}
		if err := f.ensureEmbedder(active); err != nil {
			f.mu.Unlock()
			return nil, fmt.Errorf("failed to initialize embedding server %d: %w", active, err)
		}
		emb := f.servers[active].emb
		f.mu.Unlock()

		result, err := call(emb)
		if err == nil {
			return result, nil
		}

		// 4xx = config error, don't failover. Local inference failures are
		// reported rather than silently switching to another server, whose
		// vectors belong to a different embedding space.
		if !isTransientError(err) || local {
			return nil, err
		}

		// Mark current as unhealthy, find next
		f.mu.Lock()
		f.servers[active].healthy = false
		next := f.findNextHealthy(ctx, active)
		if next < 0 {
			f.active = -1
			f.mu.Unlock()
			return nil, fmt.Errorf("all embedding servers exhausted after failover: last error: %w", err)
		}
		if f.logger != nil {
			f.logger.Warn("embedding server failed, trying next", "failed", active, "next", next, "error", err)
		}
		lastErr = err
		f.active = next
		f.mu.Unlock()
		// Loop continues with new active server
	}
}

// profileChanged reports that the active profile is no longer want, wrapping
// the server error that caused a failover, if any.
func profileChanged(want, cur profile, cause error) error {
	err := fmt.Errorf("%w: operation uses %s (%d dims) but the active server uses %s (%d dims); retry to use the new profile",
		ErrProfileChanged, want.model, want.dims, cur.model, cur.dims)
	if cause != nil {
		return fmt.Errorf("%w (after server error: %w)", err, cause)
	}
	return err
}

// serversChanged returns true if the ConfigService's server list differs from
// the cached server list (e.g. after a hot reload), including servers whose
// backend is not initialized yet. Must be called with f.mu held.
func (f *FailoverEmbedder) serversChanged() bool {
	current := f.cfg.Servers()
	if len(current) != len(f.cachedConfigs) {
		return true
	}
	for i, srv := range current {
		if srv != f.cachedConfigs[i] {
			return true
		}
		// A local model file replaced in place changes the profile.
		if srv.Backend == config.BackendLocal && LocalModelIdentity(srv) != f.profiles[i].model {
			return true
		}
	}
	return false
}

// initServers initializes the server list and probes for the first healthy
// server. Must be called with f.mu held.
func (f *FailoverEmbedder) initServers(ctx context.Context) {
	f.lastProbeTime = time.Now()
	servers := f.cfg.Servers()
	f.closeServers()
	f.servers = make([]serverEntry, len(servers))
	f.cachedConfigs = make([]config.ServerConfig, len(servers))
	copy(f.cachedConfigs, servers)
	f.profiles = make([]profile, len(servers))
	f.active = -1
	for i, srv := range servers {
		f.servers[i] = serverEntry{}
		f.profiles[i] = serverProfile(srv)
	}
	if f.logger != nil {
		f.logger.Info("probing embedding servers", "count", len(servers))
	}
	for i := range f.servers {
		if f.probeHealth(ctx, i) {
			f.servers[i].healthy = true
			err := f.ensureEmbedder(i)
			if errors.Is(err, ErrProfileChanged) {
				// The local model file was replaced since its profile was
				// computed; operations pinned to the old profile fail.
				f.profiles[i] = serverProfile(servers[i])
				err = f.ensureEmbedder(i)
			}
			if err == nil {
				f.active = i
				if f.logger != nil {
					srv := servers[i]
					f.logger.Info("selected embedding server", "server", i, "backend", srv.Backend, "host", srv.Host, "model", srv.Model)
				}
				return
			}
			// ensureEmbedder failed (e.g. unknown backend); try next server
		}
	}
	if f.logger != nil {
		f.logger.Warn("no healthy embedding server found")
	}
}

// findNextHealthy probes servers after index `after` and returns the first
// healthy one, or -1 if none found. Must be called with f.mu held.
func (f *FailoverEmbedder) findNextHealthy(ctx context.Context, after int) int {
	for i := after + 1; i < len(f.servers); i++ {
		if f.probeHealth(ctx, i) {
			f.servers[i].healthy = true
			return i
		}
	}
	return -1
}

// probeHealth checks that server i is reachable and has its configured model
// loaded. A service with no usable embedding model is not healthy.
// Must be called with f.mu held.
func (f *FailoverEmbedder) probeHealth(ctx context.Context, i int) bool {
	if i >= len(f.cachedConfigs) {
		return false
	}
	srv := f.cachedConfigs[i]
	probeCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()
	if err := ProbeServer(probeCtx, srv); err != nil {
		if f.logger != nil {
			f.logger.Warn("health probe failed", "server", i, "backend", srv.Backend, "host", srv.Host, "error", err)
		}
		return false
	}
	return true
}

// ensureEmbedder lazily initializes the backend embedder for server i from
// the configuration snapshot its profile was computed from. Must be called
// with f.mu held.
func (f *FailoverEmbedder) ensureEmbedder(i int) error {
	if f.servers[i].emb != nil {
		return nil
	}
	srv := f.cachedConfigs[i]
	dims := srv.ResolvedDims()
	ctxLen := srv.ResolvedCtxLength()

	var emb Embedder
	var err error
	switch srv.Backend {
	case "ollama":
		emb, err = NewOllama(srv.Model, dims, ctxLen, srv.Host)
	case "lmstudio":
		emb, err = NewLMStudio(srv.Model, dims, srv.Host)
	case config.BackendLocal:
		// Bind the backend to the profile operations were pinned to, and
		// reject weights replaced since that profile was computed.
		want := f.profiles[i]
		emb, err = f.newLocal(srv, want.model, dims, f.logger)
		if err == nil {
			if cur := serverProfile(srv); cur != want {
				if c, ok := emb.(io.Closer); ok {
					_ = c.Close()
				}
				return profileChanged(want, cur, nil)
			}
		}
	default:
		return fmt.Errorf("unknown backend %q", srv.Backend)
	}
	if err != nil {
		return err
	}
	f.servers[i].emb = emb
	return nil
}

// isTransientError returns true if the error represents a transient failure
// that warrants failover (5xx HTTP errors or network errors). Returns false
// for 4xx errors (configuration errors, no failover).
func isTransientError(err error) bool {
	var ee *EmbedError
	if errors.As(err, &ee) {
		return ee.StatusCode >= 500
	}
	return true // network errors are transient
}
