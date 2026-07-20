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

// findDonorFn and seedFromDonorFn are indirections so tests can exercise
// seedFromDonorIfNew without real git worktrees or a real donor database.
var (
	findDonorFn     = config.FindDonorIndex
	seedFromDonorFn = index.SeedFromDonor
)

// seedFromDonorIfNew seeds dbPath from a sibling worktree's index when dbPath
// does not yet exist, so a fresh git worktree reuses an already-indexed
// worktree's embeddings instead of re-embedding every file from scratch.
//
// The caller must hold the index lock for dbPath: this serializes seeding
// against other indexers for the same project. Seeding is best-effort — any
// failure is logged and indexing continues with a from-scratch build — so this
// never returns an error. When dbPath already exists it is a single stat on the
// hot path and returns immediately.
func seedFromDonorIfNew(dbPath, projectPath, model string, logger *slog.Logger) {
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		// Exists already, or stat failed for some other reason — nothing to do.
		return
	}

	donorPath := findDonorFn(projectPath, model)
	if donorPath == "" {
		return
	}

	logger.Info("seeding index from donor worktree",
		"project", projectPath,
		"donor_path", donorPath,
	)
	if _, err := seedFromDonorFn(donorPath, dbPath); err != nil {
		logger.Warn("seed from donor worktree failed",
			"project", projectPath,
			"donor_path", donorPath,
			"err", err,
		)
	}
}
