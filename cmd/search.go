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
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/index"
	"github.com/spf13/cobra"
)

type traceSpan struct {
	label    string
	duration time.Duration
	detail   string
}

type tracer struct {
	enabled bool
	start   time.Time
	last    time.Time
	spans   []traceSpan
}

func (t *tracer) record(label, detail string) {
	if !t.enabled {
		return
	}
	now := time.Now()
	t.spans = append(t.spans, traceSpan{
		label:    label,
		duration: now.Sub(t.last),
		detail:   detail,
	})
	t.last = now
}

func (t *tracer) print(w io.Writer) {
	if !t.enabled {
		return
	}
	const sep = "───────────────────────────────────────────────────────────────────────"
	for _, s := range t.spans {
		ms := s.duration.Milliseconds()
		_, _ = fmt.Fprintf(w, "[%4dms] %-22s → %s\n", ms, s.label, s.detail)
	}
	_, _ = fmt.Fprintln(w, sep)
	total := time.Since(t.start)
	_, _ = fmt.Fprintf(w, "[%4dms] total\n", total.Milliseconds())
}

func init() {
	searchCmd.Flags().StringP("path", "p", "", "directory to search (default: cwd)")
	searchCmd.Flags().String("cwd", "", "project root when path is a subdirectory")
	searchCmd.Flags().IntP("n-results", "n", 8, "max results to return")
	searchCmd.Flags().Float64("min-score", 0, "minimum score threshold (-1 to 1)")
	searchCmd.Flags().Bool("summary", false, "omit code snippets, return location only")
	searchCmd.Flags().Int("max-lines", 0, "truncate snippets at N lines (0 = unlimited)")
	searchCmd.Flags().BoolP("force", "f", false, "force full re-index before searching")
	searchCmd.Flags().Bool("trace", false, "print per-phase timing to stderr")
	searchCmd.Flags().StringP("model", "m", "", "embedding model override")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search an indexed project for semantically similar code",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	traceEnabled, _ := cmd.Flags().GetBool("trace")
	pathFlag, _ := cmd.Flags().GetString("path")
	cwdFlag, _ := cmd.Flags().GetString("cwd")
	nResults, _ := cmd.Flags().GetInt("n-results")
	summary, _ := cmd.Flags().GetBool("summary")
	maxLines, _ := cmd.Flags().GetInt("max-lines")
	force, _ := cmd.Flags().GetBool("force")

	var minScore *float64
	if cmd.Flags().Changed("min-score") {
		v, _ := cmd.Flags().GetFloat64("min-score")
		minScore = &v
	}

	tr := &tracer{enabled: traceEnabled}
	if traceEnabled {
		tr.start = time.Now()
		tr.last = tr.start
	}

	// Span 1: path resolution
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := applyModelFlag(cmd, &cfg); err != nil {
		return err
	}

	projectPath := pathFlag
	if projectPath == "" {
		if cwdFlag != "" {
			projectPath = cwdFlag
		} else {
			projectPath, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
		}
	}
	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// When --cwd and --path both given, cwd is the index root.
	indexRoot := projectPath
	if cwdFlag != "" {
		abs, err := filepath.Abs(cwdFlag)
		if err != nil {
			return fmt.Errorf("resolve cwd: %w", err)
		}
		indexRoot = abs
	}

	tr.record("path resolution", indexRoot)

	// Span 2: indexer setup
	idx, err := setupIndexer(&cfg, indexRoot, nil)
	if err != nil {
		return fmt.Errorf("setup indexer: %w", err)
	}
	defer func() { _ = idx.Close() }()
	tr.record("indexer setup", fmt.Sprintf("db opened, model %s", cfg.Model))

	// Span 3: merkle + freshness (BuildTree happens inside EnsureFresh/Index)
	ctx := context.Background()

	if force {
		s, err := idx.Index(ctx, indexRoot, true, nil)
		if err != nil {
			return fmt.Errorf("force reindex: %w", err)
		}
		tr.record("merkle + freshness", fmt.Sprintf("reindexed %d files", s.IndexedFiles))
		emb, err := newEmbedder(cfg)
		if err != nil {
			return fmt.Errorf("create embedder: %w", err)
		}
		return finishSearch(cmd, ctx, tr, idx, emb, query, indexRoot, projectPath, nResults, minScore, summary, maxLines, true, s.IndexedFiles)
	}

	reindexed, s, err := idx.EnsureFresh(ctx, indexRoot, nil)
	if err != nil {
		return fmt.Errorf("ensure fresh: %w", err)
	}
	var freshnessDetail string
	if reindexed {
		freshnessDetail = fmt.Sprintf("reindexed %d files", s.IndexedFiles)
	} else {
		freshnessDetail = "index is fresh (no reindex)"
	}
	tr.record("merkle + freshness", freshnessDetail)

	emb, err := newEmbedder(cfg)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}

	indexedFiles := 0
	if reindexed {
		indexedFiles = s.IndexedFiles
	}
	return finishSearch(cmd, ctx, tr, idx, emb, query, indexRoot, projectPath, nResults, minScore, summary, maxLines, reindexed, indexedFiles)
}

func finishSearch(
	_ *cobra.Command,
	ctx context.Context,
	tr *tracer,
	idx *index.Indexer,
	emb embedder.Embedder,
	query, indexRoot, searchPath string,
	nResults int,
	minScore *float64,
	summary bool,
	maxLines int,
	reindexed bool,
	indexedFiles int,
) error {
	// Span 4: query embedding
	vecs, err := emb.Embed(ctx, []string{query})
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return fmt.Errorf("embedder returned no vectors")
	}
	queryVec := vecs[0]
	tr.record("query embedding", fmt.Sprintf("%d dims", len(queryVec)))

	// Span 5: KNN search
	fetchLimit := nResults * 2
	maxDistance := computeMaxDistance(minScore, emb.ModelName(), emb.Dimensions())

	var pathPrefix string
	if searchPath != indexRoot {
		if rel, relErr := filepath.Rel(indexRoot, searchPath); relErr == nil && rel != "." {
			pathPrefix = rel
		}
	}

	results, err := idx.Search(ctx, indexRoot, queryVec, fetchLimit, maxDistance, pathPrefix)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	tr.record("knn search", fmt.Sprintf("%d candidates fetched (limit=%d, fetch=%d)", len(results), nResults, fetchLimit))

	// Span 6: post-processing
	items := make([]SearchResultItem, len(results))
	for i, r := range results {
		items[i] = SearchResultItem{
			FilePath:  r.FilePath,
			Symbol:    r.Symbol,
			Kind:      r.Kind,
			StartLine: r.StartLine,
			EndLine:   r.EndLine,
			Score:     boostedScore(float32(1.0-r.Distance), r.Kind, r.FilePath),
		}
	}
	items = mergeOverlappingResults(items)
	slices.SortStableFunc(items, func(a, b SearchResultItem) int {
		return cmp.Compare(b.Score, a.Score)
	})
	if len(items) > nResults {
		items = items[:nResults]
	}
	if !summary {
		fillSnippets(indexRoot, items, maxLines)
	}
	tr.record("post-processing", fmt.Sprintf("merged %d→%d results, filled %d snippets", len(results), len(items), len(items)))

	// Print trace to stderr, then results to stdout.
	tr.print(os.Stderr)

	out := SemanticSearchOutput{
		Results:      items,
		Reindexed:    reindexed,
		IndexedFiles: indexedFiles,
	}
	fmt.Println(formatSearchResults(searchPath, out))
	return nil
}
