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

package cmd

import (
	"fmt"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/summarizer"
)

// newEmbedder creates an Embedder based on the configured backend.
func newEmbedder(cfg config.Config) (embedder.Embedder, error) {
	switch cfg.Backend {
	case config.BackendOllama:
		return embedder.NewOllama(cfg.Model, cfg.Dims, cfg.CtxLength, cfg.OllamaHost)
	case config.BackendLMStudio:
		return embedder.NewLMStudio(cfg.Model, cfg.Dims, cfg.LMStudioHost)
	default:
		return nil, fmt.Errorf("unknown backend %q", cfg.Backend)
	}
}

// newSummarizer creates a Summarizer for the configured backend and summary model.
// Returns nil, nil when cfg.Summaries is false.
// When cfg.SummaryModel is "_mock", returns a MockSummarizer for testing.
func newSummarizer(cfg config.Config) (summarizer.Summarizer, error) {
	if !cfg.Summaries {
		return nil, nil
	}
	if cfg.SummaryModel == "_mock" {
		return &summarizer.MockSummarizer{}, nil
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
		ctxLen := 0
		if spec, ok := embedder.KnownModels[cfg.SummaryEmbedModel]; ok {
			ctxLen = spec.CtxLength
		}
		return embedder.NewOllama(cfg.SummaryEmbedModel, cfg.SummaryEmbedDims, ctxLen, cfg.OllamaHost)
	case config.BackendLMStudio:
		return embedder.NewLMStudio(cfg.SummaryEmbedModel, cfg.SummaryEmbedDims, cfg.LMStudioHost)
	default:
		return nil, fmt.Errorf("unknown backend %q", cfg.Backend)
	}
}
