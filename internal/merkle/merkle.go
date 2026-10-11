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

// Package merkle provides file-tree hashing for change detection.
package merkle

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Tree holds the Merkle tree state for a project directory.
type Tree struct {
	FilesHashed int               // files read during this scan
	FilesReused int               // hashes reused after metadata validation
	RootHash    string            // SHA-256 of the root directory
	Files       map[string]string // relative path -> content SHA-256 hash
}

// SkipFunc returns true for paths that should be skipped during tree building.
type SkipFunc func(relPath string, isDir bool) bool

// DefaultSkip skips directories in SkipDirs and non-.go files.
func DefaultSkip(relPath string, isDir bool) bool {
	base := filepath.Base(relPath)
	if isDir {
		return SkipDirs[base]
	}
	return !strings.HasSuffix(base, ".go")
}

const merkleWorkers = 8

const maxFileSize = 10 * 1024 * 1024 // 10 MB

// BuildTree walks rootDir and computes a Merkle tree.
// File reads are parallelized across up to merkleWorkers goroutines.
// If skip is nil, DefaultSkip is used.
func BuildTree(rootDir string, skip SkipFunc) (*Tree, error) {
	if skip == nil {
		skip = DefaultSkip
	}

	relPaths, err := collectFilePaths(rootDir, skip, nil)
	if err != nil {
		return nil, err
	}

	tree, err := hashFilesInParallel(rootDir, relPaths)
	if err != nil {
		return nil, err
	}

	tree.FilesHashed = len(tree.Files)
	tree.RootHash = buildDirHash(tree.Files)
	return tree, nil
}

func collectFilePaths(rootDir string, skip SkipFunc, visitDir func(string)) ([]string, error) {
	var relPaths []string
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(rootDir, path)
		if rel == "." {
			if visitDir != nil {
				visitDir(path)
			}
			return nil
		}
		if d.IsDir() {
			if skip(rel, true) {
				return filepath.SkipDir
			}
			if visitDir != nil {
				visitDir(path)
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if info, err := d.Info(); err == nil && info.Size() > maxFileSize {
			return nil
		}
		if !skip(rel, false) {
			relPaths = append(relPaths, rel)
		}
		return nil
	})
	return relPaths, err
}

func hashFilesInParallel(rootDir string, relPaths []string) (*Tree, error) {
	if len(relPaths) == 0 {
		return &Tree{Files: map[string]string{}}, nil
	}

	type result struct {
		rel  string
		hash string
		err  error
	}

	work := make(chan string, len(relPaths))
	for _, p := range relPaths {
		work <- p
	}
	close(work)

	results := make(chan result, len(relPaths))
	workers := min(merkleWorkers, len(relPaths))

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for rel := range work {
				data, err := os.ReadFile(filepath.Join(rootDir, rel))
				if err != nil {
					if os.IsPermission(err) {
						continue
					}
					results <- result{err: err}
					return
				}
				hash := fmt.Sprintf("%x", sha256.Sum256(data))
				results <- result{rel: rel, hash: hash}
			}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	tree := &Tree{
		Files: make(map[string]string, len(relPaths)),
	}
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		tree.Files[r.rel] = r.hash
	}

	return tree, nil
}

func buildDirHash(files map[string]string) string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)

	h := sha256.New()
	for _, p := range paths {
		_, _ = fmt.Fprintf(h, "%s:%s\n", p, files[p])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Diff compares two trees and returns lists of added, removed, and modified file paths.
func Diff(old, cur *Tree) (added, removed, modified []string) {
	for path, curHash := range cur.Files {
		oldHash, exists := old.Files[path]
		if !exists {
			added = append(added, path)
		} else if oldHash != curHash {
			modified = append(modified, path)
		}
	}
	for path := range old.Files {
		if _, exists := cur.Files[path]; !exists {
			removed = append(removed, path)
		}
	}
	slices.Sort(added)
	slices.Sort(removed)
	slices.Sort(modified)
	return
}
