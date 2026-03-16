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

// Package index orchestrates chunking, embedding, and storage for code indexes.
package index

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ory/lumen/internal/chunker"
	"github.com/ory/lumen/internal/embedder"
	"github.com/ory/lumen/internal/merkle"
	"github.com/ory/lumen/internal/store"
	"github.com/ory/lumen/internal/summarizer"
)

// ProgressFunc is an optional callback for reporting indexing progress.
// current is the number of items processed so far, total is the total number
// of items (0 if unknown), and message describes the current step.
type ProgressFunc func(current, total int, message string)

// Stats holds statistics from an indexing run.
type Stats struct {
	TotalFiles    int
	IndexedFiles  int
	ChunksCreated int
	FilesChanged  int
}

// StatusInfo holds information about the current index state for a project.
type StatusInfo struct {
	ProjectPath    string
	TotalFiles     int
	IndexedFiles   int
	TotalChunks    int
	LastIndexedAt  string
	EmbeddingModel string
}

// Indexer orchestrates chunking, embedding, and storage for a code index.
type Indexer struct {
	mu             sync.Mutex
	store          *store.Store
	emb            embedder.Embedder
	summaryEmb     embedder.Embedder     // nil when summaries disabled
	sumr           summarizer.Summarizer // nil when summaries disabled
	chunker        chunker.Chunker
	maxChunkTokens int
}

// NewIndexer creates a new Indexer backed by a SQLite store at dsn,
// using the given embedder for vector generation. maxChunkTokens controls
// the maximum estimated token count per chunk before splitting; 0 disables splitting.
// summaryDims is the dimensionality of the summary embedding vectors (0 = no summary vec tables).
// sumr and summaryEmb are optional; pass nil to disable summary generation.
func NewIndexer(dsn string, emb embedder.Embedder, maxChunkTokens int, summaryDims int, sumr summarizer.Summarizer, summaryEmb embedder.Embedder) (*Indexer, error) {
	s, err := store.New(dsn, emb.Dimensions(), summaryDims)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}
	return &Indexer{
		store:          s,
		emb:            emb,
		summaryEmb:     summaryEmb,
		sumr:           sumr,
		chunker:        chunker.NewMultiChunker(chunker.DefaultLanguages(maxChunkTokens)),
		maxChunkTokens: maxChunkTokens,
	}, nil
}

// Close closes the underlying store.
func (idx *Indexer) Close() error {
	return idx.store.Close()
}

