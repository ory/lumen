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

// Package llamatest writes tiny, deterministic GGUF models so the native
// inference path can be exercised on every platform without downloading a
// real model. The weights are random; only the mechanics are meaningful.
package llamatest

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
)

// TinyEmbeddingSize is the hidden width of models written by WriteTinyQwen2.
const TinyEmbeddingSize = 64

// WriteTinyQwen2 writes a two-layer Qwen2-architecture model with a byte-level
// BPE vocabulary to path. The same seed always yields the same file.
func WriteTinyQwen2(path string, seed uint64) error {
	const (
		nEmbd   = TinyEmbeddingSize
		nFF     = 128
		nLayer  = 2
		nHead   = 4
		nHeadKV = 2
		headDim = nEmbd / nHead
	)
	tokens, types := byteVocab()
	nVocab := len(tokens)

	w := &writer{}
	w.kvString("general.architecture", "qwen2")
	w.kvString("general.name", "lumen-tiny-qwen2")
	w.kvUint32("qwen2.context_length", 4096)
	w.kvUint32("qwen2.embedding_length", nEmbd)
	w.kvUint32("qwen2.block_count", nLayer)
	w.kvUint32("qwen2.feed_forward_length", nFF)
	w.kvUint32("qwen2.attention.head_count", nHead)
	w.kvUint32("qwen2.attention.head_count_kv", nHeadKV)
	w.kvFloat32("qwen2.rope.freq_base", 1000000)
	w.kvFloat32("qwen2.attention.layer_norm_rms_epsilon", 1e-6)
	w.kvString("tokenizer.ggml.model", "gpt2")
	w.kvString("tokenizer.ggml.pre", "qwen2")
	w.kvStrings("tokenizer.ggml.tokens", tokens)
	w.kvInt32s("tokenizer.ggml.token_type", types)
	w.kvStrings("tokenizer.ggml.merges", []string{"Ġ t"})
	w.kvUint32("tokenizer.ggml.eos_token_id", uint32(nVocab-1))
	w.kvUint32("tokenizer.ggml.padding_token_id", uint32(nVocab-1))
	w.kvBool("tokenizer.ggml.add_bos_token", false)

	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	random := func(n int, scale float64) []float32 {
		out := make([]float32, n)
		for i := range out {
			out[i] = float32(rng.NormFloat64() * scale)
		}
		return out
	}
	ones := func(n int) []float32 {
		out := make([]float32, n)
		for i := range out {
			out[i] = 1
		}
		return out
	}

	w.tensor("token_embd.weight", []uint64{nEmbd, uint64(nVocab)}, random(nEmbd*nVocab, 0.5))
	w.tensor("output_norm.weight", []uint64{nEmbd}, ones(nEmbd))
	for l := 0; l < nLayer; l++ {
		p := fmt.Sprintf("blk.%d.", l)
		scale := 1 / math.Sqrt(nEmbd)
		w.tensor(p+"attn_norm.weight", []uint64{nEmbd}, ones(nEmbd))
		w.tensor(p+"attn_q.weight", []uint64{nEmbd, nEmbd}, random(nEmbd*nEmbd, scale))
		w.tensor(p+"attn_k.weight", []uint64{nEmbd, nHeadKV * headDim}, random(nEmbd*nHeadKV*headDim, scale))
		w.tensor(p+"attn_v.weight", []uint64{nEmbd, nHeadKV * headDim}, random(nEmbd*nHeadKV*headDim, scale))
		w.tensor(p+"attn_output.weight", []uint64{nEmbd, nEmbd}, random(nEmbd*nEmbd, scale))
		w.tensor(p+"ffn_norm.weight", []uint64{nEmbd}, ones(nEmbd))
		w.tensor(p+"ffn_gate.weight", []uint64{nEmbd, nFF}, random(nEmbd*nFF, scale))
		w.tensor(p+"ffn_up.weight", []uint64{nEmbd, nFF}, random(nEmbd*nFF, scale))
		w.tensor(p+"ffn_down.weight", []uint64{nFF, nEmbd}, random(nFF*nEmbd, 1/math.Sqrt(nFF)))
	}
	return w.writeFile(path)
}

