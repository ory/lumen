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
	"bufio"
	"cmp"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/git"
	"github.com/ory/lumen/internal/index"
	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/merkle"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stdioCmd)
}

var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Start the MCP server on stdin/stdout",
	RunE:  runStdio,
}

// --- Tool input/output types ---

// SemanticSearchInput defines the parameters for the semantic_search tool.
type SemanticSearchInput struct {
	Query    string   `json:"query" jsonschema:"Natural language search query"`
	Path     string   `json:"path,omitempty" jsonschema:"Absolute path to search in. Defaults to cwd. When a subdirectory of cwd, results are filtered to that subtree."`
	Cwd      string   `json:"cwd,omitempty" jsonschema:"The current working directory / project root. Used as index root when provided."`
	Limit    int      `json:"limit,omitempty" jsonschema:"Max results to return, default 8. Pass a higher value for broader coverage."`
	MinScore *float64 `json:"min_score,omitempty" jsonschema:"Minimum score threshold (-1 to 1). Results below this score are excluded. Default depends on embedding model. Use -1 to return all results."`
	Summary  bool     `json:"summary,omitempty" jsonschema:"When true, return only file path, symbol, kind, line range, and score — no code content. Useful for location-only queries."`
	MaxLines int      `json:"max_lines,omitempty" jsonschema:"Truncate each code snippet to this many lines. Default: unlimited."`
}

