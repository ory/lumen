// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package merkle

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheReconcilesEditsAndRestarts(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.go")
	cachePath := filepath.Join(t.TempDir(), "scan.json")
	if err := os.WriteFile(path, []byte("package a"), 0600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	c := &Cache{closed: true}
	first, err := c.Build(root, nil, cachePath, false)
	if err != nil {
		t.Fatal(err)
	}
	// Reopen the persisted cache, then overwrite with equal-length content and
	// restore mtime. ctime/identity must prevent a false freshness result.
	c = &Cache{closed: true}
	if err := os.WriteFile(path, []byte("package b"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	second, err := c.Build(root, nil, cachePath, false)
	if err != nil {
		t.Fatal(err)
	}
	if first.RootHash == second.RootHash {
		t.Fatal("same-size, restored-mtime edit was missed")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package b"), 0600); err != nil {
		t.Fatal(err)
	}
	third, err := c.Build(root, nil, cachePath, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := third.Files["a.go"]; ok || len(third.Files) != 1 {
		t.Fatalf("stale membership: %v", third.Files)
	}
	full, err := BuildTree(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if full.RootHash != third.RootHash {
		t.Fatal("incremental and full trees differ")
	}
}

func TestCacheForceAndExpiryRehash(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.go")
	if err := os.WriteFile(path, []byte("package a"), 0600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	c := &Cache{closed: true}
	want, err := c.Build(root, nil, "", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, force := range []bool{true, false} {
		entry := c.Files["a.go"]
		entry.Hash = "stale"
		c.Files["a.go"] = entry
		if !force {
			c.Reconciled = time.Now().Add(-2 * time.Hour)
		}
		got, err := c.Build(root, nil, "", force)
		if err != nil {
			t.Fatal(err)
		}
		if got.RootHash != want.RootHash {
			t.Fatal("reconciliation reused stale hash")
		}
	}
}

func TestCacheReusesStableHashes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &Cache{closed: true}
	first, err := c.Build(root, nil, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if c.Files["a.go"].Identity == "" {
		t.Skip("platform conservatively hashes all files")
	}
	// Let both mtime and ctime leave the racy timestamp window.
	time.Sleep(2100 * time.Millisecond)
	second, err := c.Build(root, nil, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if first.RootHash != second.RootHash || second.FilesHashed != 0 || second.FilesReused != 1 {
		t.Fatalf("unchanged file was not reused: %+v", second)
	}
}

func TestCacheWatcherInvalidatesChangedPath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a.go")
	if err := os.WriteFile(path, []byte("package a"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &Cache{}
	defer func() { _ = c.Close() }()
	if _, err := c.Build(root, nil, "", false); err != nil {
		t.Fatal(err)
	}
	if c.watcher == nil {
		t.Skip("filesystem watches unavailable")
	}
	if err := os.WriteFile(path, []byte("package b"), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		_, present := c.Files["a.go"]
		c.mu.Unlock()
		if !present {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("watcher did not invalidate the changed path")
}
