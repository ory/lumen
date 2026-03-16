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
	"sync/atomic"
	"testing"
	"time"
)

// makeOpenAIResponse builds a realistic OpenAI /v1/embeddings response.
func makeOpenAIResponse(model string, embeddings [][]float32) map[string]any {
	data := make([]map[string]any, len(embeddings))
	for i, e := range embeddings {
		data[i] = map[string]any{
			"object":    "embedding",
			"embedding": e,
			"index":     i,
		}
	}
	return map[string]any{
		"object": "list",
		"data":   data,
		"model":  model,
		"usage": map[string]any{
			"prompt_tokens": 8,
			"total_tokens":  8,
		},
	}
}

func TestOpenAIEmbedder_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %s", auth)
		}
		resp := makeOpenAIResponse("text-embedding-3-small", [][]float32{
			{0.0023064255, -0.009327292, 0.015834473, 0.0069007568},
			{-0.0069352793, 0.020878976, 0.008590698, -0.012878418},
		})
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e, err := NewOpenAI("text-embedding-3-small", 4, server.URL, "test-key")
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

func TestOpenAIEmbedder_Dimensions(t *testing.T) {
	e, _ := NewOpenAI("text-embedding-3-small", 1536, "https://api.openai.com", "test-key")
	if e.Dimensions() != 1536 {
		t.Fatalf("expected 1536, got %d", e.Dimensions())
	}
}

func TestOpenAIEmbedder_ModelName(t *testing.T) {
	e, _ := NewOpenAI("text-embedding-3-small", 1536, "https://api.openai.com", "test-key")
	if e.ModelName() != "text-embedding-3-small" {
		t.Fatalf("expected text-embedding-3-small, got %s", e.ModelName())
	}
}

func TestOpenAIEmbedder_Batching(t *testing.T) {
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
		_ = json.NewEncoder(w).Encode(makeOpenAIResponse("text-embedding-3-small", embeddings))
	}))
	defer server.Close()

	e, _ := NewOpenAI("text-embedding-3-small", 4, server.URL, "test-key")
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

func TestOpenAIEmbedder_OrderingByIndex(t *testing.T) {
	// Mock returns items in reversed index order to verify sorting.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"object": "embedding", "embedding": []float32{0.9, 0.9, 0.9, 0.9}, "index": 1},
				{"object": "embedding", "embedding": []float32{0.1, 0.2, 0.3, 0.4}, "index": 0},
			},
			"model": "text-embedding-3-small",
			"usage": map[string]any{"prompt_tokens": 4, "total_tokens": 4},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e, _ := NewOpenAI("text-embedding-3-small", 4, server.URL, "test-key")
	vecs, err := e.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
	if vecs[0][0] != 0.1 {
		t.Fatalf("expected vecs[0][0]=0.1 (index:0 item), got %v", vecs[0][0])
	}
	if vecs[1][0] != 0.9 {
		t.Fatalf("expected vecs[1][0]=0.9 (index:1 item), got %v", vecs[1][0])
	}
}

func TestOpenAIEmbedder_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	e, _ := NewOpenAI("text-embedding-3-small", 4, server.URL, "test-key")
	_, err := e.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestOpenAIEmbedder_RateLimitRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		resp := makeOpenAIResponse("text-embedding-3-small", [][]float32{
			{0.1, 0.2, 0.3, 0.4},
		})
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e, _ := NewOpenAI("text-embedding-3-small", 4, server.URL, "test-key")
	vecs, err := e.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if len(vecs) != 1 {
		t.Fatalf("expected 1 vector, got %d", len(vecs))
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("expected 2 calls (1 rate-limited + 1 success), got %d", got)
	}
}

func TestOpenAI_Embed_ContextCancelledStopsRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	emb, _ := NewOpenAI("text-embedding-3-small", 4, srv.URL, "test-key")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

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

func TestOpenAIEmbedder_EmptyAPIKey(t *testing.T) {
	_, err := NewOpenAI("text-embedding-3-small", 1536, "https://api.openai.com", "")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}
