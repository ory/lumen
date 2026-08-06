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
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/index"
	"github.com/ory/lumen/internal/merkle"
	"github.com/spf13/cobra"
)

// serverStatus is the reachability outcome for a single configured embedding
// server.
type serverStatus struct {
	server    config.ServerConfig
	reachable bool
	message   string
}

// statusResult is the combined outcome of the two status checks: embedding
// service reachability (one entry per configured server) and index
// presence/freshness.
type statusResult struct {
	servers []serverStatus

	projectPath   string
	indexed       bool
	totalFiles    int
	indexedFiles  int
	totalChunks   int
	model         string
	lastIndexedAt string
	stale         bool
}

// anyServerReachable reports whether at least one configured embedding server
// responded. Lumen uses failover, so the service is usable as long as one
// server is up.
func anyServerReachable(r statusResult) bool {
	for _, s := range r.servers {
		if s.reachable {
			return true
		}
	}
	return false
}

// statusExitCode returns 0 only when at least one server is reachable, the
// index exists, and the index is fresh; otherwise 1.
func statusExitCode(r statusResult) int {
	if anyServerReachable(r) && r.indexed && !r.stale {
		return 0
	}
	return 1
}

// formatStatus renders a human-readable status report: one line per configured
// embedding server, followed by the index block (or a "not indexed" line).
func formatStatus(r statusResult) string {
	var b strings.Builder

	if len(r.servers) == 0 {
		fmt.Fprintf(&b, "Embedding service: ERROR — no servers configured\n")
	}
	for _, s := range r.servers {
		if s.reachable {
			fmt.Fprintf(&b, "Embedding service: OK (%s, %s, %s)\n", s.server.Backend, s.server.Host, s.server.Model)
		} else {
			fmt.Fprintf(&b, "Embedding service: ERROR (%s, %s) — %s\n", s.server.Backend, s.server.Host, s.message)
		}
	}

	if !r.indexed {
		fmt.Fprintf(&b, "Index: %s — not indexed (run `lumen index .`, or it will seed on first search)", r.projectPath)
		return b.String()
	}

	stale := "no"
	if r.stale {
		stale = "yes"
	}
	lastIndexed := r.lastIndexedAt
	if lastIndexed == "" {
		lastIndexed = "never"
	}
	fmt.Fprintf(&b, "Index: %s\n", r.projectPath)
	fmt.Fprintf(&b, "  Files: %d | Indexed: %d | Chunks: %d | Model: %s\n", r.totalFiles, r.indexedFiles, r.totalChunks, r.model)
	fmt.Fprintf(&b, "  Last indexed: %s | Stale: %s", lastIndexed, stale)
	return b.String()
}

// errStatusUnhealthy signals a non-zero exit without printing a cobra error
// banner; the human-readable report has already been written to stdout.
var errStatusUnhealthy = errors.New("status: unhealthy")

func init() {
	statusCmd.Flags().StringP("model", "m", "", "embedding model (default: $LUMEN_EMBED_MODEL or "+embedder.DefaultModel+")")
	statusCmd.Flags().StringP("backend", "b", "", "embedding backend to select (\"ollama\" or \"lmstudio\"); disambiguates when --model is configured on multiple backends")
	// runStatus prints the report itself; suppress cobra's error/usage output so
	// a non-zero exit on stale/unreachable does not double-print.
	statusCmd.SilenceErrors = true
	statusCmd.SilenceUsage = true
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Report embedding-service health and index freshness",
	Long: `Reports whether the configured embedding server(s) are reachable and
whether the project's index exists and is fresh. Every configured server is
reported on its own line.

With no argument, inspects the current directory. The path is normalized to the
git repository root (or an existing ancestor index) so it reports on the same
index that searches use.

status performs no indexing and never creates an index database. Exit code is 0
only when at least one server is reachable and the index exists and is fresh;
otherwise 1.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	target := "."
	if len(args) == 1 {
		target = args[0]
	}
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		return err
	}
	emb := newEmbedder(cfg)

	indexRoot, _, err := resolveIndexRoot(target, "", emb.ModelName())
	if err != nil {
		return err
	}

	r := collectStatus(cmd.Context(), cfg, emb, indexRoot)
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), formatStatus(r)); err != nil {
		return err
	}

	if statusExitCode(r) != 0 {
		return errStatusUnhealthy
	}
	return nil
}

func collectStatus(ctx context.Context, cfg *config.ConfigService, emb *embedder.FailoverEmbedder, projectPath string) statusResult {
	r := statusResult{projectPath: projectPath}

	// Probe every configured server concurrently, preserving config order in
	// the result.
	servers := cfg.Servers()
	r.servers = make([]serverStatus, len(servers))
	var wg sync.WaitGroup
	for i := range servers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			reachable, message := probeEmbeddingService(ctx, servers[i])
			r.servers[i] = serverStatus{server: servers[i], reachable: reachable, message: message}
		}(i)
	}
	wg.Wait()

	modelName := emb.ModelName()

	if unindexable, _ := merkle.IsRootUnindexable(projectPath); unindexable {
		r.indexed = false
		return r
	}

	// Detect a missing index without creating one: stat the DB file first.
	dbPath := config.DBPathForProject(projectPath, modelName)
	if _, statErr := os.Stat(dbPath); statErr != nil {
		r.indexed = false
		return r
	}

	idx, openErr := index.NewIndexer(dbPath, emb, cfg.MaxChunkTokens())
	if openErr != nil {
		r.indexed = false
		return r
	}
	defer func() { _ = idx.Close() }()

	info, infoErr := idx.Status(projectPath)
	if infoErr != nil {
		r.indexed = false
		return r
	}
	r.indexed = true
	r.totalFiles = info.TotalFiles
	r.indexedFiles = info.IndexedFiles
	r.totalChunks = info.TotalChunks
	r.model = info.EmbeddingModel
	r.lastIndexedAt = info.LastIndexedAt

	fresh, freshErr := idx.IsFresh(projectPath)
	if freshErr != nil {
		// Treat an un-checkable index as stale rather than silently fresh.
		r.stale = true
	} else {
		r.stale = !fresh
	}
	return r
}
