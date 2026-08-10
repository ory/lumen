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
	"sync"
	"time"

	"github.com/ory/lumen/internal/config"
)

const healthCheckTimeout = 5 * time.Second
const reprobeInterval = 30 * time.Second

// FailoverEmbedder wraps multiple backend embedders with ordered health-check
// failover. It probes servers on first use and fails over on transient errors.
// When the ConfigService's server list changes (e.g. via hot reload), the
// embedder re-initializes on the next Embed call.
type FailoverEmbedder struct {
	cfg           *config.ConfigService
	logger        *slog.Logger
	mu            sync.Mutex
	servers       []serverEntry
	cachedConfigs []config.ServerConfig // snapshot at last init
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
	return &FailoverEmbedder{cfg: cfg, active: -1}
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
	f.mu.Lock()
	idx := f.maybeReprobe(context.Background(), false)
	f.mu.Unlock()
	if idx < 0 {
		idx = 0
	}
	return f.cfg.ServerDims(idx)
}

// ModelName returns the model name for the active server. On first call it
// probes servers for health to ensure the returned name matches the server
// that will actually handle embeddings.
func (f *FailoverEmbedder) ModelName() string {
	f.mu.Lock()
	idx := f.maybeReprobe(context.Background(), false)
	f.mu.Unlock()
	if idx < 0 {
		idx = 0
	}
	servers := f.cfg.Servers()
	if idx < len(servers) {
		return servers[idx].Model
	}
	return ""
}

// Embed generates embeddings for the given texts. On first call it probes
// all servers for health and selects the first healthy one. On transient
// errors (5xx, network) it fails over to the next healthy server.
func (f *FailoverEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	f.mu.Lock()
	f.maybeReprobe(ctx, true)
	f.mu.Unlock()

	if f.active < 0 {
		return nil, fmt.Errorf("all embedding servers are unhealthy")
	}

	for {
		f.mu.Lock()
		active := f.active
		if active < 0 || active >= len(f.servers) {
			f.mu.Unlock()
			return nil, fmt.Errorf("all embedding servers exhausted")
		}
		emb := f.servers[active].emb
		f.mu.Unlock()

		result, err := emb.Embed(ctx, texts)
		if err == nil {
			return result, nil
		}

		if !isTransientError(err) {
			return nil, err // 4xx = config error, don't failover
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
		f.active = next
		if err := f.ensureEmbedder(next); err != nil {
			f.mu.Unlock()
			return nil, fmt.Errorf("failed to initialize fallback server %d: %w", next, err)
		}
		f.mu.Unlock()
		// Loop continues with new active server
	}
}

// serversChanged returns true if the ConfigService's server list differs from
// the cached server list (e.g. after a hot reload). Must be called with f.mu held.
func (f *FailoverEmbedder) serversChanged() bool {
	current := f.cfg.Servers()
	if len(current) != len(f.servers) {
		return true
	}
	for i, srv := range current {
		// Compare key fields — if any differ, servers have changed.
		if f.servers[i].emb == nil {
			continue // not initialized yet, can't compare
		}
		cached := f.cachedConfigs[i]
		if srv.Backend != cached.Backend || srv.Host != cached.Host || srv.Model != cached.Model {
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
	f.servers = make([]serverEntry, len(servers))
	f.cachedConfigs = make([]config.ServerConfig, len(servers))
	copy(f.cachedConfigs, servers)
	f.active = -1
	for i := range servers {
		f.servers[i] = serverEntry{}
	}
	if f.logger != nil {
		f.logger.Info("probing embedding servers", "count", len(servers))
	}
	for i := range f.servers {
		if f.probeHealth(ctx, i) {
			f.servers[i].healthy = true
			if err := f.ensureEmbedder(i); err == nil {
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
func (f *FailoverEmbedder) probeHealth(ctx context.Context, i int) bool {
	servers := f.cfg.Servers()
	if i >= len(servers) {
		return false
	}
	srv := servers[i]
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

// ensureEmbedder lazily initializes the backend embedder for server i.
// Must be called with f.mu held.
func (f *FailoverEmbedder) ensureEmbedder(i int) error {
	if f.servers[i].emb != nil {
		return nil
	}
	servers := f.cfg.Servers()
	srv := servers[i]
	dims := f.cfg.ServerDims(i)
	ctxLen := f.cfg.ServerCtxLength(i)

	var emb Embedder
	var err error
	switch srv.Backend {
	case "ollama":
		emb, err = NewOllama(srv.Model, dims, ctxLen, srv.Host)
	case "lmstudio":
		emb, err = NewLMStudio(srv.Model, dims, srv.Host)
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