// Index indexes the project at projectDir. If force is true, all files are
// re-indexed regardless of whether they have changed.
func (idx *Indexer) Index(ctx context.Context, projectDir string, force bool, progress ProgressFunc) (Stats, error) {
	// Build tree outside the lock: it is read-only and can be slow for large projects.
	curTree, err := merkle.BuildTree(projectDir, merkle.MakeSkip(projectDir, chunker.SupportedExtensions()))
	if err != nil {
		return Stats{}, fmt.Errorf("build merkle tree: %w", err)
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// If not forcing, check root hash before doing any work.
	if !force {
		storedHash, err := idx.store.GetMeta("root_hash")
		if err != nil && err != sql.ErrNoRows {
			return Stats{}, fmt.Errorf("get root_hash: %w", err)
		}
		if storedHash == curTree.RootHash {
			return Stats{}, nil
		}
	}
	return idx.indexWithTree(ctx, projectDir, force, curTree, progress)
}

// EnsureFresh checks if the index is stale and re-indexes if needed.
// Returns whether a re-index occurred, the stats, and any error.
func (idx *Indexer) EnsureFresh(ctx context.Context, projectDir string, progress ProgressFunc) (bool, Stats, error) {
	// Build tree outside the lock: it is read-only and can be slow for large projects.
	curTree, err := merkle.BuildTree(projectDir, merkle.MakeSkip(projectDir, chunker.SupportedExtensions()))
	if err != nil {
		return false, Stats{}, fmt.Errorf("build merkle tree: %w", err)
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	storedHash, err := idx.store.GetMeta("root_hash")
	if err != nil && err != sql.ErrNoRows {
		return false, Stats{}, fmt.Errorf("get root_hash: %w", err)
	}
	if storedHash == curTree.RootHash {
		return false, Stats{}, nil
	}

	stats, err := idx.indexWithTree(ctx, projectDir, false, curTree, progress)
	if err != nil {
		return false, stats, err
	}
	return true, stats, nil
}

// indexWithTree is the internal implementation of Index that accepts a pre-built
// merkle tree, so callers that already have one (e.g. EnsureFresh) do not need
// to build it again.
func (idx *Indexer) indexWithTree(ctx context.Context, projectDir string, force bool, curTree *merkle.Tree, progress ProgressFunc) (Stats, error) {
	var stats Stats

	stats.TotalFiles = len(curTree.Files)

	// Determine which files need processing.
	var filesToIndex []string
	var filesToRemove []string

	if force {
		for path := range curTree.Files {
			filesToIndex = append(filesToIndex, path)
		}
	} else {
		oldHashes, err := idx.store.GetFileHashes()
		if err != nil {
			return stats, fmt.Errorf("get file hashes: %w", err)
		}
		oldTree := &merkle.Tree{Files: oldHashes}
		added, removed, modified := merkle.Diff(oldTree, curTree)
		filesToIndex = append(filesToIndex, added...)
		filesToIndex = append(filesToIndex, modified...)
		filesToRemove = removed
	}

	stats.FilesChanged = len(filesToIndex) + len(filesToRemove)

	if progress != nil {
		progress(0, len(filesToIndex), fmt.Sprintf("Found %d files to index", len(filesToIndex)))
	}

	for _, path := range filesToRemove {
		if err := idx.store.DeleteFileChunks(path); err != nil {
			return stats, fmt.Errorf("delete chunks for %s: %w", path, err)
		}
	}

	const chunkBatchSize = 256
	var batch []chunker.Chunk
	var totalChunks int

	flushBatch := func(fileIdx int) error {
		if len(batch) == 0 {
			return nil
		}
		texts := make([]string, len(batch))
		for i, c := range batch {
			texts[i] = "// " + c.FilePath + "\n" + c.Content
		}
		vectors, err := idx.emb.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("embed batch: %w", err)
		}
		if err := idx.store.InsertChunks(batch, vectors); err != nil {
			return fmt.Errorf("insert batch: %w", err)
		}
		totalChunks += len(batch)
		batch = batch[:0]
		if progress != nil {
			progress(fileIdx, len(filesToIndex), fmt.Sprintf("Embedded %d chunks so far", totalChunks))
		}
		return nil
	}

	for fileIdx, relPath := range filesToIndex {
		if progress != nil {
			progress(fileIdx, len(filesToIndex), fmt.Sprintf("Processing file %d/%d: %s", fileIdx+1, len(filesToIndex), relPath))
		}

		absPath := filepath.Join(projectDir, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return stats, fmt.Errorf("read file %s: %w", relPath, err)
		}

		if err := idx.store.DeleteFileChunks(relPath); err != nil {
			return stats, fmt.Errorf("delete old chunks for %s: %w", relPath, err)
		}

		if err := idx.store.UpsertFile(relPath, curTree.Files[relPath]); err != nil {
			return stats, fmt.Errorf("upsert file %s: %w", relPath, err)
		}

		chunks, err := idx.chunker.Chunk(relPath, content)
		if err != nil {
			return stats, fmt.Errorf("chunk %s: %w", relPath, err)
		}

		chunks = splitOversizedChunks(chunks, idx.maxChunkTokens)
		chunks = mergeUndersizedChunks(chunks, minMergeTokens)

		batch = append(batch, chunks...)

		if len(batch) >= chunkBatchSize {
			if err := flushBatch(fileIdx + 1); err != nil {
				return stats, err
			}
		}
	}

	if err := flushBatch(len(filesToIndex)); err != nil {
		return stats, err
	}

	// Run summary passes when summarizer is configured.
	if idx.sumr != nil && idx.summaryEmb != nil && len(filesToIndex) > 0 {
		if err := idx.runSummaryPasses(ctx, projectDir, filesToIndex, progress); err != nil {
			log.Printf("warning: summary passes failed: %v", err)
		}
	}

	if progress != nil && len(filesToIndex) > 0 {
		progress(len(filesToIndex), len(filesToIndex),
			fmt.Sprintf("Indexing complete: %d files, %d chunks", len(filesToIndex), totalChunks))
	}

	stats.IndexedFiles = len(filesToIndex)
	stats.ChunksCreated = totalChunks

	if err := idx.store.SetMeta("root_hash", curTree.RootHash); err != nil {
		return stats, fmt.Errorf("set root_hash: %w", err)
	}
	if err := idx.store.SetMeta("embedding_model", idx.emb.ModelName()); err != nil {
		return stats, fmt.Errorf("set embedding_model: %w", err)
	}
	if err := idx.store.SetMeta("last_indexed_at", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return stats, fmt.Errorf("set last_indexed_at: %w", err)
	}
	if err := idx.store.SetMeta("total_files", strconv.Itoa(stats.TotalFiles)); err != nil {
		return stats, fmt.Errorf("set total_files: %w", err)
	}

	return stats, nil
}

// IsFresh checks whether the index for projectDir is up to date by comparing
// the current Merkle tree root hash against the stored one. Returns false if
// the project has never been indexed (no stored hash).
func (idx *Indexer) IsFresh(projectDir string) (bool, error) {
	curTree, err := merkle.BuildTree(projectDir, merkle.MakeSkip(projectDir, chunker.SupportedExtensions()))
	if err != nil {
		return false, fmt.Errorf("build merkle tree: %w", err)
	}

	storedHash, err := idx.store.GetMeta("root_hash")
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("get root_hash: %w", err)
	}
	if storedHash == "" {
		return false, nil
	}
	return storedHash == curTree.RootHash, nil
}

