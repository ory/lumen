package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/ory/lumen/internal/models"
)

// ServerConfig holds per-server configuration.
type ServerConfig struct {
	Backend   string  `koanf:"backend"`
	Host      string  `koanf:"host"`
	Model     string  `koanf:"model"`
	Dims      int     `koanf:"dims"`
	CtxLength int     `koanf:"ctx_length"`
	MinScore  float64 `koanf:"min_score"`
}

// ConfigService wraps koanf and provides typed config access.
type ConfigService struct {
	k             *koanf.Koanf
	mu            sync.RWMutex
	configPath    string
	modelOverride string // set via WithModelOverride, applied after env vars
	selectModel   string // set via WithServerSelection, filters server list
	selectBackend string // set via WithServerSelection, filters server list
	watcher       *fsnotify.Watcher
	stopCh        chan struct{}
	stopOnce      sync.Once
}

func defaultServerForBackend(backend string) ServerConfig {
	switch backend {
	case BackendLMStudio:
		return ServerConfig{
			Backend: BackendLMStudio,
			Host:    "http://localhost:1234",
			Model:   models.DefaultLMStudioModel,
		}
	default:
		return ServerConfig{
			Backend: BackendOllama,
			Host:    "http://localhost:11434",
			Model:   models.DefaultOllamaModel,
		}
	}
}

func defaultsMap() map[string]any {
	return map[string]any{
		"max_chunk_tokens": 512,
		"vector_storage":   "int8",
		"freshness_ttl":    "60s",
		"reindex_timeout":  "0s",
		"log_level":        "info",
		"servers": []map[string]any{
			{
				"backend": BackendOllama,
				"host":    "http://localhost:11434",
				"model":   models.DefaultOllamaModel,
			},
		},
	}
}

// Option configures a ConfigService.
type Option func(*ConfigService)

// WithModelOverride overrides the embedding model for servers[0], taking effect
// after env var processing. Use this instead of os.Setenv to avoid mutating
// global process state.
func WithModelOverride(model string) Option {
	return func(s *ConfigService) { s.modelOverride = model }
}

// WithServerSelection restricts the servers list to entries whose model and/or
// backend match the given values. An empty string for a field means "don't
// filter by that field"; passing both empty is a no-op. Model names are
// alias-resolved on both sides before comparison. If the filter matches zero
// servers, NewConfigService returns a descriptive error.
func WithServerSelection(model, backend string) Option {
	return func(s *ConfigService) {
		s.selectModel = model
		s.selectBackend = backend
	}
}

// NewConfigService creates a ConfigService. configPath is the YAML file path
// (empty string means no file). Environment variables are always loaded.
func NewConfigService(configPath string, opts ...Option) (*ConfigService, error) {
	k := koanf.New(".")

	// Layer 1: hardcoded defaults
	if err := k.Load(confmap.Provider(defaultsMap(), "."), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// Layer 2: YAML config file (optional)
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
				return nil, fmt.Errorf("loading config file %s: %w", configPath, err)
			}
		}
	}

	// Layer 3: environment variable overrides
	applyEnvOverrides(k)

	svc := &ConfigService{k: k, configPath: configPath}
	for _, opt := range opts {
		opt(svc)
	}

	// Layer 4: programmatic overrides (e.g. CLI flags via WithModelOverride)
	if svc.modelOverride != "" {
		var servers []ServerConfig
		_ = k.Unmarshal("servers", &servers)
		if len(servers) > 0 {
			servers[0].Model = svc.modelOverride
			serverMaps := make([]map[string]any, len(servers))
			for i, s := range servers {
				serverMaps[i] = map[string]any{
					"backend": s.Backend, "host": s.Host, "model": s.Model,
					"dims": s.Dims, "ctx_length": s.CtxLength, "min_score": s.MinScore,
				}
			}
			_ = k.Load(confmap.Provider(map[string]any{"servers": serverMaps}, "."), nil)
		}
	}

	// Layer 5: CLI server selection filter (WithServerSelection).
	if svc.selectModel != "" || svc.selectBackend != "" {
		var servers []ServerConfig
		_ = k.Unmarshal("servers", &servers)
		filtered := filterServers(servers, svc.selectModel, svc.selectBackend)
		if len(filtered) == 0 {
			return nil, fmt.Errorf(
				"no server matches --model=%q --backend=%q; configured: %s",
				svc.selectModel, svc.selectBackend, describeServers(servers),
			)
		}
		// Drop the existing list first — koanf merge would otherwise keep
		// stale entries beyond the filtered length.
		k.Delete("servers")
		serverMaps := make([]map[string]any, len(filtered))
		for i, s := range filtered {
			serverMaps[i] = map[string]any{
				"backend": s.Backend, "host": s.Host, "model": s.Model,
				"dims": s.Dims, "ctx_length": s.CtxLength, "min_score": s.MinScore,
			}
		}
		_ = k.Load(confmap.Provider(map[string]any{"servers": serverMaps}, "."), nil)
	}

	if err := svc.validate(); err != nil {
		return nil, err
	}
	return svc, nil
}

