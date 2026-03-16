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

package config

import (
	"strings"
	"testing"
)

func TestEnvOrDefaultInt(t *testing.T) {
	t.Setenv("TEST_DIMS", "384")
	if got := EnvOrDefaultInt("TEST_DIMS", 1024); got != 384 {
		t.Fatalf("got %d, want 384", got)
	}
	if got := EnvOrDefaultInt("TEST_DIMS_UNSET", 1024); got != 1024 {
		t.Fatalf("got %d, want 1024", got)
	}
}

func TestDBPathForProject(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project", "model-a", "")
		p2 := DBPathForProject("/home/user/project", "model-a", "")
		if p1 != p2 {
			t.Fatalf("expected same path, got %q and %q", p1, p2)
		}
	})

	t.Run("different project paths produce different hashes", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project-a", "model-a", "")
		p2 := DBPathForProject("/home/user/project-b", "model-a", "")
		if p1 == p2 {
			t.Fatalf("expected different paths, got same: %q", p1)
		}
	})

	t.Run("different models produce different hashes", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project", "model-a", "")
		p2 := DBPathForProject("/home/user/project", "model-b", "")
		if p1 == p2 {
			t.Fatalf("expected different paths, got same: %q", p1)
		}
	})

	t.Run("uses IndexVersion not runtime state", func(t *testing.T) {
		if IndexVersion == "" {
			t.Fatal("IndexVersion must not be empty")
		}
		p1 := DBPathForProject("/some/path", "some-model", "")
		p2 := DBPathForProject("/some/path", "some-model", "")
		if p1 != p2 {
			t.Fatalf("path not stable: %q vs %q", p1, p2)
		}
	})

	t.Run("ends with index.db", func(t *testing.T) {
		p := DBPathForProject("/some/path", "model", "")
		if !strings.HasSuffix(p, "index.db") {
			t.Fatalf("expected path to end with index.db, got %q", p)
		}
	})

	t.Run("empty summaryEmbedModel is deterministic", func(t *testing.T) {
		p1 := DBPathForProject("/p", "m", "")
		p2 := DBPathForProject("/p", "m", "")
		if p1 != p2 {
			t.Fatalf("empty summaryEmbedModel should be deterministic: %q vs %q", p1, p2)
		}
	})

	t.Run("non-empty summaryEmbedModel produces different hash", func(t *testing.T) {
		p1 := DBPathForProject("/p", "m", "")
		p2 := DBPathForProject("/p", "m", "nomic-embed-text")
		if p1 == p2 {
			t.Fatalf("expected different DB paths when summaryEmbedModel differs")
		}
	})
}

func TestLoad_SummaryConfig_Disabled(t *testing.T) {
	t.Setenv("LUMEN_SUMMARIES", "")
	t.Setenv("LUMEN_BACKEND", "ollama")
	t.Setenv("LUMEN_EMBED_MODEL", "nomic-embed-text")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Summaries {
		t.Fatal("expected Summaries=false when LUMEN_SUMMARIES not set")
	}
	if cfg.SummaryEmbedModel != "" {
		t.Fatalf("expected empty SummaryEmbedModel when disabled, got %q", cfg.SummaryEmbedModel)
	}
	if cfg.SummaryEmbedDims != 0 {
		t.Fatalf("expected SummaryEmbedDims=0 when disabled, got %d", cfg.SummaryEmbedDims)
	}
}

func TestLoad_SummaryConfig_Enabled_Ollama(t *testing.T) {
	t.Setenv("LUMEN_SUMMARIES", "true")
	t.Setenv("LUMEN_BACKEND", "ollama")
	t.Setenv("LUMEN_EMBED_MODEL", "nomic-embed-text")
	t.Setenv("LUMEN_SUMMARY_MODEL", "")
	t.Setenv("LUMEN_SUMMARY_EMBED_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if !cfg.Summaries {
		t.Fatal("expected Summaries=true")
	}
	if cfg.SummaryModel != "gemma3:4b" {
		t.Fatalf("expected default SummaryModel=gemma3:4b, got %q", cfg.SummaryModel)
	}
	if cfg.SummaryEmbedModel != "nomic-embed-text" {
		t.Fatalf("expected default SummaryEmbedModel=nomic-embed-text for ollama, got %q", cfg.SummaryEmbedModel)
	}
	if cfg.SummaryEmbedDims != 768 {
		t.Fatalf("expected SummaryEmbedDims=768, got %d", cfg.SummaryEmbedDims)
	}
}

func TestLoad_SummaryConfig_Enabled_LMStudio(t *testing.T) {
	t.Setenv("LUMEN_SUMMARIES", "true")
	t.Setenv("LUMEN_BACKEND", "lmstudio")
	t.Setenv("LUMEN_EMBED_MODEL", "nomic-ai/nomic-embed-code-GGUF")
	t.Setenv("LUMEN_SUMMARY_MODEL", "")
	t.Setenv("LUMEN_SUMMARY_EMBED_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.SummaryEmbedModel != "nomic-ai/nomic-embed-text-GGUF" {
		t.Fatalf("expected default SummaryEmbedModel=nomic-ai/nomic-embed-text-GGUF for lmstudio, got %q", cfg.SummaryEmbedModel)
	}
	if cfg.SummaryEmbedDims != 768 {
		t.Fatalf("expected SummaryEmbedDims=768, got %d", cfg.SummaryEmbedDims)
	}
}

func TestLoad_SummaryConfig_UnknownModel_FallbackDims(t *testing.T) {
	t.Setenv("LUMEN_SUMMARIES", "true")
	t.Setenv("LUMEN_BACKEND", "ollama")
	t.Setenv("LUMEN_EMBED_MODEL", "nomic-embed-text")
	t.Setenv("LUMEN_SUMMARY_EMBED_MODEL", "some-unknown-embed-model")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.SummaryEmbedDims != 768 {
		t.Fatalf("expected fallback SummaryEmbedDims=768 for unknown model, got %d", cfg.SummaryEmbedDims)
	}
}
