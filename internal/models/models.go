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

package models

import "strings"

// ModelSpec describes the known configuration for an embedding model.
type ModelSpec struct {
	Dims      int
	CtxLength int
	Backend   string  // "ollama", "lmstudio", or "" for both
	MinScore  float64 // default minimum cosine similarity threshold for search
}

// DefaultOllamaModel is the default model when using the Ollama backend.
const DefaultOllamaModel = "ordis/jina-embeddings-v2-base-code"

// DefaultLMStudioModel is the default model when using the LM Studio backend.
const DefaultLMStudioModel = "nomic-ai/nomic-embed-code-GGUF"

// DefaultLocalModel is the default model for the in-process local backend.
const DefaultLocalModel = "nomic-ai/nomic-embed-code-GGUF"

// DefaultModel is an alias for DefaultOllamaModel for backward compatibility.
const DefaultModel = DefaultOllamaModel

// ModelAliases maps alternative model names to their canonical names.
// LM Studio exposes some models under different names than their repository ID.
var ModelAliases = map[string]string{
	"text-embedding-nomic-embed-code": "nomic-ai/nomic-embed-code-GGUF",
}

// BaseName strips the artifact/profile qualifier ("@...") that identifies a
// local-backend collection, returning the registry model name.
func BaseName(identity string) string {
	name, _, _ := strings.Cut(identity, "@")
	return name
}

// DefaultMinScore is the fallback noise-floor threshold used when the active
// model is not in KnownModels and dimensions are unknown.
const DefaultMinScore = 0.20

// DimensionAwareMinScore returns a noise-floor threshold appropriate for the
// given embedding dimensionality. Higher-dimensional spaces compress cosine
// similarity ranges (concentration of measure), so they need lower floors.
func DimensionAwareMinScore(dims int) float64 {
	switch {
	case dims > 3072:
		return 0.15
	case dims > 1024:
		return 0.20
	case dims > 512:
		return 0.25
	default:
		return 0.20
	}
}

// KnownModels maps model names to their specifications.
var KnownModels = map[string]ModelSpec{
	"ordis/jina-embeddings-v2-base-code": {Dims: 768, CtxLength: 8192, Backend: "ollama", MinScore: 0.35},
	"nomic-embed-text":                   {Dims: 768, CtxLength: 8192, Backend: "ollama", MinScore: 0.30},
	"nomic-ai/nomic-embed-code-GGUF":     {Dims: 3584, CtxLength: 8192, Backend: "lmstudio", MinScore: 0.15},
	"qwen3-embedding:8b":                 {Dims: 4096, CtxLength: 40960, Backend: "ollama", MinScore: 0.30},
	"qwen3-embedding:4b":                 {Dims: 2560, CtxLength: 40960, Backend: "ollama", MinScore: 0.30},
	"qwen3-embedding:0.6b":               {Dims: 1024, CtxLength: 32768, Backend: "ollama", MinScore: 0.30},
	"all-minilm":                         {Dims: 384, CtxLength: 512, Backend: "ollama", MinScore: 0.20},
	"manutic/nomic-embed-code:7b":        {Dims: 3584, CtxLength: 32768, Backend: "ollama", MinScore: 0.15},
}

// LocalModelSpec pins the GGUF artifact and the model-specific preprocessing
// the in-process llama.cpp backend uses for a model.
type LocalModelSpec struct {
	// File is the managed cache file name of the pinned artifact.
	File string
	// URL is an immutable (revision-pinned) download location of the artifact.
	URL string
	// SHA256 is the expected hex digest of the artifact.
	SHA256 string
	// Size is the expected artifact size in bytes.
	Size int64
	// Architecture is the expected GGUF general.architecture of any file used
	// for this model, managed or user-supplied.
	Architecture string
	// Pooling is the sequence pooling the model was trained with ("last").
	Pooling string
	// QueryInstruction is prepended to search queries but not to indexed code.
	QueryInstruction string
}

// LocalModels lists the models the local backend can run in-process.
var LocalModels = map[string]LocalModelSpec{
	"nomic-ai/nomic-embed-code-GGUF": {
		File:             "nomic-embed-code.Q4_K_M.gguf",
		URL:              "https://huggingface.co/nomic-ai/nomic-embed-code-GGUF/resolve/ff2ddedde976ea623178981f18e36af33c0c2a94/nomic-embed-code.Q4_K_M.gguf",
		SHA256:           "4354a73ee9ff5d811efe552a515dfd518667ff25fdfc4ee9e10af3f617f96eec",
		Size:             4376511808,
		Architecture:     "qwen2",
		Pooling:          "last",
		QueryInstruction: "Represent this query for searching relevant code: ",
	},
}
