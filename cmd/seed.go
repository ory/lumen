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
	"log/slog"
	"os"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/index"
)

// seedFromDonorIfNew copies a sibling git-worktree's index to dbPath when no
// index exists there yet, so indexing a fresh worktree reuses the parent's
// embeddings and only re-indexes changed files instead of re-embedding the
// whole tree from scratch. It is best-effort: any failure is logged and
// indexing proceeds from scratch.
//
// This mirrors the seeding the MCP server performs in
// indexerCache.getOrCreate. The background indexer that lumen index runs at
// SessionStart also creates a fresh DB, and it normally wins the race against
// the MCP path. Without seeding here, that background indexer leaves the
// worktree fully re-indexed from scratch and the getOrCreate seed is then
// skipped because the DB already exists.
func seedFromDonorIfNew(dbPath, projectPath, model string, logger *slog.Logger) {
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		// The DB already exists (or stat failed unexpectedly) — nothing to seed.
		return
	}
	donorPath := config.FindDonorIndex(projectPath, model)
	if donorPath == "" {
		return
	}
	seeded, err := index.SeedFromDonor(donorPath, dbPath)
	if err != nil {
		logger.Warn("seed from donor worktree failed",
			"project", projectPath, "donor_path", donorPath, "error", err)
		return
	}
	if seeded {
		logger.Info("seeded index from donor worktree",
			"project", projectPath, "donor_path", donorPath)
	}
}
