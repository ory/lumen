// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/models"
)

func localHealth(t *testing.T, modelPath string) (string, bool) {
	t.Helper()
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_MODEL_PATH"} {
		t.Setenv(k, "")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	y := "servers:\n  - backend: local\n    model: " + models.DefaultLocalModel + "\n"
	if modelPath != "" {
		y += "    model_path: " + modelPath + "\n"
	}
	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(y), 0o644); err != nil {
		t.Fatal(err)
	}
	svc, err := config.NewConfigService(cfgFile)
	if err != nil {
		t.Fatalf("NewConfigService: %v", err)
	}
	ic := &indexerCache{embedder: embedder.NewFailoverEmbedder(svc), cfg: svc}
	result, _, err := ic.handleHealthCheck(context.Background(), &mcp.CallToolRequest{}, HealthCheckInput{})
	if err != nil {
		t.Fatalf("handleHealthCheck: %v", err)
	}
	return mustTextResult(t, result), result.IsError
}

func TestHandleHealthCheck_LocalBackendReportsLocalReadiness(t *testing.T) {
	gguf := filepath.Join(t.TempDir(), "nomic.gguf")
	if err := os.WriteFile(gguf, []byte("GGUF model"), 0o644); err != nil {
		t.Fatal(err)
	}
	notGGUF := filepath.Join(t.TempDir(), "model.bin")
	if err := os.WriteFile(notGGUF, []byte("not a model"), 0o644); err != nil {
		t.Fatal(err)
	}

	for name, tc := range map[string]struct {
		path    string
		isError bool
		want    string
	}{
		"managed not downloaded": {path: "", want: "will be downloaded"},
		"user file present":      {path: gguf, want: "is present"},
		"user file missing":      {path: filepath.Join(t.TempDir(), "missing.gguf"), isError: true, want: "does not exist"},
		"user file not gguf":     {path: notGGUF, isError: true, want: "not a GGUF model file"},
	} {
		t.Run(name, func(t *testing.T) {
			text, isError := localHealth(t, tc.path)
			if isError != tc.isError || !strings.Contains(text, tc.want) {
				t.Fatalf("health (error=%v):\n%s\nwant error=%v and %q", isError, text, tc.isError, tc.want)
			}
			if !strings.Contains(text, "Backend: local") || !strings.Contains(text, "Host: in-process") {
				t.Errorf("health does not describe the in-process backend:\n%s", text)
			}
			if entries, _ := os.ReadDir(embedder.ManagedModelDir()); len(entries) != 0 {
				t.Errorf("health check touched the model cache: %v", entries)
			}
		})
	}
}

// queryEmbedder distinguishes query encoding from document encoding.
type queryEmbedder struct{ stubEmbedder }

func (q *queryEmbedder) EmbedQuery(context.Context, string) ([]float32, error) {
	return []float32{9, 9, 9, 9}, nil
}

func TestEmbedQuery_UsesQueryEncoding(t *testing.T) {
	ic := &indexerCache{embedder: &queryEmbedder{}}
	vec, err := ic.embedQuery(context.Background(), ic.currentModel(), "find the parser")
	if err != nil {
		t.Fatal(err)
	}
	if vec[0] != 9 {
		t.Errorf("embedQuery = %v, want the query encoding", vec)
	}
}

func TestComputeMaxDistance_LocalIdentityUsesModelDefault(t *testing.T) {
	identity := models.DefaultLocalModel + "@local-sha256-0123456789abcdef-p1"
	if d := computeMaxDistance(nil, identity, 0); d != 0.85 {
		t.Fatalf("computeMaxDistance = %v, want 0.85 from the nomic registry entry", d)
	}
}

func TestLoadConfigWithFlags_LocalBackend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nomic.gguf")
	cmd := flagsTestHarness(t, fmt.Sprintf(`
servers:
  - backend: ollama
    host: http://localhost:11434
    model: all-minilm
  - backend: local
    model: %s
    model_path: %s
`, models.DefaultLocalModel, path))
	if err := cmd.Flags().Set("backend", "local"); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		t.Fatalf("loadConfigWithFlags: %v", err)
	}
	servers := cfg.Servers()
	if len(servers) != 1 || servers[0].Backend != config.BackendLocal || servers[0].ModelPath != path {
		t.Fatalf("servers = %+v, want only the local server", servers)
	}
}

func TestResolveIndexRoot_LocalIdentityFindsExistingCollection(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	for _, k := range []string{"LUMEN_BACKEND", "LUMEN_EMBED_MODEL", "LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX", "OLLAMA_HOST", "LM_STUDIO_HOST", "LUMEN_MODEL_PATH"} {
		t.Setenv(k, "")
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("servers:\n  - backend: local\n    model: "+models.DefaultLocalModel+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.NewConfigService(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	identity := embedder.LocalModelIdentity(cfg.Servers()[0])

	// newIndexedProject creates a non-Git project with an existing local
	// collection, as written by indexing, and an unindexed subdirectory.
	newIndexedProject := func(t *testing.T) (project, child string) {
		t.Helper()
		project, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		child = filepath.Join(project, "src")
		if err := os.Mkdir(child, 0o755); err != nil {
			t.Fatal(err)
		}
		db := configuredDBPath(cfg, nil, project, identity)
		if err := os.MkdirAll(filepath.Dir(db), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(db, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		return project, child
	}

	t.Run("ancestor", func(t *testing.T) {
		project, child := newIndexedProject(t)
		if got := findAncestorIndex(child, identity); got != project {
			t.Errorf("findAncestorIndex = %q, want %q", got, project)
		}
		if got, _, err := resolveIndexRoot(child, "", identity); err != nil || got != project {
			t.Errorf("resolveIndexRoot = %q, %v; want %q", got, err, project)
		}
	})

	t.Run("explicit cwd", func(t *testing.T) {
		project, _ := newIndexedProject(t)
		elsewhere, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if got, _, err := resolveIndexRoot(elsewhere, project, identity); err != nil || got != project {
			t.Errorf("resolveIndexRoot = %q, %v; want cwd %q", got, err, project)
		}
	})
}
