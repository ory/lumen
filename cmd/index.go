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
	"github.com/ory/lumen/internal/git"
	"github.com/ory/lumen/internal/index"
	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/merkle"
	"github.com/ory/lumen/internal/tui"
	"github.com/spf13/cobra"
)

func init() {
	indexCmd.Flags().StringP("model", "m", "", "embedding model (default: $LUMEN_EMBED_MODEL or "+embedder.DefaultModel+")")
	indexCmd.Flags().StringP("backend", "b", "", "embedding backend to select (\"ollama\" or \"lmstudio\"); disambiguates when --model is configured on multiple backends")
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
	projectPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Refuse roots that have declared themselves un-indexable — either via a
	// .lumenignore catch-all (e.g. $HOME with `**`) or via the hardcoded
	// refusal list. Without this, the indexer would walk the entire tree,
	// ignore every file, produce an empty index, and on macOS trigger TCC
	// prompts for protected folders along the way.
	if unindexable, reason := merkle.IsRootUnindexable(projectPath); unindexable {
		return fmt.Errorf("refusing to index %s: %s", projectPath, reason)
	}

	logger, logFile := newDebugLogger()
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}

	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		return err
	}

	emb := newEmbedder(cfg)
	emb.SetLogger(logger)

	modelName := emb.ModelName()

	// Normalize to the git repository root when inside a git repo so that
	// indexing a subdirectory produces the same DB as indexing the repo root.
	// For non-git directories, walk up to reuse an existing ancestor index.
	if root, err := git.RepoRoot(projectPath); err == nil {
		projectPath = root
	} else if ancestor := findAncestorIndex(projectPath, modelName); ancestor != "" {
		projectPath = ancestor
	}

	// Re-check after normalization: git.RepoRoot can resolve upward to an
	// un-indexable root (e.g. a git repo rooted at $HOME) and bypass the
	// pre-normalization guard. findAncestorIndex already filters, but the git
	// path does not.
	if unindexable, reason := merkle.IsRootUnindexable(projectPath); unindexable {
		return fmt.Errorf("refusing to index %s: %s", projectPath, reason)
	}

	// When the project directory is not a git repo, discover nested git repos
	// and index each one separately before indexing the parent (which will
	// only contain "loose" files not belonging to any nested repo).
	for _, repo := range git.DiscoverNestedGitRepos(projectPath) {
		p := tui.NewProgress(os.Stderr)
		p.Info(fmt.Sprintf("Indexing nested repo %s", repo))
		stats, elapsed, skipped, err := runIndexer(cmd, cfg, emb, repo, p, logger)
		switch {
		case err != nil:
			logger.Error("indexing nested repo failed", "repo", repo, "err", err)
			fmt.Fprintf(os.Stderr, "Warning: failed to index nested repo %s: %v\n", repo, err)
		case skipped:
			logger.Info("index skipped: another indexer is already running", "project", repo)
			fmt.Fprintf(os.Stderr, "Skipping %s: another indexer is already running.\n", repo)
		default:
			logger.Info("nested repo indexed", "project", repo,
				"indexed_files", stats.IndexedFiles, "files_skipped", stats.FilesSkipped,
				"chunks_created", stats.ChunksCreated, "elapsed", elapsed.String())
			if stats.FilesSkipped > 0 {
				fmt.Printf("Nested repo %s: %d files, %d chunks, skipped %d files (parse errors) in %s.\n",
					repo, stats.IndexedFiles, stats.ChunksCreated, stats.FilesSkipped, elapsed)
			} else {
				fmt.Printf("Nested repo %s: %d files, %d chunks in %s.\n", repo, stats.IndexedFiles, stats.ChunksCreated, elapsed)
			}
		}
	}

	p := tui.NewProgress(os.Stderr)
	dims := cfg.ServerDims(0)
	logger.Info("indexing started", "project", projectPath, "model", modelName, "dims", dims)
	p.Info(fmt.Sprintf("Indexing %s (model: %s, dims: %d)", projectPath, modelName, dims))
	stats, elapsed, skipped, err := runIndexer(cmd, cfg, emb, projectPath, p, logger)
	if err != nil {
		logger.Error("indexing failed", "project", projectPath, "err", err)
		return err
	}
	if skipped {
		logger.Info("index skipped: another indexer is already running", "project", projectPath)
		fmt.Fprintln(os.Stderr, "Another indexer is already running for this project. Skipping.")
		return nil
	}

	if stats.Reason == "already fresh" {
		logger.Info("index already fresh", "project", projectPath, "elapsed", elapsed.String())
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
			"files_skipped", stats.FilesSkipped,
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
	if stats.FilesSkipped > 0 {
		fmt.Printf("Done. Indexed %d files, %d chunks, skipped %d files (parse errors — see debug log) in %s.\n",
			stats.IndexedFiles, stats.ChunksCreated, stats.FilesSkipped, elapsed)
	} else {
		fmt.Printf("Done. Indexed %d files, %d chunks in %s.\n",
			stats.IndexedFiles, stats.ChunksCreated, elapsed)
	}
	return nil
}

