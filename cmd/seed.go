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
	"os"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/index"
)

type seedOptions struct {
	findDonor func(string, string) string
	seed      func(context.Context, string, string, string) (bool, error)
	status    func(string)
}

// seedFromDonorIfNew seeds dbPath from a sibling worktree's index when dbPath
// does not yet exist, so a fresh git worktree reuses an already-indexed
// worktree's embeddings instead of re-embedding every file from scratch.
//
// Callers may additionally hold the index lock for dbPath, but SeedFromDonor
// has its own advisory lock so CLI and MCP callers cannot duplicate the copy.
// Seeding is best-effort — any failure is logged and indexing continues with a
// from-scratch build. The returned warning is suitable for surfacing to MCP
// clients. When dbPath already exists it is a single stat on the hot path.
func seedFromDonorIfNew(ctx context.Context, dbPath, projectPath, model string, logger *slog.Logger, opts seedOptions) string {
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		// Exists already, or stat failed for some other reason — nothing to do.
		return ""
	}

	findDonor := opts.findDonor
	if findDonor == nil {
		findDonor = config.FindDonorIndex
	}
	donorPath := findDonor(projectPath, model)
	if donorPath == "" {
		return ""
	}

	logger.Info("seeding index from donor worktree",
		"project_path", projectPath,
		"donor_path", donorPath,
	)
	if opts.status != nil {
		opts.status("Seeding index from sibling worktree...")
	}

	seed := opts.seed
	if seed == nil {
		seed = index.SeedFromDonorContext
	}
	seeded, err := seed(ctx, donorPath, dbPath, projectPath)
	if err != nil {
		logger.Warn("seed from donor worktree failed",
			"project_path", projectPath,
			"donor_path", donorPath,
			"error", err,
		)
		warning := fmt.Sprintf("index seeded from scratch (sibling copy failed: %v)", err)
		if opts.status != nil {
			opts.status(fmt.Sprintf("Sibling index copy failed: %v; indexing from scratch.", err))
		}
		return warning
	}
	if seeded && opts.status != nil {
		opts.status("Seeded index from sibling worktree.")
	} else if opts.status != nil {
		if _, statErr := os.Stat(dbPath); statErr == nil {
			opts.status("Index was seeded by another process.")
		} else {
			opts.status("Sibling index could not be reused; indexing from scratch.")
		}
	}
	return ""
}
