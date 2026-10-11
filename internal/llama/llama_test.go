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

package llama

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ory/lumen/internal/llama/llamatest"
)

func tinyModelPath(t *testing.T) string {
	t.Helper()
	skipUnsupported(t)
	path := filepath.Join(t.TempDir(), "tiny.gguf")
	if err := llamatest.WriteTinyQwen2(path, 1); err != nil {
		t.Fatal(err)
	}
	return path
}

// skipUnsupported skips native inference on CPUs the bundled kernels do not
// target (e.g. x86 emulators without AVX2); TestSupported covers the error.
func skipUnsupported(t *testing.T) {
	t.Helper()
	if err := Supported(); err != nil {
		t.Skipf("native inference unavailable: %v", err)
	}
}

func loadTiny(t *testing.T, cpuOnly bool) (*Model, *Context) {
	t.Helper()
	m, err := LoadModel(context.Background(), tinyModelPath(t), cpuOnly)
	if err != nil {
		t.Fatalf("LoadModel: %v", err)
	}
	c, err := m.NewContext(ContextParams{ContextSize: 256, MaxSequences: 4, Threads: 2, Pooling: PoolingLast})
	if err != nil {
		_ = m.Close()
		t.Fatalf("NewContext: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Close()
		_ = m.Close()
	})
	return m, c
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	return dot / math.Sqrt(na*nb)
}

func tokenize(t *testing.T, m *Model, texts ...string) [][]int32 {
	t.Helper()
	out := make([][]int32, len(texts))
	for i, s := range texts {
		tok, err := m.Tokenize(s)
		if err != nil {
			t.Fatalf("Tokenize(%q): %v", s, err)
		}
		out[i] = tok
	}
	return out
}

func TestTinyModel_LoadTokenizeEmbed(t *testing.T) {
	for _, cpuOnly := range []bool{true, false} {
		m, c := loadTiny(t, cpuOnly)
		if got := m.EmbeddingSize(); got != llamatest.TinyEmbeddingSize {
			t.Fatalf("EmbeddingSize = %d, want %d", got, llamatest.TinyEmbeddingSize)
		}
		if arch, ok := m.Meta("general.architecture"); !ok || arch != "qwen2" {
			t.Errorf("Meta(general.architecture) = %q, %v", arch, ok)
		}
		if _, ok := m.Meta("no.such.key"); ok {
			t.Error("Meta reported a missing key as present")
		}

		seqs := tokenize(t, m, "func main() {}", "hello world", "x")
		// Byte-level BPE: one token per byte except the "Ġt" merge, no BOS/EOS.
		if got := len(seqs[2]); got != 1 {
			t.Errorf("Tokenize(%q) = %v, want one token", "x", seqs[2])
		}
		out, err := c.Embed(context.Background(), seqs)
		if err != nil {
			t.Fatalf("Embed: %v", err)
		}
		if len(out) != len(seqs) {
			t.Fatalf("Embed returned %d vectors for %d sequences", len(out), len(seqs))
		}
		for i, v := range out {
			if len(v) != llamatest.TinyEmbeddingSize {
				t.Errorf("vector %d has %d dims", i, len(v))
			}
			for _, x := range v {
				if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
					t.Fatalf("vector %d is not finite", i)
				}
			}
		}
		if cosine(out[0], out[1]) > 0.9999 {
			t.Error("different inputs produced identical embeddings")
		}
	}
}

func TestEmbed_BatchMatchesIndividualSequences(t *testing.T) {
	m, c := loadTiny(t, true)
	seqs := tokenize(t, m, "alpha beta", "gamma", "the quick brown fox", "delta epsilon zeta")
	batched, err := c.Embed(context.Background(), seqs)
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range seqs {
		single, err := c.Embed(context.Background(), [][]int32{s})
		if err != nil {
			t.Fatal(err)
		}
		if cos := cosine(single[0], batched[i]); cos < 0.9999 {
			t.Errorf("sequence %d: batched vs single cosine = %f", i, cos)
		}
	}
}

