// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package index

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/ory/lumen/internal/merkle"
	"github.com/ory/lumen/internal/store"
)

// indexSharedWithTree indexes one project membership in a repository-scoped
// content-addressed collection. Existing path+content revisions and exact
// embedding inputs are reused without calling the embedding backend.
func (idx *Indexer) indexSharedWithTree(ctx context.Context, projectDir, _ string, force bool, curTree *merkle.Tree, progress ProgressFunc) (Stats, error) {
	stats := Stats{TotalFiles: len(curTree.Files)}
	oldHashes, err := idx.store.GetFileHashes()
	if err != nil {
		return stats, fmt.Errorf("get project file hashes: %w", err)
	}
	for path := range oldHashes {
		if !supportedExts[filepath.Ext(path)] {
			if err := idx.store.DeleteFileChunks(path); err != nil {
				return stats, fmt.Errorf("purge stale file %s: %w", path, err)
			}
			delete(oldHashes, path)
		}
	}

	var filesToIndex, filesToRemove []string
	if force {
		for path := range curTree.Files {
			filesToIndex = append(filesToIndex, path)
		}
		for path := range oldHashes {
			if _, ok := curTree.Files[path]; !ok {
				filesToRemove = append(filesToRemove, path)
			}
		}
		stats.FilesAdded = len(filesToIndex)
		stats.FilesRemoved = len(filesToRemove)
	} else {
		added, removed, modified := merkle.Diff(&merkle.Tree{Files: oldHashes}, curTree)
		filesToIndex = append(filesToIndex, added...)
		filesToIndex = append(filesToIndex, modified...)
		filesToRemove = removed
		stats.FilesAdded = len(added)
		stats.FilesModified = len(modified)
		stats.FilesRemoved = len(removed)
	}
	slices.Sort(filesToIndex)
	slices.Sort(filesToRemove)
	stats.FilesChanged = len(filesToIndex) + len(filesToRemove)

	for _, path := range filesToRemove {
		if err := idx.store.DeleteFileChunks(path); err != nil {
			return stats, fmt.Errorf("remove project file %s: %w", path, err)
		}
	}

	if progress != nil {
		progress(0, len(filesToIndex), fmt.Sprintf("Found %d files to index", len(filesToIndex)))
	}

	for fileIndex, relativePath := range filesToIndex {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		if progress != nil {
			progress(fileIndex, len(filesToIndex), fmt.Sprintf("Processing file %d/%d: %s", fileIndex+1, len(filesToIndex), relativePath))
		}
		contentHash := curTree.Files[relativePath]
		if !force {
			attached, err := idx.store.AttachExistingFileRevision(relativePath, contentHash)
			if err != nil {
				return stats, fmt.Errorf("reuse file revision %s: %w", relativePath, err)
			}
			if attached {
				stats.IndexedFiles++
				continue
			}
		}

		content, err := os.ReadFile(filepath.Join(projectDir, relativePath))
		if err != nil {
			if os.IsPermission(err) {
				stats.FilesSkipped++
				continue
			}
			return stats, fmt.Errorf("read file %s: %w", relativePath, err)
		}
		if isBinaryContent(content) {
			if err := idx.store.DeleteFileChunks(relativePath); err != nil {
				return stats, fmt.Errorf("remove binary file %s: %w", relativePath, err)
			}
			continue
		}

		chunks, err := idx.chunker.Chunk(relativePath, content)
		if err != nil {
			if idx.logger != nil {
				idx.logger.Warn("skipping unchunkable file", "path", relativePath, "error", err)
			}
			stats.FilesSkipped++
			if _, err := idx.store.StoreFileRevision(relativePath, contentHash, nil, nil); err != nil {
				return stats, fmt.Errorf("record skipped file %s: %w", relativePath, err)
			}
			continue
		}
		chunks = splitOversizedChunks(chunks, idx.maxChunkTokens)
		chunks = mergeUndersizedChunks(chunks)
		chunks = splitOversizedChunks(chunks, idx.maxChunkTokens)

		var missing []int
		if force {
			missing = make([]int, len(chunks))
			for i := range chunks {
				missing[i] = i
			}
		} else {
			missing, err = idx.store.MissingChunkInputs(chunks)
			if err != nil {
				return stats, fmt.Errorf("check shared vectors for %s: %w", relativePath, err)
			}
		}
		vectors := make(map[int][]float32, len(missing))
		const embedBatchSize = 256
		var needsEmbedding []int
		for _, position := range missing {
			h := sha256.Sum256([]byte(store.EmbeddingInput(chunks[position])))
			if vector, ok := idx.legacyVectors[h]; ok {
				vectors[position] = vector
			} else {
				needsEmbedding = append(needsEmbedding, position)
			}
		}
		for start := 0; start < len(needsEmbedding); start += embedBatchSize {
			end := min(start+embedBatchSize, len(needsEmbedding))
			positions := needsEmbedding[start:end]
			texts := make([]string, len(positions))
			for i, position := range positions {
				texts[i] = store.EmbeddingInput(chunks[position])
			}
			embedded, err := idx.emb.Embed(ctx, texts)
			if err != nil {
				return stats, fmt.Errorf("embed %s: %w", relativePath, err)
			}
			if len(embedded) != len(positions) {
				return stats, fmt.Errorf("embed %s returned %d vectors for %d inputs", relativePath, len(embedded), len(positions))
			}
			for i, position := range positions {
				vectors[position] = embedded[i]
			}
			if progress != nil {
				progress(fileIndex+1, len(filesToIndex), fmt.Sprintf("Embedded %d chunks for %s", len(positions), relativePath))
			}
		}
		created, err := idx.store.StoreFileRevision(relativePath, contentHash, chunks, vectors)
		if err != nil {
			return stats, fmt.Errorf("store file revision %s: %w", relativePath, err)
		}
		if created || force {
			stats.ChunksCreated += len(chunks)
		}
		stats.IndexedFiles++
	}

	if len(filesToIndex) > 0 {
		idx.store.Analyze()
	}
	if err := idx.store.SetMeta("root_hash", curTree.RootHash); err != nil {
		return stats, err
	}
	for key, value := range map[string]string{
		"embedding_model": idx.emb.ModelName(),
		"project_path":    projectDir,
		"last_indexed_at": time.Now().UTC().Format(time.RFC3339),
		"total_files":     strconv.Itoa(stats.TotalFiles),
		"vector_storage":  idx.vectorStorage,
	} {
		if err := idx.store.SetMeta(key, value); err != nil {
			return stats, fmt.Errorf("store %s metadata: %w", key, err)
		}
	}
	if progress != nil && len(filesToIndex) > 0 {
		progress(len(filesToIndex), len(filesToIndex), fmt.Sprintf("Indexing complete: %d files, %d new chunks", len(filesToIndex), stats.ChunksCreated))
	}
	idx.finishLegacyMigration(curTree)
	return stats, nil
}
