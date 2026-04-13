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
	if err := applyModelFlag(cmd); err != nil {
		return err
	}

	cfg, err := config.NewConfigService(config.DefaultConfigPath())
	if err != nil {
		return err
	}

	emb := newEmbedder(cfg)
	emb.SetLogger(logger)

	projectPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	modelName := emb.ModelName()

	// Normalize to the git repository root when inside a git repo so that
	// indexing a subdirectory produces the same DB as indexing the repo root.
	// For non-git directories, walk up to reuse an existing ancestor index.
	if root, err := git.RepoRoot(projectPath); err == nil {
		projectPath = root
	} else if ancestor := findAncestorIndex(projectPath, modelName); ancestor != "" {
		projectPath = ancestor
	}

	// When the project directory is not a git repo, discover nested git repos
	// and index each one separately before indexing the parent (which will
	// only contain "loose" files not belonging to any nested repo).
	for _, repo := range git.DiscoverNestedGitRepos(projectPath) {
		p := tui.NewProgress(os.Stderr)
		defer p.RestoreCursor()
		p.Info(fmt.Sprintf("Indexing nested repo %s", repo))
		stats, elapsed, skipped, err := runIndexer(cmd, cfg, emb, repo, p)
		switch {
		case err != nil:
			fmt.Fprintf(os.Stderr, "Warning: failed to index nested repo %s: %v\n", repo, err)
		case skipped:
			fmt.Fprintf(os.Stderr, "Skipping %s: another indexer is already running.\n", repo)
		default:
			fmt.Printf("Nested repo %s: %d files, %d chunks in %s.\n", repo, stats.IndexedFiles, stats.ChunksCreated, elapsed)
		}
	}

	p := tui.NewProgress(os.Stderr)
	defer p.RestoreCursor()
	dims := cfg.ServerDims(0)
	p.Info(fmt.Sprintf("Indexing %s (model: %s, dims: %d)", projectPath, modelName, dims))
	stats, elapsed, skipped, err := runIndexer(cmd, cfg, emb, projectPath, p)
	if err != nil {
		return err
	}
	if skipped {
		fmt.Fprintln(os.Stderr, "Another indexer is already running for this project. Skipping.")
		return nil
	}

	if stats.Reason == "already fresh" {
		fmt.Printf("Index already fresh: %s\n", elapsed)
	} else {
		fmt.Printf("Indexing complete: %d files, %d chunks in %s.\n", stats.IndexedFiles, stats.ChunksCreated, elapsed)
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

// applyModelFlag sets LUMEN_EMBED_MODEL so that NewConfigService picks it up.
func applyModelFlag(cmd *cobra.Command) error {
	m, _ := cmd.Flags().GetString("model")
	if m == "" {
		return nil
	}
	canonical := m
	if c, ok := embedder.ModelAliases[m]; ok {
		canonical = c
	}
	if _, ok := embedder.KnownModels[canonical]; !ok {
		return fmt.Errorf("unknown embedding model %q", m)
	}
	_ = os.Setenv("LUMEN_EMBED_MODEL", m)
	return nil
}

// setupIndexer receives dbPath so it is computed exactly once in runIndex.
func setupIndexer(cfg *config.ConfigService, emb *embedder.FailoverEmbedder, dbPath string) (*index.Indexer, error) {
	idx, err := index.NewIndexer(dbPath, emb, cfg.MaxChunkTokens())
	if err != nil {
		return nil, fmt.Errorf("create indexer: %w", err)
	}
	return idx, nil
}

// runIndexer acquires the index lock for projectPath, runs performIndexing, and
// returns the stats and elapsed time. skipped is true when another indexer holds
// the lock. err is nil if indexing was cancelled by a signal.
func runIndexer(cmd *cobra.Command, cfg *config.ConfigService, emb *embedder.FailoverEmbedder, projectPath string, p *tui.Progress) (stats index.Stats, elapsed time.Duration, skipped bool, err error) {
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

	// Cancel context on SIGTERM or SIGINT so the indexer stops cleanly and
	// the deferred lock.Release() runs before exit.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	idx, setupErr := setupIndexer(cfg, emb, dbPath)
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
