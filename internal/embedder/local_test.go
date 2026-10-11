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
	"math"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ory/lumen/internal/llama"
	"github.com/ory/lumen/internal/llama/llamatest"
	"github.com/ory/lumen/internal/models"
)

var tinySpec = models.LocalModelSpec{Architecture: "qwen2", Pooling: "last", QueryInstruction: "query: "}

func tinyModel(t *testing.T) string {
	t.Helper()
	if err := llama.Supported(); err != nil {
		t.Skipf("native inference unavailable: %v", err)
	}
	path := filepath.Join(t.TempDir(), "tiny.gguf")
	if err := llamatest.WriteTinyQwen2(path, 7); err != nil {
		t.Fatal(err)
	}
	return path
}

func newTinyLocal(t *testing.T, path string) (*Local, *atomic.Int32) {
	t.Helper()
	var resolves atomic.Int32
	l := newLocal("tiny@test", tinySpec, llamatest.TinyEmbeddingSize, func(context.Context, DownloadProgress) (string, error) {
		resolves.Add(1)
		return path, nil
	})
	t.Cleanup(func() { _ = l.Close() })
	return l, &resolves
}

func dot(a, b []float32) float64 {
	var s float64
	for i := range a {
		s += float64(a[i]) * float64(b[i])
	}
	return s
}

func TestLocal_EmbedReturnsNormalizedVectorsInOrder(t *testing.T) {
	l, resolves := newTinyLocal(t, tinyModel(t))
	if resolves.Load() != 0 {
		t.Fatal("model resolved before first use")
	}

	texts := make([]string, 40) // more than one native batch
	for i := range texts {
		texts[i] = strings.Repeat("token ", i%7+1) + string(rune('a'+i%26))
	}
	vecs, err := l.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != len(texts) {
		t.Fatalf("got %d vectors for %d texts", len(vecs), len(texts))
	}
	for i, v := range vecs {
		if len(v) != llamatest.TinyEmbeddingSize {
			t.Fatalf("vector %d has %d dims", i, len(v))
		}
		if n := math.Sqrt(dot(v, v)); math.Abs(n-1) > 1e-4 {
			t.Errorf("vector %d norm = %f, want 1", i, n)
		}
		single, err := l.Embed(context.Background(), texts[i:i+1])
		if err != nil {
			t.Fatal(err)
		}
		if cos := dot(single[0], v); cos < 0.9999 {
			t.Errorf("text %d: batched result does not correspond to its input (cosine %f)", i, cos)
		}
	}
	if n := resolves.Load(); n != 1 {
		t.Errorf("model resolved %d times, want once for the embedder lifetime", n)
	}
}

func TestLocal_EmbedQueryAddsInstructionOnce(t *testing.T) {
	l, _ := newTinyLocal(t, tinyModel(t))
	ctx := context.Background()

	plain, err := l.Embed(ctx, []string{"find the parser"})
	if err != nil {
		t.Fatal(err)
	}
	prefixed, err := l.Embed(ctx, []string{"query: find the parser"})
	if err != nil {
		t.Fatal(err)
	}
	q, err := l.EmbedQuery(ctx, "find the parser")
	if err != nil {
		t.Fatal(err)
	}
	again, err := l.EmbedQuery(ctx, "query: find the parser")
	if err != nil {
		t.Fatal(err)
	}
	if dot(q, prefixed[0]) < 0.9999 {
		t.Error("EmbedQuery did not apply the query instruction")
	}
	if dot(q, plain[0]) > 0.9999 {
		t.Error("query instruction had no effect; Embed must not add it")
	}
	if dot(again, q) < 0.9999 {
		t.Error("EmbedQuery added the instruction twice")
	}
	if got, err := EmbedQuery(ctx, l, "find the parser"); err != nil || dot(got, q) < 0.9999 {
		t.Errorf("EmbedQuery helper did not use the local query encoding: %v", err)
	}
}

func TestLocal_TruncatesLongTexts(t *testing.T) {
	l, _ := newTinyLocal(t, tinyModel(t))
	l.ctxSize = 32
	long := strings.Repeat("abcdefgh", 20) // 160 byte-level tokens
	vecs, err := l.Embed(context.Background(), []string{long, "short"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	truncated, err := l.Embed(context.Background(), []string{long[:32]})
	if err != nil {
		t.Fatal(err)
	}
	if dot(vecs[0], truncated[0]) < 0.9999 {
		t.Error("long text was not truncated to the context size")
	}
}

func TestLocal_RejectsIncompatibleModelFile(t *testing.T) {
	path := tinyModel(t)
	for want, l := range map[string]*Local{
		"64-dimensional": newLocal("x", tinySpec, 3584, func(context.Context, DownloadProgress) (string, error) { return path, nil }),
		"architecture": newLocal("x", models.LocalModelSpec{Architecture: "bert", Pooling: "last"}, llamatest.TinyEmbeddingSize,
			func(context.Context, DownloadProgress) (string, error) { return path, nil }),
	} {
		_, err := l.Embed(context.Background(), []string{"x"})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Embed error = %v, want mention of %q", err, want)
		}
		_ = l.Close()
	}

	notGGUF := filepath.Join(t.TempDir(), "model.gguf")
	l := newLocal("x", tinySpec, llamatest.TinyEmbeddingSize, func(context.Context, DownloadProgress) (string, error) { return notGGUF, nil })
	defer func() { _ = l.Close() }()
	if _, err := l.Embed(context.Background(), []string{"x"}); err == nil {
		t.Error("Embed succeeded with a missing model file")
	}
}

func TestLocal_RejectsArtifactThatNoLongerMatchesIdentity(t *testing.T) {
	path := tinyModel(t)
	l, _ := newTinyLocal(t, path)
	l.identify = func() string { return "tiny@replaced" }
	if _, err := l.Embed(context.Background(), []string{"x"}); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("Embed error = %v, want model file changed", err)
	}
}

