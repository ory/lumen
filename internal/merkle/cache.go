// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package merkle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Cache retains hashes only while the file's identity and metadata agree.
// It still walks directory entries, so additions, removals, and ignore-rule
// changes are discovered without relying on event delivery.
type Cache struct {
	watcher    *fsnotify.Watcher
	watched    map[string]bool
	closed     bool
	mu         sync.Mutex
	Files      map[string]cachedFile `json:"files"`
	Reconciled time.Time             `json:"reconciled"`
}

type cachedFile struct {
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
	Mode     uint32 `json:"mode"`
	Identity string `json:"identity"`
	Changed  int64  `json:"changed"`
}

// Build reconciles metadata on every call and content at least hourly. force
// bypasses all cached hashes. Cache files are disposable and atomically replaced.
func (c *Cache) Build(root string, skip SkipFunc, cachePath string, force bool) (*Tree, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Files == nil && cachePath != "" {
		if data, err := os.ReadFile(cachePath); err == nil {
			var saved struct {
				Files      map[string]cachedFile `json:"files"`
				Reconciled time.Time             `json:"reconciled"`
			}
			if json.Unmarshal(data, &saved) == nil {
				c.Files, c.Reconciled = saved.Files, saved.Reconciled
			}
		}
	}
	now := time.Now()
	reconcile := force || c.Reconciled.IsZero() || now.Sub(c.Reconciled) >= time.Hour || now.Before(c.Reconciled)
	if skip == nil {
		skip = DefaultSkip
	}
	c.watchLocked(root)
	paths, err := collectFilePaths(root, skip, c.watchDirectoryLocked)
	if err != nil {
		return nil, err
	}
	next := make(map[string]cachedFile, len(paths))
	tree := &Tree{Files: make(map[string]string, len(paths))}
	for _, rel := range paths {
		path := filepath.Join(root, rel)
		before, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !before.Mode().IsRegular() || before.Size() > maxFileSize {
			continue
		}
		stamp := stampFile(before)
		old, ok := c.Files[rel]
		// Recently modified files are deliberately rehashed: coarse timestamps
		// can hide a second write in the same timestamp tick.
		reusable := ok && len(old.Hash) == 64 && !reconcile && stamp.Identity != "" && stamp.Changed < now.Add(-2*time.Second).UnixNano() && before.ModTime().Before(now.Add(-2*time.Second)) && old.Size == stamp.Size && old.Modified == stamp.Modified && old.Mode == stamp.Mode && old.Identity == stamp.Identity
		if reusable {
			stamp.Hash = old.Hash
			tree.FilesReused++
		} else {
			data, err := os.ReadFile(path)
			if os.IsPermission(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
			after, err := os.Lstat(path)
			if err != nil {
				return nil, err
			}
			if stampFile(after) != stamp {
				return nil, fmt.Errorf("file changed while hashing: %s", rel)
			}
			stamp.Hash = fmt.Sprintf("%x", sha256.Sum256(data))
			tree.FilesHashed++
		}
		next[rel] = stamp
		tree.Files[rel] = stamp.Hash
	}
	c.Files = next
	if reconcile {
		c.Reconciled = now
	}
	tree.RootHash = buildDirHash(tree.Files)
	if cachePath != "" {
		// A failed cache write affects performance only, never index validity.
		if data, err := json.Marshal(c); err == nil {
			if f, err := os.CreateTemp(filepath.Dir(cachePath), ".scan-*"); err == nil {
				name := f.Name()
				_, writeErr := f.Write(data)
				closeErr := f.Close()
				if writeErr == nil && closeErr == nil {
					_ = os.Rename(name, cachePath)
				}
				_ = os.Remove(name)
			}
		}
	}
	return tree, nil
}

func stampFile(info os.FileInfo) cachedFile {
	identity, changed := fileIdentity(info)
	return cachedFile{Size: info.Size(), Modified: info.ModTime().UnixNano(), Mode: uint32(info.Mode()), Identity: identity, Changed: changed}
}