// filterServers returns the subset of servers matching model and/or backend.
// Empty model or empty backend means "don't filter by that field". Model
// names are alias-resolved on both sides before comparison.
func filterServers(servers []ServerConfig, model, backend string) []ServerConfig {
	want := resolveModel(model)
	var out []ServerConfig
	for _, s := range servers {
		if model != "" && resolveModel(s.Model) != want {
			continue
		}
		if backend != "" && s.Backend != backend {
			continue
		}
		out = append(out, s)
	}
	return out
}

// describeServers renders a compact list of configured (backend, model) pairs
// for inclusion in error messages.
func describeServers(servers []ServerConfig) string {
	if len(servers) == 0 {
		return "[]"
	}
	parts := make([]string, len(servers))
	for i, s := range servers {
		parts[i] = fmt.Sprintf("(%s, %s)", s.Backend, s.Model)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// applyEnvOverrides reads legacy env vars and merges them into koanf.
func applyEnvOverrides(k *koanf.Koanf) {
	globals := make(map[string]any)

	if v := os.Getenv("LUMEN_MAX_CHUNK_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			globals["max_chunk_tokens"] = n
		}
	}
	if v := os.Getenv("LUMEN_FRESHNESS_TTL"); v != "" {
		globals["freshness_ttl"] = v
	}
	if v := os.Getenv("LUMEN_REINDEX_TIMEOUT"); v != "" {
		globals["reindex_timeout"] = v
	}
	if v := os.Getenv("LUMEN_LOG_LEVEL"); v != "" {
		globals["log_level"] = v
	}
	if v := os.Getenv("LUMEN_VECTOR_STORAGE"); v != "" {
		globals["vector_storage"] = strings.ToLower(v)
	}
	if len(globals) > 0 {
		_ = k.Load(confmap.Provider(globals, "."), nil)
	}

	// Server env vars → merge into servers[0]
	backendEnv := os.Getenv("LUMEN_BACKEND")
	model := os.Getenv("LUMEN_EMBED_MODEL")
	dims := os.Getenv("LUMEN_EMBED_DIMS")
	ctx := os.Getenv("LUMEN_EMBED_CTX")
	ollamaHost := os.Getenv("OLLAMA_HOST")
	lmStudioHost := os.Getenv("LM_STUDIO_HOST")

	// Only apply if at least one server env var is explicitly set
	hasOverride := backendEnv != "" || model != "" || dims != "" || ctx != "" || ollamaHost != "" || lmStudioHost != ""
	if !hasOverride {
		return
	}

	// Read current servers from koanf, merge env into [0]
	var servers []ServerConfig
	_ = k.Unmarshal("servers", &servers)
	if len(servers) == 0 {
		servers = append(servers, defaultServerForBackend(BackendOllama))
	}
	srv := servers[0]

	// If backend is explicitly overridden, reset server[0] to backend-specific
	// defaults first to avoid mixed config (e.g. lmstudio backend with Ollama host/model).
	if backendEnv != "" {
		srv = defaultServerForBackend(backendEnv)
	}

	if model != "" {
		srv.Model = model
	}

	// Host env vars are backend-specific. If backend is explicitly set, apply
	// the host var for that backend. Otherwise, apply host var for the current
	// configured backend of server[0].
	selectedBackend := srv.Backend
	if selectedBackend == "" {
		selectedBackend = BackendOllama
	}
	switch selectedBackend {
	case BackendLMStudio:
		if lmStudioHost != "" {
			srv.Host = lmStudioHost
		}
	default:
		if ollamaHost != "" {
			srv.Host = ollamaHost
		}
	}

	if dims != "" {
		if n, err := strconv.Atoi(dims); err == nil {
			srv.Dims = n
		}
	}
	if ctx != "" {
		if n, err := strconv.Atoi(ctx); err == nil {
			srv.CtxLength = n
		}
	}
	servers[0] = srv

	// Re-marshal servers back into koanf
	serverMaps := make([]map[string]any, len(servers))
	for i, s := range servers {
		serverMaps[i] = map[string]any{
			"backend": s.Backend, "host": s.Host, "model": s.Model,
			"dims": s.Dims, "ctx_length": s.CtxLength, "min_score": s.MinScore,
		}
	}
	_ = k.Load(confmap.Provider(map[string]any{"servers": serverMaps}, "."), nil)
}

func (s *ConfigService) MaxChunkTokens() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.k.Int("max_chunk_tokens")
}

