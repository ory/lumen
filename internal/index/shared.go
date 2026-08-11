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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/ory/lumen/internal/chunker"
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

	type pendingSharedFile struct {
		relativePath string
		contentHash  string
		chunks       []chunker.Chunk
		vectors      map[int][]float32
		remaining    int
		fileIndex    int
		skipped      bool
	}
	type chunkRef struct {
		file     *pendingSharedFile
		position int
	}

	embedBatchSize := idx.embedBatchSize
	if embedBatchSize <= 0 {
		embedBatchSize = 256
	}
	var pending []*pendingSharedFile
	var batchTexts []string
	var batchRefs []chunkRef

	embedAllFileChunks := func(file *pendingSharedFile) (map[int][]float32, error) {
		vectors := make(map[int][]float32, len(file.chunks))
		for start := 0; start < len(file.chunks); start += embedBatchSize {
			end := min(start+embedBatchSize, len(file.chunks))
			texts := make([]string, end-start)
			for i := start; i < end; i++ {
				texts[i-start] = store.EmbeddingInput(file.chunks[i])
			}
			embedded, err := idx.emb.Embed(ctx, texts)
			if err != nil {
				return nil, fmt.Errorf("re-embed %s: %w", file.relativePath, err)
			}
			if len(embedded) != len(texts) {
				return nil, fmt.Errorf("re-embed %s returned %d vectors for %d inputs", file.relativePath, len(embedded), len(texts))
			}
			for i, vector := range embedded {
				vectors[start+i] = vector
			}
		}
		return vectors, nil
	}

	storeFile := func(file *pendingSharedFile) (bool, error) {
		created, err := idx.store.StoreFileRevision(file.relativePath, file.contentHash, file.chunks, file.vectors)
		if !errors.Is(err, store.ErrVectorVanished) {
			return created, err
		}
		// A concurrent cleanup can remove a vector after MissingChunkInputs
		// observes it. Re-embedding every position makes the retry independent
		// of all shared vector rows and closes that TOCTOU window.
		vectors, embedErr := embedAllFileChunks(file)
		if embedErr != nil {
			return false, embedErr
		}
		created, err = idx.store.StoreFileRevision(file.relativePath, file.contentHash, file.chunks, vectors)
		return created, err
	}

	drainReady := func() error {
		for len(pending) > 0 && pending[0].remaining == 0 {
			file := pending[0]
			created, err := storeFile(file)
			if err != nil {
				return fmt.Errorf("store file revision %s: %w", file.relativePath, err)
			}
			if created || force {
				stats.ChunksCreated += len(file.chunks)
			}
			if !file.skipped {
				stats.IndexedFiles++
			}
			pending = pending[1:]
		}
		return nil
	}

	flushBatch := func() error {
		if len(batchTexts) == 0 {
			return drainReady()
		}
		embedded, err := idx.emb.Embed(ctx, batchTexts)
		if err != nil {
			return fmt.Errorf("embed shared batch: %w", err)
		}
		if len(embedded) != len(batchRefs) {
			return fmt.Errorf("embed shared batch returned %d vectors for %d inputs", len(embedded), len(batchRefs))
		}
		for i, ref := range batchRefs {
			ref.file.vectors[ref.position] = embedded[i]
			ref.file.remaining--
		}
		if progress != nil {
			last := batchRefs[len(batchRefs)-1].file
			progress(last.fileIndex+1, len(filesToIndex), fmt.Sprintf("Embedded %d shared chunks", len(batchRefs)))
		}
		batchTexts = batchTexts[:0]
		batchRefs = batchRefs[:0]
		return drainReady()
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
			pending = append(pending, &pendingSharedFile{
				relativePath: relativePath,
				contentHash:  contentHash,
				vectors:      map[int][]float32{},
				fileIndex:    fileIndex,
				skipped:      true,
			})
			if err := drainReady(); err != nil {
				return stats, err
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
		file := &pendingSharedFile{
			relativePath: relativePath,
			contentHash:  contentHash,
			chunks:       chunks,
			vectors:      make(map[int][]float32, len(missing)),
			fileIndex:    fileIndex,
		}
		pending = append(pending, file)
		var needsEmbedding []int
		for _, position := range missing {
			h := sha256.Sum256([]byte(store.EmbeddingInput(chunks[position])))
			if vector, ok := idx.legacyVectors[h]; ok {
				file.vectors[position] = vector
			} else {
				needsEmbedding = append(needsEmbedding, position)
			}
		}
		file.remaining = len(needsEmbedding)
		for _, position := range needsEmbedding {
			batchTexts = append(batchTexts, store.EmbeddingInput(chunks[position]))
			batchRefs = append(batchRefs, chunkRef{file: file, position: position})
			if len(batchTexts) == embedBatchSize {
				if err := flushBatch(); err != nil {
					return stats, err
				}
			}
		}
		if err := drainReady(); err != nil {
			return stats, err
		}
	}
	if err := flushBatch(); err != nil {
		return stats, err
	}
	if len(pending) != 0 {
		return stats, fmt.Errorf("shared embedding queue left %d files pending", len(pending))
	}

	if len(filesToIndex) > 0 {
		idx.store.Analyze()
	}
	metadata := []struct{ key, value string }{
		{"embedding_model", idx.emb.ModelName()},
		{"project_path", projectDir},
		{"last_indexed_at", time.Now().UTC().Format(time.RFC3339)},
		{"total_files", strconv.Itoa(stats.TotalFiles)},
		{"vector_storage", idx.vectorStorage},
	}
	for _, item := range metadata {
		if err := idx.store.SetMeta(item.key, item.value); err != nil {
			return stats, fmt.Errorf("store %s metadata: %w", item.key, err)
		}
	}
	// root_hash is the commit marker and must be written after every other
	// durable file, vector, and metadata update.
	if err := idx.store.SetMeta("root_hash", curTree.RootHash); err != nil {
		return stats, fmt.Errorf("store root_hash metadata: %w", err)
	}
	if progress != nil && len(filesToIndex) > 0 {
		progress(len(filesToIndex), len(filesToIndex), fmt.Sprintf("Indexing complete: %d files, %d new chunks", len(filesToIndex), stats.ChunksCreated))
	}
	idx.finishLegacyMigration(curTree)
	return stats, nil
}
