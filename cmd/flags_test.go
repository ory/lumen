// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/config"
	"github.com/spf13/cobra"
)

// flagsTestHarness sets up an isolated XDG_CONFIG_HOME with the given YAML,
// clears server env vars, and returns the cobra command to exercise.
func flagsTestHarness(t *testing.T, yaml string) *cobra.Command {
	// This harness exercises embedding server selection, which is opt-in.
	t.Setenv("LUMEN_RETRIEVAL", "semantic")
	t.Helper()

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	for _, k := range []string{
		"LUMEN_BACKEND", "LUMEN_EMBED_MODEL",
		"OLLAMA_HOST", "LM_STUDIO_HOST",
		"LUMEN_EMBED_DIMS", "LUMEN_EMBED_CTX",
	} {
		t.Setenv(k, "")
	}

	if yaml != "" {
		dir := filepath.Join(home, "lumen")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringP("model", "m", "", "")
	cmd.Flags().StringP("backend", "b", "", "")
	return cmd
}

func TestLoadConfigWithFlags_NoFlagsLoadsYAML(t *testing.T) {
	cmd := flagsTestHarness(t, threeServerFixture)
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		t.Fatalf("loadConfigWithFlags: %v", err)
	}
	if got := len(cfg.Servers()); got != 3 {
		t.Errorf("Servers() len = %d, want 3", got)
	}
}

func TestLoadConfigWithFlags_ModelInYAML_UnambiguousAcrossBackends(t *testing.T) {
	cmd := flagsTestHarness(t, threeServerFixture)
	_ = cmd.Flags().Set("model", "ordis/jina-embeddings-v2-base-code")
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		t.Fatalf("loadConfigWithFlags: %v", err)
	}
	servers := cfg.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Backend != config.BackendOllama {
		t.Errorf("Backend = %q, want %q", servers[0].Backend, config.BackendOllama)
	}
}

func TestLoadConfigWithFlags_ModelInYAML_LMStudioOnlyName(t *testing.T) {
	cmd := flagsTestHarness(t, threeServerFixture)
	_ = cmd.Flags().Set("model", "text-embedding-jina-embeddings-v2-base-code")
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		t.Fatalf("loadConfigWithFlags: %v", err)
	}
	servers := cfg.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Backend != config.BackendLMStudio {
		t.Errorf("Backend = %q, want %q", servers[0].Backend, config.BackendLMStudio)
	}
}

func TestLoadConfigWithFlags_ModelAndBackend(t *testing.T) {
	cmd := flagsTestHarness(t, threeServerFixture)
	_ = cmd.Flags().Set("model", "ordis/jina-embeddings-v2-base-code")
	_ = cmd.Flags().Set("backend", "ollama")
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		t.Fatalf("loadConfigWithFlags: %v", err)
	}
	servers := cfg.Servers()
	if len(servers) != 1 || servers[0].Backend != config.BackendOllama {
		t.Fatalf("unexpected servers: %+v", servers)
	}
}

func TestLoadConfigWithFlags_BackendOnly(t *testing.T) {
	cmd := flagsTestHarness(t, threeServerFixture)
	_ = cmd.Flags().Set("backend", "ollama")
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		t.Fatalf("loadConfigWithFlags: %v", err)
	}
	servers := cfg.Servers()
	if len(servers) != 1 || servers[0].Backend != config.BackendOllama {
		t.Fatalf("unexpected servers: %+v", servers)
	}
}

func TestLoadConfigWithFlags_KnownModelFallback_NoYAML(t *testing.T) {
	// No YAML. User passes a KnownModels entry that isn't the default. The
	// fallback should rewrite servers[0].model via WithModelOverride.
	cmd := flagsTestHarness(t, "")
	_ = cmd.Flags().Set("model", "all-minilm")
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		t.Fatalf("loadConfigWithFlags: %v", err)
	}
	servers := cfg.Servers()
	if len(servers) != 1 {
		t.Fatalf("Servers() len = %d, want 1", len(servers))
	}
	if servers[0].Model != "all-minilm" {
		t.Errorf("Model = %q, want all-minilm", servers[0].Model)
	}
}

