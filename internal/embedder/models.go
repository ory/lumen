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

// DefaultModel is the model used when AGENT_INDEX_EMBED_MODEL is not set.
const DefaultModel = "qwen3-embedding:4b"

// KnownModels maps Ollama model names to their specifications.
var KnownModels = map[string]ModelSpec{
	"ordis/jina-embeddings-v2-base-code": {768, 8192, "~323MB"},
	"nomic-embed-text":                   {768, 8192, "~274MB"},
	"qwen3-embedding:8b":                 {4096, 40960, "~4.7GB"},
	"qwen3-embedding:4b":                 {2560, 40960, "~2.6GB"},
	"qwen3-embedding:0.6b":               {1024, 32768, "~522MB"},
	"all-minilm":                         {384, 512, "~33MB"},
}
