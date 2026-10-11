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
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/llama"
	"github.com/ory/lumen/internal/models"
)

const (
	// localContextSize bounds the tokens decoded per native call (and thus
	// the compute buffers). Longer texts are truncated to this many tokens.
	localContextSize = 2048
	// localMaxSequences bounds the texts decoded per native call.
	localMaxSequences = 16
	// localMaxThreads caps CPU threads so background indexing leaves the
	// machine responsive.
	localMaxThreads = 8
)

// ErrLocalClosed is returned by a Local embedder after Close.
var ErrLocalClosed = errors.New("local embedder is closed")

// Local generates embeddings in-process with the bundled llama.cpp library.
// The model is resolved (downloaded if managed), loaded, and kept for the
// lifetime of the embedder on first use; Close releases it.
type Local struct {
	name    string
	spec    models.LocalModelSpec
	dims    int
	ctxSize int
	resolve func(context.Context, DownloadProgress) (string, error)
	// identify recomputes the artifact identity; nil skips the check.
	identify func() string
	logger   atomic.Pointer[slog.Logger]

	life     context.Context
	stopLife context.CancelFunc

	mu      sync.Mutex
	closed  bool
	loading chan struct{} // closed when the current load attempt finishes
	loadErr error
	model   *llama.Model
	lctx    *llama.Context
}

// NewLocal creates an in-process embedder for a local-backend server. It does
// not touch the model file until the first Embed call.
func NewLocal(srv config.ServerConfig, dims int, logger *slog.Logger) (*Local, error) {
	return newLocalFor(srv, LocalModelIdentity(srv), dims, logger)
}

// newLocalFor creates a local embedder bound to identity, a previously
// computed LocalModelIdentity of srv. Loading fails if the model file no
// longer is the artifact identity names.
func newLocalFor(srv config.ServerConfig, identity string, dims int, logger *slog.Logger) (*Local, error) {
	spec, err := localSpec(srv)
	if err != nil {
		return nil, err
	}
	l := newLocal(identity, spec, dims, func(ctx context.Context, p DownloadProgress) (string, error) {
		return EnsureLocalModel(ctx, srv, p)
	})
	l.identify = func() string { return LocalModelIdentity(srv) }
	l.SetLogger(logger)
	return l, nil
}

func newLocal(name string, spec models.LocalModelSpec, dims int, resolve func(context.Context, DownloadProgress) (string, error)) *Local {
	life, stop := context.WithCancel(context.Background())
	return &Local{
		name:     name,
		spec:     spec,
		dims:     dims,
		ctxSize:  localContextSize,
		resolve:  resolve,
		life:     life,
		stopLife: stop,
	}
}

// SetLogger sets the logger for model download, loading, and native
// llama.cpp diagnostics.
func (l *Local) SetLogger(logger *slog.Logger) {
	l.logger.Store(logger)
	if logger != nil {
		llama.SetLogger(logger)
	}
}

func (l *Local) log() *slog.Logger {
	if lg := l.logger.Load(); lg != nil {
		return lg
	}
	return slog.New(slog.DiscardHandler)
}

// Dimensions returns the embedding dimensionality.
func (l *Local) Dimensions() int { return l.dims }

// ModelName returns the collection identity: model name plus artifact and
// preprocessing fingerprint.
func (l *Local) ModelName() string { return l.name }

// Embed embeds indexed texts (code chunks) without an instruction prefix.
func (l *Local) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return l.embed(ctx, texts)
}

// EmbedQuery embeds a search query with the model's query instruction. A
// query that already starts with the instruction is not prefixed again.
func (l *Local) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if !strings.HasPrefix(query, l.spec.QueryInstruction) {
		query = l.spec.QueryInstruction + query
	}
	vecs, err := l.embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// Close releases the native model and context. In-flight inference finishes
// first; an in-progress download or load is cancelled. Close is idempotent.
func (l *Local) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	loading := l.loading
	l.mu.Unlock()

	l.stopLife()
	if loading != nil {
		<-loading
	}

	l.mu.Lock()
	lctx, model := l.lctx, l.model
	l.lctx, l.model = nil, nil
	l.mu.Unlock()
	if lctx != nil {
		_ = lctx.Close()
	}
	if model != nil {
		_ = model.Close()
	}
	return nil
}

// ready returns the loaded context, starting the load on first use. The
// download and load run under the embedder's lifetime rather than the
// caller's context, so a short query timeout does not abort a multi-gigabyte
// download; the caller just stops waiting.
func (l *Local) ready(ctx context.Context) (*llama.Model, *llama.Context, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, nil, ErrLocalClosed
	}
	if l.lctx != nil {
		m, c := l.model, l.lctx
		l.mu.Unlock()
		return m, c, nil
	}
	if l.loading == nil {
		l.loading = make(chan struct{})
		go l.load(l.loading)
	}
	loading := l.loading
	l.mu.Unlock()

	select {
	case <-loading:
	case <-ctx.Done():
		return nil, nil, fmt.Errorf("local model is still loading: %w", ctx.Err())
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, nil, ErrLocalClosed
	}
	if l.lctx == nil {
		err := l.loadErr
		// Let the next call retry (e.g. after a network failure).
		if l.loading == loading {
			l.loading = nil
		}
		return nil, nil, err
	}
	return l.model, l.lctx, nil
}