// loadConfigWithFlags loads the ConfigService, honouring the --model and
// --backend flags if set.
//
// Selection flow:
//  1. No flags set → use YAML/env as-is.
//  2. --model and/or --backend set → filter the configured servers to matching
//     entries via WithServerSelection.
//  3. If no configured server matches and --backend is unset, fall back to
//     WithModelOverride for models present in the static KnownModels registry.
//     This preserves backward compatibility for users with no YAML config who
//     pass e.g. `--model all-minilm`.
func loadConfigWithFlags(cmd *cobra.Command) (*config.ConfigService, error) {
	model, _ := cmd.Flags().GetString("model")
	backend, _ := cmd.Flags().GetString("backend")
	path := config.DefaultConfigPath()

	if model == "" && backend == "" {
		return config.NewConfigService(path)
	}
	if backend != "" && backend != config.BackendOllama && backend != config.BackendLMStudio {
		return nil, fmt.Errorf("unknown backend %q (must be %q or %q)",
			backend, config.BackendOllama, config.BackendLMStudio)
	}

	cfg, selErr := config.NewConfigService(path, config.WithServerSelection(model, backend))
	if selErr == nil {
		return cfg, nil
	}

	// Fallback: legacy WithModelOverride path for users with no YAML who rely
	// on the static model registry. Only applies when --backend is unset.
	if model != "" && backend == "" {
		canonical := model
		if c, ok := embedder.ModelAliases[model]; ok {
			canonical = c
		}
		if _, ok := embedder.KnownModels[canonical]; ok {
			return config.NewConfigService(path, config.WithModelOverride(model))
		}
	}
	return nil, selErr
}

// setupIndexer receives dbPath so it is computed exactly once in runIndex.
func setupIndexer(cfg *config.ConfigService, emb *embedder.FailoverEmbedder, dbPath string, logger *slog.Logger) (*index.Indexer, error) {
	idx, err := index.NewIndexer(dbPath, emb, cfg.MaxChunkTokens())
	if err != nil {
		return nil, fmt.Errorf("create indexer: %w", err)
	}
	idx.SetLogger(logger)
	return idx, nil
}

// runIndexer acquires the index lock for projectPath, runs performIndexing, and
// returns the stats and elapsed time. skipped is true when another indexer holds
// the lock. err is nil if indexing was cancelled by a signal.
func runIndexer(cmd *cobra.Command, cfg *config.ConfigService, emb *embedder.FailoverEmbedder, projectPath string, p *tui.Progress, logger *slog.Logger) (stats index.Stats, elapsed time.Duration, skipped bool, err error) {
	dbPath := config.DBPathForProject(projectPath, emb.ModelName())
	if mkErr := os.MkdirAll(filepath.Dir(dbPath), 0o755); mkErr != nil {
		err = fmt.Errorf("create db directory: %w", mkErr)
		return
	}

	lockPath := indexlock.LockPathForDB(dbPath)
	lock, lockErr := indexlock.TryAcquire(lockPath)
	if lockErr != nil {
		err = fmt.Errorf("acquire index lock: %w", lockErr)
		return
	}
	if lock == nil {
		skipped = true
		return
	}
	defer lock.Release()

	// Seed before opening the DB, under the index lock acquired above so only
	// one indexer seeds. Doing it here (not just in the MCP getOrCreate path)
	// covers the background indexer that lumen index runs at SessionStart.
	seedFromDonorIfNew(dbPath, projectPath, emb.ModelName(), logger)

	// Cancel context on SIGTERM or SIGINT so the indexer stops cleanly and
	// the deferred lock.Release() runs before exit.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	idx, setupErr := setupIndexer(cfg, emb, dbPath, logger)
	if setupErr != nil {
		err = setupErr
		return
	}
	defer func() { _ = idx.Close() }()

	start := time.Now()
	stats, err = performIndexing(ctx, cmd, idx, projectPath, p)
	elapsed = time.Since(start).Round(time.Millisecond)
	if err != nil && ctx.Err() != nil {
		// A signal arrived; treat as clean exit.
		logger.Info("indexing cancelled by signal", "project", projectPath)
		err = nil
	}
	return
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