func TestLocal_ResolveFailureIsRetried(t *testing.T) {
	path := tinyModel(t)
	var calls atomic.Int32
	l := newLocal("x", tinySpec, llamatest.TinyEmbeddingSize, func(context.Context, DownloadProgress) (string, error) {
		if calls.Add(1) == 1 {
			return "", errors.New("network unreachable")
		}
		return path, nil
	})
	defer func() { _ = l.Close() }()
	if _, err := l.Embed(context.Background(), []string{"x"}); err == nil || !strings.Contains(err.Error(), "network unreachable") {
		t.Fatalf("first Embed error = %v", err)
	}
	if _, err := l.Embed(context.Background(), []string{"x"}); err != nil {
		t.Fatalf("second Embed did not retry: %v", err)
	}
}

func TestLocal_CallerTimeoutDoesNotAbortLoad(t *testing.T) {
	path := tinyModel(t)
	release := make(chan struct{})
	l := newLocal("x", tinySpec, llamatest.TinyEmbeddingSize, func(ctx context.Context, _ DownloadProgress) (string, error) {
		select {
		case <-release:
			return path, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	})
	defer func() { _ = l.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := l.Embed(ctx, []string{"x"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Embed error = %v, want deadline exceeded while loading", err)
	}
	close(release)
	if _, err := l.Embed(context.Background(), []string{"x"}); err != nil {
		t.Fatalf("Embed after load completed: %v", err)
	}
}

func TestLocal_CloseCancelsLoadAndRejectsUse(t *testing.T) {
	started := make(chan struct{})
	l := newLocal("x", tinySpec, llamatest.TinyEmbeddingSize, func(ctx context.Context, _ DownloadProgress) (string, error) {
		close(started)
		<-ctx.Done() // a download that only stops when cancelled
		return "", ctx.Err()
	})
	errc := make(chan error, 1)
	go func() {
		_, err := l.Embed(context.Background(), []string{"x"})
		errc <- err
	}()
	<-started
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-errc; !errors.Is(err, ErrLocalClosed) {
		t.Errorf("Embed during Close = %v, want ErrLocalClosed", err)
	}
	if _, err := l.Embed(context.Background(), []string{"x"}); !errors.Is(err, ErrLocalClosed) {
		t.Errorf("Embed after Close = %v, want ErrLocalClosed", err)
	}
	if err := l.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestLocal_ConcurrentEmbedAndClose(t *testing.T) {
	l, _ := newTinyLocal(t, tinyModel(t))
	if _, err := l.Embed(context.Background(), []string{"warm up"}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			_, err := l.Embed(context.Background(), []string{strings.Repeat("x", i+1)})
			if err != nil && !errors.Is(err, ErrLocalClosed) {
				t.Errorf("Embed: %v", err)
			}
		})
	}
	_ = l.Close()
	wg.Wait()
}

func TestLocal_EmptyInputs(t *testing.T) {
	l, resolves := newTinyLocal(t, tinyModel(t))
	if vecs, err := l.Embed(context.Background(), nil); err != nil || vecs != nil {
		t.Errorf("Embed(nil) = %v, %v", vecs, err)
	}
	if resolves.Load() != 0 {
		t.Error("empty input loaded the model")
	}
	if _, err := l.Embed(context.Background(), []string{""}); err == nil {
		t.Error("Embed of empty text succeeded without tokens")
	}
}

func TestNormalize(t *testing.T) {
	v := []float32{3, 4}
	if err := normalize(v, 2); err != nil || v[0] != 0.6 || v[1] != 0.8 {
		t.Errorf("normalize = %v, %v", v, err)
	}
	for name, bad := range map[string][]float32{
		"dims":     {1},
		"nan":      {float32(math.NaN()), 1},
		"infinite": {float32(math.Inf(1)), 1},
		"zero":     {0, 0},
	} {
		if err := normalize(bad, 2); err == nil {
			t.Errorf("%s: normalize accepted invalid embedding", name)
		}
	}
}
