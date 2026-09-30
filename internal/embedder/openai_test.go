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
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func makeOpenAIResponse(embeddings [][]float32) openaiEmbedResponse {
	data := make([]openaiEmbedItem, len(embeddings))
	for i, e := range embeddings {
		data[i] = openaiEmbedItem{Embedding: e, Index: i}
	}
	return openaiEmbedResponse{Data: data}
}

func TestOpenAIEmbedder_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := makeOpenAIResponse([][]float32{
			{0.1, 0.2, 0.3, 0.4},
			{0.5, 0.6, 0.7, 0.8},
		})
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e, err := NewOpenAI("text-embedding-3-small", 4, server.URL, "")
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

func TestOpenAIEmbedder_OrderingByIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := openaiEmbedResponse{
			Data: []openaiEmbedItem{
				{Embedding: []float32{0.9, 0.9, 0.9, 0.9}, Index: 1},
				{Embedding: []float32{0.1, 0.2, 0.3, 0.4}, Index: 0},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	e, _ := NewOpenAI("text-embedding-3-small", 4, server.URL, "")
	vecs, err := e.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if vecs[0][0] != 0.1 {
		t.Fatalf("expected vecs[0][0]=0.1 (index:0 item), got %v", vecs[0][0])
	}
	if vecs[1][0] != 0.9 {
		t.Fatalf("expected vecs[1][0]=0.9 (index:1 item), got %v", vecs[1][0])
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
		_ = json.NewEncoder(w).Encode(makeOpenAIResponse(embeddings))
	}))
	defer server.Close()

	e, _ := NewOpenAI("text-embedding-3-small", 4, server.URL, "")
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

func TestOpenAIEmbedder_Dimensions(t *testing.T) {
	e, _ := NewOpenAI("text-embedding-3-small", 768, "http://localhost:8080", "")
	if e.Dimensions() != 768 {
		t.Fatalf("expected 768, got %d", e.Dimensions())
	}
}

func TestOpenAIEmbedder_ModelName(t *testing.T) {
	e, _ := NewOpenAI("text-embedding-3-small", 768, "http://localhost:8080", "")
	if e.ModelName() != "text-embedding-3-small" {
		t.Fatalf("expected text-embedding-3-small, got %s", e.ModelName())
	}
}

func TestOpenAIEmbedder_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	e, _ := NewOpenAI("text-embedding-3-small", 4, server.URL, "")
	_, err := e.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestOpenAI_Embed_ContextCancelledStopsRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	emb, _ := NewOpenAI("text-embedding-3-small", 4, srv.URL, "")

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

func TestOpenAIEmbedder_NoAuthHeaderWhenKeyEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		_ = json.NewEncoder(w).Encode(makeOpenAIResponse([][]float32{{0.1, 0.2}}))
	}))
	defer server.Close()

	e, err := NewOpenAI("m", 2, server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Embed(context.Background(), []string{"hello"}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIEmbedder_BearerHeaderWhenKeySet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer sk-test-key" {
			t.Errorf("expected Bearer sk-test-key, got %q", auth)
		}
		_ = json.NewEncoder(w).Encode(makeOpenAIResponse([][]float32{{0.1, 0.2}}))
	}))
	defer server.Close()

	e, err := NewOpenAI("m", 2, server.URL, "sk-test-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Embed(context.Background(), []string{"hello"}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIEmbedder_TrailingV1InBaseURLNotDoubled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("expected path /v1/embeddings, got %q (base URL's /v1 must not be duplicated)", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(makeOpenAIResponse([][]float32{{0.1, 0.2}}))
	}))
	defer server.Close()

	e, err := NewOpenAI("m", 2, server.URL+"/v1", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Embed(context.Background(), []string{"hello"}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIEmbedder_RetriesOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(makeOpenAIResponse([][]float32{{0.1, 0.2}}))
	}))
	defer server.Close()

	e, _ := NewOpenAI("m", 2, server.URL, "")
	vecs, err := e.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("expected retry to succeed after 429, got: %v", err)
	}
	if len(vecs) != 1 {
		t.Fatalf("expected 1 vector, got %d", len(vecs))
	}
	if attempts < 2 {
		t.Fatalf("expected at least 2 attempts, got %d", attempts)
	}
}

// TestOpenAIEmbedder_AuthenticatedRedirects guards the bearer token against
// redirect-based downgrades: net/http forwards Authorization on same-host
// redirects regardless of scheme, so an https→http hop would leak the key in
// plaintext despite config validation requiring an https host.
func TestOpenAIEmbedder_AuthenticatedRedirects(t *testing.T) {
	var plainHits atomic.Int32
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		plainHits.Add(1)
		_ = json.NewEncoder(w).Encode(makeOpenAIResponse([][]float32{{0.1, 0.2, 0.3}}))
	}))
	defer plain.Close()

	var target atomic.Value
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
				t.Errorf("Authorization = %q, want Bearer sk-test", got)
			}
			_ = json.NewEncoder(w).Encode(makeOpenAIResponse([][]float32{{0.1, 0.2, 0.3}}))
			return
		}
		http.Redirect(w, r, target.Load().(string), http.StatusTemporaryRedirect)
	}))
	defer secure.Close()

	tests := []struct {
		name    string
		target  string
		wantErr string
	}{
		{name: "https to http downgrade is refused", target: plain.URL + "/v1/embeddings", wantErr: "refusing redirect"},
		{name: "https to https is followed", target: secure.URL + "/final"},
		{name: "redirect limit is retained", target: secure.URL + "/v1/embeddings", wantErr: "stopped after 10 redirects"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target.Store(tt.target)
			plainHits.Store(0)

			emb, err := NewOpenAI("test-model", 3, secure.URL, "sk-test")
			if err != nil {
				t.Fatalf("NewOpenAI: %v", err)
			}
			emb.client.Transport = secure.Client().Transport

			_, err = emb.Embed(context.Background(), []string{"hello"})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Embed: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Embed error = %v, want containing %q", err, tt.wantErr)
			}
			if n := plainHits.Load(); n != 0 {
				t.Errorf("plaintext destination contacted %d times, want 0", n)
			}
		})
	}
}
