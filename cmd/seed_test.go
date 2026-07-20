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
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// withSeedStubs swaps the donor-discovery and seed indirections for the
// duration of a test and restores them afterward.
func withSeedStubs(t *testing.T, find func(string, string) string, seed func(string, string) (bool, error)) {
	t.Helper()
	origFind, origSeed := findDonorFn, seedFromDonorFn
	findDonorFn, seedFromDonorFn = find, seed
	t.Cleanup(func() { findDonorFn, seedFromDonorFn = origFind, origSeed })
}

func TestSeedFromDonorIfNew_SkipsWhenDBExists(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	if err := os.WriteFile(dbPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	findCalled := false
	withSeedStubs(t,
		func(string, string) string { findCalled = true; return "/donor.db" },
		func(string, string) (bool, error) { t.Fatal("seed must not run when DB exists"); return false, nil },
	)

	seedFromDonorIfNew(dbPath, "/project", "model", discardLogger())
	if findCalled {
		t.Fatal("donor discovery must not run when the DB already exists")
	}
}

func TestSeedFromDonorIfNew_SeedsWhenMissing(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db") // does not exist

	var gotDonor, gotDst string
	withSeedStubs(t,
		func(project, model string) string { return "/donor.db" },
		func(donor, dst string) (bool, error) { gotDonor, gotDst = donor, dst; return true, nil },
	)

	seedFromDonorIfNew(dbPath, "/project", "model", discardLogger())
	if gotDonor != "/donor.db" || gotDst != dbPath {
		t.Fatalf("seed called with (%q, %q), want (%q, %q)", gotDonor, gotDst, "/donor.db", dbPath)
	}
}

func TestSeedFromDonorIfNew_NoDonorFound(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")

	withSeedStubs(t,
		func(string, string) string { return "" },
		func(string, string) (bool, error) { t.Fatal("seed must not run without a donor"); return false, nil },
	)

	seedFromDonorIfNew(dbPath, "/project", "model", discardLogger())
}

func TestSeedFromDonorIfNew_SeedErrorIsSwallowed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")

	withSeedStubs(t,
		func(string, string) string { return "/donor.db" },
		func(string, string) (bool, error) { return false, errors.New("copy failed") },
	)

	// Best-effort: a seed failure must not panic or propagate.
	seedFromDonorIfNew(dbPath, "/project", "model", discardLogger())
}