// byteVocab returns the 256 GPT-2 byte-level tokens, one merged token, and an
// end-of-text control token.
func byteVocab() ([]string, []int32) {
	var bs []int
	for b := '!'; b <= '~'; b++ {
		bs = append(bs, int(b))
	}
	for b := '¡'; b <= '¬'; b++ {
		bs = append(bs, int(b))
	}
	for b := '®'; b <= 'ÿ'; b++ {
		bs = append(bs, int(b))
	}
	mapped := map[int]rune{}
	for _, b := range bs {
		mapped[b] = rune(b)
	}
	n := 0
	for b := 0; b < 256; b++ {
		if _, ok := mapped[b]; !ok {
			mapped[b] = rune(256 + n)
			n++
		}
	}
	tokens := make([]string, 0, 258)
	types := make([]int32, 0, 258)
	for b := 0; b < 256; b++ {
		tokens = append(tokens, string(mapped[b]))
		types = append(types, 1) // normal
	}
	tokens = append(tokens, "Ġt") // result of the single merge rule
	types = append(types, 1)
	tokens = append(tokens, "<|endoftext|>")
	types = append(types, 3) // control
	return tokens, types
}

const (
	ggufTypeUint32  = 4
	ggufTypeInt32   = 5
	ggufTypeFloat32 = 6
	ggufTypeBool    = 7
	ggufTypeString  = 8
	ggufTypeArray   = 9
	ggufAlignment   = 32
)

type tensor struct {
	name string
	dims []uint64
	data []float32
}

type writer struct {
	kv      []func(*bufio.Writer)
	tensors []tensor
}

func put(b *bufio.Writer, v any) { _ = binary.Write(b, binary.LittleEndian, v) }

func putString(b *bufio.Writer, s string) {
	put(b, uint64(len(s)))
	_, _ = b.WriteString(s)
}

func (w *writer) kvHeader(b *bufio.Writer, key string, typ uint32) {
	putString(b, key)
	put(b, typ)
}

func (w *writer) kvString(key, v string) {
	w.kv = append(w.kv, func(b *bufio.Writer) { w.kvHeader(b, key, ggufTypeString); putString(b, v) })
}

func (w *writer) kvUint32(key string, v uint32) {
	w.kv = append(w.kv, func(b *bufio.Writer) { w.kvHeader(b, key, ggufTypeUint32); put(b, v) })
}

func (w *writer) kvFloat32(key string, v float32) {
	w.kv = append(w.kv, func(b *bufio.Writer) { w.kvHeader(b, key, ggufTypeFloat32); put(b, v) })
}

func (w *writer) kvBool(key string, v bool) {
	w.kv = append(w.kv, func(b *bufio.Writer) { w.kvHeader(b, key, ggufTypeBool); put(b, v) })
}

func (w *writer) kvStrings(key string, v []string) {
	w.kv = append(w.kv, func(b *bufio.Writer) {
		w.kvHeader(b, key, ggufTypeArray)
		put(b, uint32(ggufTypeString))
		put(b, uint64(len(v)))
		for _, s := range v {
			putString(b, s)
		}
	})
}

func (w *writer) kvInt32s(key string, v []int32) {
	w.kv = append(w.kv, func(b *bufio.Writer) {
		w.kvHeader(b, key, ggufTypeArray)
		put(b, uint32(ggufTypeInt32))
		put(b, uint64(len(v)))
		put(b, v)
	})
}

func (w *writer) tensor(name string, dims []uint64, data []float32) {
	w.tensors = append(w.tensors, tensor{name: name, dims: dims, data: data})
}

func padding(n uint64) uint64 { return (ggufAlignment - n%ggufAlignment) % ggufAlignment }

func (w *writer) writeFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	b := bufio.NewWriter(f)
	cw := &countingWriter{w: b}
	bw := bufio.NewWriterSize(cw, 1)

	_, _ = bw.WriteString("GGUF")
	put(bw, uint32(3))
	put(bw, uint64(len(w.tensors)))
	put(bw, uint64(len(w.kv)))
	for _, kv := range w.kv {
		kv(bw)
	}
	var offset uint64
	for _, t := range w.tensors {
		putString(bw, t.name)
		put(bw, uint32(len(t.dims)))
		for _, d := range t.dims {
			put(bw, d)
		}
		put(bw, uint32(0)) // GGML_TYPE_F32
		put(bw, offset)
		size := uint64(len(t.data) * 4)
		offset += size + padding(size)
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	_, _ = b.Write(make([]byte, padding(cw.n)))
	for _, t := range w.tensors {
		put(b, t.data)
		_, _ = b.Write(make([]byte, padding(uint64(len(t.data)*4))))
	}
	if err := b.Flush(); err != nil {
		return err
	}
	return f.Close()
}

type countingWriter struct {
	w *bufio.Writer
	n uint64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += uint64(len(p))
	return c.w.Write(p)
}
