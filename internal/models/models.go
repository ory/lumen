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

// DefaultModel is an alias for DefaultOllamaModel for backward compatibility.
const DefaultModel = DefaultOllamaModel

// ModelAliases maps alternative model names to their canonical names.
// LM Studio exposes some models under different names than their repository ID.
var ModelAliases = map[string]string{
	"text-embedding-nomic-embed-code": "nomic-ai/nomic-embed-code-GGUF",
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
	"ordis/jina-embeddings-v2-base-code":  {Dims: 768, CtxLength: 8192, Backend: "ollama", MinScore: 0.35},
	"nomic-embed-text":                    {Dims: 768, CtxLength: 8192, Backend: "ollama", MinScore: 0.30},
	"nomic-ai/nomic-embed-code-GGUF":      {Dims: 3584, CtxLength: 8192, Backend: "lmstudio", MinScore: 0.15},
	"qwen3-embedding:8b":                  {Dims: 4096, CtxLength: 40960, Backend: "ollama", MinScore: 0.30},
	"qwen3-embedding:4b":                  {Dims: 2560, CtxLength: 40960, Backend: "ollama", MinScore: 0.30},
	"qwen3-embedding:0.6b":                {Dims: 1024, CtxLength: 32768, Backend: "ollama", MinScore: 0.30},
	"all-minilm":                          {Dims: 384, CtxLength: 512, Backend: "ollama", MinScore: 0.20},
	"manutic/nomic-embed-code:7b":         {Dims: 3584, CtxLength: 32768, Backend: "ollama", MinScore: 0.15},
}
