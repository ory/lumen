// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/ory/lumen/internal/store"
)

func TestLegacyMigrationReusesUnchangedVectors(t *testing.T) {
	projectDir := t.TempDir()
	content := []byte("package demo\n\nfunc Hello() {}\n")
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	emb := &mockEmbedder{dims: 4, model: "test-model"}
	newPath := filepath.Join(t.TempDir(), "shared.db")
	idx, err := NewIndexerForProject(newPath, emb, 512, "int8", projectDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()

	chunks, err := idx.chunker.Chunk("main.go", content)
	if err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(t.TempDir(), "index.db")
	legacy, err := store.New(legacyPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if err := legacy.UpsertFile("main.go", hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	vectors := make([][]float32, len(chunks))
	for i := range vectors {
		vectors[i] = []float32{1, 0, 0, 0}
	}
	if err := legacy.InsertChunks(chunks, vectors); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	legacyBefore, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	legacyDigest := sha256.Sum256(legacyBefore)

	if err := idx.PrepareLegacyMigration(projectDir, legacyPath); err != nil {
		t.Fatal(err)
	}
	legacyAfter, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256.Sum256(legacyAfter); got != legacyDigest {
		t.Fatal("PrepareLegacyMigration modified the read-only legacy database")
	}
	stats, err := idx.Index(context.Background(), projectDir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.ChunksCreated == 0 {
		t.Fatal("expected migrated chunks")
	}
	if emb.callCount != 0 {
		t.Fatalf("legacy vectors should avoid embedding, got %d calls", emb.callCount)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy database should be removed after verification, stat err=%v", err)
	}
}

func TestLegacyMigrationReembedsChangedContentAndRemovesSource(t *testing.T) {
	projectDir := t.TempDir()
	oldContent := []byte("package demo\n\nfunc Before() {}\n")
	newContent := []byte("package demo\n\nfunc After() {}\n")
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), newContent, 0o644); err != nil {
		t.Fatal(err)
	}
	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexerForProject(filepath.Join(t.TempDir(), "shared.db"), emb, 512, "int8", projectDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()
	legacyPath := filepath.Join(t.TempDir(), "index.db")
	legacy, err := store.New(legacyPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	oldHash := sha256.Sum256(oldContent)
	if err := legacy.UpsertFile("main.go", hex.EncodeToString(oldHash[:])); err != nil {
		t.Fatal(err)
	}
	chunks, err := idx.chunker.Chunk("main.go", oldContent)
	if err != nil {
		t.Fatal(err)
	}
	vectors := make([][]float32, len(chunks))
	for i := range vectors {
		vectors[i] = []float32{1, 0, 0, 0}
	}
	if err := legacy.InsertChunks(chunks, vectors); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	if err := idx.PrepareLegacyMigration(projectDir, legacyPath); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	if emb.callCount == 0 {
		t.Fatal("changed content should be embedded")
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy database should be removed after successful reindex, stat err=%v", err)
	}
}
