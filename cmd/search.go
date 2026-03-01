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
	"path/filepath"

	"github.com/aeneasr/agent-index/internal/config"
	"github.com/aeneasr/agent-index/internal/embedder"
	"github.com/aeneasr/agent-index/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	searchCmd.Flags().StringP("model", "m", "", "embedding model (default: $AGENT_INDEX_EMBED_MODEL or "+embedder.DefaultModel+")")
	searchCmd.Flags().IntP("limit", "l", 20, "max results to return")
	searchCmd.Flags().Float64P("min-score", "s", 0.5, "minimum score threshold (-1 to 1); use -1 to return all results")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search <query> <project-path>",
	Short: "Search an indexed project using natural language",
	Args:  cobra.ExactArgs(2),
	RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if m, _ := cmd.Flags().GetString("model"); m != "" {
		spec, ok := embedder.KnownModels[m]
		if !ok {
			return fmt.Errorf("unknown embedding model %q", m)
		}
		cfg.Model = m
		cfg.Dims = spec.Dims
		cfg.CtxLength = spec.CtxLength
	}

	projectPath, err := filepath.Abs(args[1])
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	dbPath := config.DBPathForProject(projectPath, cfg.Model)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("no index found for %s (model: %s)\nRun: agent-index index %s", projectPath, cfg.Model, args[1])
	}

	emb, err := newEmbedder(cfg)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}

	st, err := store.New(dbPath, cfg.Dims)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	vecs, err := emb.Embed(context.Background(), []string{query})
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}

	limit, _ := cmd.Flags().GetInt("limit")
	minScore, _ := cmd.Flags().GetFloat64("min-score")
	var maxDistance float64
	if minScore > -1 {
		maxDistance = 1.0 - minScore
	}
	results, err := st.Search(vecs[0], limit, maxDistance)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No results found.")
		return nil
	}

	snippets := extractSnippets(projectPath, results)

	// Build SearchResultItems for formatting.
	items := make([]SearchResultItem, len(results))
	for i, r := range results {
		items[i] = SearchResultItem{
			FilePath:  r.FilePath,
			Symbol:    r.Symbol,
			Kind:      r.Kind,
			StartLine: r.StartLine,
			EndLine:   r.EndLine,
			Score:     float32(1.0 - r.Distance),
			Content:   snippets[i],
		}
	}

	out := SemanticSearchOutput{Results: items}
	_, _ = fmt.Fprint(os.Stdout, formatSearchResults(projectPath, out))

	return nil
}