// SearchResultItem represents a single search result returned to the caller.
type SearchResultItem struct {
	FilePath  string  `json:"file_path"`
	Symbol    string  `json:"symbol"`
	Kind      string  `json:"kind"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
	Content   string  `json:"content,omitempty"`
}

// SemanticSearchOutput is the structured output of the semantic_search tool.
type SemanticSearchOutput struct {
	Results      []SearchResultItem `json:"results"`
	Reindexed    bool               `json:"reindexed"`
	IndexedFiles int                `json:"indexed_files,omitempty"`
	FilteredHint string             `json:"filtered_hint,omitempty"`
	SeedWarning  string             `json:"seed_warning,omitempty"`
	StaleWarning string             `json:"stale_warning,omitempty"`
}

// IndexStatusInput defines the parameters for the index_status tool.
type IndexStatusInput struct {
	Path string `json:"path" jsonschema:"Absolute path to the project root. Defaults to cwd."`
	Cwd  string `json:"cwd,omitempty" jsonschema:"The current working directory / project root. Used as index root when provided."`
}

// IndexStatusOutput is the structured output of the index_status tool.
type IndexStatusOutput struct {
	ProjectPath    string `json:"project_path"`
	TotalFiles     int    `json:"total_files"`
	IndexedFiles   int    `json:"indexed_files"`
	TotalChunks    int    `json:"total_chunks"`
	LastIndexedAt  string `json:"last_indexed_at"`
	EmbeddingModel string `json:"embedding_model"`
	Stale          bool   `json:"stale"`
}

// HealthCheckInput defines the parameters for the health_check tool.
type HealthCheckInput struct{}

// HealthCheckOutput is the structured output of the health_check tool.
type HealthCheckOutput struct {
	Backend   string `json:"backend"`
	Host      string `json:"host"`
	Model     string `json:"model"`
	Reachable bool   `json:"reachable"`
	Message   string `json:"message"`
}

// --- indexerCache ---

// defaultFreshnessTTL is how long a confirmed-fresh index is trusted before
// the merkle tree is re-walked. Within a session Claude often issues many
// searches in quick succession; re-walking thousands of files on every call
// adds 1-3s of pure filesystem I/O even when nothing has changed.
// Override with LUMEN_FRESHNESS_TTL (e.g. "1s", "30s") for testing.
const defaultFreshnessTTL = 30 * time.Second
const defaultReindexTimeout = 15 * time.Second
const defaultSearchTimeout = 20 * time.Second
const backgroundReindexMaxDuration = 10 * time.Minute

type cacheEntry struct {
	idx           *index.Indexer
	effectiveRoot string
	model         string
	lastCheckedAt time.Time // zero means never checked
}

// cacheKey scopes cache entries by both project path and embedding model.
func cacheKey(projectPath, model string) string {
	return projectPath + "\x00" + model
}

func (ic *indexerCache) currentModel() string {
	if ic.embedder == nil {
		return ""
	}
	return ic.embedder.ModelName()
}

func (ic *indexerCache) cacheGet(projectPath, model string) (cacheEntry, bool) {
	if entry, ok := ic.cache[cacheKey(projectPath, model)]; ok {
		return entry, true
	}
	entry, ok := ic.cache[projectPath]
	if !ok {
		return cacheEntry{}, false
	}
	if entry.model != "" && entry.model != model {
		return cacheEntry{}, false
	}
	return entry, true
}

func (ic *indexerCache) cacheSet(projectPath, model string, entry cacheEntry) {
	if entry.model == "" {
		entry.model = model
	}
	ic.cache[cacheKey(projectPath, model)] = entry
	// Backward-compatible path-only alias used by existing tests and helper logic.
	ic.cache[projectPath] = entry
}

// indexerCache manages one *index.Indexer per project path, creating them
// lazily with a shared embedder.
type indexerCache struct {
	mu              sync.RWMutex
	cache           map[string]cacheEntry
	reindexing      map[string]bool // projects with an active background reindex goroutine
	embedder        embedder.Embedder
	cfg             *config.ConfigService
	freshnessTTL    time.Duration                                                                                                            // override for tests; 0 reads from cfg, then defaultFreshnessTTL
	reindexTimeout  time.Duration                                                                                                            // override for tests; 0 reads from cfg, then defaultReindexTimeout
	findDonorFunc   func(string, string) string                                                                                              // nil uses config.FindDonorIndex
	seedFunc        func(string, string) (bool, error)                                                                                       // nil uses index.SeedFromDonor
	ensureFreshFunc func(ctx context.Context, idx *index.Indexer, projectDir string, progress index.ProgressFunc) (bool, index.Stats, error) // nil uses idx.EnsureFresh
	log             *slog.Logger
	wg              sync.WaitGroup     // tracks background reindex goroutines
	closeCtx        context.Context    // cancelled by Close() to signal background goroutines
	closeFn         context.CancelFunc // cancels closeCtx
}

// getFreshnessTTL returns the effective freshness TTL, checking the override
// field first, then cfg, then the default constant.
func (ic *indexerCache) getFreshnessTTL() time.Duration {
	if ic.freshnessTTL != 0 {
		return ic.freshnessTTL
	}
	if ic.cfg != nil {
		if ttl := ic.cfg.FreshnessTTL(); ttl != 0 {
			return ttl
		}
	}
	return defaultFreshnessTTL
}

// getReindexTimeout returns the effective reindex timeout, checking the override
// field first, then cfg, then the default constant.
func (ic *indexerCache) getReindexTimeout() time.Duration {
	if ic.reindexTimeout != 0 {
		return ic.reindexTimeout
	}
	if ic.cfg != nil {
		if t := ic.cfg.ReindexTimeout(); t != 0 {
			return t
		}
	}
	return defaultReindexTimeout
}

// logger returns ic.log, falling back to a discarding logger when the field
// is nil (e.g. in unit tests that construct indexerCache directly).
func (ic *indexerCache) logger() *slog.Logger {
	if ic.log == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return ic.log
}

// Close cancels all background reindex goroutines, waits for them to drain
// (up to 30 seconds), then closes all cached indexers. Call on MCP server
// shutdown.
func (ic *indexerCache) Close() {
	// Signal all background goroutines to stop.
	if ic.closeFn != nil {
		ic.closeFn()
	}

	// Wait for goroutines to finish, with a hard ceiling so a stuck
	// goroutine (e.g. unresponsive embedder ignoring context) does not
	// block shutdown indefinitely.
	done := make(chan struct{})
	go func() {
		ic.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
	}

	ic.mu.Lock()
	defer ic.mu.Unlock()
	seen := make(map[*index.Indexer]bool)
	for _, entry := range ic.cache {
		if !seen[entry.idx] {
			seen[entry.idx] = true
			_ = entry.idx.Close()
		}
	}
	ic.cache = nil
}

// findEffectiveRoot walks up the directory tree from path's parent to find an
// existing parent index (either in cache or on disk). Returns path unchanged
// if no parent index is found. Must be called under ic.mu write lock.
//
// A candidate parent is skipped when the relative path from that parent to
// path passes through a directory in merkle.SkipDirs (e.g. "testdata"). Such
// a parent index would never contain path's files, so it is not useful.
func (ic *indexerCache) findEffectiveRoot(path string, model ...string) string {
	modelName := ic.currentModel()
	if len(model) > 0 && model[0] != "" {
		modelName = model[0]
	}
	// Cap the upward walk at the git repository root. This prevents lumen
	// from adopting a large ancestor index (e.g. a GOPATH index) that
	// happens to contain path as a subdirectory, which would cause
	// EnsureFresh to scan the entire ancestor tree.
	//
	// RepoRoot resolves symlinks (via git rev-parse --show-toplevel), so we
	// must also resolve candidate paths before comparing. On macOS /var is a
	// symlink to /private/var; without resolution the Rel check produces a
	// "../" prefix and breaks the walk prematurely.
	gitRoot, gitErr := git.RepoRoot(path)

	candidate := filepath.Dir(path)
	for {
		// Do not walk above the git repo root. Resolve symlinks on candidate
		// to match gitRoot (which is already resolved by git).
		if gitErr == nil {
			resolvedCandidate := candidate
			if r, err := filepath.EvalSymlinks(candidate); err == nil {
				resolvedCandidate = r
			}
			if rel, relErr := filepath.Rel(gitRoot, resolvedCandidate); relErr != nil || strings.HasPrefix(rel, "..") {
				break
			}
		}

		if !pathCrossesSkipDir(candidate, path) {
			if _, ok := ic.cacheGet(candidate, modelName); ok {
				return candidate
			}
			if _, err := os.Stat(config.DBPathForProject(candidate, modelName)); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
		candidate = parent
	}
	// No existing index found anywhere in the ancestry. Default to the git
	// repository root so all searches within the same repo share one index,
	// rather than creating a separate index per subdirectory.
	//
	// Skip the git-root fallback if the search path lives under a SkipDir
	// (e.g. testdata/). A git-root index would never include those files, so
	// defaulting to it only wastes time indexing the whole tree. Use the
	// search path itself instead, so test fixtures and similar directories get
	// their own scoped index.
	if gitErr == nil && !pathCrossesSkipDir(gitRoot, path) {
		return gitRoot
	}
	return path
}

// pathCrossesSkipDir reports whether the relative path from root to sub passes
// through any directory whose base name is in merkle.SkipDirs.
func pathCrossesSkipDir(root, sub string) bool {
	rel, err := filepath.Rel(root, sub)
	if err != nil {
		return false
	}
	for part := range strings.SplitSeq(rel, string(filepath.Separator)) {
		if merkle.SkipDirs[part] {
			return true
		}
	}
	return false
}

// hasIndex reports whether projectPath has an in-memory cached indexer or an
// on-disk SQLite database. Callers must hold ic.mu.
func (ic *indexerCache) hasIndex(projectPath string, model ...string) bool {
	modelName := ic.currentModel()
	if len(model) > 0 && model[0] != "" {
		modelName = model[0]
	}
	if _, ok := ic.cacheGet(projectPath, modelName); ok {
		return true
	}
	_, err := os.Stat(config.DBPathForProject(projectPath, modelName))
	return err == nil
}

// getOrCreate returns an existing Indexer for the given project path (or a
// parent index if one exists), along with the effective root directory used by
// the indexer, and a non-empty seedWarning if seeding from a sibling worktree
// failed. Creates a new indexer if none exists.
//
// When preferredRoot is non-empty it is used as the effective root directly,
// bypassing the findEffectiveRoot walk. This lets callers pass the known
// project root (e.g. cwd from Claude) so that sub-directory paths index the
// whole project.
func (ic *indexerCache) getOrCreate(projectPath string, preferredRoot string, model ...string) (*index.Indexer, string, string, error) {
	modelName := ic.currentModel()
	if len(model) > 0 && model[0] != "" {
		modelName = model[0]
	}

	// Fast path: read lock for already-cached indexers.
	ic.mu.RLock()
	if ic.cache != nil {
		if entry, ok := ic.cacheGet(projectPath, modelName); ok {
			ic.mu.RUnlock()
			ic.logger().Debug("indexer cache hit",
				"project_path", projectPath,
				"effective_root", entry.effectiveRoot,
				"model", modelName,
			)
			return entry.idx, entry.effectiveRoot, "", nil
		}
	}
	ic.mu.RUnlock()

	// Slow path: acquire write lock to create.
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if ic.cache == nil {
		ic.cache = make(map[string]cacheEntry)
	}
	// Double-check: another goroutine may have created it while we waited.
	if entry, ok := ic.cacheGet(projectPath, modelName); ok {
		return entry.idx, entry.effectiveRoot, "", nil
	}

	// Determine the effective root: prefer explicit root, then walk up.
	// When projectPath is a git worktree (has a .git FILE, not dir), only
	// use preferredRoot if it already has an index — otherwise fall back to
	// findEffectiveRoot to avoid adopting the outer monorepo root.
	var effectiveRoot string
	if preferredRoot != "" && !git.IsWorktree(projectPath) {
		clean := filepath.Clean(preferredRoot)
		// Only adopt the preferred root if an index already exists there.
		// Creating a brand-new index at cwd when path is a small subdirectory
		// would scan and embed the entire ancestor tree (e.g. a monorepo root),
		// making every first search prohibitively slow. Once an index exists at
		// the preferred root, subsequent searches reuse it and benefit from the
		// shared project-wide index.
		if _, err := os.Stat(config.DBPathForProject(clean, modelName)); err == nil {
			effectiveRoot = clean
		} else {
			effectiveRoot = ic.findEffectiveRoot(projectPath, modelName)
		}
	} else if git.IsWorktree(projectPath) && preferredRoot != "" && ic.hasIndex(preferredRoot, modelName) {
		// projectPath is a worktree but the parent project (preferredRoot) is
		// already indexed — reuse the parent index instead of creating a new
		// one. This handles the case where a search path points into an
		// internal .worktrees/ subdir of an already-indexed project.
		effectiveRoot = filepath.Clean(preferredRoot)
	} else {
		effectiveRoot = ic.findEffectiveRoot(projectPath, modelName)
	}

	// If a parent index is already cached, alias and return.
	// Guard: only reuse the cached entry if it is the actual owner of effectiveRoot
	// (entry.effectiveRoot == effectiveRoot). If the cache holds a guest-alias entry
	// for effectiveRoot (e.g. cache["src"] = {ccIdx, effectiveRoot:"cc"} written when
	// "src" was a projectPath routed to cc), reusing it here would pair the wrong
	// indexer with the wrong directory scope, causing EnsureFresh to scan one directory
	// and write results into a different DB.
	if effectiveRoot != projectPath {
		if entry, ok := ic.cacheGet(effectiveRoot, modelName); ok && entry.effectiveRoot == effectiveRoot {
			ic.cacheSet(projectPath, modelName, cacheEntry{idx: entry.idx, effectiveRoot: effectiveRoot, model: modelName})
			return entry.idx, effectiveRoot, "", nil
		}
	}

	dbPath := config.DBPathForProject(effectiveRoot, modelName)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, "", "", fmt.Errorf("create db directory: %w", err)
	}

	// Seed from sibling worktree if this is a new index.
	var seedWarning string
	isNewDB := false
	if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
		isNewDB = true
		ic.logger().Info("creating new index database",
			"effective_root", effectiveRoot,
			"db_path", dbPath,
			"model", modelName,
			"index_version", config.IndexVersion,
		)
		findDonor := ic.findDonorFunc
		if findDonor == nil {
			findDonor = config.FindDonorIndex
		}
		if donorPath := findDonor(effectiveRoot, modelName); donorPath != "" {
			ic.logger().Info("seeding index from donor worktree",
				"effective_root", effectiveRoot,
				"donor_path", donorPath,
			)
			seedFn := ic.seedFunc
			if seedFn == nil {
				seedFn = index.SeedFromDonor
			}
			if _, seedErr := seedFn(donorPath, dbPath); seedErr != nil {
				ic.logger().Warn("seed from donor worktree failed",
					"effective_root", effectiveRoot,
					"donor_path", donorPath,
					"error", seedErr,
				)
				seedWarning = fmt.Sprintf("index seeded from scratch (sibling copy failed: %v)", seedErr)
			}
		}
	}

	idx, err := index.NewIndexer(dbPath, ic.embedder, ic.cfg.MaxChunkTokens())
	if err != nil {
		return nil, "", "", fmt.Errorf("create indexer: %w", err)
	}
	idx.SetLogger(ic.logger())

	// Pre-populate the freshness TTL if the index was recently stamped by
	// background pre-warming (SessionStart hook). This avoids a redundant
	// merkle walk on the very first search in a new session.
	entry := cacheEntry{idx: idx, effectiveRoot: effectiveRoot, model: modelName}
	if lastAt, ok := idx.LastIndexedAt(); ok {
		ttl := ic.getFreshnessTTL()
		if time.Since(lastAt) < ttl {
			entry.lastCheckedAt = lastAt
		}
	}

	ic.logger().Info("indexer created",
		"project_path", projectPath,
		"effective_root", effectiveRoot,
		"db_path", dbPath,
		"new_index", isNewDB,
		"model", modelName,
		"index_version", config.IndexVersion,
	)

	ic.cacheSet(effectiveRoot, modelName, entry)
	if effectiveRoot != projectPath {
		ic.cacheSet(projectPath, modelName, cacheEntry{idx: idx, effectiveRoot: effectiveRoot, model: modelName, lastCheckedAt: entry.lastCheckedAt})
	}
	return idx, effectiveRoot, seedWarning, nil
}

// handleSemanticSearch is the tool handler for the semantic_search tool.
// Uses Out=any so the SDK does not set StructuredContent — the LLM sees
// only the plaintext in Content.
func (ic *indexerCache) handleSemanticSearch(ctx context.Context, req *mcp.CallToolRequest, input SemanticSearchInput) (*mcp.CallToolResult, any, error) {
	if err := validateSearchInput(&input); err != nil {
		return nil, nil, err
	}

	ic.logger().Debug("semantic search request",
		"cwd", input.Cwd,
		"search_path", input.Path,
		"limit", input.Limit,
	)

	modelName := ic.currentModel()
	idx, effectiveRoot, seedWarning, err := ic.getOrCreate(input.Path, input.Cwd, modelName)
	if err != nil {
		return nil, nil, fmt.Errorf("get indexer: %w", err)
	}

	progress := buildProgressFunc(ctx, req)

	dbPath := config.DBPathForProject(effectiveRoot, modelName)
	out, err := ic.ensureIndexed(idx, input, effectiveRoot, dbPath, progress)
	if err != nil {
		return nil, nil, err
	}
	out.SeedWarning = seedWarning

	queryVec, err := ic.embedQuery(ctx, input.Query)
	if err != nil {
		return nil, nil, err
	}

	maxDistance := computeMaxDistance(input.MinScore, modelName, ic.embedder.Dimensions())

	// When searching a subdirectory, filter results to that prefix only.
	var pathPrefix string
	if input.Path != effectiveRoot {
		if rel, relErr := filepath.Rel(effectiveRoot, input.Path); relErr == nil && rel != "." {
			pathPrefix = rel
		}
	}

	// Over-fetch from KNN so that merging overlapping split chunks doesn't
	// reduce the final result count below the requested limit.
	fetchLimit := input.Limit * 2

	// Defense-in-depth: bound the search call so that even if a future
	// regression reintroduces mutex contention, we return within a
	// predictable time rather than hanging indefinitely.
	searchCtx, searchCancel := context.WithTimeout(ctx, defaultSearchTimeout)
	defer searchCancel()

	results, err := idx.Search(searchCtx, effectiveRoot, queryVec, fetchLimit, maxDistance, pathPrefix)
	if err != nil {
		return nil, nil, fmt.Errorf("search: %w", err)
	}

	// When the noise floor filtered out all results, check whether unfiltered
	// results exist so we can tell the caller why the search came up empty.
	if len(results) == 0 && maxDistance > 0 {
		unfiltered, ufErr := idx.Search(searchCtx, effectiveRoot, queryVec, 1, 0, pathPrefix)
		if ufErr == nil && len(unfiltered) > 0 {
			bestScore := 1.0 - unfiltered[0].Distance
			noiseFloor := 1.0 - maxDistance
			out.FilteredHint = fmt.Sprintf(
				"Results exist but were below the %.2f noise floor (best match scored %.2f). "+
					"Use min_score=-1 to see all results, or lower min_score.",
				noiseFloor, bestScore,
			)
		}
	}

	// Convert store results to SearchResultItems with boosted scores.
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

	// Merge overlapping/adjacent chunks from the same file to eliminate
	// duplicates caused by split chunks and to present cohesive results.
	items = mergeOverlappingResults(items)

	// Re-sort by boosted score so documentation does not outrank source code.
	slices.SortStableFunc(items, func(a, b SearchResultItem) int {
		return cmp.Compare(b.Score, a.Score)
	})

	// Cap to the originally requested limit after merging.
	if len(items) > input.Limit {
		items = items[:input.Limit]
	}

	// Extract snippets for merged results.
	if !input.Summary {
		fillSnippets(effectiveRoot, items, input.MaxLines)
	}

	out.Results = items
	text := formatSearchResults(input.Path, out)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func validateSearchInput(input *SemanticSearchInput) error {
	if input.Cwd != "" {
		input.Cwd = filepath.Clean(input.Cwd)
		if !filepath.IsAbs(input.Cwd) {
			return fmt.Errorf("cwd must be an absolute path")
		}
		if resolved, err := filepath.EvalSymlinks(input.Cwd); err == nil {
			input.Cwd = resolved
		}
	}

	if input.Path == "" && input.Cwd != "" {
		input.Path = input.Cwd
	}
	if input.Path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("path is required (or provide cwd): %w", err)
		}
		input.Path = wd
		if input.Cwd == "" {
			input.Cwd = wd
		}
	}

	input.Path = filepath.Clean(input.Path)
	if resolved, err := filepath.EvalSymlinks(input.Path); err == nil {
		input.Path = resolved
	}

	if input.Cwd != "" && input.Path != input.Cwd {
		rel, err := filepath.Rel(input.Cwd, input.Path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("path must be equal to or under cwd")
		}
	}

	if input.Query == "" {
		return fmt.Errorf("query is required")
	}
	if input.Limit <= 0 {
		input.Limit = 8
	}
	return nil
}

func buildProgressFunc(ctx context.Context, req *mcp.CallToolRequest) index.ProgressFunc {
	token := req.Params.GetProgressToken()
	if token == nil {
		return nil
	}
	return func(current, total int, message string) {
		if total == 0 {
			// Skip indeterminate notifications — MCP progress requires Total > 0.
			return
		}
		_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
			ProgressToken: token,
			Progress:      float64(current),
			Total:         float64(total),
			Message:       message,
		})
	}
}

func (ic *indexerCache) ensureIndexed(idx *index.Indexer, input SemanticSearchInput, projectDir string, dbPath string, progress index.ProgressFunc, model ...string) (SemanticSearchOutput, error) {
	start := time.Now()
	out := SemanticSearchOutput{}
	modelName := ic.currentModel()
	if len(model) > 0 && model[0] != "" {
		modelName = model[0]
	}
	reindexKey := cacheKey(projectDir, modelName)

	// Skip the merkle tree walk if we confirmed freshness recently. The walk
	// costs 1-3s on large projects even when nothing changed.
	if ic.recentlyChecked(projectDir, modelName) {
		ic.logger().Debug("freshness TTL hit, skipping merkle check",
			"cwd", input.Cwd,
			"effective_root", projectDir,
			"elapsed_ms", time.Since(start).Milliseconds(),
		)
		return out, nil
	}

	// If a background indexer holds the exclusive flock, skip EnsureFresh to
	// avoid duplicating the in-progress Merkle walk. The TOCTOU race is benign:
	// worst case is redundant work, not corruption (SQLite WAL mode).
	if indexlock.IsHeld(indexlock.LockPathForDB(dbPath)) {
		ic.logger().Info("skipping reindex: background indexer is running", "project", projectDir)
		out.StaleWarning = "Index is being updated in the background. Results may be incomplete or outdated. Use standard tools for the next 10 tool calls before trying semantic_search again."
		return out, nil
	}

	ic.logger().Debug("freshness TTL expired or first check, building merkle tree",
		"cwd", input.Cwd,
		"effective_root", projectDir,
	)

	// If an in-process background goroutine is already reindexing this
	// project, skip spawning another one. This prevents redundant merkle
	// tree walks and avoids macOS flock(2) same-process reentrance issues.
	ic.mu.Lock()
	if ic.reindexing != nil && ic.reindexing[reindexKey] {
		ic.mu.Unlock()
		ic.logger().Debug("skipping reindex: in-process background goroutine already running", "project", projectDir)
		out.StaleWarning = "Index is being updated in the background. Results may be incomplete or outdated. Use standard tools for the next 10 tool calls before trying semantic_search again."
		return out, nil
	}
	if ic.reindexing == nil {
		ic.reindexing = make(map[string]bool)
	}
	ic.reindexing[reindexKey] = true
	ic.mu.Unlock()

	// Run EnsureFresh in a goroutine with a 15s timeout. If reindexing
	// takes longer, return stale results with a warning while the
	// goroutine continues in the background.
	type freshResult struct {
		reindexed bool
		stats     index.Stats
		err       error
		skipped   bool // true when flock was held by another process
	}
	done := make(chan freshResult, 1) // buffered: goroutine must never block on send

	bgParent := context.Background()
	if ic.closeCtx != nil {
		bgParent = ic.closeCtx
	}
	bgCtx, bgCancel := context.WithTimeout(bgParent, backgroundReindexMaxDuration)

	lockPath := indexlock.LockPathForDB(dbPath)
	ic.wg.Go(func() {
		defer bgCancel()
		defer func() {
			ic.mu.Lock()
			delete(ic.reindexing, reindexKey)
			ic.mu.Unlock()
		}()

		lk, lockErr := indexlock.TryAcquire(lockPath)
		if lockErr != nil {
			ic.logger().Warn("background reindex: failed to acquire lock", "project", projectDir, "err", lockErr)
			done <- freshResult{skipped: true}
			return
		}
		if lk == nil {
			// Another process grabbed the lock between our IsHeld check and now.
			ic.logger().Debug("background reindex: lock held by another process, skipping", "project", projectDir)
			done <- freshResult{skipped: true}
			return
		}
		defer lk.Release()

		// If a recent external process (e.g. lumen index from SessionStart)
		// already updated the index within freshnessTTL, trust the DB timestamp
		// and skip the expensive merkle tree walk.
		if lastAt, ok := idx.LastIndexedAt(); ok {
			ttl := ic.getFreshnessTTL()
			if ttl == 0 {
				ttl = defaultFreshnessTTL
			}
			if time.Since(lastAt) < ttl {
				ic.logger().Debug("skipping merkle walk: index recently updated by external process",
					"project", projectDir,
					"last_indexed_at", lastAt,
				)
				ic.touchChecked(projectDir, modelName)
				done <- freshResult{}
				return
			}
		}

		ensureFresh := ic.ensureFreshFunc
		if ensureFresh == nil {
			ensureFresh = func(ctx context.Context, idx *index.Indexer, dir string, p index.ProgressFunc) (bool, index.Stats, error) {
				return idx.EnsureFresh(ctx, dir, p)
			}
		}
		reindexed, stats, err := ensureFresh(bgCtx, idx, projectDir, progress)
		if err != nil {
			ic.logger().Warn("background reindex failed", "project", projectDir, "err", err)
		} else {
			ic.touchChecked(projectDir, modelName)
		}
		done <- freshResult{reindexed: reindexed, stats: stats, err: err}
	})

	timeout := ic.getReindexTimeout()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-done:
		bgCancel() // release context resources early
		if result.skipped {
			ic.logger().Info("reindex skipped: lock held by another process", "project", projectDir)
			out.StaleWarning = "Index is being updated in the background. Results may be incomplete or outdated. Use standard tools for the next 10 tool calls before trying semantic_search again."
			return out, nil
		}
		if result.err != nil {
			return out, fmt.Errorf("ensure fresh: %w", result.err)
		}
		elapsed := time.Since(start)
		if !result.reindexed {
			ic.logger().Debug("index fresh, caching result",
				"cwd", input.Cwd,
				"effective_root", projectDir,
				"elapsed_ms", elapsed.Milliseconds(),
			)
		} else {
			ic.logger().Info("reindex triggered",
				"cwd", input.Cwd,
				"search_path", input.Path,
				"effective_root", projectDir,
				"reason", result.stats.Reason,
				"total_project_files", result.stats.TotalFiles,
				"files_indexed", result.stats.IndexedFiles,
				"chunks_created", result.stats.ChunksCreated,
				"files_added", result.stats.FilesAdded,
				"files_modified", result.stats.FilesModified,
				"files_removed", result.stats.FilesRemoved,
				"old_root_hash", result.stats.OldRootHash,
				"new_root_hash", result.stats.NewRootHash,
				"elapsed_ms", elapsed.Milliseconds(),
			)
		}
		out.Reindexed = result.reindexed
		if result.reindexed {
			out.IndexedFiles = result.stats.IndexedFiles
		}
		return out, nil

	case <-timer.C:
		ic.logger().Info("reindex timeout, returning stale results",
			"project", projectDir,
			"timeout", timeout,
		)
		out.StaleWarning = "Index is being updated in the background. Results may be incomplete or outdated. Use standard tools for the next 10 tool calls before trying semantic_search again."
		return out, nil
	}
}

// recentlyChecked reports whether the index for projectDir was confirmed fresh
// within freshnessTTL. Reads under RLock so it is safe to call concurrently.
func (ic *indexerCache) recentlyChecked(projectDir string, model ...string) bool {
	modelName := ic.currentModel()
	if len(model) > 0 && model[0] != "" {
		modelName = model[0]
	}
	ic.mu.RLock()
	entry, ok := ic.cacheGet(projectDir, modelName)
	ic.mu.RUnlock()
	ttl := ic.getFreshnessTTL()
	if ttl == 0 {
		ttl = defaultFreshnessTTL
	}
	return ok && !entry.lastCheckedAt.IsZero() && time.Since(entry.lastCheckedAt) < ttl
}

// touchChecked records the current time as the last freshness-check time for
// projectDir. It updates both the projectDir entry and its effectiveRoot entry
// (which may differ when projectDir is a subdirectory alias).
func (ic *indexerCache) touchChecked(projectDir string, model ...string) {
	modelName := ic.currentModel()
	if len(model) > 0 && model[0] != "" {
		modelName = model[0]
	}
	now := time.Now()
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if entry, ok := ic.cacheGet(projectDir, modelName); ok {
		entry.lastCheckedAt = now
		ic.cacheSet(projectDir, modelName, entry)
		if entry.effectiveRoot != projectDir {
			if root, ok := ic.cacheGet(entry.effectiveRoot, modelName); ok {
				root.lastCheckedAt = now
				ic.cacheSet(entry.effectiveRoot, modelName, root)
			}
		}
	}
}

func (ic *indexerCache) embedQuery(ctx context.Context, query string) ([]float32, error) {
	vecs, err := ic.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embedder returned no vectors")
	}
	return vecs[0], nil
}

// computeMaxDistance converts a user-facing min_score into the cosine distance
// ceiling used by sqlite-vec. When no explicit score is given, the noise floor
// is determined from KnownModels or, for unknown models, from the embedding
// dimensionality via DimensionAwareMinScore.
func computeMaxDistance(minScore *float64, model string, dims int) float64 {
	if minScore == nil {
		if spec, ok := embedder.KnownModels[model]; ok && spec.MinScore > 0 {
			return 1.0 - spec.MinScore
		}
		if dims > 0 {
			return 1.0 - embedder.DimensionAwareMinScore(dims)
		}
		return 1.0 - embedder.DefaultMinScore
	}
	if *minScore > -1 {
		return 1.0 - *minScore
	}
	return 0 // -1 means no filter
}

// handleIndexStatus is the tool handler for the index_status tool.
// Uses Out=any so the SDK does not set StructuredContent.
func (ic *indexerCache) handleIndexStatus(_ context.Context, _ *mcp.CallToolRequest, input IndexStatusInput) (*mcp.CallToolResult, any, error) {
	if input.Path == "" && input.Cwd != "" {
		input.Path = input.Cwd
	}
	if input.Path == "" {
		return nil, nil, fmt.Errorf("path is required (or provide cwd)")
	}

	modelName := ic.currentModel()
	idx, effectiveRoot, _, err := ic.getOrCreate(input.Path, input.Cwd, modelName)
	if err != nil {
		return nil, nil, fmt.Errorf("get indexer: %w", err)
	}

	info, err := idx.Status(effectiveRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("get status: %w", err)
	}

	out := IndexStatusOutput{
		ProjectPath:    info.ProjectPath,
		TotalFiles:     info.TotalFiles,
		IndexedFiles:   info.IndexedFiles,
		TotalChunks:    info.TotalChunks,
		LastIndexedAt:  info.LastIndexedAt,
		EmbeddingModel: info.EmbeddingModel,
	}

	fresh, err := idx.IsFresh(effectiveRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("check freshness: %w", err)
	}
	out.Stale = !fresh

	text := formatIndexStatus(out)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

// handleHealthCheck pings the configured embedding service and reports status.
// It checks the currently active server (after failover), not always server[0].
func (ic *indexerCache) handleHealthCheck(ctx context.Context, _ *mcp.CallToolRequest, _ HealthCheckInput) (*mcp.CallToolResult, any, error) {
	servers := ic.cfg.Servers()
	idx := 0
	if fe, ok := ic.embedder.(*embedder.FailoverEmbedder); ok {
		if active := fe.ActiveServerIndex(); active >= 0 && active < len(servers) {
			idx = active
		}
	}
	srv := servers[idx]
	backend := srv.Backend
	host := srv.Host
	model := srv.Model
	probeURL := host + "/api/tags"
	if backend == config.BackendLMStudio {
		probeURL = host + "/v1/models"
	}

	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, probeURL, nil)
	if err != nil {
		return healthResult(backend, host, model, false,
			fmt.Sprintf("failed to create request: %v", err)), nil, nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return healthResult(backend, host, model, false,
			fmt.Sprintf("service unreachable: %v", err)), nil, nil
	}
	_ = resp.Body.Close()

	if resp.StatusCode >= 500 {
		return healthResult(backend, host, model, false,
			fmt.Sprintf("service returned HTTP %d", resp.StatusCode)), nil, nil
	}

	return healthResult(backend, host, model, true, "service is healthy"), nil, nil
}

func healthResult(backend, host, model string, reachable bool, message string) *mcp.CallToolResult {
	status := "OK"
	if !reachable {
		status = "ERROR"
	}
	text := fmt.Sprintf("Backend: %s\nHost: %s\nModel: %s\nStatus: %s\nMessage: %s",
		backend, host, model, status, message)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		IsError: !reachable,
	}
}

func readFileLines(projectPath, filePath string) []string {
	absPath := filepath.Join(projectPath, filePath)
	f, err := os.Open(absPath)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // up to 1MB per line
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// truncateLines returns at most maxLines lines from a string.
func truncateLines(s string, maxLines int) string {
	lines := strings.SplitN(s, "\n", maxLines+1)
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

func normalizeLineRange(startLine, endLine, totalLines int) (int, int) {
	start := max(startLine-1, 0)
	end := min(endLine, totalLines)
	return start, end
}

// mergeOverlappingResults combines search results from the same file whose
// line ranges overlap or are adjacent (within 5 lines). This eliminates
// duplicates from split chunks and produces cohesive results. The highest
// score among merged items is kept.
func mergeOverlappingResults(items []SearchResultItem) []SearchResultItem {
	if len(items) == 0 {
		return nil
	}

	// Group by file path.
	type group struct {
		items []SearchResultItem
	}
	groups := make(map[string]*group)
	var order []string
	for _, item := range items {
		g, ok := groups[item.FilePath]
		if !ok {
			g = &group{}
			groups[item.FilePath] = g
			order = append(order, item.FilePath)
		}
		g.items = append(g.items, item)
	}

	var merged []SearchResultItem
	const adjacencyGap = 5

	for _, fp := range order {
		g := groups[fp]
		// Sort by StartLine to find overlaps.
		slices.SortFunc(g.items, func(a, b SearchResultItem) int {
			return cmp.Compare(a.StartLine, b.StartLine)
		})

		// Merge overlapping/adjacent items.
		current := g.items[0]
		for _, next := range g.items[1:] {
			if next.StartLine <= current.EndLine+adjacencyGap {
				// Merge: extend line range, keep best score, join symbols.
				if next.EndLine > current.EndLine {
					current.EndLine = next.EndLine
				}
				if next.Score > current.Score {
					current.Score = next.Score
					current.Kind = next.Kind
				}
				if !strings.Contains(current.Symbol, next.Symbol) {
					current.Symbol = current.Symbol + "+" + next.Symbol
				}
			} else {
				merged = append(merged, current)
				current = next
			}
		}
		merged = append(merged, current)
	}

	return merged
}

// fillSnippets reads source files and populates Content for each result item.
func fillSnippets(projectPath string, items []SearchResultItem, maxLines int) {
	// Group by file to read each file once.
	type ref struct {
		idx       int
		startLine int
		endLine   int
	}
	byFile := make(map[string][]ref)
	for i, item := range items {
		byFile[item.FilePath] = append(byFile[item.FilePath], ref{i, item.StartLine, item.EndLine})
	}

	for filePath, refs := range byFile {
		lines := readFileLines(projectPath, filePath)
		if lines == nil {
			continue
		}
		for _, r := range refs {
			start, end := normalizeLineRange(r.startLine, r.endLine, len(lines))
			if start >= end {
				continue
			}
			content := strings.Join(lines[start:end], "\n")
			if maxLines > 0 && content != "" {
				content = truncateLines(content, maxLines)
			}
			items[r.idx].Content = content
		}
	}
}

// sourceCodeKinds lists chunk kinds that represent source code declarations.
// These receive a score boost to outrank documentation and changelog chunks.
var sourceCodeKinds = map[string]bool{
	"function":  true,
	"method":    true,
	"type":      true,
	"interface": true,
	"const":     true,
	"var":       true,
}

// boostedScore adjusts the raw cosine score of a chunk based on its kind and
// file type. Source code declarations get a 1.15x boost; test files are
// demoted by 0.75x so that implementation code clearly outranks test helpers
// for concept queries. The result is capped at 1.0.
func boostedScore(score float32, kind, filePath string) float32 {
	if sourceCodeKinds[kind] {
		if boosted := score * 1.15; boosted < 1.0 {
			score = boosted
		} else {
			score = 1.0
		}
	}
	if isTestFile(filePath) {
		score *= 0.75
	}
	return score
}

// isTestFile reports whether filePath looks like a test file across common
// language conventions: Go (*_test.go), Rust (*_test.rs), Ruby (*_spec.rb),
// JS/TS (*.test.*, *.spec.*), Python (test_*.py, tests/ directory),
// Java (src/test/), PHP (tests/).
func isTestFile(filePath string) bool {
	lower := strings.ToLower(filepath.ToSlash(filePath))
	base := filepath.Base(lower)
	ext := filepath.Ext(base)
	nameNoExt := strings.TrimSuffix(base, ext)

	// Filename-based patterns.
	switch {
	case strings.HasSuffix(nameNoExt, "_test"),
		strings.HasSuffix(nameNoExt, "_spec"),
		strings.HasSuffix(nameNoExt, ".test"),
		strings.HasSuffix(nameNoExt, ".spec"),
		strings.HasPrefix(nameNoExt, "test_"),
		strings.Contains(base, ".test."),
		strings.Contains(base, ".spec."):
		return true
	}

	// Directory-based patterns: check each path segment.
	for dir := filepath.Dir(lower); dir != "." && dir != "/"; dir = filepath.Dir(dir) {
		switch filepath.Base(dir) {
		case "test", "tests", "__tests__":
			return true
		}
	}
	return false
}

var xmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
)

// formatSearchResults builds an XML-tagged representation of search results
// for LLM consumption. File paths are shown relative to the project root.
// Chunks from the same file are grouped under a <result:file> element to
// reduce repetition. Overlapping chunks are merged before this function is
// called, so no per-file cap is needed.
func formatSearchResults(projectPath string, out SemanticSearchOutput) string {
	if len(out.Results) == 0 {
		var b strings.Builder
		b.WriteString("No results found.")
		if out.Reindexed {
			fmt.Fprintf(&b, " (indexed %d files)", out.IndexedFiles)
		}
		if out.SeedWarning != "" {
			b.WriteString("\nWarning: ")
			b.WriteString(out.SeedWarning)
		}
		if out.StaleWarning != "" {
			b.WriteString("\nWarning: ")
			b.WriteString(out.StaleWarning)
		}
		if out.FilteredHint != "" {
			b.WriteString("\n")
			b.WriteString(out.FilteredHint)
		}
		return b.String()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d results", len(out.Results))
	if out.Reindexed {
		fmt.Fprintf(&b, " (indexed %d files)", out.IndexedFiles)
	}
	if out.SeedWarning != "" {
		fmt.Fprintf(&b, "\nWarning: %s", out.SeedWarning)
	}
	if out.StaleWarning != "" {
		fmt.Fprintf(&b, "\nWarning: %s", out.StaleWarning)
	}
	b.WriteString(":\n")

	// Group results by relative file path.
	type fileGroup struct {
		rel      string
		results  []SearchResultItem
		maxScore float32
	}
	var order []string
	groups := make(map[string]*fileGroup)
	for _, r := range out.Results {
		rel, err := filepath.Rel(projectPath, r.FilePath)
		if err != nil {
			rel = r.FilePath
		}
		if _, ok := groups[rel]; !ok {
			order = append(order, rel)
			groups[rel] = &fileGroup{rel: rel}
		}
		g := groups[rel]
		g.results = append(g.results, r)
		if r.Score > g.maxScore {
			g.maxScore = r.Score
		}
	}

	// Sort files by best chunk score descending.
	slices.SortFunc(order, func(a, b string) int {
		return cmp.Compare(groups[b].maxScore, groups[a].maxScore)
	})

	for _, rel := range order {
		g := groups[rel]
		// Sort chunks within each file by score descending.
		slices.SortFunc(g.results, func(a, b SearchResultItem) int {
			return cmp.Compare(b.Score, a.Score)
		})
		fmt.Fprintf(&b, "\n<result:file filename=\"%s\">\n", xmlEscaper.Replace(g.rel))
		for _, r := range g.results {
			fmt.Fprintf(&b, "  <result:chunk line-start=\"%d\" line-end=\"%d\" symbol=\"%s\" kind=\"%s\" score=\"%.2f\">\n",
				r.StartLine, r.EndLine, xmlEscaper.Replace(r.Symbol), xmlEscaper.Replace(r.Kind), r.Score)
			if r.Content != "" {
				b.WriteString(r.Content)
				b.WriteByte('\n')
			}
			b.WriteString("  </result:chunk>\n")
		}
		b.WriteString("</result:file>")
	}

	return b.String()
}

// formatIndexStatus builds a compact plaintext representation of index status.
func formatIndexStatus(out IndexStatusOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Index: %s\n", out.ProjectPath)
	fmt.Fprintf(&b, "Files: %d | Indexed: %d | Chunks: %d | Model: %s\n", out.TotalFiles, out.IndexedFiles, out.TotalChunks, out.EmbeddingModel)
	stale := "no"
	if out.Stale {
		stale = "yes"
	}
	lastIndexed := out.LastIndexedAt
	if lastIndexed == "" {
		lastIndexed = "never"
	}
	fmt.Fprintf(&b, "Last indexed: %s | Stale: %s", lastIndexed, stale)
	return b.String()
}

func runStdio(_ *cobra.Command, _ []string) error {
	cfg, err := config.NewConfigService(config.DefaultConfigPath())
	if err != nil {
		return err
	}
	if err := cfg.Watch(); err != nil {
		return fmt.Errorf("watch config: %w", err)
	}
	defer cfg.Stop()

	emb := newEmbedder(cfg)

	logger, logFile := newDebugLogger()
	emb.SetLogger(logger)
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}

	logger.Info("lumen config",
		"model", emb.ModelName(),
		"backend", cfg.Servers()[0].Backend,
		"freshness_ttl", cfg.FreshnessTTL().String(),
	)

	closeCtx, closeFn := context.WithCancel(context.Background())
	indexers := &indexerCache{
		embedder: emb,
		cfg:      cfg,
		log:      logger,
		closeCtx: closeCtx,
		closeFn:  closeFn,
	}
	defer indexers.Close()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "lumen",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name: "semantic_search",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
			Title:        "Semantic Code Search",
		},
		Description: `Search indexed codebase using natural language. ALWAYS use semantic_search as the FIRST tool for code discovery and exploration.

Do NOT default to Grep, Glob, or Read for search tasks — only use them for exact literal string lookups.

Before using Search, Grep, Glob, Find, or Read for any search, stop and ask:

> "Do I already know the exact literal string I'm searching for?"

- **No** — understanding how something works, finding where something is implemented, exploring
  unfamiliar code → use **semantic search**
- **Yes** — a specific function name, import path, variable name, or error message you already
  know exists → Grep/Glob is acceptable for that exact string only

# ALWAYS use semantic search as the first tool for code discovery

This includes:

- Understanding how a system or feature works
- Finding where functionality is implemented
- Discovering what calls what or how components connect
- Locating code related to a concept or domain term
- Finding relevant code before making changes

Auto-indexes if the index is stale or empty.

Tip: If a search returns no results, retry with a lower min_score (e.g. 0.0 or -1) before trying a completely different query. Increase limit (e.g. limit: 20) for broader coverage.`,
	}, indexers.handleSemanticSearch)

	mcp.AddTool(server, &mcp.Tool{
		Name: "health_check",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
			Title:        "Embedding Service Health Check",
		},
		Description: `Check if the configured embedding service (Ollama or LM Studio) is reachable and healthy.

Reports backend type, host, model name, and connection status. Use this to diagnose embedding failures or verify service availability.`,
	}, indexers.handleHealthCheck)

	mcp.AddTool(server, &mcp.Tool{
		Name: "index_status",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
			Title:        "Code Index Status",
		},
		Description: `Check the indexing status of a project. Shows total files, indexed chunks, and embedding model.

Use this to verify a project is indexed before searching, or to check if the index is up to date.

Note: You do NOT need to call index_status before semantic_search. Semantic search auto-indexes automatically. Only use this tool when the user explicitly asks about index status, or to diagnose why search results seem incomplete.`,
	}, indexers.handleIndexStatus)

	return server.Run(context.Background(), &mcp.StdioTransport{})
}