func (l *Local) load(done chan struct{}) {
	defer close(done)
	model, lctx, err := l.open(l.life)

	l.mu.Lock()
	defer l.mu.Unlock()
	if err == nil && l.closed {
		_ = lctx.Close()
		_ = model.Close()
		err = ErrLocalClosed
	}
	l.loadErr = err
	if err == nil {
		l.model, l.lctx = model, lctx
	}
}

func (l *Local) open(ctx context.Context) (*llama.Model, *llama.Context, error) {
	log := l.log()
	if err := llama.Supported(); err != nil {
		return nil, nil, err
	}
	path, err := l.resolve(ctx, func(done, total int64) {
		log.Info("downloading local embedding model", "model", l.name, "done_bytes", done, "total_bytes", total,
			"percent", math.Round(float64(done)*1000/float64(max(total, 1)))/10)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("prepare local model: %w", err)
	}
	if err := checkGGUF(path); err != nil {
		return nil, nil, err
	}

	log.Info("loading local embedding model", "model", l.name, "path", path)
	model, lctx, err := l.openWith(ctx, path, false)
	if err != nil && ctx.Err() == nil {
		// Accelerator setup can fail (e.g. Metal unavailable in a VM); the CPU
		// path produces the same embedding space.
		log.Warn("local model failed to load with GPU acceleration, falling back to CPU", "error", err)
		model, lctx, err = l.openWith(ctx, path, true)
	}
	if err != nil {
		return nil, nil, err
	}
	// The identity was computed before loading; reject weights that are not
	// the artifact it names (e.g. a model file replaced in the meantime).
	if l.identify != nil {
		if id := l.identify(); id != l.name {
			_ = lctx.Close()
			_ = model.Close()
			return nil, nil, fmt.Errorf("model file %s changed after it was selected (now %s, want %s)", path, id, l.name)
		}
	}
	log.Info("local embedding model ready", "model", l.name)
	return model, lctx, nil
}

func (l *Local) openWith(ctx context.Context, path string, cpuOnly bool) (*llama.Model, *llama.Context, error) {
	model, err := llama.LoadModel(ctx, path, cpuOnly)
	if err != nil {
		return nil, nil, err
	}
	if arch, _ := model.Meta("general.architecture"); arch != l.spec.Architecture {
		_ = model.Close()
		return nil, nil, fmt.Errorf("model file %s has architecture %q, want %q", path, arch, l.spec.Architecture)
	}
	if got := model.EmbeddingSize(); got != l.dims {
		_ = model.Close()
		return nil, nil, fmt.Errorf("model file %s produces %d-dimensional embeddings, want %d", path, got, l.dims)
	}
	lctx, err := model.NewContext(llama.ContextParams{
		ContextSize:  l.ctxSize,
		MaxSequences: localMaxSequences,
		Threads:      max(1, min(runtime.NumCPU(), localMaxThreads)),
		Pooling:      poolingFor(l.spec.Pooling),
	})
	if err != nil {
		_ = model.Close()
		return nil, nil, err
	}
	return model, lctx, nil
}

func poolingFor(name string) llama.Pooling {
	switch name {
	case "mean":
		return llama.PoolingMean
	case "cls":
		return llama.PoolingCLS
	default:
		return llama.PoolingLast
	}
}

func (l *Local) embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	model, lctx, err := l.ready(ctx)
	if err != nil {
		return nil, err
	}

	tokens := make([][]int32, len(texts))
	for i, text := range texts {
		tok, err := model.Tokenize(text)
		if err != nil {
			return nil, l.closedOr(fmt.Errorf("tokenize text %d: %w", i, err))
		}
		if len(tok) == 0 {
			return nil, fmt.Errorf("text %d produced no tokens", i)
		}
		if len(tok) > l.ctxSize {
			tok = tok[:l.ctxSize]
		}
		tokens[i] = tok
	}

	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(tokens); {
		end, budget := start, 0
		for end < len(tokens) && end-start < localMaxSequences && budget+len(tokens[end]) <= l.ctxSize {
			budget += len(tokens[end])
			end++
		}
		vecs, err := lctx.Embed(ctx, tokens[start:end])
		if err != nil {
			return nil, l.closedOr(fmt.Errorf("local inference: %w", err))
		}
		for i, v := range vecs {
			if err := normalize(v, l.dims); err != nil {
				return nil, fmt.Errorf("text %d: %w", start+i, err)
			}
		}
		out = append(out, vecs...)
		start = end
	}
	return out, nil
}

// closedOr maps native use-after-close errors to ErrLocalClosed.
func (l *Local) closedOr(err error) error {
	if errors.Is(err, llama.ErrClosed) {
		return ErrLocalClosed
	}
	return err
}

// normalize validates and L2-normalizes v in place.
func normalize(v []float32, dims int) error {
	if len(v) != dims {
		return fmt.Errorf("embedding has %d dimensions, want %d", len(v), dims)
	}
	var sum float64
	for _, x := range v {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return errors.New("embedding contains non-finite values")
		}
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return errors.New("embedding is all zeros")
	}
	inv := 1 / math.Sqrt(sum)
	for i := range v {
		v[i] = float32(float64(v[i]) * inv)
	}
	return nil
}
