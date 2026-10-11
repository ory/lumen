// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package embedder

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/flock"

	"github.com/ory/lumen/internal/config"
)

func TestBudgetHonorsQueryIntentAndCancellation(t *testing.T) {
	dir := t.TempDir()
	intent := flock.New(filepath.Join(dir, "query.lock"))
	if ok, err := intent.TryRLock(); err != nil || !ok {
		t.Fatalf("intent: %v %v", ok, err)
	}
	defer func() { _ = intent.Close() }()
	ctx, cancel := context.WithTimeout(BackgroundContext(context.Background()), 40*time.Millisecond)
	defer cancel()
	if release, err := acquireBudget(ctx, dir); err == nil {
		release()
		t.Fatal("background request jumped ahead of query")
	}
	if err := intent.Unlock(); err != nil {
		t.Fatal(err)
	}
	release, err := acquireBudget(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel2()
	if other, err := acquireBudget(ctx2, dir); err == nil {
		other()
		t.Fatal("budget admitted overlapping requests")
	}
	release()
	next, err := acquireBudget(BackgroundContext(context.Background()), dir)
	if err != nil {
		t.Fatal(err)
	}
	next()
}

// TestQueryPrecedesNextDocumentHTTPRequest proves the compute budget is admitted
// per backend HTTP request: a query that announces intent while the first
// 32-input document request is running must be served before the next one.
func TestQueryPrecedesNextDocumentHTTPRequest(t *testing.T) {
	for _, backend := range []string{config.BackendOllama, config.BackendLMStudio} {
		t.Run(backend, func(t *testing.T) {
			const query = "where is the query handled"
			arrived := make(chan struct{})
			release := make(chan struct{})
			var mu sync.Mutex
			var order []string
			var inFlight, maxInFlight int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/tags":
					_, _ = fmt.Fprint(w, `{"models":[{"name":"test"}]}`)
					return
				case "/v1/models":
					_, _ = fmt.Fprint(w, `{"data":[{"id":"test"}]}`)
					return
				}
				var body struct {
					Input []string `json:"input"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					w.WriteHeader(400)
					return
				}
				mu.Lock()
				inFlight++
				maxInFlight = max(maxInFlight, inFlight)
				label := fmt.Sprintf("doc:%d", len(body.Input))
				if len(body.Input) == 1 && body.Input[0] == query {
					label = "query"
				}
				order = append(order, label)
				first := len(order) == 1
				mu.Unlock()
				if first {
					close(arrived)
					<-release
				}
				mu.Lock()
				inFlight--
				mu.Unlock()
				vectors := make([][]float32, len(body.Input))
				for i := range vectors {
					vectors[i] = []float32{1, 0, 0}
				}
				if r.URL.Path == "/api/embed" {
					_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": vectors})
					return
				}
				data := make([]map[string]any, len(vectors))
				for i, v := range vectors {
					data[i] = map[string]any{"index": i, "embedding": v}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
			}))
			defer srv.Close()

			cfg := testConfigService(t, config.ServerConfig{Backend: backend, Host: srv.URL, Model: "test", Dims: 3})
			budgetDir := t.TempDir()
			indexer := NewFailoverEmbedder(cfg)
			indexer.SetComputeBudget(budgetDir)
			searcher := NewFailoverEmbedder(cfg)
			searcher.SetComputeBudget(budgetDir)

			docs := make([]string, 40)
			for i := range docs {
				docs[i] = fmt.Sprintf("// f.go\nfunc F%d() {}", i)
			}
			docDone := make(chan error, 1)
			go func() {
				_, err := indexer.Embed(BackgroundContext(context.Background()), docs)
				docDone <- err
			}()
			select {
			case <-arrived:
			case <-time.After(10 * time.Second):
				t.Fatal("first document request never arrived")
			}
			queryDone := make(chan error, 1)
			go func() {
				_, err := searcher.Embed(context.Background(), []string{query})
				queryDone <- err
			}()
			// Wait until the query holds its shared intent lock.
			intent := flock.New(filepath.Join(budgetDir, "query.lock"))
			deadline := time.Now().Add(10 * time.Second)
			for {
				ok, err := intent.TryLock()
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					break
				}
				_ = intent.Unlock()
				if time.Now().After(deadline) {
					t.Fatal("query never announced intent")
				}
				time.Sleep(5 * time.Millisecond)
			}
			_ = intent.Close()
			close(release)
			for name, done := range map[string]chan error{"documents": docDone, "query": queryDone} {
				select {
				case err := <-done:
					if err != nil {
						t.Fatalf("%s: %v", name, err)
					}
				case <-time.After(10 * time.Second):
					t.Fatalf("%s embed did not finish", name)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"doc:32", "query", "doc:8"}
			if !slices.Equal(order, want) {
				t.Fatalf("HTTP request order = %v, want %v", order, want)
			}
			if maxInFlight != 1 {
				t.Fatalf("max in-flight embedding requests = %d, want 1", maxInFlight)
			}
		})
	}
}
