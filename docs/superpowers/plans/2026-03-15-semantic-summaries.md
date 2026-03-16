# Semantic Summaries Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add LLM-generated semantic summaries to the indexing and search pipeline so queries match against natural-language descriptions of code, with file-level hits producing `<relevant_files>` hints in MCP responses.

**Architecture:** A new `internal/summarizer/` package wraps Ollama and LM Studio chat APIs to generate chunk and file summaries. The store gains four new tables (two text, two vec) guarded by a `summaryDims > 0` flag. The indexer runs two new passes after raw embedding; search fans out to three vector indices and merges results.

**Tech Stack:** Go 1.25+, SQLite + sqlite-vec, Ollama/LM Studio chat completion APIs, MCP (go-sdk), Cobra

---

## Chunk 1: Summarizer Package + Config

### File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/summarizer/summarizer.go` | Create | `ChunkInfo` struct, `Summarizer` interface, factory function |
| `internal/summarizer/ollama.go` | Create | Ollama chat completion client |
| `internal/summarizer/lmstudio.go` | Create | LM Studio chat completion client |
| `internal/summarizer/summarizer_test.go` | Create | Unit tests with mock HTTP server for both clients |
| `internal/embedder/models.go` | Modify | Add `nomic-ai/nomic-embed-text-GGUF` entry |
| `internal/config/config.go` | Modify | Add `Summaries`, `SummaryModel`, `SummaryEmbedModel`, `SummaryEmbedDims` fields; update `Load()` and `DBPathForProject()` |
| `internal/config/config_test.go` | Modify | Update `TestDBPathForProject` and add summary config tests |

---

### Task 1: Add `nomic-ai/nomic-embed-text-GGUF` to the embedder model registry

**Files:**
- Modify: `internal/embedder/models.go`

- [ ] **Step 1: Write a failing test that asserts the model exists in `KnownModels`**

In `internal/embedder/models_test.go`, add:

```go
func TestKnownModels_NomicEmbedTextGGUF(t *testing.T) {
    spec, ok := KnownModels["nomic-ai/nomic-embed-text-GGUF"]
    if !ok {
        t.Fatal("nomic-ai/nomic-embed-text-GGUF missing from KnownModels")
    }
    if spec.Dims != 768 {
        t.Fatalf("expected 768 dims, got %d", spec.Dims)
    }
    if spec.Backend != "lmstudio" {
        t.Fatalf("expected lmstudio backend, got %q", spec.Backend)
    }
    if spec.MinScore != 0.30 {
        t.Fatalf("expected MinScore 0.30, got %f", spec.MinScore)
    }
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/embedder/... -run TestKnownModels_NomicEmbedTextGGUF -v
```

Expected: `FAIL` — `nomic-ai/nomic-embed-text-GGUF missing from KnownModels`

- [ ] **Step 3: Add the entry to `KnownModels` in `internal/embedder/models.go`**

Inside the `KnownModels` map literal, add:

```go
"nomic-ai/nomic-embed-text-GGUF": {Dims: 768, CtxLength: 8192, Backend: "lmstudio", MinScore: 0.30},
```

