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

// ModelSpec describes the known configuration for an embedding model.
type ModelSpec struct {
	Dims      int
	CtxLength int
	SizeHint  string
}

// DefaultOllamaModel is the default model when using the Ollama backend.
const DefaultOllamaModel = "ordis/jina-embeddings-v2-base-code"

// DefaultLMStudioModel is the default model when using the LM Studio backend.
const DefaultLMStudioModel = "nomic-ai/nomic-embed-code-GGUF"

// DefaultModel is an alias for DefaultOllamaModel for backward compatibility.
const DefaultModel = DefaultOllamaModel

// KnownModels maps model names to their specifications.
var KnownModels = map[string]ModelSpec{
	"ordis/jina-embeddings-v2-base-code": {768, 8192, "~323MB"},
	"nomic-embed-text":                   {768, 8192, "~274MB"},
	"nomic-ai/nomic-embed-code-GGUF":     {3584, 8192, "~274MB"},
	"qwen3-embedding:8b":                 {4096, 40960, "~4.7GB"},
	"qwen3-embedding:4b":                 {2560, 40960, "~2.6GB"},
	"qwen3-embedding:0.6b":               {1024, 32768, "~522MB"},
	"all-minilm":                         {384, 512, "~33MB"},
}
