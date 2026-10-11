// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package index

import (
	"path/filepath"

	"github.com/ory/lumen/internal/merkle"
	"github.com/ory/lumen/internal/store"
)

func (idx *Indexer) buildTree(projectDir string, force bool) (*merkle.Tree, error) {
	root, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}
	value, _ := idx.scanCaches.LoadOrStore(root, &merkle.Cache{})
	cachePath := ""
	if idx.dsn != "" && idx.dsn != ":memory:" {
		cachePath = store.ScanCachePath(idx.dsn, root)
	}
	tree, err := value.(*merkle.Cache).Build(root, makeSkip(root), cachePath, force)
	if err == nil && idx.logger != nil {
		idx.logger.Info("scan completed",
			"project", root,
			"forced", force,
			"files_total", len(tree.Files),
			"files_hashed", tree.FilesHashed,
			"files_reused", tree.FilesReused,
			"root_hash", tree.RootHash,
		)
	}
	return tree, err
}

// logIndexPass records the work one indexing pass actually performed. It is
// emitted for failed passes too, so interrupted work remains observable.
func (idx *Indexer) logIndexPass(projectDir string, force bool, rootHash string, stats Stats, err error) {
	if idx.logger == nil {
		return
	}
	args := []any{
		"project", projectDir,
		"forced", force,
		"files_total", stats.TotalFiles,
		"files_changed", stats.FilesChanged,
		"files_removed", stats.FilesRemoved,
		"files_parsed", stats.FilesParsed,
		"revisions_attached", stats.RevisionsAttached,
		"chunks_created", stats.ChunksCreated,
		"root_hash", rootHash,
	}
	if err != nil {
		idx.logger.Warn("index pass failed", append(args, "error", err.Error())...)
		return
	}
	idx.logger.Info("index pass completed", args...)
}