func TestLoadConfigWithFlags_UnknownModelYAMLConfigured(t *testing.T) {
	// Model not in KnownModels, not in YAML → should error with the listing
	// of configured servers.
	cmd := flagsTestHarness(t, threeServerFixture)
	_ = cmd.Flags().Set("model", "totally-nonexistent")
	_, err := loadConfigWithFlags(cmd)
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
	if !strings.Contains(err.Error(), "no server matches") {
		t.Errorf("error should name the filter mismatch, got %q", err.Error())
	}
}

func TestLoadConfigWithFlags_InvalidBackend(t *testing.T) {
	cmd := flagsTestHarness(t, threeServerFixture)
	_ = cmd.Flags().Set("backend", "bogus")
	_, err := loadConfigWithFlags(cmd)
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
	if !strings.Contains(err.Error(), "unknown backend") {
		t.Errorf("error should mention unknown backend, got %q", err.Error())
	}
}

func TestLoadConfigWithFlags_KnownModelButBackendMismatch(t *testing.T) {
	// KnownModels fallback must NOT kick in when --backend is set — we don't
	// want to silently mask a real selection error by falling back to
	// servers[0] mutation.
	cmd := flagsTestHarness(t, threeServerFixture)
	// ollama-only model, but the user forces --backend lmstudio → no match,
	// no fallback, clear error.
	_ = cmd.Flags().Set("model", "ordis/jina-embeddings-v2-base-code")
	_ = cmd.Flags().Set("backend", "lmstudio")
	_, err := loadConfigWithFlags(cmd)
	if err == nil {
		t.Fatal("expected error when --backend contradicts --model")
	}
}

const threeServerFixture = `
servers:
  - backend: lmstudio
    host: http://remote:1234
    model: text-embedding-nomic-embed-code
    dims: 3584
  - backend: lmstudio
    host: http://remote:1234
    model: text-embedding-jina-embeddings-v2-base-code
    dims: 768
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
    dims: 768
`

func TestSemanticFlagsApplyBeforeProfileValidation(t *testing.T) {
	t.Run("known model repairs invalid base model", func(t *testing.T) {
		cmd := flagsTestHarness(t, "servers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: unknown-model\n")
		_ = cmd.Flags().Set("model", "all-minilm")
		cfg, err := loadConfigWithFlags(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Servers()[0].Model != "all-minilm" {
			t.Fatalf("wrong model: %+v", cfg.Servers())
		}
	})
	t.Run("selection excludes invalid unselected server", func(t *testing.T) {
		cmd := flagsTestHarness(t, "servers:\n  - backend: invalid\n    host: wrong\n    model: unknown-model\n  - backend: ollama\n    host: http://localhost:11434\n    model: all-minilm\n")
		_ = cmd.Flags().Set("model", "all-minilm")
		cfg, err := loadConfigWithFlags(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.Servers()) != 1 || cfg.Servers()[0].Model != "all-minilm" {
			t.Fatalf("wrong servers: %+v", cfg.Servers())
		}
	})
}

func TestEmbeddingFlagsRejectedInBM25Mode(t *testing.T) {
	yaml := "servers:\n  - backend: ollama\n    host: http://localhost:11434\n    model: all-minilm\n"
	for _, flags := range []map[string]string{
		{"model": "all-minilm"},
		{"backend": "ollama"},
		{"model": "all-minilm", "backend": "ollama"},
	} {
		cmd := flagsTestHarness(t, yaml)
		t.Setenv("LUMEN_RETRIEVAL", "bm25")
		for name, value := range flags {
			_ = cmd.Flags().Set(name, value)
		}
		_, err := loadConfigWithFlags(cmd)
		if err == nil || !strings.Contains(err.Error(), "retrieval=semantic") {
			t.Fatalf("%v in bm25 mode: want rejection mentioning retrieval=semantic, got %v", flags, err)
		}
	}
	cmd := flagsTestHarness(t, yaml)
	_ = cmd.Flags().Set("model", "all-minilm")
	if _, err := loadConfigWithFlags(cmd); err != nil {
		t.Fatalf("semantic mode with --model: %v", err)
	}
}
