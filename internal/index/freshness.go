// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package index

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/ory/lumen/internal/merkle"
)

func (idx *Indexer) buildTree(projectDir string, force bool) (*merkle.Tree, error) {
	root, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}
	value, _ := idx.scanCaches.LoadOrStore(root, &merkle.Cache{})
	cachePath := ""
	if idx.dsn != "" && idx.dsn != ":memory:" {
		cachePath = fmt.Sprintf("%s.scan-%x.json", idx.dsn, sha256.Sum256([]byte(root)))
	}
	return value.(*merkle.Cache).Build(root, makeSkip(root), cachePath, force)
}
