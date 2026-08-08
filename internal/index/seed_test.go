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
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ory/lumen/internal/indexlock"
	"github.com/ory/lumen/internal/store"
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
	seedProjectDir := t.TempDir()
	seeded, err := SeedFromDonor(donorPath, dstPath, seedProjectDir)
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
	seedMeta, err := store.ReadMetaAt(dstPath, "project_path")
	if err != nil {
		t.Fatal(err)
	}
	if seedMeta["project_path"] != seedProjectDir {
		t.Fatalf("seeded project_path = %q, want %q", seedMeta["project_path"], seedProjectDir)
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

	seeded, err := SeedFromDonor(donorPath, dstPath, t.TempDir())
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

func TestSeedFromDonor_MissingDonorIsNotCreated(t *testing.T) {
	donorPath := filepath.Join(t.TempDir(), "missing.db")
	dstPath := filepath.Join(t.TempDir(), "seeded.db")

	seeded, err := SeedFromDonor(donorPath, dstPath, t.TempDir())
	if err == nil {
		t.Fatal("expected missing donor to return an error")
	}
	if seeded {
		t.Fatal("expected seeded=false for missing donor")
	}
	if _, statErr := os.Stat(donorPath); !os.IsNotExist(statErr) {
		t.Fatalf("missing donor was created: stat error = %v", statErr)
	}
	if _, statErr := os.Stat(dstPath); !os.IsNotExist(statErr) {
		t.Fatalf("destination was created: stat error = %v", statErr)
	}
}

func TestSeedFromDonor_ConcurrentSeedersExactlyOneWins(t *testing.T) {
	// Build a real, complete donor DB.
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

	// Many goroutines race to seed the same destination.
	dstPath := filepath.Join(t.TempDir(), "sub", "seeded.db")
	const n = 8
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		wins     int
		firstErr error
	)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			seeded, err := SeedFromDonor(donorPath, dstPath, projectDir)
			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if seeded {
				wins++
			}
		}()
	}
	wg.Wait()

	if firstErr != nil {
		t.Fatalf("concurrent SeedFromDonor returned error: %v", firstErr)
	}
	if wins != 1 {
		t.Fatalf("expected exactly one seeder to win, got %d", wins)
	}

	// No seed temp files should be left behind. The advisory lock file itself
	// may remain on disk, just like the regular index lock file.
	entries, err := os.ReadDir(filepath.Dir(dstPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == filepath.Base(dstPath)+".seed-tmp" {
			t.Fatalf("unexpected leftover file after concurrent seeding: %q", e.Name())
		}
	}

	// The published DB must be usable.
	idx2, err := NewIndexer(dstPath, emb, 0)
	if err != nil {
		t.Fatalf("seeded DB is not openable: %v", err)
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

func TestSeedFromDonor_WaitsBeforeCopying(t *testing.T) {
	projectDir := t.TempDir()
	writeGoFile(t, projectDir, "main.go", "package main\n\nfunc Hello() {}\n")

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

	dstPath := filepath.Join(t.TempDir(), "seeded.db")
	held, err := indexlock.Acquire(context.Background(), dstPath+".seed.lock")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := SeedFromDonor(donorPath, dstPath, projectDir)
		done <- err
	}()

	select {
	case err := <-done:
		t.Fatalf("seeder returned before lock was released: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := os.Stat(dstPath + ".seed-tmp"); !os.IsNotExist(err) {
		t.Fatalf("seeder copied before acquiring the seed lock: stat error = %v", err)
	}

	held.Release()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("seeder did not finish after lock release")
	}
}

func TestPublishSeed_FallsBackToRename(t *testing.T) {
	dir := t.TempDir()
	tmp := filepath.Join(dir, "index.db.seed-tmp")
	dst := filepath.Join(dir, "index.db")
	if err := os.WriteFile(tmp, []byte("seed"), 0o600); err != nil {
		t.Fatal(err)
	}

	seeded, err := publishSeed(
		tmp,
		dst,
		func(_, _ string) error { return errors.New("hard links unsupported") },
		os.Rename,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !seeded {
		t.Fatal("expected rename fallback to publish the seed")
	}
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "seed" {
		t.Fatalf("published content = %q, want seed", content)
	}
}

func TestSeedFromDonor_IncompleteDonor(t *testing.T) {
	// Create a donor DB that has the schema but no root_hash (simulates
	// a donor whose first indexing pass hasn't finished yet).
	donorPath := filepath.Join(t.TempDir(), "donor.db")
	emb := &mockEmbedder{dims: 4, model: "test-model"}
	idx, err := NewIndexer(donorPath, emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Close without indexing — root_hash will not be set.
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}

	dstPath := filepath.Join(t.TempDir(), "seeded.db")
	seeded, err := SeedFromDonor(donorPath, dstPath, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if seeded {
		t.Fatal("expected seeded=false for incomplete donor (no root_hash)")
	}

	// Destination should not have been created.
	if _, err := os.Stat(dstPath); err == nil {
		t.Fatal("expected dst to not exist when donor is incomplete")
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
	if _, err := SeedFromDonor(donorPath, dstPath, projectDir); err != nil {
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
