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

// Package llama is a minimal cgo bridge to the vendored llama.cpp library
// (see csrc/ and update-llama.sh). It exposes only what in-process embedding
// needs: loading a GGUF model, tokenizing text, and decoding batches of token
// sequences into pooled embeddings.
//
// A Model may be shared by several Contexts. A Context serializes its own
// calls. Close releases native memory; using a closed value returns an error.
package llama

// #cgo CPPFLAGS: -I${SRCDIR}/csrc/include -I${SRCDIR}/csrc/ggml/include
// #cgo CFLAGS: -std=gnu11 -O2
// #include <stdlib.h>
// #include "bridge.h"
import "C"

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	_ "github.com/ory/lumen/internal/llama/internal/ggmlbase"    // ggml core
	_ "github.com/ory/lumen/internal/llama/internal/ggmlcpu"     // CPU backend
	_ "github.com/ory/lumen/internal/llama/internal/llamacore"   // llama runtime
	_ "github.com/ory/lumen/internal/llama/internal/llamamodels" // model graphs
)

// Pooling selects how token states are reduced to one sequence embedding.
type Pooling int32

// Pooling types, matching enum llama_pooling_type.
const (
	PoolingMean Pooling = C.LLAMA_POOLING_TYPE_MEAN
	PoolingCLS  Pooling = C.LLAMA_POOLING_TYPE_CLS
	PoolingLast Pooling = C.LLAMA_POOLING_TYPE_LAST
)

// ErrClosed is returned when a Model or Context is used after Close.
var ErrClosed = errors.New("llama: use of closed model or context")

// ErrUnsupportedCPU is returned when the CPU lacks the instruction set the
// bundled x86-64 kernels were compiled for.
var ErrUnsupportedCPU = errors.New("llama: local embeddings require an x86-64 CPU with AVX2, FMA, and BMI2 (x86-64-v3)")

var initOnce sync.Once

func initBackend() {
	initOnce.Do(func() { C.lumen_llama_init() })
}

// cancelFlag is native memory polled by llama.cpp callbacks. Go writes it
// atomically; C reads it as a volatile int32.
type cancelFlag struct{ p *C.int32_t }

func newCancelFlag() cancelFlag {
	p := (*C.int32_t)(C.calloc(1, C.size_t(unsafe.Sizeof(C.int32_t(0)))))
	return cancelFlag{p: p}
}

func (f cancelFlag) set(v int32) { atomic.StoreInt32((*int32)(unsafe.Pointer(f.p)), v) }
func (f cancelFlag) free()       { C.free(unsafe.Pointer(f.p)) }

// watch sets the flag when ctx is cancelled until the returned stop is called.
func (f cancelFlag) watch(ctx context.Context) (stop func()) {
	f.set(0)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		select {
		case <-ctx.Done():
			f.set(1)
		case <-done:
		}
	})
	return func() {
		close(done)
		wg.Wait()
	}
}

// Model is a loaded GGUF model.
type Model struct {
	mu      sync.RWMutex
	ptr     *C.struct_llama_model
	vocab   *C.struct_llama_vocab
	cpuOnly bool
}

// LoadModel loads the GGUF file at path. Weights are memory-mapped. When
// cpuOnly is false, layers are offloaded to an available accelerator (Metal on
// Apple Silicon); otherwise inference stays on the CPU. Cancelling ctx aborts
// loading.
func LoadModel(ctx context.Context, path string, cpuOnly bool) (*Model, error) {
	if err := Supported(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	initBackend()

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	flag := newCancelFlag()
	defer flag.free()
	stop := flag.watch(ctx)
	ptr := C.lumen_llama_load_model(cpath, boolInt(cpuOnly), flag.p)
	stop()
	if ptr == nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("llama: failed to load model %s", path)
	}
	return &Model{ptr: ptr, vocab: C.llama_model_get_vocab(ptr), cpuOnly: cpuOnly}, nil
}

// Close frees the model. Contexts created from it must be closed first.
func (m *Model) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ptr != nil {
		C.llama_model_free(m.ptr)
		m.ptr = nil
		m.vocab = nil
	}
	return nil
}

// EmbeddingSize returns the width of the model's output embeddings.
func (m *Model) EmbeddingSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ptr == nil {
		return 0
	}
	return int(C.llama_model_n_embd_out(m.ptr))
}

// TrainContextSize returns the context length the model was trained with.
func (m *Model) TrainContextSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ptr == nil {
		return 0
	}
	return int(C.llama_model_n_ctx_train(m.ptr))
}