// VectorStorage returns the sqlite-vec element type used for persisted
// embeddings. int8 is the default; float32 is retained as an opt-in override.
func (s *ConfigService) VectorStorage() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.k.String("vector_storage")
}

func (s *ConfigService) FreshnessTTL() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, _ := time.ParseDuration(s.k.String("freshness_ttl"))
	return d
}

func (s *ConfigService) ReindexTimeout() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, _ := time.ParseDuration(s.k.String("reindex_timeout"))
	return d
}

func (s *ConfigService) LogLevel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.k.String("log_level")
}

func (s *ConfigService) Servers() []ServerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var servers []ServerConfig
	_ = s.k.Unmarshal("servers", &servers)
	return servers
}

// serverAt returns the ServerConfig for index i without acquiring s.mu.
// Callers must either hold s.mu before calling this, or call it on a struct
// that has not yet been shared with other goroutines.
func (s *ConfigService) serverAt(i int) (ServerConfig, bool) {
	var servers []ServerConfig
	_ = s.k.Unmarshal("servers", &servers)
	if i < 0 || i >= len(servers) {
		return ServerConfig{}, false
	}
	return servers[i], true
}

// resolveModel applies alias resolution for a model name, returning the
// canonical name if an alias exists, or the original name otherwise.
func resolveModel(model string) string {
	if canonical, ok := models.ModelAliases[model]; ok {
		return canonical
	}
	return model
}

// serverDims returns dims for server i without locking (caller must hold lock).
func (s *ConfigService) serverDims(i int) int {
	srv, ok := s.serverAt(i)
	if !ok {
		return 0
	}
	if srv.Dims != 0 {
		return srv.Dims
	}
	canonical := resolveModel(srv.Model)
	if spec, ok := models.KnownModels[canonical]; ok {
		return spec.Dims
	}
	return 0
}

func (s *ConfigService) ServerDims(i int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serverDims(i)
}

func (s *ConfigService) ServerCtxLength(i int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	srv, ok := s.serverAt(i)
	if !ok {
		return 0
	}
	if srv.CtxLength != 0 {
		return srv.CtxLength
	}
	canonical := resolveModel(srv.Model)
	if spec, ok := models.KnownModels[canonical]; ok {
		return spec.CtxLength
	}
	return 0
}

