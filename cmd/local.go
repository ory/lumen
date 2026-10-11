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
	"context"
	"fmt"
	"log/slog"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/tui"
)

// prepareLocalModel downloads the managed local model before interactive
// indexing or search so the user sees download progress instead of a silent
// multi-gigabyte wait inside the first embedding call. It does nothing for
// remote backends, user-supplied model files, or an already cached model.
func prepareLocalModel(ctx context.Context, cfg *config.ConfigService, emb *embedder.FailoverEmbedder, p *tui.Progress, logger *slog.Logger) error {
	servers := cfg.Servers()
	idx := max(emb.ActiveServerIndex(), 0)
	if idx >= len(servers) || servers[idx].Backend != config.BackendLocal {
		return nil
	}
	srv := servers[idx]
	info, err := embedder.LocalModel(srv)
	if err != nil || !info.Managed || info.Present {
		return err
	}

	logger.Info("downloading local embedding model", "path", info.Path, "bytes", info.Size)
	const mib = 1 << 20
	p.Start(fmt.Sprintf("Downloading %s (%.2f GiB, once)", srv.Model, float64(info.Size)/(1<<30)), int(info.Size/mib))
	_, err = embedder.EnsureLocalModel(ctx, srv, func(done, total int64) {
		p.Update(int(done/mib), "Downloading local embedding model")
	})
	p.Stop()
	if err != nil {
		logger.Error("local embedding model download failed", "error", err)
		return fmt.Errorf("download local embedding model: %w", err)
	}
	logger.Info("local embedding model downloaded", "path", info.Path)
	return nil
}
