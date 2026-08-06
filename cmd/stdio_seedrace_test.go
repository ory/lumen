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

package cmd

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/indexlock"
)

// TestGetOrCreate_SkipsSeedWhenIndexLockHeld reproduces the cross-process race
// between the MCP server (getOrCreate) and the background indexer (`lumen
// index`, spawned by the SessionStart hook). Both target the same fresh
// worktree DB. The background indexer holds the exclusive index flock while it
// creates and seeds the DB; getOrCreate must NOT run SeedFromDonor at the same
// time — two concurrent seeds against the same dbPath corrupt the SQLite file
// (SeedFromDonor copies through a shared temp path and renames over the DB).
//
// getOrCreate must defer to the lock holder instead of racing it.
func TestGetOrCreate_SkipsSeedWhenIndexLockHeld(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	const model = "test-model"
	dbPath := config.DBPathForProject(projectDir, model)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// Simulate the background indexer holding the exclusive index lock while it
	// creates + seeds the DB.
	lk, err := indexlock.TryAcquire(indexlock.LockPathForDB(dbPath))
	if err != nil || lk == nil {
		t.Fatalf("acquire index lock: err=%v lk=%v", err, lk)
	}
	defer lk.Release()

	var seedCalls int32
	ic := &indexerCache{
		embedder:      &stubEmbedder{},
		cfg:           newTestConfigService(t, 512),
		findDonorFunc: func(_, _ string) string { return "/fake/donor.db" },
		seedFunc: func(_, _ string) (bool, error) {
			atomic.AddInt32(&seedCalls, 1)
			return true, nil
		},
		// Keep the wait-for-peer bounded so the test is fast.
		createWaitTimeout: 50 * time.Millisecond,
	}

	idx, _, _, err := ic.getOrCreate(projectDir, "", model)
	if err != nil {
		t.Fatalf("getOrCreate: %v", err)
	}
	t.Cleanup(func() { _ = idx.Close() })

	if n := atomic.LoadInt32(&seedCalls); n != 0 {
		t.Fatalf("getOrCreate ran SeedFromDonor (%d times) while another process held "+
			"the index lock — this races the background indexer and can corrupt the DB", n)
	}
}
