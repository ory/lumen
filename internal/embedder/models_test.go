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

import "testing"

func TestKnownModels(t *testing.T) {
	expected := map[string]ModelSpec{
		"ordis/jina-embeddings-v2-base-code": {Dims: 768, CtxLength: 8192, Backend: "ollama", MinScore: 0.35},
		"nomic-embed-text":                   {Dims: 768, CtxLength: 8192, Backend: "ollama", MinScore: 0.30},
		"nomic-ai/nomic-embed-code-GGUF":     {Dims: 3584, CtxLength: 8192, Backend: "lmstudio", MinScore: 0.15},
		"qwen3-embedding:8b":                 {Dims: 4096, CtxLength: 40960, Backend: "ollama", MinScore: 0.30},
		"qwen3-embedding:4b":                 {Dims: 2560, CtxLength: 40960, Backend: "ollama", MinScore: 0.30},
		"qwen3-embedding:0.6b":               {Dims: 1024, CtxLength: 32768, Backend: "ollama", MinScore: 0.30},
		"all-minilm":                         {Dims: 384, CtxLength: 512, Backend: "ollama", MinScore: 0.20},
		"manutic/nomic-embed-code:7b":        {Dims: 3584, CtxLength: 32768, Backend: "ollama", MinScore: 0.15},
	}

	for name, want := range expected {
		got, ok := KnownModels[name]
		if !ok {
			t.Errorf("model %q missing from KnownModels", name)
			continue
		}
		if got != want {
			t.Errorf("model %q: got %+v, want %+v", name, got, want)
		}
	}

	if len(KnownModels) != len(expected) {
		t.Errorf("KnownModels has %d entries, expected %d", len(KnownModels), len(expected))
	}
}

func TestDefaultModelInRegistry(t *testing.T) {
	if _, ok := KnownModels[DefaultModel]; !ok {
		t.Errorf("DefaultModel %q is not in KnownModels", DefaultModel)
	}
}

func TestDefaultOllamaModelInRegistry(t *testing.T) {
	if _, ok := KnownModels[DefaultOllamaModel]; !ok {
		t.Errorf("DefaultOllamaModel %q is not in KnownModels", DefaultOllamaModel)
	}
}

func TestDefaultLMStudioModelInRegistry(t *testing.T) {
	if _, ok := KnownModels[DefaultLMStudioModel]; !ok {
		t.Errorf("DefaultLMStudioModel %q is not in KnownModels", DefaultLMStudioModel)
	}
}

func TestDimensionAwareMinScore(t *testing.T) {
	tests := []struct {
		dims int
		want float64
	}{
		{384, 0.20},  // small dims (all-minilm)
		{512, 0.20},  // boundary
		{513, 0.25},  // just above 512
		{768, 0.25},  // medium dims (jina, nomic-text)
		{1024, 0.25}, // boundary
		{1025, 0.20}, // just above 1024
		{2560, 0.20}, // medium-high (qwen3-4b)
		{3072, 0.20}, // boundary
		{3073, 0.15}, // just above 3072
		{3584, 0.15}, // high dims (nomic-embed-code)
		{4096, 0.15}, // very high dims (qwen3-8b)
	}

	for _, tt := range tests {
		got := DimensionAwareMinScore(tt.dims)
		if got != tt.want {
			t.Errorf("DimensionAwareMinScore(%d) = %v, want %v", tt.dims, got, tt.want)
		}
	}
}
