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
	"os"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/config"
)

func TestStatusExitCode(t *testing.T) {
	// reach builds a single-server slice with the given reachability.
	reach := func(ok bool) []serverStatus { return []serverStatus{{reachable: ok}} }
	tests := []struct {
		name string
		r    statusResult
		want int
	}{
		{"healthy and fresh", statusResult{servers: reach(true), indexed: true, stale: false}, 0},
		{"service unreachable", statusResult{servers: reach(false), indexed: true, stale: false}, 1},
		{"index missing", statusResult{servers: reach(true), indexed: false}, 1},
		{"index stale", statusResult{servers: reach(true), indexed: true, stale: true}, 1},
		{"all bad", statusResult{servers: reach(false), indexed: false, stale: true}, 1},
		// Failover semantics: at least one reachable server means the service
		// is usable, so this is healthy.
		{"one of two reachable", statusResult{servers: []serverStatus{{reachable: false}, {reachable: true}}, indexed: true, stale: false}, 0},
		{"all servers unreachable", statusResult{servers: []serverStatus{{reachable: false}, {reachable: false}}, indexed: true, stale: false}, 1},
		{"no servers configured", statusResult{servers: nil, indexed: true, stale: false}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusExitCode(tt.r); got != tt.want {
				t.Errorf("statusExitCode(%+v) = %d, want %d", tt.r, got, tt.want)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	okServer := func() serverStatus {
		return serverStatus{
			server:    config.ServerConfig{Backend: "ollama", Host: "http://localhost:11434", Model: "jina"},
			reachable: true,
			message:   "service is healthy",
		}
	}

	t.Run("healthy and fresh", func(t *testing.T) {
		r := statusResult{
			servers:       []serverStatus{okServer()},
			projectPath:   "/repo",
			indexed:       true,
			totalFiles:    158,
			indexedFiles:  158,
			totalChunks:   1646,
			model:         "jina",
			lastIndexedAt: "2026-05-30T12:00:00Z",
			stale:         false,
		}
		out := formatStatus(r)
		if !strings.Contains(out, "Embedding service: OK (ollama, http://localhost:11434, jina)") {
			t.Errorf("missing healthy service line:\n%s", out)
		}
		if !strings.Contains(out, "Files: 158 | Indexed: 158 | Chunks: 1646 | Model: jina") {
			t.Errorf("missing index stats line:\n%s", out)
		}
		if !strings.Contains(out, "Stale: no") {
			t.Errorf("expected Stale: no:\n%s", out)
		}
	})

	t.Run("service unreachable", func(t *testing.T) {
		r := statusResult{
			servers: []serverStatus{{
				server:    config.ServerConfig{Backend: "ollama", Host: "http://localhost:11434"},
				reachable: false,
				message:   "service unreachable: connection refused",
			}},
		}
		out := formatStatus(r)
		if !strings.Contains(out, "Embedding service: ERROR (ollama, http://localhost:11434) — service unreachable: connection refused") {
			t.Errorf("missing error service line:\n%s", out)
		}
	})

	t.Run("multiple servers each reported", func(t *testing.T) {
		r := statusResult{
			servers: []serverStatus{
				okServer(),
				{
					server:    config.ServerConfig{Backend: "lmstudio", Host: "http://localhost:1234"},
					reachable: false,
					message:   "service unreachable: nope",
				},
			},
			projectPath: "/repo",
			indexed:     false,
		}
		out := formatStatus(r)
		if !strings.Contains(out, "Embedding service: OK (ollama, http://localhost:11434, jina)") {
			t.Errorf("missing ollama OK line:\n%s", out)
		}
		if !strings.Contains(out, "Embedding service: ERROR (lmstudio, http://localhost:1234) — service unreachable: nope") {
			t.Errorf("missing lmstudio ERROR line:\n%s", out)
		}
	})

	t.Run("no servers configured", func(t *testing.T) {
		r := statusResult{servers: nil, projectPath: "/repo", indexed: false}
		out := formatStatus(r)
		if !strings.Contains(out, "Embedding service: ERROR — no servers configured") {
			t.Errorf("missing no-servers line:\n%s", out)
		}
	})

	t.Run("not indexed", func(t *testing.T) {
		r := statusResult{
			servers:     []serverStatus{okServer()},
			projectPath: "/repo",
			indexed:     false,
		}
		out := formatStatus(r)
		if !strings.Contains(out, "/repo — not indexed") {
			t.Errorf("missing not-indexed line:\n%s", out)
		}
	})

	t.Run("indexed but never indexed shows never", func(t *testing.T) {
		r := statusResult{
			servers:       []serverStatus{okServer()},
			projectPath:   "/repo",
			indexed:       true,
			lastIndexedAt: "",
			stale:         true,
		}
		out := formatStatus(r)
		if !strings.Contains(out, "Last indexed: never") {
			t.Errorf("expected 'Last indexed: never' when lastIndexedAt empty:\n%s", out)
		}
	})

	t.Run("stale shows yes", func(t *testing.T) {
		r := statusResult{
			servers:       []serverStatus{okServer()},
			projectPath:   "/repo",
			indexed:       true,
			lastIndexedAt: "2026-05-30T12:00:00Z",
			stale:         true,
		}
		out := formatStatus(r)
		if !strings.Contains(out, "Stale: yes") {
			t.Errorf("expected Stale: yes:\n%s", out)
		}
	})
}

func TestRunStatusMissingIndexNoSideEffect(t *testing.T) {
	// A temp dir with no git repo and no existing index: collectStatus must
	// report not-indexed and must NOT create a DB file.
	tmp := t.TempDir()

	// Isolate XDG_DATA_HOME so we observe DB creation cleanly.
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	cfg, err := config.NewConfigService(config.DefaultConfigPath())
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	emb := newEmbedder(cfg)

	r := collectStatus(context.Background(), cfg, emb, tmp)

	if r.indexed {
		t.Error("expected not indexed for empty temp dir, got indexed=true")
	}

	// collectStatus must probe every configured server.
	if len(r.servers) != len(cfg.Servers()) {
		t.Errorf("expected %d server results, got %d", len(cfg.Servers()), len(r.servers))
	}

	dbPath := config.DBPathForProject(tmp, emb.ModelName())
	if _, statErr := os.Stat(dbPath); statErr == nil {
		t.Errorf("collectStatus created a DB file at %s; it must be read-only", dbPath)
	}
}
