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

package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/config"
)

func TestProbeEmbeddingService(t *testing.T) {
	t.Run("reachable when ollama returns 200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/tags" {
				t.Errorf("expected /api/tags, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		reachable, msg := probeEmbeddingService(context.Background(),
			config.ServerConfig{Backend: config.BackendOllama, Host: srv.URL, Model: "m"})
		if !reachable {
			t.Fatalf("expected reachable, got message %q", msg)
		}
	})
	t.Run("reachable when lmstudio returns 200 on /v1/models", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/models" {
				t.Errorf("expected /v1/models, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		reachable, _ := probeEmbeddingService(context.Background(),
			config.ServerConfig{Backend: config.BackendLMStudio, Host: srv.URL, Model: "m"})
		if !reachable {
			t.Fatal("expected reachable for lmstudio 200")
		}
	})
	t.Run("not reachable on 5xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		reachable, msg := probeEmbeddingService(context.Background(),
			config.ServerConfig{Backend: config.BackendOllama, Host: srv.URL, Model: "m"})
		if reachable {
			t.Fatal("expected not reachable on 500")
		}
		if !strings.Contains(msg, "500") {
			t.Errorf("expected message to mention 500, got %q", msg)
		}
	})
	t.Run("not reachable when connection refused", func(t *testing.T) {
		// Start a server and close it immediately so srv.URL is a refused
		// address — hermetic, no reliance on a well-known closed port.
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		srv.Close()
		reachable, msg := probeEmbeddingService(context.Background(),
			config.ServerConfig{Backend: config.BackendOllama, Host: srv.URL, Model: "m"})
		if reachable {
			t.Fatal("expected not reachable for refused connection")
		}
		if msg == "" {
			t.Error("expected non-empty failure message")
		}
	})
	t.Run("reachable on 4xx (only 5xx and transport errors are unhealthy)", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()
		reachable, _ := probeEmbeddingService(context.Background(),
			config.ServerConfig{Backend: config.BackendOllama, Host: srv.URL, Model: "m"})
		if !reachable {
			t.Error("expected 404 to be treated as reachable")
		}
	})
}
