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

package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSeedFromDonor_CopiesDB(t *testing.T) {
	// Create a real SQLite DB with indexed data.
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	donorPath := filepath.Join(t.TempDir(), "donor.db")
	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(donorPath, emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}

	// Seed to a new path.
	dstPath := filepath.Join(t.TempDir(), "sub", "seeded.db")
	seeded, err := SeedFromDonor(donorPath, dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if !seeded {
		t.Fatal("expected seeded=true")
	}

	// Verify the seeded DB works.
	idx2, err := NewIndexer(dstPath, emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx2.Close() }()

	status, err := idx2.Status(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if status.IndexedFiles == 0 {
		t.Fatal("expected seeded DB to have indexed files")
	}
}

func TestSeedFromDonor_DstExists(t *testing.T) {
	donorPath := filepath.Join(t.TempDir(), "donor.db")
	if err := os.WriteFile(donorPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	dstPath := filepath.Join(t.TempDir(), "existing.db")
	if err := os.WriteFile(dstPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	seeded, err := SeedFromDonor(donorPath, dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if seeded {
		t.Fatal("expected seeded=false when dst exists")
	}

	// Verify original content preserved.
	content, _ := os.ReadFile(dstPath)
	if string(content) != "existing" {
		t.Fatalf("expected dst unchanged, got %q", content)
	}
}

func TestSeedFromDonor_IncrementalUpdate(t *testing.T) {
	// Create donor with one file.
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", `package main

func Hello() {}
`)

	donorPath := filepath.Join(t.TempDir(), "donor.db")
	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(donorPath, emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Index(context.Background(), projectDir, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}
	callsAfterDonor := emb.callCount

	// Seed to new path.
	dstPath := filepath.Join(t.TempDir(), "seeded.db")
	if _, err := SeedFromDonor(donorPath, dstPath); err != nil {
		t.Fatal(err)
	}

	// Open seeded DB and add a new file to the project.
	writeGoFile(t, projectDir, "extra.go", `package main

func Extra() {}
`)

	idx2, err := NewIndexer(dstPath, emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx2.Close() }()

	// EnsureFresh should do an incremental update (not full re-index).
	reindexed, stats, err := idx2.EnsureFresh(context.Background(), projectDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reindexed {
		t.Fatal("expected reindexed=true after adding a file")
	}
	// Only the new file should be indexed, not the original.
	if stats.IndexedFiles != 1 {
		t.Fatalf("expected 1 file indexed incrementally, got %d", stats.IndexedFiles)
	}
	if emb.callCount == callsAfterDonor {
		t.Fatal("expected embed calls for the new file")
	}
}