// Search performs a vector similarity search against the index.
// If maxDistance > 0, results with distance >= maxDistance are excluded.
// If pathPrefix != "", only chunks under that relative path are returned.
func (idx *Indexer) Search(_ context.Context, _ string, queryVec []float32, limit int, maxDistance float64, pathPrefix string) ([]store.SearchResult, error) {
	return idx.store.Search(queryVec, limit, maxDistance, pathPrefix)
}

// SearchChunkSummaries searches the chunk summary vector index.
func (idx *Indexer) SearchChunkSummaries(queryVec []float32, limit int, maxDistance float64, pathPrefix string) ([]store.SearchResult, error) {
	return idx.store.SearchChunkSummaries(queryVec, limit, maxDistance, pathPrefix)
}

// SearchFileSummaries searches the file summary vector index.
func (idx *Indexer) SearchFileSummaries(queryVec []float32, limit int, maxDistance float64) ([]store.FileSummaryResult, error) {
	return idx.store.SearchFileSummaries(queryVec, limit, maxDistance)
}

// TopChunksByFile returns the top n chunks from a file by raw-code distance.
func (idx *Indexer) TopChunksByFile(filePath string, queryVec []float32, maxDistance float64, n int) ([]store.SearchResult, error) {
	return idx.store.TopChunksByFile(filePath, queryVec, maxDistance, n)
}

// Status returns information about the current index state for a project.
// All values are read from the database; no filesystem walk is performed.
func (idx *Indexer) Status(projectDir string) (StatusInfo, error) {
	var info StatusInfo
	info.ProjectPath = projectDir

	storeStats, err := idx.store.Stats()
	if err != nil {
		return info, fmt.Errorf("get store stats: %w", err)
	}
	info.IndexedFiles = storeStats.TotalFiles
	info.TotalChunks = storeStats.TotalChunks

	meta, err := idx.store.GetMetaBatch([]string{"embedding_model", "last_indexed_at", "total_files"})
	if err != nil {
		return info, fmt.Errorf("get meta batch: %w", err)
	}
	info.EmbeddingModel = meta["embedding_model"]
	info.LastIndexedAt = meta["last_indexed_at"]
	if n, err := strconv.Atoi(meta["total_files"]); err == nil {
		info.TotalFiles = n
	}

	return info, nil
}