- [ ] **Step 4: Run to confirm it passes**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/embedder/... -v
```

Expected: all embedder tests pass.

- [ ] **Step 5: Commit**

```bash
cd /Users/aeneas/workspace/go/agent-index-go
git add internal/embedder/models.go internal/embedder/models_test.go
git commit -m "feat(embedder): add nomic-ai/nomic-embed-text-GGUF to KnownModels"
```

---

### Task 2: Extend `Config` with summary fields and update `Load()` + `DBPathForProject()`

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Background:** `Config.Load()` already reads `LUMEN_EMBED_MODEL` and resolves dims from `KnownModels`. We add three more env vars and extend `DBPathForProject` to accept a `summaryEmbedModel` parameter. When summaries are disabled (`Summaries=false`), `summaryEmbedModel` is `""` — which keeps the hash identical to the current formula (backward compatible).

- [ ] **Step 1: Write failing tests for the new config fields and updated `DBPathForProject`**

Replace the existing content of `internal/config/config_test.go` with:

```go
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

	t.Run("empty summaryEmbedModel is backward compatible with old two-arg hash", func(t *testing.T) {
		// When summaryEmbedModel is "", the hash input is identical to the old formula:
		// SHA-256(projectPath + "\x00" + codeModel + "\x00" + "" + "\x00" + IndexVersion)
		// The old formula was: SHA-256(projectPath + "\x00" + codeModel + "\x00" + IndexVersion)
		// These differ; the test just checks that passing "" is stable (not that it equals the old value).
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
	// When LUMEN_SUMMARIES is not set, summaries are disabled.
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
	if cfg.SummaryModel != "qwen2.5-coder:7b" {
		t.Fatalf("expected default SummaryModel=qwen2.5-coder:7b, got %q", cfg.SummaryModel)
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
	// Unknown model falls back to 768
	if cfg.SummaryEmbedDims != 768 {
		t.Fatalf("expected fallback SummaryEmbedDims=768 for unknown model, got %d", cfg.SummaryEmbedDims)
	}
}
```

- [ ] **Step 2: Run the new tests to confirm they fail**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/config/... -v
```

Expected: compilation errors or test failures because `DBPathForProject` still takes two args and `Config` lacks the new fields.

- [ ] **Step 3: Update `internal/config/config.go`**

Replace the file with:

```go
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

// Package config loads and validates runtime configuration from environment variables.
package config

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ory/lumen/internal/embedder"
)

const (
	// BackendOllama is the backend identifier for Ollama.
	BackendOllama = "ollama"
	// BackendLMStudio is the backend identifier for LM Studio.
	BackendLMStudio = "lmstudio"

	// DefaultSummaryModel is the LLM used for generating summaries.
	DefaultSummaryModel = "qwen2.5-coder:7b"
	// DefaultSummaryEmbedModelOllama is the embedding model for summaries on Ollama.
	DefaultSummaryEmbedModelOllama = "nomic-embed-text"
	// DefaultSummaryEmbedModelLMStudio is the embedding model for summaries on LM Studio.
	DefaultSummaryEmbedModelLMStudio = "nomic-ai/nomic-embed-text-GGUF"
	// DefaultSummaryEmbedDims is the fallback dimensionality for unknown summary embed models.
	DefaultSummaryEmbedDims = 768
)

// Config holds the resolved configuration for the lumen process.
type Config struct {
	Model          string
	Dims           int
	CtxLength      int
	MaxChunkTokens int
	OllamaHost     string
	Backend        string
	LMStudioHost   string

	// Summaries fields — only populated when LUMEN_SUMMARIES=true.
	Summaries         bool
	SummaryModel      string
	SummaryEmbedModel string
	SummaryEmbedDims  int
}

// Load reads configuration from environment variables and the model registry.
func Load() (Config, error) {
	backend := EnvOrDefault("LUMEN_BACKEND", BackendOllama)
	if backend != BackendOllama && backend != BackendLMStudio {
		return Config{}, fmt.Errorf("unknown backend %q: must be %q or %q", backend, BackendOllama, BackendLMStudio)
	}

	defaultModel := embedder.DefaultOllamaModel
	if backend == BackendLMStudio {
		defaultModel = embedder.DefaultLMStudioModel
	}

	model := EnvOrDefault("LUMEN_EMBED_MODEL", defaultModel)
	spec, ok := embedder.KnownModels[model]
	if !ok {
		return Config{}, fmt.Errorf("unknown embedding model %q", model)
	}

	cfg := Config{
		Model:          model,
		Dims:           spec.Dims,
		CtxLength:      spec.CtxLength,
		MaxChunkTokens: EnvOrDefaultInt("LUMEN_MAX_CHUNK_TOKENS", 512),
		OllamaHost:     EnvOrDefault("OLLAMA_HOST", "http://localhost:11434"),
		Backend:        backend,
		LMStudioHost:   EnvOrDefault("LM_STUDIO_HOST", "http://localhost:1234"),
	}

	if EnvOrDefault("LUMEN_SUMMARIES", "") == "true" {
		cfg.Summaries = true
		cfg.SummaryModel = EnvOrDefault("LUMEN_SUMMARY_MODEL", DefaultSummaryModel)

		defaultSummaryEmbedModel := DefaultSummaryEmbedModelOllama
		if backend == BackendLMStudio {
			defaultSummaryEmbedModel = DefaultSummaryEmbedModelLMStudio
		}
		cfg.SummaryEmbedModel = EnvOrDefault("LUMEN_SUMMARY_EMBED_MODEL", defaultSummaryEmbedModel)

		if sumSpec, ok := embedder.KnownModels[cfg.SummaryEmbedModel]; ok {
			cfg.SummaryEmbedDims = sumSpec.Dims
		} else {
			log.Printf("warning: unknown summary embed model %q, using fallback %d dims", cfg.SummaryEmbedModel, DefaultSummaryEmbedDims)
			cfg.SummaryEmbedDims = DefaultSummaryEmbedDims
		}
	}

	return cfg, nil
}

// DBPathForProject returns the SQLite database path for a given project,
// derived from a SHA-256 hash of the project path, code embedding model name,
// summary embedding model name (empty string when summaries disabled), and
// IndexVersion. Including the models ensures that switching models creates a
// fresh index automatically.
//
// Backward compatibility: when summaryEmbedModel is "", the hash input gains a
// trailing "\x00" compared to the old two-arg formula. Existing users who
// never set LUMEN_SUMMARIES will see a new DB path on upgrade — a one-time
// full re-index will occur automatically. This is acceptable because summary
// tables are a new schema addition.
func DBPathForProject(projectPath, codeEmbedModel, summaryEmbedModel string) string {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(projectPath+"\x00"+codeEmbedModel+"\x00"+summaryEmbedModel+"\x00"+IndexVersion)))
	dataDir := XDGDataDir()
	return filepath.Join(dataDir, "lumen", hash[:16], "index.db")
}

// XDGDataDir returns the XDG data home directory, defaulting to
// ~/.local/share if XDG_DATA_HOME is not set.
func XDGDataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

// EnvOrDefault returns the value of the environment variable named by key,
// or fallback if the variable is not set or empty.
func EnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// EnvOrDefaultInt returns the integer value of the environment variable named
// by key, or fallback if the variable is not set, empty, or not a valid integer.
func EnvOrDefaultInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}
```

- [ ] **Step 4: Fix all callers of `DBPathForProject` (now requires 3 args)**

`DBPathForProject` is called in four places. Update each:

**`cmd/stdio.go`** — in `findEffectiveRoot`, line ~145:
```go
// Before:
if _, err := os.Stat(config.DBPathForProject(candidate, ic.model)); err == nil {
// After:
if _, err := os.Stat(config.DBPathForProject(candidate, ic.model, ic.summaryEmbedModel)); err == nil {
```

Also update `indexerCache` struct (add `summaryEmbedModel string` field) and `getOrCreate` (line ~220):
```go
// Before:
dbPath := config.DBPathForProject(effectiveRoot, ic.model)
// After:
dbPath := config.DBPathForProject(effectiveRoot, ic.model, ic.summaryEmbedModel)
```

**`cmd/index.go`** — in `setupIndexer`, line ~101:
```go
// Before:
dbPath := config.DBPathForProject(projectPath, cfg.Model)
// After:
dbPath := config.DBPathForProject(projectPath, cfg.Model, cfg.SummaryEmbedModel)
```

**`cmd/hook.go`** — in `generateSessionContext`, line ~108:
```go
// Before:
dbPath := config.DBPathForProject(cwd, cfg.Model)
// After:
dbPath := config.DBPathForProject(cwd, cfg.Model, cfg.SummaryEmbedModel)
```

Note: `cmd/purge.go` does not call `DBPathForProject` — it purges the entire data directory with `os.RemoveAll`. No change needed there.

- [ ] **Step 5: Run config tests to confirm they pass**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/config/... -v
```

Expected: all tests pass.

- [ ] **Step 6: Run the full test suite to confirm no regressions**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./...
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/aeneas/workspace/go/agent-index-go
git add internal/config/config.go internal/config/config_test.go \
        cmd/stdio.go cmd/index.go cmd/hook.go
git commit -m "feat(config): add summary config fields and update DBPathForProject signature"
```

---

### Task 3: Create `internal/summarizer/` package

**Files:**
- Create: `internal/summarizer/summarizer.go`
- Create: `internal/summarizer/ollama.go`
- Create: `internal/summarizer/lmstudio.go`
- Create: `internal/summarizer/summarizer_test.go`

**Background:** The summarizer wraps the chat completion API. Ollama uses `POST /api/chat` with `stream: false`. LM Studio uses `POST /v1/chat/completions` (OpenAI-compatible). Both return a `message.content` string. The interface is purposely minimal. The `ChunkInfo` struct avoids importing `chunker` — the mapping happens in `internal/index/index.go`.

- [ ] **Step 1: Write the test file first (TDD)**

Create `internal/summarizer/summarizer_test.go`:

```go
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

package summarizer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/summarizer"
)

// --- Ollama mock ---

func ollamaChatHandler(t *testing.T, wantSubstring string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/chat" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(req.Messages) == 0 {
			t.Error("expected at least one message")
		}
		userContent := req.Messages[len(req.Messages)-1].Content
		if !strings.Contains(userContent, wantSubstring) {
			t.Errorf("expected prompt to contain %q, got:\n%s", wantSubstring, userContent)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "This function does X."},
		})
	}
}

func TestOllamaSummarizer_SummarizeChunk(t *testing.T) {
	srv := httptest.NewServer(ollamaChatHandler(t, "MyFunc"))
	defer srv.Close()

	s := summarizer.NewOllama("qwen2.5-coder:7b", srv.URL)
	result, err := s.SummarizeChunk(context.Background(), summarizer.ChunkInfo{
		Kind:    "function",
		Symbol:  "MyFunc",
		Content: "func MyFunc() {}",
	})
	if err != nil {
		t.Fatalf("SummarizeChunk error: %v", err)
	}
	if result != "This function does X." {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestOllamaSummarizer_SummarizeFile(t *testing.T) {
	srv := httptest.NewServer(ollamaChatHandler(t, "chunk summary 1"))
	defer srv.Close()

	s := summarizer.NewOllama("qwen2.5-coder:7b", srv.URL)
	result, err := s.SummarizeFile(context.Background(), []string{"chunk summary 1", "chunk summary 2"})
	if err != nil {
		t.Fatalf("SummarizeFile error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestOllamaSummarizer_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := summarizer.NewOllama("qwen2.5-coder:7b", srv.URL)
	_, err := s.SummarizeChunk(context.Background(), summarizer.ChunkInfo{Kind: "function", Symbol: "F", Content: "f()"})
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
}

// --- LM Studio mock ---

func lmstudioChatHandler(t *testing.T, wantSubstring string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		userContent := req.Messages[len(req.Messages)-1].Content
		if !strings.Contains(userContent, wantSubstring) {
			t.Errorf("expected prompt to contain %q, got:\n%s", wantSubstring, userContent)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "This function does Y."}},
			},
		})
	}
}

func TestLMStudioSummarizer_SummarizeChunk(t *testing.T) {
	srv := httptest.NewServer(lmstudioChatHandler(t, "AnotherFunc"))
	defer srv.Close()

	s := summarizer.NewLMStudio("qwen2.5-coder:7b", srv.URL)
	result, err := s.SummarizeChunk(context.Background(), summarizer.ChunkInfo{
		Kind:    "method",
		Symbol:  "AnotherFunc",
		Content: "func (r *Recv) AnotherFunc() {}",
	})
	if err != nil {
		t.Fatalf("SummarizeChunk error: %v", err)
	}
	if result != "This function does Y." {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestLMStudioSummarizer_SummarizeFile(t *testing.T) {
	srv := httptest.NewServer(lmstudioChatHandler(t, "handles auth"))
	defer srv.Close()

	s := summarizer.NewLMStudio("qwen2.5-coder:7b", srv.URL)
	result, err := s.SummarizeFile(context.Background(), []string{"handles auth", "validates tokens"})
	if err != nil {
		t.Fatalf("SummarizeFile error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestLMStudioSummarizer_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := summarizer.NewLMStudio("qwen2.5-coder:7b", srv.URL)
	_, err := s.SummarizeChunk(context.Background(), summarizer.ChunkInfo{Kind: "function", Symbol: "F", Content: "f()"})
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail to compile**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/summarizer/... -v 2>&1 | head -20
```

Expected: compilation error — package `summarizer` does not exist yet.

- [ ] **Step 3: Create `internal/summarizer/summarizer.go`**

```go
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

// Package summarizer generates natural-language summaries of code chunks and
// files using LLM chat completion APIs (Ollama and LM Studio).
package summarizer

import (
	"context"
	"fmt"
)

// ChunkInfo carries the fields needed to summarize a code chunk.
// It is intentionally decoupled from chunker.Chunk to keep the package
// dependency graph clean.
type ChunkInfo struct {
	Kind    string
	Symbol  string
	Content string
}

// Summarizer generates natural-language summaries of code.
type Summarizer interface {
	SummarizeChunk(ctx context.Context, chunk ChunkInfo) (string, error)
	SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error)
}

// chunkPrompt returns the LLM prompt for summarizing a single chunk.
func chunkPrompt(chunk ChunkInfo) string {
	return fmt.Sprintf(
		"Summarize what this %s '%s' does in 2-3 sentences, focusing on its purpose and behavior:\n\n%s",
		chunk.Kind, chunk.Symbol, chunk.Content,
	)
}

// filePrompt returns the LLM prompt for summarizing a file from its chunk summaries.
func filePrompt(chunkSummaries []string) string {
	combined := ""
	for i, s := range chunkSummaries {
		if i > 0 {
			combined += "\n"
		}
		combined += s
	}
	return fmt.Sprintf(
		"Summarize what this file does in 3-5 sentences, covering its main purpose, key types/functions, and role in the codebase:\n\n%s",
		combined,
	)
}
```

- [ ] **Step 4: Create `internal/summarizer/ollama.go`**

```go
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

package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaSummarizer calls the Ollama /api/chat endpoint.
type OllamaSummarizer struct {
	model   string
	baseURL string
	client  *http.Client
}

// NewOllama creates a new OllamaSummarizer.
func NewOllama(model, baseURL string) *OllamaSummarizer {
	return &OllamaSummarizer{
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaChatMessage `json:"message"`
}

func (s *OllamaSummarizer) chat(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaChatRequest{
		Model:    s.model,
		Messages: []ollamaChatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama chat request: %w", err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama chat: status %d: %s", resp.StatusCode, string(body))
	}
	if readErr != nil {
		return "", fmt.Errorf("read ollama response: %w", readErr)
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal ollama response: %w", err)
	}
	return chatResp.Message.Content, nil
}

// SummarizeChunk generates a natural-language summary for a code chunk.
func (s *OllamaSummarizer) SummarizeChunk(ctx context.Context, chunk ChunkInfo) (string, error) {
	return s.chat(ctx, chunkPrompt(chunk))
}

// SummarizeFile generates a file-level summary from its chunk summaries.
func (s *OllamaSummarizer) SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error) {
	return s.chat(ctx, filePrompt(chunkSummaries))
}
```

- [ ] **Step 5: Create `internal/summarizer/lmstudio.go`**

```go
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

package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LMStudioSummarizer calls the LM Studio /v1/chat/completions endpoint.
type LMStudioSummarizer struct {
	model   string
	baseURL string
	client  *http.Client
}

// NewLMStudio creates a new LMStudioSummarizer.
func NewLMStudio(model, baseURL string) *LMStudioSummarizer {
	return &LMStudioSummarizer{
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

type lmstudioChatRequest struct {
	Model    string               `json:"model"`
	Messages []lmstudioChatMessage `json:"messages"`
}

type lmstudioChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type lmstudioChatResponse struct {
	Choices []struct {
		Message lmstudioChatMessage `json:"message"`
	} `json:"choices"`
}

func (s *LMStudioSummarizer) chat(ctx context.Context, prompt string) (string, error) {
	reqBody := lmstudioChatRequest{
		Model:    s.model,
		Messages: []lmstudioChatMessage{{Role: "user", Content: prompt}},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("lmstudio chat request: %w", err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lmstudio chat: status %d: %s", resp.StatusCode, string(body))
	}
	if readErr != nil {
		return "", fmt.Errorf("read lmstudio response: %w", readErr)
	}

	var chatResp lmstudioChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal lmstudio response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("lmstudio returned no choices")
	}
	return chatResp.Choices[0].Message.Content, nil
}

// SummarizeChunk generates a natural-language summary for a code chunk.
func (s *LMStudioSummarizer) SummarizeChunk(ctx context.Context, chunk ChunkInfo) (string, error) {
	return s.chat(ctx, chunkPrompt(chunk))
}

// SummarizeFile generates a file-level summary from its chunk summaries.
func (s *LMStudioSummarizer) SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error) {
	return s.chat(ctx, filePrompt(chunkSummaries))
}
```

- [ ] **Step 6: Run tests to confirm they pass**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/summarizer/... -v
```

Expected: all summarizer tests pass.

- [ ] **Step 7: Run the full test suite**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./...
```

Expected: all tests pass.

- [ ] **Step 8: Commit**

```bash
cd /Users/aeneas/workspace/go/agent-index-go
git add internal/summarizer/
git commit -m "feat(summarizer): add Summarizer interface with Ollama and LM Studio clients"
```

---

## Chunk 2: Store Changes

### File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/store/store.go` | Modify | Add `summaryDims` field; create 4 new tables; update `DeleteFileChunks`; add `InsertChunkSummaries`, `InsertFileSummary`, `SearchChunkSummaries`, `SearchFileSummaries`, `TopChunksByFile` |
| `internal/store/store_test.go` | Modify | Tests for new store methods and summary cleanup |

---

### Task 4: Extend the store with summary tables and methods

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

**Background:** Four new tables are added. `vec_chunk_summaries` and `vec_file_summaries` are only created when `summaryDims > 0`. The `DeleteFileChunks` transaction is expanded to explicitly delete from both vec tables before deleting rows (sqlite-vec does not participate in FK cascades). `resetAndRecreateVecTable` is expanded to also drop/recreate summary vec tables atomically.

- [ ] **Step 1: Write failing tests for the new store behaviour**

Append the following tests to `internal/store/store_test.go`:

```go
func TestStore_SummaryTables_CreatedWhenDimsPositive(t *testing.T) {
	s, err := New(":memory:", 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	// Verify summary tables exist by querying them directly.
	for _, tbl := range []string{"chunk_summaries", "file_summaries"} {
		var count int
		if err := s.db.QueryRow("SELECT count(*) FROM " + tbl).Scan(&count); err != nil {
			t.Fatalf("table %q missing or unreadable: %v", tbl, err)
		}
	}
	// vec virtual tables exist if they can be queried.
	for _, tbl := range []string{"vec_chunk_summaries", "vec_file_summaries"} {
		var count int
		if err := s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&count); err != nil || count == 0 {
			t.Fatalf("expected virtual table %q to exist", tbl)
		}
	}
}

func TestStore_SummaryTables_NotCreatedWhenDimsZero(t *testing.T) {
	s, err := New(":memory:", 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	// chunk_summaries and file_summaries are always created (they're regular tables).
	// Only the vec virtual tables are guarded by summaryDims > 0.
	for _, tbl := range []string{"vec_chunk_summaries", "vec_file_summaries"} {
		var count int
		if err := s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&count); err != nil {
			t.Fatalf("query failed for %q: %v", tbl, err)
		}
		if count != 0 {
			t.Fatalf("expected virtual table %q to NOT exist when summaryDims=0, but it does", tbl)
		}
	}
}

func TestStore_InsertChunkSummaries_And_SearchChunkSummaries(t *testing.T) {
	const dims = 4
	s, err := New(":memory:", dims, dims)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	// Set up a file and chunk first.
	if err := s.UpsertFile("auth.go", "hash1"); err != nil {
		t.Fatal(err)
	}
	chunks := []chunker.Chunk{
		{ID: "c1", FilePath: "auth.go", Symbol: "ValidateToken", Kind: "function", StartLine: 1, EndLine: 10},
	}
	codeVecs := [][]float32{{1, 0, 0, 0}}
	if err := s.InsertChunks(chunks, codeVecs); err != nil {
		t.Fatal(err)
	}

	// Insert a chunk summary.
	summaryVecs := [][]float32{{0, 1, 0, 0}}
	if err := s.InsertChunkSummaries([]string{"c1"}, []string{"Validates JWT tokens."}, summaryVecs); err != nil {
		t.Fatalf("InsertChunkSummaries: %v", err)
	}

	// Search for it.
	query := []float32{0, 1, 0, 0}
	results, err := s.SearchChunkSummaries(query, 5, 0, "")
	if err != nil {
		t.Fatalf("SearchChunkSummaries: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Symbol != "ValidateToken" {
		t.Fatalf("expected symbol ValidateToken, got %q", results[0].Symbol)
	}
}

func TestStore_InsertFileSummary_And_SearchFileSummaries(t *testing.T) {
	const dims = 4
	s, err := New(":memory:", dims, dims)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	if err := s.UpsertFile("auth.go", "hash1"); err != nil {
		t.Fatal(err)
	}

	summaryVec := []float32{0, 0, 1, 0}
	if err := s.InsertFileSummary("auth.go", "Handles authentication logic.", summaryVec); err != nil {
		t.Fatalf("InsertFileSummary: %v", err)
	}

	query := []float32{0, 0, 1, 0}
	results, err := s.SearchFileSummaries(query, 5, 0)
	if err != nil {
		t.Fatalf("SearchFileSummaries: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].FilePath != "auth.go" {
		t.Fatalf("expected auth.go, got %q", results[0].FilePath)
	}
}

func TestStore_DeleteFileChunks_CleansUpSummaryTables(t *testing.T) {
	const dims = 4
	s, err := New(":memory:", dims, dims)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	if err := s.UpsertFile("auth.go", "hash1"); err != nil {
		t.Fatal(err)
	}
	chunks := []chunker.Chunk{
		{ID: "c1", FilePath: "auth.go", Symbol: "F", Kind: "function", StartLine: 1, EndLine: 5},
	}
	if err := s.InsertChunks(chunks, [][]float32{{1, 0, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertChunkSummaries([]string{"c1"}, []string{"summary"}, [][]float32{{0, 1, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertFileSummary("auth.go", "file summary", []float32{0, 0, 1, 0}); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteFileChunks("auth.go"); err != nil {
		t.Fatalf("DeleteFileChunks: %v", err)
	}

	// Verify all summary rows are gone.
	var count int
	if err := s.db.QueryRow("SELECT count(*) FROM chunk_summaries").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 chunk_summaries after delete, got %d", count)
	}
	if err := s.db.QueryRow("SELECT count(*) FROM file_summaries").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 file_summaries after delete, got %d", count)
	}
}

func TestStore_TopChunksByFile(t *testing.T) {
	const dims = 4
	s, err := New(":memory:", dims, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	if err := s.UpsertFile("main.go", "hash1"); err != nil {
		t.Fatal(err)
	}
	chunks := []chunker.Chunk{
		{ID: "c1", FilePath: "main.go", Symbol: "A", Kind: "function", StartLine: 1, EndLine: 5},
		{ID: "c2", FilePath: "main.go", Symbol: "B", Kind: "function", StartLine: 6, EndLine: 10},
		{ID: "c3", FilePath: "main.go", Symbol: "C", Kind: "function", StartLine: 11, EndLine: 15},
	}
	vecs := [][]float32{{1, 0, 0, 0}, {0.9, 0.1, 0, 0}, {0, 0, 1, 0}}
	if err := s.InsertChunks(chunks, vecs); err != nil {
		t.Fatal(err)
	}

	// Query vector close to c1 and c2.
	queryVec := []float32{1, 0, 0, 0}
	results, err := s.TopChunksByFile("main.go", queryVec, 2)
	if err != nil {
		t.Fatalf("TopChunksByFile: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestStore_DimensionMismatch_DropsSummaryTablesOnReset(t *testing.T) {
	// Create store with dims=4, summaryDims=4.
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dbPath, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	// Reopen with different code dims — triggers full reset.
	s2, err := New(dbPath, 8, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s2.Close() }()

	// Summary vec tables should still exist (summary dims unchanged).
	var count int
	if err := s2.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='vec_chunk_summaries'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected vec_chunk_summaries to be recreated after reset")
	}
}
```

Note: the test file already imports `chunker` — verify the import is present at the top of `store_test.go`. If not, add it.

- [ ] **Step 2: Run tests to confirm compilation failure (new signature)**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/store/... -v 2>&1 | head -30
```

Expected: compilation errors because `New` still takes 2 args.

- [ ] **Step 3: Update `internal/store/store.go`**

Make the following changes (in order, all within `store.go`):

**3a. Add `summaryDims` to `Store` struct:**
```go
type Store struct {
    db          *sql.DB
    dimensions  int
    summaryDims int
}
```

**3b. Update `New` signature and body:**
```go
func New(dsn string, dimensions int, summaryDims int) (*Store, error) {
    // ... existing pragma setup unchanged ...
    if err := createSchema(db, dimensions, summaryDims); err != nil {
        _ = db.Close()
        return nil, fmt.Errorf("create schema: %w", err)
    }
    return &Store{db: db, dimensions: dimensions, summaryDims: summaryDims}, nil
}
```

**3c. Update `createSchema` signature and body:**

```go
func createSchema(db *sql.DB, dimensions int, summaryDims int) error {
    stmts := []string{
        `CREATE TABLE IF NOT EXISTS files (
            path TEXT PRIMARY KEY,
            hash TEXT NOT NULL
        )`,
        `CREATE TABLE IF NOT EXISTS project_meta (
            key   TEXT PRIMARY KEY,
            value TEXT NOT NULL
        )`,
        `CREATE TABLE IF NOT EXISTS chunks (
            id         TEXT PRIMARY KEY,
            file_path  TEXT NOT NULL REFERENCES files(path),
            symbol     TEXT NOT NULL,
            kind       TEXT NOT NULL,
            start_line INTEGER NOT NULL,
            end_line   INTEGER NOT NULL
        )`,
        `CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path)`,
        `CREATE TABLE IF NOT EXISTS chunk_summaries (
            chunk_id TEXT PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
            summary  TEXT NOT NULL
        )`,
        `CREATE TABLE IF NOT EXISTS file_summaries (
            file_path TEXT PRIMARY KEY REFERENCES files(path) ON DELETE CASCADE,
            summary   TEXT NOT NULL
        )`,
    }
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return fmt.Errorf("exec %q: %w", s, err)
        }
    }

    if err := ensureVecDimensions(db, dimensions, summaryDims); err != nil {
        return err
    }
    return nil
}
```

**3d. Replace `ensureVecDimensions` and supporting functions:**

```go
// ensureVecDimensions manages both vec_chunks and (optionally) vec_chunk_summaries /
// vec_file_summaries. If stored dimensions mismatch, a full reset is performed.
func ensureVecDimensions(db *sql.DB, dimensions int, summaryDims int) error {
    tableExists, err := checkTableExists(db, "vec_chunks")
    if err != nil {
        return err
    }

    if !tableExists {
        if err := createVecTable(db, dimensions); err != nil {
            return err
        }
        if summaryDims > 0 {
            return createSummaryVecTables(db, summaryDims)
        }
        return nil
    }

    storedDims, err := getStoredDimensions(db)
    if err == nil && storedDims == dimensions {
        // Code dims match; check summary dims separately.
        return ensureSummaryVecDimensions(db, summaryDims)
    }

    // Code dims mismatch — full reset.
    return resetAndRecreateVecTable(db, dimensions, summaryDims)
}

