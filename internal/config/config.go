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
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ory/lumen/internal/embedder"
)

const (
	// BackendOllama is the backend identifier for Ollama.
	BackendOllama = "ollama"
	// BackendLMStudio is the backend identifier for LM Studio.
	BackendLMStudio = "lmstudio"
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
	FreshnessTTL   time.Duration
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

	// Allow fully custom models via LUMEN_EMBED_DIMS + LUMEN_EMBED_CTX,
	// bypassing the KnownModels registry. Useful for any OpenAI-compat server.
	var dims, ctxLength int
	if d := EnvOrDefaultInt("LUMEN_EMBED_DIMS", 0); d > 0 {
		dims = d
		ctxLength = EnvOrDefaultInt("LUMEN_EMBED_CTX", 8192)
	} else {
		spec, ok := embedder.KnownModels[model]
		if !ok {
			return Config{}, fmt.Errorf("unknown embedding model %q: set LUMEN_EMBED_DIMS to use an unlisted model", model)
		}
		dims = spec.Dims
		ctxLength = spec.CtxLength
	}

	return Config{
		Model:          model,
		Dims:           dims,
		CtxLength:      ctxLength,
		MaxChunkTokens: EnvOrDefaultInt("LUMEN_MAX_CHUNK_TOKENS", 512),
		OllamaHost:     EnvOrDefault("OLLAMA_HOST", "http://localhost:11434"),
		Backend:        backend,
		LMStudioHost:   EnvOrDefault("LM_STUDIO_HOST", "http://localhost:1234"),
		FreshnessTTL:   EnvOrDefaultDuration("LUMEN_FRESHNESS_TTL", 60*time.Second),
	}, nil
}

// DBPathForProject returns the SQLite database path for a given project,
// derived from a SHA-256 hash of the project path, embedding model name, and
// IndexVersion. Including the model ensures that switching models creates a
// fresh index automatically. Including IndexVersion ensures that incompatible
// chunker/index format changes never share an index with older data.
func DBPathForProject(projectPath, model string) string {
	return DBPathForProjectBase(XDGDataDir(), projectPath, model)
}

// DBPathForProjectBase returns the SQLite database path for a given project
// using an explicit data directory instead of reading XDG_DATA_HOME from the
// environment. Safe to call from parallel goroutines.
func DBPathForProjectBase(dataDir, projectPath, model string) string {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(projectPath+"\x00"+model+"\x00"+IndexVersion)))
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

// EnvOrDefaultDuration returns the time.Duration value of the environment
// variable named by key, or fallback if the variable is not set, empty, or
// not a valid duration string (e.g. "30s", "1m").
func EnvOrDefaultDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