func TestEmbed_RejectsInvalidBatches(t *testing.T) {
	m, c := loadTiny(t, true)
	one := tokenize(t, m, "abc")[0]
	long := make([]int32, 257)
	for i := range long {
		long[i] = one[0]
	}
	for name, seqs := range map[string][][]int32{
		"too many sequences": {one, one, one, one, one},
		"too many tokens":    {long},
		"empty sequence":     {one, {}},
	} {
		if _, err := c.Embed(context.Background(), seqs); err == nil {
			t.Errorf("%s: Embed succeeded", name)
		}
	}
	if out, err := c.Embed(context.Background(), nil); err != nil || out != nil {
		t.Errorf("Embed(nil) = %v, %v", out, err)
	}
}

func TestEmbed_CancelledContext(t *testing.T) {
	m, c := loadTiny(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Embed(ctx, tokenize(t, m, "abc")); !errors.Is(err, context.Canceled) {
		t.Errorf("Embed error = %v, want context.Canceled", err)
	}
}

func TestEmbed_QueuedCallerHonorsCancellation(t *testing.T) {
	m, c := loadTiny(t, true)
	seqs := tokenize(t, m, "queued")
	c.busy <- struct{}{} // simulate a long-running inference holding the context
	defer func() { <-c.busy }()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := c.Embed(ctx, seqs)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Embed error = %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("queued Embed ignored cancellation while the context was busy")
	}
}

func TestEmbed_ConcurrentCallersAreSerialized(t *testing.T) {
	m, c := loadTiny(t, true)
	seqs := tokenize(t, m, "concurrent embedding", "check")
	want, err := c.Embed(context.Background(), seqs)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			got, err := c.Embed(context.Background(), seqs)
			if err != nil {
				t.Error(err)
				return
			}
			for i := range got {
				if cos := cosine(got[i], want[i]); cos < 0.9999 {
					t.Errorf("concurrent result %d cosine = %f", i, cos)
				}
			}
		})
	}
	wg.Wait()
}

func TestClose_UseAfterCloseFails(t *testing.T) {
	m, err := LoadModel(context.Background(), tinyModelPath(t), true)
	if err != nil {
		t.Fatal(err)
	}
	c, err := m.NewContext(ContextParams{ContextSize: 64, MaxSequences: 1, Threads: 1, Pooling: PoolingLast})
	if err != nil {
		t.Fatal(err)
	}
	seqs := tokenize(t, m, "abc")
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	if _, err := c.Embed(context.Background(), seqs); !errors.Is(err, ErrClosed) {
		t.Errorf("Embed after Close = %v, want ErrClosed", err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Tokenize("abc"); !errors.Is(err, ErrClosed) {
		t.Errorf("Tokenize after Close = %v, want ErrClosed", err)
	}
	if _, err := m.NewContext(ContextParams{ContextSize: 64, MaxSequences: 1, Threads: 1}); !errors.Is(err, ErrClosed) {
		t.Errorf("NewContext after Close = %v, want ErrClosed", err)
	}
}

func TestLoadModel_Errors(t *testing.T) {
	dir := t.TempDir()
	notGGUF := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(notGGUF, []byte("not a model"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{
		"missing":  filepath.Join(dir, "missing.gguf"),
		"not gguf": notGGUF,
	} {
		if m, err := LoadModel(context.Background(), path, true); err == nil {
			_ = m.Close()
			t.Errorf("%s: LoadModel succeeded", name)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadModel(ctx, tinyModelPath(t), true); !errors.Is(err, context.Canceled) {
		t.Errorf("LoadModel with cancelled context = %v", err)
	}
}

func TestSupported_UnsupportedCPUFailsCleanly(t *testing.T) {
	if Supported() == nil {
		t.Skip("CPU supports the bundled kernels")
	}
	if _, err := LoadModel(context.Background(), "unused.gguf", true); !errors.Is(err, ErrUnsupportedCPU) {
		t.Fatalf("LoadModel on unsupported CPU = %v, want ErrUnsupportedCPU", err)
	}
}
