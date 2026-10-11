// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package store

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ory/lumen/internal/chunker"
)

func TestCleanupCollectionAtRemovesOnlyOrphanScanCaches(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")
	active, stale, vanished := t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "gone")
	s, err := NewCollection(dbPath, 4, "int8", active)
	if err != nil {
		t.Fatal(err)
	}
	chunk := chunker.Chunk{ID: "a", FilePath: "a.go", Symbol: "A", Kind: "function", StartLine: 1, EndLine: 1, Content: "func A() {}"}
	if _, err := s.StoreFileRevision("a.go", "aa", []chunker.Chunk{chunk}, map[int][]float32{0: {1, 0, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	if err := s.UseProject(stale); err != nil {
		t.Fatal(err)
	}
	if attached, err := s.AttachExistingFileRevision("a.go", "aa"); err != nil || !attached {
		t.Fatalf("attach: %v %v", attached, err)
	}
	old := time.Now().Add(-60 * 24 * time.Hour).UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`UPDATE projects SET last_accessed_at = ? WHERE path = ?`, old, stale); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	scan := func(project string) string {
		return fmt.Sprintf("%s.scan-%x.json", dbPath, sha256.Sum256([]byte(project)))
	}
	keep := []string{
		scan(active),
		filepath.Join(dir, ".scan-123456"),                         // in-progress atomic write
		dbPath + ".lock",                                           // collection guard
		dbPath + ".project-0123456789abcdef.lock",                  // per-project guard
		dbPath + ".scan-NOTHEX.json",                               // unknown naming
		dbPath + ".scan-" + fmt.Sprintf("%x", [8]byte{}) + ".json", // wrong length
		filepath.Join(dir, "notes.txt"),
	}
	remove := []string{scan(stale), scan(vanished)}
	for _, path := range append(append([]string{}, keep...), remove...) {
		if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	stats, shared, err := CleanupCollectionAt(dbPath, time.Now().Add(-30*24*time.Hour))
	if err != nil || !shared {
		t.Fatalf("cleanup: shared=%v err=%v", shared, err)
	}
	if stats.ProjectsRemoved != 1 || stats.ProjectsLeft != 1 {
		t.Fatalf("unexpected cleanup stats: %+v", stats)
	}
	for _, path := range keep {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("cleanup removed %s: %v", filepath.Base(path), err)
		}
	}
	for _, path := range remove {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("orphan scan cache %s survived cleanup: %v", filepath.Base(path), err)
		}
	}
}