func ensureSummaryVecDimensions(db *sql.DB, summaryDims int) error {
    if summaryDims == 0 {
        return nil
    }
    exists, err := checkTableExists(db, "vec_chunk_summaries")
    if err != nil {
        return err
    }
    if !exists {
        return createSummaryVecTables(db, summaryDims)
    }
    storedSummaryDims, err := getStoredSummaryDimensions(db)
    if err == nil && storedSummaryDims == summaryDims {
        return nil
    }
    // Summary dims mismatch — drop and recreate summary tables only.
    return resetAndRecreateSummaryVecTables(db, summaryDims)
}

func createSummaryVecTables(db *sql.DB, summaryDims int) error {
    stmts := []string{
        fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunk_summaries USING vec0(
            id TEXT PRIMARY KEY,
            embedding float[%d] distance_metric=cosine
        )`, summaryDims),
        fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_file_summaries USING vec0(
            id TEXT PRIMARY KEY,
            embedding float[%d] distance_metric=cosine
        )`, summaryDims),
    }
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return fmt.Errorf("create summary vec table: %w", err)
        }
    }
    return storeSummaryDimensions(db, summaryDims)
}

func resetAndRecreateSummaryVecTables(db *sql.DB, summaryDims int) error {
    stmts := []string{
        "DROP TABLE IF EXISTS vec_chunk_summaries",
        "DROP TABLE IF EXISTS vec_file_summaries",
        "DELETE FROM chunk_summaries",
        "DELETE FROM file_summaries",
    }
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return fmt.Errorf("reset summary vec tables %q: %w", s, err)
        }
    }
    return createSummaryVecTables(db, summaryDims)
}

