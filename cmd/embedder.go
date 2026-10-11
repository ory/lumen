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
	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
)

// newEmbedder creates a FailoverEmbedder from the given ConfigService.
func newEmbedder(cfg *config.ConfigService) *embedder.FailoverEmbedder {
	if cfg.Retrieval() == "bm25" {
		return nil
	}
	return embedder.NewFailoverEmbedder(cfg)
}

func retrievalModel(emb *embedder.FailoverEmbedder) string {
	if emb == nil {
		return "lumen-bm25-v1"
	}
	return emb.ModelName()
}