// Meta returns a GGUF metadata value by key.
func (m *Model) Meta(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ptr == nil {
		return "", false
	}
	ckey := C.CString(key)
	defer C.free(unsafe.Pointer(ckey))
	buf := make([]byte, 256)
	for {
		n := C.llama_model_meta_val_str(m.ptr, ckey, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
		if n < 0 {
			return "", false
		}
		if int(n) < len(buf) {
			return string(buf[:n]), true
		}
		buf = make([]byte, int(n)+1)
	}
}

// Tokenize converts text to token IDs the way the model's reference tokenizer
// does: special tokens configured by the GGUF (BOS/EOS) are added and special
// token text in the input is recognized.
func (m *Model) Tokenize(text string) ([]int32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ptr == nil {
		return nil, ErrClosed
	}
	var ctext *C.char
	if len(text) > 0 {
		ctext = (*C.char)(unsafe.Pointer(unsafe.StringData(text)))
	}
	if int64(len(text)) > int64(^uint32(0)>>1)-8 {
		return nil, fmt.Errorf("llama: text too long to tokenize (%d bytes)", len(text))
	}
	tokens := make([]int32, len(text)+8)
	for {
		n := C.llama_tokenize(m.vocab, ctext, C.int32_t(len(text)),
			(*C.llama_token)(unsafe.Pointer(&tokens[0])), C.int32_t(len(tokens)), true, true)
		if n >= 0 {
			return tokens[:n], nil
		}
		if int(-n) <= len(tokens) {
			return nil, fmt.Errorf("llama: tokenization failed")
		}
		tokens = make([]int32, -n)
	}
}

// ContextParams configures an embedding context.
type ContextParams struct {
	// ContextSize is the maximum number of tokens decoded in one Embed call,
	// across all sequences.
	ContextSize int
	// MaxSequences is the maximum number of sequences per Embed call.
	MaxSequences int
	// Threads is the number of CPU threads used for computation.
	Threads int
	Pooling Pooling
}

// Context is an embedding inference context bound to a Model.
type Context struct {
	// busy holds a token while the context is in use. Unlike a mutex, a
	// queued Embed can stop waiting when its context is cancelled.
	busy   chan struct{}
	model  *Model
	ptr    *C.struct_llama_context
	cancel cancelFlag
	params ContextParams
	nEmbd  int
}

// NewContext creates an embedding context.
func (m *Model) NewContext(p ContextParams) (*Context, error) {
	if p.ContextSize <= 0 || p.MaxSequences <= 0 || p.Threads <= 0 {
		return nil, fmt.Errorf("llama: invalid context params %+v", p)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ptr == nil {
		return nil, ErrClosed
	}
	flag := newCancelFlag()
	ptr := C.lumen_llama_new_context(m.ptr, C.int32_t(p.ContextSize), C.int32_t(p.MaxSequences),
		C.int32_t(p.Threads), C.int32_t(p.Pooling), boolInt(m.cpuOnly), flag.p)
	if ptr == nil {
		flag.free()
		return nil, fmt.Errorf("llama: failed to create context (n_ctx=%d, n_seq_max=%d)", p.ContextSize, p.MaxSequences)
	}
	return &Context{
		busy:   make(chan struct{}, 1),
		model:  m,
		ptr:    ptr,
		cancel: flag,
		params: p,
		nEmbd:  int(C.llama_model_n_embd_out(m.ptr)),
	}, nil
}

// Close frees the context.
func (c *Context) Close() error {
	c.busy <- struct{}{}
	defer func() { <-c.busy }()
	if c.ptr != nil {
		C.llama_free(c.ptr)
		c.ptr = nil
		c.cancel.free()
	}
	return nil
}

// Embed decodes the token sequences as one batch and returns one pooled,
// unnormalized embedding per sequence in input order. The total token count
// must not exceed ContextSize and the number of sequences must not exceed
// MaxSequences. Cancelling ctx aborts CPU computation between graph nodes.
func (c *Context) Embed(ctx context.Context, seqs [][]int32) ([][]float32, error) {
	if len(seqs) == 0 {
		return nil, nil
	}
	if len(seqs) > c.params.MaxSequences {
		return nil, fmt.Errorf("llama: %d sequences exceed the context limit of %d", len(seqs), c.params.MaxSequences)
	}
	total := 0
	lens := make([]int32, len(seqs))
	for i, s := range seqs {
		if len(s) == 0 {
			return nil, fmt.Errorf("llama: sequence %d is empty", i)
		}
		lens[i] = int32(len(s))
		total += len(s)
	}
	if total > c.params.ContextSize {
		return nil, fmt.Errorf("llama: %d tokens exceed the context size of %d", total, c.params.ContextSize)
	}
	flat := make([]int32, 0, total)
	for _, s := range seqs {
		flat = append(flat, s...)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case c.busy <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-c.busy }()
	if c.ptr == nil {
		return nil, ErrClosed
	}

	out := make([]float32, len(seqs)*c.nEmbd)
	stop := c.cancel.watch(ctx)
	rc := C.lumen_llama_embed(c.ptr,
		(*C.int32_t)(unsafe.Pointer(&flat[0])), (*C.int32_t)(unsafe.Pointer(&lens[0])), C.int32_t(len(seqs)),
		(*C.float)(unsafe.Pointer(&out[0])), C.int32_t(c.nEmbd))
	stop()
	switch {
	case rc == 0:
	case rc == 2 && ctx.Err() != nil:
		return nil, ctx.Err()
	case rc <= -1000:
		return nil, fmt.Errorf("llama: no embedding produced for sequence %d", -1000-int(rc))
	default:
		return nil, fmt.Errorf("llama: decode failed with status %d", int(rc))
	}

	res := make([][]float32, len(seqs))
	for i := range res {
		res[i] = out[i*c.nEmbd : (i+1)*c.nEmbd : (i+1)*c.nEmbd]
	}
	return res, nil
}

func boolInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}
