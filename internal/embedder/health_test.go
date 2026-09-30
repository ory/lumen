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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ory/lumen/internal/config"
)

func TestProbeServer_OpenAI_TrailingV1InHostNotDoubled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("expected path /v1/models, got %q (host's /v1 must not be duplicated)", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"test-model"}]}`))
	}))
	defer server.Close()

	srv := config.ServerConfig{Backend: config.BackendOpenAI, Host: server.URL + "/v1", Model: "test-model"}
	if err := ProbeServer(context.Background(), srv); err != nil {
		t.Fatalf("ProbeServer: %v", err)
	}
}

// TestProbeServer_AuthenticatedRedirects mirrors the embedder redirect test
// for the /v1/models health probe, which carries the same bearer token.
func TestProbeServer_AuthenticatedRedirects(t *testing.T) {
	var plainHits atomic.Int32
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		plainHits.Add(1)
		_, _ = w.Write([]byte(`{"data":[{"id":"test-model"}]}`))
	}))
	defer plain.Close()

	var target atomic.Value
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
				t.Errorf("Authorization = %q, want Bearer sk-test", got)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"test-model"}]}`))
			return
		}
		http.Redirect(w, r, target.Load().(string), http.StatusTemporaryRedirect)
	}))
	defer secure.Close()

	orig := authProbeClient
	authProbeClient = &http.Client{Transport: secure.Client().Transport, CheckRedirect: orig.CheckRedirect}
	t.Cleanup(func() { authProbeClient = orig })

	tests := []struct {
		name    string
		target  string
		wantErr string
	}{
		{name: "https to http downgrade is refused", target: plain.URL + "/v1/models", wantErr: "refusing redirect"},
		{name: "https to https is followed", target: secure.URL + "/final"},
		{name: "redirect limit is retained", target: secure.URL + "/v1/models", wantErr: "stopped after 10 redirects"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target.Store(tt.target)
			plainHits.Store(0)

			srv := config.ServerConfig{Backend: config.BackendOpenAI, Host: secure.URL, Model: "test-model", APIKey: "sk-test"}
			err := ProbeServer(context.Background(), srv)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ProbeServer: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ProbeServer error = %v, want containing %q", err, tt.wantErr)
			}
			if n := plainHits.Load(); n != 0 {
				t.Errorf("plaintext destination contacted %d times, want 0", n)
			}
		})
	}
}