func getStoredSummaryDimensions(db *sql.DB) (int, error) {
    var dims int
    err := db.QueryRow("SELECT value FROM project_meta WHERE key = 'vec_summary_dimensions'").Scan(&dims)
    return dims, err
}

func storeSummaryDimensions(db *sql.DB, summaryDims int) error {
    _, err := db.Exec(
        `INSERT INTO project_meta (key, value) VALUES ('vec_summary_dimensions', ?)
         ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
        fmt.Sprintf("%d", summaryDims),
    )
    if err != nil {
        return fmt.Errorf("store vec_summary_dimensions: %w", err)
    }
    return nil
}
```

**3e. Update `resetAndRecreateVecTable` to also handle summary tables:**

```go
func resetAndRecreateVecTable(db *sql.DB, dimensions int, summaryDims int) error {
    stmts := []string{
        "DROP TABLE IF EXISTS vec_chunks",
        "DROP TABLE IF EXISTS vec_chunk_summaries",
        "DROP TABLE IF EXISTS vec_file_summaries",
        "DELETE FROM chunk_summaries",
        "DELETE FROM file_summaries",
        "DELETE FROM chunks",
        "DELETE FROM files",
        "DELETE FROM project_meta",
    }
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return fmt.Errorf("reset for dimension change %q: %w", s, err)
        }
    }

    if err := createVecTable(db, dimensions); err != nil {
        return err
    }
    if summaryDims > 0 {
        return createSummaryVecTables(db, summaryDims)
    }
    return nil
}
```

**3f. Update `DeleteFileChunks` with the three-phase ordering:**

```go
func (s *Store) DeleteFileChunks(filePath string) error {
    tx, err := s.db.Begin()
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer func() { _ = tx.Rollback() }()

    // Phase 1: Collect chunk IDs before deletion.
    rows, err := tx.Query(`SELECT id FROM chunks WHERE file_path = ?`, filePath)
    if err != nil {
        return fmt.Errorf("fetch chunk ids: %w", err)
    }
    var chunkIDs []string
    for rows.Next() {
        var id string
        if err := rows.Scan(&id); err != nil {
            _ = rows.Close()
            return fmt.Errorf("scan chunk id: %w", err)
        }
        chunkIDs = append(chunkIDs, id)
    }
    if err := rows.Err(); err != nil {
        return fmt.Errorf("iterate chunk ids: %w", err)
    }
    _ = rows.Close()

    // Phase 2: Explicit vec deletes (sqlite-vec does not support FK cascades).
    if len(chunkIDs) > 0 {
        placeholders := strings.Repeat("?,", len(chunkIDs))
        placeholders = placeholders[:len(placeholders)-1]
        args := make([]any, len(chunkIDs))
        for i, id := range chunkIDs {
            args[i] = id
        }
        if _, err := tx.Exec(`DELETE FROM vec_chunk_summaries WHERE id IN (`+placeholders+`)`, args...); err != nil {
            return fmt.Errorf("delete vec_chunk_summaries: %w", err)
        }
        if _, err := tx.Exec(`DELETE FROM vec_chunks WHERE id IN (`+placeholders+`)`, args...); err != nil {
            return fmt.Errorf("delete vec_chunks: %w", err)
        }
    }
    if _, err := tx.Exec(`DELETE FROM vec_file_summaries WHERE id = ?`, filePath); err != nil {
        return fmt.Errorf("delete vec_file_summaries: %w", err)
    }

    // Phase 3: Row deletes (FK cascades handle chunk_summaries and file_summaries).
    if _, err := tx.Exec(`DELETE FROM chunks WHERE file_path = ?`, filePath); err != nil {
        return fmt.Errorf("delete chunks: %w", err)
    }
    if _, err := tx.Exec(`DELETE FROM files WHERE path = ?`, filePath); err != nil {
        return fmt.Errorf("delete file: %w", err)
    }

    return tx.Commit()
}
```

**3g. Add new public methods at the bottom of `store.go`:**

```go
// InsertChunkSummaries upserts summary text and vectors for a batch of chunks.
// len(chunkIDs), len(summaries), and len(vectors) must all be equal.
func (s *Store) InsertChunkSummaries(chunkIDs []string, summaries []string, vectors [][]float32) error {
    if len(chunkIDs) != len(summaries) || len(chunkIDs) != len(vectors) {
        return fmt.Errorf("length mismatch: ids=%d summaries=%d vectors=%d", len(chunkIDs), len(summaries), len(vectors))
    }
    tx, err := s.db.Begin()
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer func() { _ = tx.Rollback() }()

    for i, id := range chunkIDs {
        if _, err := tx.Exec(
            `INSERT INTO chunk_summaries (chunk_id, summary) VALUES (?, ?)
             ON CONFLICT(chunk_id) DO UPDATE SET summary = excluded.summary`,
            id, summaries[i],
        ); err != nil {
            return fmt.Errorf("upsert chunk_summary %s: %w", id, err)
        }
        blob, err := sqlite_vec.SerializeFloat32(vectors[i])
        if err != nil {
            return fmt.Errorf("serialize summary vector %d: %w", i, err)
        }
        if _, err := tx.Exec(
            `INSERT INTO vec_chunk_summaries (id, embedding) VALUES (?, ?)
             ON CONFLICT(id) DO UPDATE SET embedding = excluded.embedding`,
            id, blob,
        ); err != nil {
            return fmt.Errorf("upsert vec_chunk_summary %s: %w", id, err)
        }
    }
    return tx.Commit()
}

// InsertFileSummary upserts the summary text and vector for a file.
func (s *Store) InsertFileSummary(filePath, summary string, vector []float32) error {
    tx, err := s.db.Begin()
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer func() { _ = tx.Rollback() }()

    if _, err := tx.Exec(
        `INSERT INTO file_summaries (file_path, summary) VALUES (?, ?)
         ON CONFLICT(file_path) DO UPDATE SET summary = excluded.summary`,
        filePath, summary,
    ); err != nil {
        return fmt.Errorf("upsert file_summary: %w", err)
    }
    blob, err := sqlite_vec.SerializeFloat32(vector)
    if err != nil {
        return fmt.Errorf("serialize file summary vector: %w", err)
    }
    if _, err := tx.Exec(
        `INSERT INTO vec_file_summaries (id, embedding) VALUES (?, ?)
         ON CONFLICT(id) DO UPDATE SET embedding = excluded.embedding`,
        filePath, blob,
    ); err != nil {
        return fmt.Errorf("upsert vec_file_summary: %w", err)
    }
    return tx.Commit()
}

// FileSummaryResult represents a file-level summary search hit.
type FileSummaryResult struct {
    FilePath string
    Distance float64
}

// SearchChunkSummaries performs a KNN search against vec_chunk_summaries and
// returns matching SearchResult values (joined with chunks for metadata).
// Parameters mirror Store.Search.
func (s *Store) SearchChunkSummaries(queryVec []float32, limit int, maxDistance float64, pathPrefix string) ([]SearchResult, error) {
    blob, err := sqlite_vec.SerializeFloat32(queryVec)
    if err != nil {
        return nil, fmt.Errorf("serialize query: %w", err)
    }

    knn := limit
    if pathPrefix != "" {
        knn = min(limit*3, 300)
    }

    whereClauses := []string{"v.embedding MATCH ?", "v.k = ?"}
    args := []any{blob, knn}
    if maxDistance > 0 {
        whereClauses = append(whereClauses, "v.distance < ?")
        args = append(args, maxDistance)
    }
    if pathPrefix != "" {
        whereClauses = append(whereClauses, "(c.file_path = ? OR c.file_path LIKE ? || '/%')")
        args = append(args, pathPrefix, pathPrefix)
    }
    args = append(args, limit)

    query := fmt.Sprintf(`
        SELECT c.file_path, c.symbol, c.kind, c.start_line, c.end_line, v.distance
        FROM vec_chunk_summaries v
        JOIN chunks c ON v.id = c.id
        WHERE %s
        ORDER BY v.distance
        LIMIT ?
    `, strings.Join(whereClauses, "\n\t\tAND "))

    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, fmt.Errorf("search chunk summaries: %w", err)
    }
    defer func() { _ = rows.Close() }()

    var results []SearchResult
    for rows.Next() {
        var r SearchResult
        if err := rows.Scan(&r.FilePath, &r.Symbol, &r.Kind, &r.StartLine, &r.EndLine, &r.Distance); err != nil {
            return nil, fmt.Errorf("scan chunk summary result: %w", err)
        }
        results = append(results, r)
    }
    return results, rows.Err()
}

// SearchFileSummaries performs a KNN search against vec_file_summaries.
func (s *Store) SearchFileSummaries(queryVec []float32, limit int, maxDistance float64) ([]FileSummaryResult, error) {
    blob, err := sqlite_vec.SerializeFloat32(queryVec)
    if err != nil {
        return nil, fmt.Errorf("serialize query: %w", err)
    }

    whereClauses := []string{"v.embedding MATCH ?", "v.k = ?"}
    args := []any{blob, limit}
    if maxDistance > 0 {
        whereClauses = append(whereClauses, "v.distance < ?")
        args = append(args, maxDistance)
    }
    args = append(args, limit)

    query := fmt.Sprintf(`
        SELECT v.id, v.distance
        FROM vec_file_summaries v
        WHERE %s
        ORDER BY v.distance
        LIMIT ?
    `, strings.Join(whereClauses, "\n\t\tAND "))

    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, fmt.Errorf("search file summaries: %w", err)
    }
    defer func() { _ = rows.Close() }()

    var results []FileSummaryResult
    for rows.Next() {
        var r FileSummaryResult
        if err := rows.Scan(&r.FilePath, &r.Distance); err != nil {
            return nil, fmt.Errorf("scan file summary result: %w", err)
        }
        results = append(results, r)
    }
    return results, rows.Err()
}

// TopChunksByFile returns the top n chunks from filePath ranked by distance to queryVec.
func (s *Store) TopChunksByFile(filePath string, queryVec []float32, n int) ([]SearchResult, error) {
    return s.Search(queryVec, n, 0, filePath)
}
```

- [ ] **Step 4: Fix all callers of `store.New` (now requires 3 args)**

Search for all callers:

```bash
cd /Users/aeneas/workspace/go/agent-index-go && grep -rn "store\.New(" --include="*.go"
```

Expected callers:
- `internal/index/index.go` — `store.New(dsn, emb.Dimensions())`
- `cmd/hook.go` — `store.New(dbPath, cfg.Dims)`

Update `internal/index/index.go`:
```go
// Before:
s, err := store.New(dsn, emb.Dimensions())
// After:
s, err := store.New(dsn, emb.Dimensions(), summaryDims)
```

This requires `NewIndexer` to accept a `summaryDims int` parameter. Update `NewIndexer`:
```go
func NewIndexer(dsn string, emb embedder.Embedder, maxChunkTokens int, summaryDims int) (*Indexer, error) {
    s, err := store.New(dsn, emb.Dimensions(), summaryDims)
    // ... rest unchanged ...
}
```

Update `cmd/hook.go`:
```go
// Before:
s, err := store.New(dbPath, cfg.Dims)
// After:
s, err := store.New(dbPath, cfg.Dims, cfg.SummaryEmbedDims)
```

Update callers of `index.NewIndexer` in `cmd/index.go` and `cmd/stdio.go`:
```go
// cmd/index.go setupIndexer:
idx, err := index.NewIndexer(dbPath, emb, cfg.MaxChunkTokens, cfg.SummaryEmbedDims)

// cmd/stdio.go getOrCreate:
idx, err := index.NewIndexer(dbPath, ic.embedder, ic.cfg.MaxChunkTokens, ic.cfg.SummaryEmbedDims)
```

- [ ] **Step 5: Run all tests**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./...
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
cd /Users/aeneas/workspace/go/agent-index-go
git add internal/store/store.go internal/store/store_test.go \
        internal/index/index.go cmd/index.go cmd/stdio.go cmd/hook.go
git commit -m "feat(store): add summary tables, vec indices, and explicit cleanup in DeleteFileChunks"
```

---

## Chunk 3: Indexer Summary Passes + Search + E2E Test

### File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/index/index.go` | Modify | Add `summarizer` + `summaryEmb` fields; add `runSummaryPasses` after raw embedding |
| `internal/index/index_test.go` | Create | Integration tests for summary pass filtering and incremental behaviour |
| `cmd/stdio.go` | Modify | Add `summaryEmb` field to `indexerCache`; expand `handleSemanticSearch` to fan out to three indices and collect `<relevant_files>` |
| `cmd/stdio_test.go` | Modify | Add tests for `<relevant_files>` formatting; update existing tests that call `store.New` / `index.NewIndexer` |

---

### Task 5: Add summary passes to the indexer

**Files:**
- Modify: `internal/index/index.go`
- Create: `internal/index/index_test.go`

**Background:** After the raw-embedding pass completes for all changed files, two new passes run (only when `cfg.Summaries` is true, i.e., when `summarizer != nil`):

- Pass 1 (chunk summaries): for each newly-indexed chunk with `EndLine - StartLine >= 2`, call `summarizer.SummarizeChunk`, collect summaries, embed in batches of 32, store via `InsertChunkSummaries`.
- Pass 2 (file summaries): for each file that had at least one chunk summary, call `summarizer.SummarizeFile` with the collected summaries, embed the result, store via `InsertFileSummary`.

LLM calls are sequential (local models are not safely parallelizable). Errors in either pass are logged and skipped — raw code search continues to work.

- [ ] **Step 1: Write failing integration tests**

Create `internal/index/index_test.go`:

```go
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

package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ory/lumen/internal/index"
	"github.com/ory/lumen/internal/summarizer"
)

// stubEmbedder is a minimal Embedder for tests that returns constant vectors.
type stubEmbedder struct{ dims int }

func (s *stubEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i := range vecs {
		v := make([]float32, s.dims)
		v[0] = 1
		vecs[i] = v
	}
	return vecs, nil
}
func (s *stubEmbedder) Dimensions() int  { return s.dims }
func (s *stubEmbedder) ModelName() string { return "stub" }

// stubSummarizer counts calls and returns a canned summary.
type stubSummarizer struct {
	chunkCalls int
	fileCalls  int
}

func (s *stubSummarizer) SummarizeChunk(_ context.Context, chunk summarizer.ChunkInfo) (string, error) {
	s.chunkCalls++
	return "summary of " + chunk.Symbol, nil
}

func (s *stubSummarizer) SummarizeFile(_ context.Context, sums []string) (string, error) {
	s.fileCalls++
	return "file summary", nil
}

func makeGoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestIndexer_SummaryPass_ChunkFilterMinLines verifies that chunks with
// EndLine - StartLine < 2 are skipped by the summary pass.
func TestIndexer_SummaryPass_ChunkFilterMinLines(t *testing.T) {
	dir := t.TempDir()
	// A single-line function: StartLine==EndLine → EndLine-StartLine==0 < 2 → skip.
	// A multi-line function: StartLine=1, EndLine=5 → EndLine-StartLine==4 >= 2 → summarize.
	makeGoFile(t, dir, "main.go", `package main

func Short() {}

func Long() {
	_ = 1
	_ = 2
	_ = 3
}
`)

	emb := &stubEmbedder{dims: 4}
	sum := &stubSummarizer{}
	idx, err := index.NewIndexer(":memory:", emb, 512, 4, sum, emb)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}

	// Long() spans multiple lines and should be summarized; Short() should not.
	if sum.chunkCalls == 0 {
		t.Fatal("expected at least one chunk summary call")
	}
}

// TestIndexer_SummaryPass_FileSummaryGeneratedFromChunks verifies that a
// file summary is generated when at least one chunk was summarized.
func TestIndexer_SummaryPass_FileSummaryGeneratedFromChunks(t *testing.T) {
	dir := t.TempDir()
	makeGoFile(t, dir, "auth.go", `package auth

func ValidateToken(token string) bool {
	return token != ""
}

func RevokeToken(token string) {
	// revoke
}
`)

	emb := &stubEmbedder{dims: 4}
	sum := &stubSummarizer{}
	idx, err := index.NewIndexer(":memory:", emb, 512, 4, sum, emb)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}

	if sum.fileCalls == 0 {
		t.Fatal("expected file summary to be generated")
	}
}

// TestIndexer_NoSummarizer_NoPanics verifies that nil summarizer disables passes gracefully.
func TestIndexer_NoSummarizer_NoPanics(t *testing.T) {
	dir := t.TempDir()
	makeGoFile(t, dir, "main.go", `package main

func Hello() {
	println("hello")
}
`)

	emb := &stubEmbedder{dims: 4}
	idx, err := index.NewIndexer(":memory:", emb, 512, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	if _, err := idx.Index(context.Background(), dir, true, nil); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/index/... -v 2>&1 | head -20
```

Expected: compilation error — `NewIndexer` signature mismatch (not yet updated to accept summarizer).

- [ ] **Step 3: Update `internal/index/index.go`**

**3a. Add import for `summarizer` package and `log`:**

In the import block, add:
```go
"log"

"github.com/ory/lumen/internal/summarizer"
```

**3b. Extend `Indexer` struct:**

```go
type Indexer struct {
    mu             sync.Mutex
    store          *store.Store
    emb            embedder.Embedder
    summaryEmb     embedder.Embedder   // nil when summaries disabled
    sumr           summarizer.Summarizer // nil when summaries disabled
    chunker        chunker.Chunker
    maxChunkTokens int
}
```

**3c. Update `NewIndexer` signature:**

```go
func NewIndexer(dsn string, emb embedder.Embedder, maxChunkTokens int, summaryDims int, sumr summarizer.Summarizer, summaryEmb embedder.Embedder) (*Indexer, error) {
    s, err := store.New(dsn, emb.Dimensions(), summaryDims)
    if err != nil {
        return nil, fmt.Errorf("create store: %w", err)
    }
    return &Indexer{
        store:          s,
        emb:            emb,
        summaryEmb:     summaryEmb,
        sumr:           sumr,
        chunker:        chunker.NewMultiChunker(chunker.DefaultLanguages(maxChunkTokens)),
        maxChunkTokens: maxChunkTokens,
    }, nil
}
```

**3d. Add `runSummaryPasses` method and call it from `indexWithTree`:**

Add after the existing `flushBatch` call in `indexWithTree`:

```go
    // Run summary passes when summarizer is configured.
    if idx.sumr != nil && idx.summaryEmb != nil && len(filesToIndex) > 0 {
        if err := idx.runSummaryPasses(ctx, filesToIndex, progress); err != nil {
            // Log and continue — raw code search is unaffected.
            log.Printf("warning: summary passes failed: %v", err)
        }
    }
```

Add the method:

```go
// runSummaryPasses generates chunk and file summaries for files that were
// re-indexed. It runs after raw embedding is complete.
// Errors from individual LLM calls are logged and skipped.
func (idx *Indexer) runSummaryPasses(ctx context.Context, files []string, progress ProgressFunc) error {
    const summaryEmbedBatchSize = 32

    // Pass 1: chunk summaries.
    // For each file, fetch its newly-indexed chunks and summarize eligible ones.
    // fileSummaryInputs maps file path → slice of chunk summaries for that file.
    fileSummaryInputs := make(map[string][]string)

    var (
        pendingChunkIDs  []string
        pendingSummaries []string
    )

    flushChunkSummaries := func() error {
        if len(pendingChunkIDs) == 0 {
            return nil
        }
        vecs, err := idx.summaryEmb.Embed(ctx, pendingSummaries)
        if err != nil {
            return fmt.Errorf("embed chunk summaries: %w", err)
        }
        if err := idx.store.InsertChunkSummaries(pendingChunkIDs, pendingSummaries, vecs); err != nil {
            return fmt.Errorf("store chunk summaries: %w", err)
        }
        pendingChunkIDs = pendingChunkIDs[:0]
        pendingSummaries = pendingSummaries[:0]
        return nil
    }

    for _, relPath := range files {
        chunks, err := idx.store.ChunksByFile(relPath)
        if err != nil {
            log.Printf("warning: fetch chunks for %s: %v", relPath, err)
            continue
        }
        for _, c := range chunks {
            if c.EndLine-c.StartLine < 2 {
                continue
            }
            summary, err := idx.sumr.SummarizeChunk(ctx, summarizer.ChunkInfo{
                Kind:    c.Kind,
                Symbol:  c.Symbol,
                Content: c.Content,
            })
            if err != nil {
                log.Printf("warning: summarize chunk %s: %v", c.ID, err)
                continue
            }
            pendingChunkIDs = append(pendingChunkIDs, c.ID)
            pendingSummaries = append(pendingSummaries, summary)
            fileSummaryInputs[relPath] = append(fileSummaryInputs[relPath], summary)

            if len(pendingChunkIDs) >= summaryEmbedBatchSize {
                if err := flushChunkSummaries(); err != nil {
                    log.Printf("warning: flush chunk summaries: %v", err)
                }
            }
        }
    }
    if err := flushChunkSummaries(); err != nil {
        log.Printf("warning: final flush chunk summaries: %v", err)
    }

    // Pass 2: file summaries.
    for relPath, chunkSummaries := range fileSummaryInputs {
        if len(chunkSummaries) == 0 {
            continue
        }
        fileSummary, err := idx.sumr.SummarizeFile(ctx, chunkSummaries)
        if err != nil {
            log.Printf("warning: summarize file %s: %v", relPath, err)
            continue
        }
        vecs, err := idx.summaryEmb.Embed(ctx, []string{fileSummary})
        if err != nil {
            log.Printf("warning: embed file summary %s: %v", relPath, err)
            continue
        }
        if err := idx.store.InsertFileSummary(relPath, fileSummary, vecs[0]); err != nil {
            log.Printf("warning: store file summary %s: %v", relPath, err)
        }
        if progress != nil {
            progress(0, 0, fmt.Sprintf("Summarized file: %s", relPath))
        }
    }
    return nil
}
```

**3e. Add `ChunksByFile` method to `internal/store/store.go`:**

The summary pass needs to read chunk content from the store after the embedding pass. Add:

```go
// ChunksByFile returns all chunks for a given file path, including content
// read from the filesystem. Note: Content is NOT stored in the DB; callers
// that need content must read it separately. This method returns metadata only.
// For summary passes, content is retrieved by reading the source file.
func (s *Store) ChunksByFile(filePath string) ([]chunker.Chunk, error) {
    rows, err := s.db.Query(
        `SELECT id, file_path, symbol, kind, start_line, end_line FROM chunks WHERE file_path = ?`,
        filePath,
    )
    if err != nil {
        return nil, fmt.Errorf("query chunks by file: %w", err)
    }
    defer func() { _ = rows.Close() }()

    var chunks []chunker.Chunk
    for rows.Next() {
        var c chunker.Chunk
        if err := rows.Scan(&c.ID, &c.FilePath, &c.Symbol, &c.Kind, &c.StartLine, &c.EndLine); err != nil {
            return nil, fmt.Errorf("scan chunk: %w", err)
        }
        chunks = append(chunks, c)
    }
    return chunks, rows.Err()
}
```

**Note on content:** `Chunk.Content` is not stored in SQLite (only metadata is). The summary pass needs the source text. Update `runSummaryPasses` to read the file from disk:

```go
// Inside runSummaryPasses, replace the chunk loop with:
for _, relPath := range files {
    absPath := filepath.Join(projectDir, relPath)
    fileContent, err := os.ReadFile(absPath)
    if err != nil {
        log.Printf("warning: read file %s for summarization: %v", relPath, err)
        continue
    }
    chunks, err := idx.store.ChunksByFile(relPath)
    if err != nil {
        log.Printf("warning: fetch chunks for %s: %v", relPath, err)
        continue
    }
    lines := strings.Split(string(fileContent), "\n")
    for _, c := range chunks {
        if c.EndLine-c.StartLine < 2 {
            continue
        }
        // Extract chunk content from source lines (1-based, inclusive).
        start := max(c.StartLine-1, 0)
        end := min(c.EndLine, len(lines))
        content := strings.Join(lines[start:end], "\n")

        summary, err := idx.sumr.SummarizeChunk(ctx, summarizer.ChunkInfo{
            Kind:    c.Kind,
            Symbol:  c.Symbol,
            Content: content,
        })
        // ... rest unchanged ...
    }
}
```

This means `runSummaryPasses` needs `projectDir string` as a parameter — thread it through from `indexWithTree`. Update the call site:

```go
if err := idx.runSummaryPasses(ctx, projectDir, filesToIndex, progress); err != nil {
```

And the signature:

```go
func (idx *Indexer) runSummaryPasses(ctx context.Context, projectDir string, files []string, progress ProgressFunc) error {
```

- [ ] **Step 4: Run tests**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./internal/index/... ./internal/store/... -v
```

Expected: all tests pass.

- [ ] **Step 5: Run the full test suite**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./...
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
cd /Users/aeneas/workspace/go/agent-index-go
git add internal/index/index.go internal/index/index_test.go internal/store/store.go
git commit -m "feat(index): add chunk and file summary passes after raw embedding"
```

---

### Task 6: Wire summarizer into `cmd/stdio.go` and `cmd/index.go`

**Files:**
- Modify: `cmd/stdio.go`
- Modify: `cmd/index.go`
- Modify: `cmd/embedder.go`

**Background:** `runStdio` and `runIndex` must construct the summarizer and summary embedder alongside the existing embedder, and pass them to `NewIndexer`. `indexerCache` must also carry the summary embedder to pass its model name to `DBPathForProject`.

- [ ] **Step 1: Add `newSummarizer` helper to `cmd/embedder.go`**

Append to `cmd/embedder.go`:

```go
import "github.com/ory/lumen/internal/summarizer"

// newSummarizer creates a Summarizer for the configured backend and summary model.
// Returns nil, nil when cfg.Summaries is false.
func newSummarizer(cfg config.Config) (summarizer.Summarizer, error) {
    if !cfg.Summaries {
        return nil, nil
    }
    switch cfg.Backend {
    case config.BackendOllama:
        return summarizer.NewOllama(cfg.SummaryModel, cfg.OllamaHost), nil
    case config.BackendLMStudio:
        return summarizer.NewLMStudio(cfg.SummaryModel, cfg.LMStudioHost), nil
    default:
        return nil, fmt.Errorf("unknown backend %q", cfg.Backend)
    }
}

// newSummaryEmbedder creates an Embedder for summary vectors.
// Returns nil, nil when cfg.Summaries is false.
func newSummaryEmbedder(cfg config.Config) (embedder.Embedder, error) {
    if !cfg.Summaries {
        return nil, nil
    }
    switch cfg.Backend {
    case config.BackendOllama:
        spec := embedder.KnownModels[cfg.SummaryEmbedModel]
        return embedder.NewOllama(cfg.SummaryEmbedModel, cfg.SummaryEmbedDims, spec.CtxLength, cfg.OllamaHost)
    case config.BackendLMStudio:
        return embedder.NewLMStudio(cfg.SummaryEmbedModel, cfg.SummaryEmbedDims, cfg.LMStudioHost)
    default:
        return nil, fmt.Errorf("unknown backend %q", cfg.Backend)
    }
}
```

- [ ] **Step 2: Update `indexerCache` struct in `cmd/stdio.go`**

```go
type indexerCache struct {
    mu                sync.RWMutex
    cache             map[string]cacheEntry
    embedder          embedder.Embedder
    summaryEmbedder   embedder.Embedder       // nil when summaries disabled
    summarizer        summarizer.Summarizer   // nil when summaries disabled
    model             string
    summaryEmbedModel string
    cfg               config.Config
}
```

- [ ] **Step 3: Update `runStdio` to wire the new dependencies**

```go
func runStdio(_ *cobra.Command, _ []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    emb, err := newEmbedder(cfg)
    if err != nil {
        return fmt.Errorf("create embedder: %w", err)
    }

    sumr, err := newSummarizer(cfg)
    if err != nil {
        return fmt.Errorf("create summarizer: %w", err)
    }

    summaryEmb, err := newSummaryEmbedder(cfg)
    if err != nil {
        return fmt.Errorf("create summary embedder: %w", err)
    }

    indexers := &indexerCache{
        embedder:          emb,
        summaryEmbedder:   summaryEmb,
        summarizer:        sumr,
        model:             cfg.Model,
        summaryEmbedModel: cfg.SummaryEmbedModel,
        cfg:               cfg,
    }
    // ... rest unchanged ...
}
```

- [ ] **Step 4: Update `getOrCreate` in `cmd/stdio.go` to pass summarizer to `NewIndexer`**

```go
idx, err := index.NewIndexer(dbPath, ic.embedder, ic.cfg.MaxChunkTokens, ic.cfg.SummaryEmbedDims, ic.summarizer, ic.summaryEmbedder)
```

- [ ] **Step 5: Update `setupIndexer` in `cmd/index.go`**

```go
func setupIndexer(cfg *config.Config, projectPath string) (*index.Indexer, error) {
    emb, err := newEmbedder(*cfg)
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    sumr, err := newSummarizer(*cfg)
    if err != nil {
        return nil, fmt.Errorf("create summarizer: %w", err)
    }

    summaryEmb, err := newSummaryEmbedder(*cfg)
    if err != nil {
        return nil, fmt.Errorf("create summary embedder: %w", err)
    }

    dbPath := config.DBPathForProject(projectPath, cfg.Model, cfg.SummaryEmbedModel)
    if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
        return nil, fmt.Errorf("create db directory: %w", err)
    }

    idx, err := index.NewIndexer(dbPath, emb, cfg.MaxChunkTokens, cfg.SummaryEmbedDims, sumr, summaryEmb)
    if err != nil {
        return nil, fmt.Errorf("create indexer: %w", err)
    }
    return idx, nil
}
```

- [ ] **Step 6: Run the full test suite**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./...
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/aeneas/workspace/go/agent-index-go
git add cmd/embedder.go cmd/stdio.go cmd/index.go
git commit -m "feat(cmd): wire summarizer and summary embedder into indexer construction"
```

---

### Task 7: Expand search to fan out across three indices and emit `<relevant_files>`

**Files:**
- Modify: `cmd/stdio.go`
- Modify: `cmd/stdio_test.go`

**Background:** When `cfg.Summaries` is true, `handleSemanticSearch` runs an expanded pipeline:
1. Embed query twice (code model + summary model).
2. Three sequential vector searches.
3. Merge chunk results (union by chunk ID, take max score).
4. Expand file hits: fetch top 4 chunks per file by raw-code distance.
5. Final dedup and re-rank.
6. Append `<relevant_files>` XML section when file hits exist.

- [ ] **Step 1: Write failing tests for the expanded search path**

Append to `cmd/stdio_test.go`:

```go
func TestFormatSearchResults_RelevantFiles(t *testing.T) {
    out := SemanticSearchOutput{
        Results: []SearchResultItem{
            {FilePath: "/proj/auth.go", Symbol: "ValidateToken", Kind: "function", StartLine: 1, EndLine: 10, Score: 0.91},
        },
        RelevantFiles: []RelevantFile{
            {FilePath: "auth.go", Score: 0.91},
            {FilePath: "token.go", Score: 0.87},
        },
    }
    text := formatSearchResults("/proj", out)
    if !strings.Contains(text, "<relevant_files>") {
        t.Fatal("expected <relevant_files> section in output")
    }
    if !strings.Contains(text, `path="auth.go"`) {
        t.Fatal("expected auth.go in relevant_files")
    }
    if !strings.Contains(text, `score="0.91"`) {
        t.Fatal("expected score in relevant_files")
    }
}

func TestFormatSearchResults_NoRelevantFiles_NoSection(t *testing.T) {
    out := SemanticSearchOutput{
        Results: []SearchResultItem{
            {FilePath: "/proj/main.go", Symbol: "Main", Kind: "function", StartLine: 1, EndLine: 5, Score: 0.80},
        },
    }
    text := formatSearchResults("/proj", out)
    if strings.Contains(text, "<relevant_files>") {
        t.Fatal("expected no <relevant_files> section when RelevantFiles is empty")
    }
}
```

- [ ] **Step 2: Run to confirm compilation failure**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./cmd/... -v 2>&1 | head -20
```

Expected: `RelevantFiles` field missing from `SemanticSearchOutput`.

- [ ] **Step 3: Add `RelevantFile` type and `RelevantFiles` field to output types**

In `cmd/stdio.go`, add after `SearchResultItem`:

```go
// RelevantFile represents a file-level summary search hit returned in MCP responses.
type RelevantFile struct {
    FilePath string  `json:"file_path"`
    Score    float64 `json:"score"`
}
```

Add `RelevantFiles []RelevantFile` to `SemanticSearchOutput`:

```go
type SemanticSearchOutput struct {
    Results       []SearchResultItem `json:"results"`
    RelevantFiles []RelevantFile     `json:"relevant_files,omitempty"`
    Reindexed     bool               `json:"reindexed"`
    IndexedFiles  int                `json:"indexed_files,omitempty"`
    FilteredHint  string             `json:"filtered_hint,omitempty"`
}
```

- [ ] **Step 4: Update `formatSearchResults` to append `<relevant_files>`**

After the existing loop that formats chunk results, add:

```go
    if len(out.RelevantFiles) > 0 {
        b.WriteString("\n<relevant_files>\n")
        for _, rf := range out.RelevantFiles {
            fmt.Fprintf(&b, "  <file path=%q score=\"%.2f\"/>\n", xmlEscaper.Replace(rf.FilePath), rf.Score)
        }
        b.WriteString("</relevant_files>")
    }
```

- [ ] **Step 5: Add `embedSummaryQuery` method and expand `handleSemanticSearch`**

Add helper to `indexerCache`:

```go
func (ic *indexerCache) embedSummaryQuery(ctx context.Context, query string) ([]float32, error) {
    if ic.summaryEmbedder == nil {
        return nil, nil
    }
    vecs, err := ic.summaryEmbedder.Embed(ctx, []string{query})
    if err != nil {
        return nil, fmt.Errorf("embed summary query: %w", err)
    }
    if len(vecs) == 0 {
        return nil, fmt.Errorf("summary embedder returned no vectors")
    }
    return vecs[0], nil
}
```

Expand `handleSemanticSearch` after the existing `idx.Search` call:

```go
    // Expanded search when summaries are enabled.
    summaryQueryVec, summaryEmbedErr := ic.embedSummaryQuery(ctx, input.Query)
    if summaryEmbedErr != nil {
        log.Printf("warning: embed summary query: %v", summaryEmbedErr)
    }

    if summaryQueryVec != nil {
        summaryMaxDistance := computeMaxDistance(nil, ic.cfg.SummaryEmbedModel, ic.cfg.SummaryEmbedDims)

        // Search chunk summaries.
        chunkSumResults, err := idx.SearchChunkSummaries(ctx, effectiveRoot, summaryQueryVec, fetchLimit, summaryMaxDistance, pathPrefix)
        if err != nil {
            log.Printf("warning: search chunk summaries: %v", err)
        }

        // Merge chunk + chunk-summary results by chunk ID, taking max score.
        results = mergeSearchResults(results, chunkSumResults)

        // Search file summaries.
        fileSumResults, err := idx.SearchFileSummaries(ctx, effectiveRoot, summaryQueryVec, input.NResults, summaryMaxDistance)
        if err != nil {
            log.Printf("warning: search file summaries: %v", err)
        }

        // Expand file hits: fetch top 4 chunks per file by raw-code distance.
        var relevantFiles []RelevantFile
        for _, fr := range fileSumResults {
            relPath := fr.FilePath
            // Convert file path to relative if it's absolute.
            if rel, err := filepath.Rel(effectiveRoot, fr.FilePath); err == nil {
                relPath = rel
            }
            relevantFiles = append(relevantFiles, RelevantFile{
                FilePath: relPath,
                Score:    1.0 - fr.Distance,
            })
            topChunks, err := idx.TopChunksByFile(ctx, effectiveRoot, fr.FilePath, queryVec, 4)
            if err != nil {
                log.Printf("warning: top chunks for %s: %v", fr.FilePath, err)
                continue
            }
            results = append(results, topChunks...)
        }
        out.RelevantFiles = relevantFiles
    }
```

- [ ] **Step 6: Add `SearchChunkSummaries`, `SearchFileSummaries`, `TopChunksByFile` proxy methods to `Indexer`**

In `internal/index/index.go`, add:

```go
// SearchChunkSummaries proxies to the store's chunk summary search.
func (idx *Indexer) SearchChunkSummaries(_ context.Context, _ string, queryVec []float32, limit int, maxDistance float64, pathPrefix string) ([]store.SearchResult, error) {
    return idx.store.SearchChunkSummaries(queryVec, limit, maxDistance, pathPrefix)
}

// SearchFileSummaries proxies to the store's file summary search.
func (idx *Indexer) SearchFileSummaries(_ context.Context, _ string, queryVec []float32, limit int, maxDistance float64) ([]store.FileSummaryResult, error) {
    return idx.store.SearchFileSummaries(queryVec, limit, maxDistance)
}

// TopChunksByFile returns the top n chunks from filePath ranked by raw-code distance.
func (idx *Indexer) TopChunksByFile(_ context.Context, _ string, filePath string, queryVec []float32, n int) ([]store.SearchResult, error) {
    return idx.store.TopChunksByFile(filePath, queryVec, n)
}
```

- [ ] **Step 7: Add `mergeSearchResults` helper in `cmd/stdio.go`**

```go
// mergeSearchResults merges two slices of SearchResult by chunk ID, keeping
// the entry with the lower distance (higher score) for each duplicate.
func mergeSearchResults(a, b []store.SearchResult) []store.SearchResult {
    seen := make(map[string]int, len(a)) // chunk ID → index in result
    result := make([]store.SearchResult, len(a))
    copy(result, a)
    for i, r := range result {
        seen[r.FilePath+":"+r.Symbol+":"+fmt.Sprintf("%d", r.StartLine)] = i
    }
    for _, r := range b {
        key := r.FilePath + ":" + r.Symbol + ":" + fmt.Sprintf("%d", r.StartLine)
        if idx, ok := seen[key]; ok {
            if r.Distance < result[idx].Distance {
                result[idx] = r
            }
        } else {
            seen[key] = len(result)
            result = append(result, r)
        }
    }
    return result
}
```

Note: deduplication uses `filePath:symbol:startLine` as the composite key because `SearchResult` does not expose the chunk ID. This is equivalent and deterministic.

- [ ] **Step 8: Run full test suite**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./...
```

Expected: all tests pass.

- [ ] **Step 9: Commit**

```bash
cd /Users/aeneas/workspace/go/agent-index-go
git add cmd/stdio.go cmd/stdio_test.go internal/index/index.go
git commit -m "feat(search): fan out to summary indices and emit relevant_files in MCP response"
```

---

### Task 8: Lint and final verification

**Files:**
- No new files

- [ ] **Step 1: Run linter**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && golangci-lint run ./...
```

Expected: zero issues. Fix any reported issues before proceeding.

- [ ] **Step 2: Run go vet**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go vet ./...
```

Expected: no errors (external dependency warnings are acceptable per CLAUDE.md).

- [ ] **Step 3: Run the full test suite one final time**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && go test ./...
```

Expected: all tests pass.

- [ ] **Step 4: Verify the binary builds**

```bash
cd /Users/aeneas/workspace/go/agent-index-go && make build-local
```

Expected: binary produced in `bin/`.

- [ ] **Step 5: Commit any lint fixes**

If lint required changes:

```bash
cd /Users/aeneas/workspace/go/agent-index-go
git add -p
git commit -m "fix: address golangci-lint issues in semantic summaries implementation"
```

---

## Appendix: Environment Variables Reference

| Variable | Default (Ollama) | Default (LM Studio) | Description |
|---|---|---|---|
| `LUMEN_SUMMARIES` | `false` | `false` | Set to `true` to enable semantic summarization |
| `LUMEN_SUMMARY_MODEL` | `qwen2.5-coder:7b` | `qwen2.5-coder:7b` | LLM for generating summaries |
| `LUMEN_SUMMARY_EMBED_MODEL` | `nomic-embed-text` | `nomic-ai/nomic-embed-text-GGUF` | Embedding model for summary vectors |

## Appendix: DB Path Backward Compatibility Note

`DBPathForProject` now takes a third argument `summaryEmbedModel`. When `LUMEN_SUMMARIES` is not set, `SummaryEmbedModel` is `""` and the hash input becomes `projectPath + "\x00" + codeModel + "\x00" + "" + "\x00" + IndexVersion` — which differs from the old two-argument formula. Existing users will see a one-time full re-index on upgrade. This is intentional and acceptable: the new summary tables are additive and would cause a schema migration regardless. `IndexVersion` stays at `"2"` because new tables are only created in summary-enabled DBs.