// ServerMinScore returns min score for server i. Uses unlocked serverDims
// to avoid recursive RLock (which deadlocks under writer contention).
func (s *ConfigService) ServerMinScore(i int) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	srv, ok := s.serverAt(i)
	if !ok {
		return 0
	}
	if srv.MinScore != 0 {
		return srv.MinScore
	}
	canonical := resolveModel(srv.Model)
	if spec, ok := models.KnownModels[canonical]; ok && spec.MinScore != 0 {
		return spec.MinScore
	}
	return models.DimensionAwareMinScore(s.serverDims(i))
}

func (s *ConfigService) ServersForModel(model string) ([]int, error) {
	servers := s.Servers()
	var indices []int
	for i, srv := range servers {
		if srv.Model == model {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("no server configured for model %q", model)
	}
	return indices, nil
}

// validate checks the current configuration for correctness.
// Must be called either on a ConfigService that has not yet been shared with
// other goroutines, or on a temporary ConfigService (as in reload). Must not
// be called while holding s.mu — it acquires RLock via Servers().
func (s *ConfigService) validate() error {
	storage := s.VectorStorage()
	if storage != "int8" && storage != "float32" {
		return fmt.Errorf("config: vector_storage must be int8 or float32, got %q", storage)
	}
	servers := s.Servers()
	if len(servers) == 0 {
		return fmt.Errorf("config: servers list is empty")
	}
	for i, srv := range servers {
		if srv.Backend == "" {
			return fmt.Errorf("config: servers[%d]: backend is required", i)
		}
		if srv.Backend != BackendOllama && srv.Backend != BackendLMStudio {
			return fmt.Errorf("config: servers[%d]: unknown backend %q", i, srv.Backend)
		}
		if srv.Model == "" {
			return fmt.Errorf("config: servers[%d]: model is required", i)
		}
		if srv.Host == "" {
			return fmt.Errorf("config: servers[%d]: host is required", i)
		}
		u, err := url.Parse(srv.Host)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("config: servers[%d]: host %q must be a valid http/https URL", i, srv.Host)
		}
		if s.serverDims(i) == 0 {
			return fmt.Errorf("config: servers[%d]: cannot resolve dims for model %q — set dims explicitly", i, srv.Model)
		}
	}
	return nil
}

// Watch starts watching the config file for changes (MCP server only).
func (s *ConfigService) Watch() error {
	if s.configPath == "" {
		return nil
	}
	// Watch the directory (handles file renames/recreates).
	// If the directory doesn't exist yet, skip watching — no config file
	// can be present in a non-existent directory.
	dir := filepath.Dir(s.configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	if err := w.Add(dir); err != nil {
		_ = w.Close()
		return fmt.Errorf("watching %s: %w", dir, err)
	}
	s.mu.Lock()
	s.stopCh = make(chan struct{})
	s.watcher = w
	s.mu.Unlock()
	go s.watchLoop()
	return nil
}

// Stop stops the file watcher. Safe to call multiple times.
func (s *ConfigService) Stop() {
	s.stopOnce.Do(func() {
		s.mu.RLock()
		stopCh := s.stopCh
		watcher := s.watcher
		s.mu.RUnlock()
		if stopCh != nil {
			close(stopCh)
		}
		if watcher != nil {
			_ = watcher.Close()
		}
	})
}

func (s *ConfigService) watchLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != filepath.Base(s.configPath) {
				continue
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				s.reload()
			}
		case _, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (s *ConfigService) reload() {
	k := koanf.New(".")
	_ = k.Load(confmap.Provider(defaultsMap(), "."), nil)

	if s.configPath != "" {
		if _, err := os.Stat(s.configPath); err == nil {
			if err := k.Load(file.Provider(s.configPath), yaml.Parser()); err != nil {
				return // keep previous config
			}
		}
	}

	applyEnvOverrides(k)

	tmp := &ConfigService{k: k}
	if err := tmp.validate(); err != nil {
		return // keep previous config on validation error
	}

	s.mu.Lock()
	s.k = k
	s.mu.Unlock()
}
