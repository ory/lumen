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
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/index"
	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/tui"
	"github.com/spf13/cobra"
)

func init() {
	indexCmd.Flags().StringP("model", "m", "", "embedding model (default: $LUMEN_EMBED_MODEL or "+embedder.DefaultModel+")")
	indexCmd.Flags().BoolP("force", "f", false, "force full re-index")
	rootCmd.AddCommand(indexCmd)
}

var indexCmd = &cobra.Command{
	Use:   "index <project-path>",
	Short: "Index a project for semantic search",
	Args:  cobra.ExactArgs(1),
	RunE:  runIndex,
}

func runIndex(cmd *cobra.Command, args []string) error {
	logger, logFile := newDebugLogger()
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := applyModelFlag(cmd, &cfg); err != nil {
		return err
	}

	projectPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	dbPath := config.DBPathForProject(projectPath, cfg.Model)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	lockPath := indexlock.LockPathForDB(dbPath)
	lock, err := indexlock.TryAcquire(lockPath)
	if err != nil {
		return fmt.Errorf("acquire index lock: %w", err)
	}
	if lock == nil {
		// Another indexer is already running for this project — skip silently.
		// This is the normal case when multiple Claude terminals are open.
		logger.Info("index skipped: another indexer is already running", "project", projectPath)
		fmt.Fprintln(os.Stderr, "Another indexer is already running for this project. Skipping.")
		return nil
	}
	defer lock.Release()

	// Cancel context on SIGTERM or SIGINT so the indexer stops cleanly and
	// the deferred lock.Release() runs before exit.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	idx, err := setupIndexer(&cfg, dbPath, logger)
	if err != nil {
		return err
	}
	defer func() { _ = idx.Close() }()

	logger.Info("indexing started", "project", projectPath, "model", cfg.Model, "dims", cfg.Dims)
	p := tui.NewProgress(os.Stderr)
	p.Info(fmt.Sprintf("Indexing %s (model: %s, dims: %d)", projectPath, cfg.Model, cfg.Dims))

	start := time.Now()
	stats, err := performIndexing(ctx, cmd, idx, projectPath, p)
	if err != nil {
		if ctx.Err() != nil {
			// A signal arrived; treat as clean exit. If an unrelated error
			// also occurred in the same instant, it is intentionally dropped —
			// the cancellation is the primary cause and the lock will be released.
			logger.Info("indexing cancelled by signal", "project", projectPath)
			return nil
		}
		logger.Error("indexing failed", "project", projectPath, "err", err)
		return err
	}

	elapsed := time.Since(start).Round(time.Millisecond)
	if stats.Reason == "already fresh" {
		logger.Info("index already fresh",
			"project", projectPath,
			"elapsed", elapsed.String(),
		)
	} else {
		logger.Info("indexing complete",
			"project", projectPath,
			"reason", stats.Reason,
			"total_files", stats.TotalFiles,
			"files_unchanged", stats.TotalFiles-stats.FilesChanged,
			"files_added", stats.FilesAdded,
			"files_modified", stats.FilesModified,
			"files_removed", stats.FilesRemoved,
			"indexed_files", stats.IndexedFiles,
			"chunks_created", stats.ChunksCreated,
			"old_root_hash", stats.OldRootHash,
			"new_root_hash", stats.NewRootHash,
			"elapsed", elapsed.String(),
		)
	}
	if stats.Reason != "" {
		fmt.Printf("Reason: %s\n", stats.Reason)
	}
	if stats.OldRootHash != "" {
		fmt.Printf("Root hash: %s -> %s\n", stats.OldRootHash[:16], stats.NewRootHash[:16])
	} else if stats.NewRootHash != "" {
		fmt.Printf("Root hash: (none) -> %s\n", stats.NewRootHash[:16])
	}
	fmt.Printf("Files: %d added, %d modified, %d removed (%d total in project)\n",
		stats.FilesAdded, stats.FilesModified, stats.FilesRemoved, stats.TotalFiles)
	fmt.Printf("Done. Indexed %d files, %d chunks in %s.\n",
		stats.IndexedFiles, stats.ChunksCreated, elapsed)
	return nil
}

func applyModelFlag(cmd *cobra.Command, cfg *config.Config) error {
	m, _ := cmd.Flags().GetString("model")
	if m == "" {
		return nil
	}
	spec, ok := embedder.KnownModels[m]
	if !ok {
		return fmt.Errorf("unknown embedding model %q", m)
	}
	cfg.Model = m
	cfg.Dims = spec.Dims
	cfg.CtxLength = spec.CtxLength
	return nil
}

// setupIndexer receives dbPath so it is computed exactly once in runIndex.
func setupIndexer(cfg *config.Config, dbPath string, logger *slog.Logger) (*index.Indexer, error) {
	emb, err := newEmbedder(*cfg)
	if err != nil {
		return nil, fmt.Errorf("create embedder: %w", err)
	}

	idx, err := index.NewIndexer(dbPath, emb, cfg.MaxChunkTokens)
	if err != nil {
		return nil, fmt.Errorf("create indexer: %w", err)
	}
	idx.SetLogger(logger)
	return idx, nil
}

func performIndexing(ctx context.Context, cmd *cobra.Command, idx *index.Indexer, projectPath string, p *tui.Progress) (index.Stats, error) {
	force, _ := cmd.Flags().GetBool("force")

	progress := p.AsProgressFunc()

	if force {
		return idx.Index(ctx, projectPath, true, progress)
	}

	reindexed, stats, err := idx.EnsureFresh(ctx, projectPath, progress)
	if err != nil {
		return stats, fmt.Errorf("indexing: %w", err)
	}

	if !reindexed {
		stats.Reason = "already fresh"
		fmt.Println("Index is already up to date.")
	}

	return stats, nil
}
