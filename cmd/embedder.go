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

	"github.com/aeneasr/agent-index/internal/config"
	"github.com/aeneasr/agent-index/internal/embedder"
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
