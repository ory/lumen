// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package store

import (
	"testing"
	"time"

	"github.com/ory/lumen/internal/chunker"
)

func TestRevisionCacheReuseAndBound(t *testing.T) {
	s, err := NewCollection(":memory:", 4, "int8", "/project")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	for _, version := range []string{"a", "b"} {
		c := chunker.Chunk{ID: version, FilePath: "a.go", Content: version}
		if _, err := s.StoreFileRevision("a.go", version, []chunker.Chunk{c}, map[int][]float32{0: {1, 0, 0, 0}}); err != nil {
			t.Fatal(err)
		}
	}
	attached, err := s.AttachExistingFileRevision("a.go", "a")
	if err != nil || !attached {
		t.Fatalf("switch-back reuse: %v, %v", attached, err)
	}
	if err := s.SetRevisionCachePolicy(1, time.Hour); err != nil {
		t.Fatal(err)
	}
	attached, err = s.AttachExistingFileRevision("a.go", "b")
	if err != nil || attached {
		t.Fatalf("over-budget revision retained: %v, %v", attached, err)
	}
	stats, err := s.CollectionStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.UniqueVectors != 1 || stats.ProjectFiles != 1 {
		t.Fatalf("eviction damaged active membership: %+v", stats)
	}
}

func TestRevisionCacheExpires(t *testing.T) {
	s, err := NewCollection(":memory:", 4, "float32", "/project")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if _, err := s.StoreFileRevision("a.go", "a", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFileChunks("a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE revision_cache SET unused_at=?`, time.Now().Add(-8*24*time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	if err := s.SetRevisionCachePolicy(256<<20, 7*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	attached, err := s.AttachExistingFileRevision("a.go", "a")
	if err != nil || attached {
		t.Fatalf("expired revision retained: %v %v", attached, err)
	}
}
