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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func makeLMStudioResponse(embeddings [][]float32) lmstudioEmbedResponse {
	data := make([]lmstudioEmbedItem, len(embeddings))
	for i, e := range embeddings {
		data[i] = lmstudioEmbedItem{Embedding: e, Index: i}
	}
	return lmstudioEmbedResponse{Data: data}
}

func TestLMStudioEmbedder_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := makeLMStudioResponse([][]float32{
			{0.1, 0.2, 0.3, 0.4},
			{0.5, 0.6, 0.7, 0.8},
		})
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e, err := NewLMStudio("nomic-embed-code", 4, server.URL)
	if err != nil {
		t.Fatal(err)
	}

	vecs, err := e.Embed(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
	if len(vecs[0]) != 4 {
		t.Fatalf("expected 4 dimensions, got %d", len(vecs[0]))
	}
}

func TestLMStudioEmbedder_OrderingByIndex(t *testing.T) {
	// Mock returns items in reversed index order to verify sorting.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := lmstudioEmbedResponse{
			Data: []lmstudioEmbedItem{
				{Embedding: []float32{0.9, 0.9, 0.9, 0.9}, Index: 1},
				{Embedding: []float32{0.1, 0.2, 0.3, 0.4}, Index: 0},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e, _ := NewLMStudio("nomic-embed-code", 4, server.URL)
	vecs, err := e.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
	// vecs[0] should correspond to index:0, which has embedding {0.1, 0.2, 0.3, 0.4}
	if vecs[0][0] != 0.1 {
		t.Fatalf("expected vecs[0][0]=0.1 (index:0 item), got %v", vecs[0][0])
	}
	// vecs[1] should correspond to index:1, which has embedding {0.9, 0.9, 0.9, 0.9}
	if vecs[1][0] != 0.9 {
		t.Fatalf("expected vecs[1][0]=0.9 (index:1 item), got %v", vecs[1][0])
	}
}

func TestLMStudioEmbedder_Batching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		input := req["input"].([]any)

		embeddings := make([][]float32, len(input))
		for i := range input {
			embeddings[i] = []float32{0.1, 0.2, 0.3, 0.4}
		}
		_ = json.NewEncoder(w).Encode(makeLMStudioResponse(embeddings))
	}))
	defer server.Close()

	e, _ := NewLMStudio("nomic-embed-code", 4, server.URL)
	texts := make([]string, 50)
	for i := range texts {
		texts[i] = "text"
	}

	vecs, err := e.Embed(context.Background(), texts)
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 50 {
		t.Fatalf("expected 50 vectors, got %d", len(vecs))
	}
	if callCount != 2 {
		t.Fatalf("expected 2 batch calls (32+18), got %d", callCount)
	}
}

func TestLMStudioEmbedder_Dimensions(t *testing.T) {
	e, _ := NewLMStudio("nomic-embed-code", 768, "http://localhost:1234")
	if e.Dimensions() != 768 {
		t.Fatalf("expected 768, got %d", e.Dimensions())
	}
}

func TestLMStudioEmbedder_ModelName(t *testing.T) {
	e, _ := NewLMStudio("nomic-embed-code", 768, "http://localhost:1234")
	if e.ModelName() != "nomic-embed-code" {
		t.Fatalf("expected nomic-embed-code, got %s", e.ModelName())
	}
}

func TestLMStudioEmbedder_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	e, _ := NewLMStudio("nomic-embed-code", 4, server.URL)
	_, err := e.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestLMStudio_Embed_ContextCancelledStopsRetry(t *testing.T) {
	// Server always returns 500 to force retry attempts.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	emb, _ := NewLMStudio("nomic-embed-code", 4, srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before any request

	start := time.Now()
	_, err := emb.Embed(ctx, []string{"hello"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected fast failure on pre-cancelled context, took %v", elapsed)
	}
}