// runSummaryPasses generates per-chunk and per-file summaries for the given
// files, embeds them via summaryEmb, and stores them in the summary tables.
func (idx *Indexer) runSummaryPasses(ctx context.Context, projectDir string, files []string, progress ProgressFunc) error {
	const summaryEmbedBatchSize = 32

	if progress != nil {
		progress(0, len(files), fmt.Sprintf("Generating summaries for %d files", len(files)))
	}

	fileSummaryInputs := make(map[string][]string)
	var pendingChunkIDs []string
	var pendingSummaries []string

	flushChunkSummaries := func() error {
		if len(pendingChunkIDs) == 0 {
			return nil
		}
		vecs, err := idx.summaryEmb.Embed(ctx, pendingSummaries)
		if err != nil {
			return fmt.Errorf("embed chunk summaries: %w", err)
		}
		if err := idx.store.InsertChunkSummaries(pendingChunkIDs, pendingSummaries, vecs); err != nil {
			return fmt.Errorf("store chunk summaries: %w", err)
		}
		pendingChunkIDs = pendingChunkIDs[:0]
		pendingSummaries = pendingSummaries[:0]
		return nil
	}

	for fileIdx, relPath := range files {
		if progress != nil {
			progress(fileIdx, len(files), fmt.Sprintf("Summarizing chunks %d/%d: %s", fileIdx+1, len(files), relPath))
		}

		absPath := filepath.Join(projectDir, relPath)
		fileContent, err := os.ReadFile(absPath)
		if err != nil {
			log.Printf("warning: read file %s for summarization: %v", relPath, err)
			continue
		}
		chunks, err := idx.store.ChunksByFile(relPath)
		if err != nil {
			log.Printf("warning: fetch chunks for %s: %v", relPath, err)
			continue
		}
		lines := strings.Split(string(fileContent), "\n")
		for _, c := range chunks {
			if c.EndLine-c.StartLine < 2 {
				continue
			}
			start := max(c.StartLine-1, 0)
			end := min(c.EndLine, len(lines))
			content := strings.Join(lines[start:end], "\n")

			summary, err := idx.sumr.SummarizeChunk(ctx, summarizer.ChunkInfo{
				Kind:    c.Kind,
				Symbol:  c.Symbol,
				Content: content,
			})
			if err != nil {
				log.Printf("warning: summarize chunk %s: %v", c.ID, err)
				continue
			}
			pendingChunkIDs = append(pendingChunkIDs, c.ID)
			pendingSummaries = append(pendingSummaries, summary)
			fileSummaryInputs[relPath] = append(fileSummaryInputs[relPath], summary)

			if len(pendingChunkIDs) >= summaryEmbedBatchSize {
				if err := flushChunkSummaries(); err != nil {
					log.Printf("warning: flush chunk summaries: %v", err)
				}
			}
		}
	}
	if err := flushChunkSummaries(); err != nil {
		log.Printf("warning: final flush chunk summaries: %v", err)
	}

	if progress != nil {
		progress(len(files), len(files), fmt.Sprintf("Generating file-level summaries for %d files", len(fileSummaryInputs)))
	}

	fileIdx := 0
	for _, relPath := range files {
		chunkSummaries := fileSummaryInputs[relPath]
		if len(chunkSummaries) == 0 {
			continue
		}
		fileIdx++
		if progress != nil {
			progress(fileIdx, len(fileSummaryInputs), fmt.Sprintf("Summarizing file %d/%d: %s", fileIdx, len(fileSummaryInputs), relPath))
		}
		fileSummary, err := idx.sumr.SummarizeFile(ctx, chunkSummaries)
		if err != nil {
			log.Printf("warning: summarize file %s: %v", relPath, err)
			continue
		}
		vecs, err := idx.summaryEmb.Embed(ctx, []string{fileSummary})
		if err != nil {
			log.Printf("warning: embed file summary %s: %v", relPath, err)
			continue
		}
		if err := idx.store.InsertFileSummary(relPath, fileSummary, vecs[0]); err != nil {
			log.Printf("warning: store file summary %s: %v", relPath, err)
		}
	}
	return nil
}
