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
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/models"
)

// nomicReference holds embeddings produced by upstream llama.cpp's
// llama-embedding tool for the pinned nomic-embed-code Q4_K_M artifact. The
// exact command is recorded in the fixture.
type nomicReference struct {
	ModelSHA256 string `json:"model_sha256"`
	Command     string `json:"command"`
	Documents   []struct {
		Text      string    `json:"text"`
		Embedding []float32 `json:"embedding"`
	} `json:"documents"`
	Queries []struct {
		Query     string    `json:"query"`
		Embedding []float32 `json:"embedding"`
		// Relevant is the index of the document the query describes.
		Relevant int `json:"relevant"`
	} `json:"queries"`
}

// TestLocal_NomicMatchesLlamaCppReference runs real nomic-embed-code
// inference in-process and checks it against the upstream reference. It needs
// the official 4 GiB model, so it only runs when LUMEN_TEST_NOMIC_GGUF points
// to nomic-embed-code.Q4_K_M.gguf, e.g. the managed copy in
// ~/.cache/lumen/models/.
func TestLocal_NomicMatchesLlamaCppReference(t *testing.T) {
	path := os.Getenv("LUMEN_TEST_NOMIC_GGUF")
	if path == "" {
		t.Skip("set LUMEN_TEST_NOMIC_GGUF to the official nomic-embed-code.Q4_K_M.gguf to run real-model inference")
	}

	ref := loadNomicReference(t)
	if got := fileSHA256(t, path); got != ref.ModelSHA256 {
		t.Fatalf("model sha256 = %s, reference was produced with %s", got, ref.ModelSHA256)
	}
	if spec := models.LocalModels[models.DefaultLocalModel]; spec.SHA256 != ref.ModelSHA256 {
		t.Fatalf("reference artifact %s is not the managed artifact %s", ref.ModelSHA256, spec.SHA256)
	}

	srv := config.ServerConfig{Backend: config.BackendLocal, Model: models.DefaultLocalModel, ModelPath: path}
	l, err := NewLocal(srv, 3584, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	ctx := context.Background()
	texts := make([]string, len(ref.Documents))
	for i, d := range ref.Documents {
		texts[i] = d.Text
	}
	start := time.Now()
	docs, err := l.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	t.Logf("embedded %d documents in %s (including model load)", len(docs), time.Since(start))

	// Hardware and kernel differences (Metal vs CPU, SIMD width) perturb
	// floating-point results slightly; the embedding direction must not move.
	const minCosine = 0.99
	for i, d := range ref.Documents {
		if cos := dot(docs[i], d.Embedding); cos < minCosine {
			t.Errorf("document %d: cosine to reference = %.5f, want >= %.2f", i, cos, minCosine)
		} else {
			t.Logf("document %d: cosine to reference = %.5f", i, cos)
		}
	}

	for i, q := range ref.Queries {
		vec, err := l.EmbedQuery(ctx, q.Query)
		if err != nil {
			t.Fatalf("EmbedQuery: %v", err)
		}
		if cos := dot(vec, q.Embedding); cos < minCosine {
			t.Errorf("query %d: cosine to reference = %.5f, want >= %.2f", i, cos, minCosine)
		} else {
			t.Logf("query %d: cosine to reference = %.5f", i, cos)
		}
		best, bestScore, second := -1, -2.0, -2.0
		for j, d := range docs {
			s := dot(vec, d)
			switch {
			case s > bestScore:
				best, bestScore, second = j, s, bestScore
			case s > second:
				second = s
			}
		}
		if best != q.Relevant {
			t.Errorf("query %q ranked document %d first (%.3f), want %d", q.Query, best, bestScore, q.Relevant)
		}
		if bestScore-second < 0.05 {
			t.Errorf("query %q: top result margin %.3f is too small to be meaningful", q.Query, bestScore-second)
		}
	}
}

func loadNomicReference(t *testing.T) nomicReference {
	t.Helper()
	f, err := os.Open("testdata/nomic-embed-code-q4km-reference.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var ref nomicReference
	if err := json.NewDecoder(zr).Decode(&ref); err != nil {
		t.Fatal(err)
	}
	return ref
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
