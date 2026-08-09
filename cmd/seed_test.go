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
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func seedTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSeedFromDonorIfNew(t *testing.T) {
	tests := []struct {
		name     string
		setupDB  bool // pre-create the destination DB
		donor    string
		seedErr  error // error returned by the seed stub
		wantFind bool  // donor discovery should run
		wantSeed bool  // seed should run
	}{
		{
			name:     "skips when DB already exists",
			setupDB:  true,
			donor:    "/donor.db",
			wantFind: false,
			wantSeed: false,
		},
		{
			name:     "seeds when DB missing and donor found",
			donor:    "/donor.db",
			wantFind: true,
			wantSeed: true,
		},
		{
			name:     "no seed when no donor found",
			donor:    "",
			wantFind: true,
			wantSeed: false,
		},
		{
			name:     "seed error is swallowed",
			donor:    "/donor.db",
			seedErr:  errors.New("copy failed"),
			wantFind: true,
			wantSeed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "index.db")
			if tt.setupDB {
				if err := os.WriteFile(dbPath, []byte("existing"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			var findCalled, seedCalled bool
			var gotDonor, gotDst string
			var statuses []string
			warning := seedFromDonorIfNew(
				context.Background(),
				dbPath,
				"/project",
				"model",
				seedTestLogger(),
				seedOptions{
					findDonor: func(_, _ string) string {
						findCalled = true
						return tt.donor
					},
					seed: func(_ context.Context, donor, dst, projectPath string) (bool, error) {
						seedCalled = true
						gotDonor, gotDst = donor, dst
						if projectPath != "/project" {
							t.Errorf("seed project path = %q, want /project", projectPath)
						}
						return tt.seedErr == nil, tt.seedErr
					},
					status: func(message string) {
						statuses = append(statuses, message)
					},
				},
			)

			if findCalled != tt.wantFind {
				t.Errorf("donor discovery called = %v, want %v", findCalled, tt.wantFind)
			}
			if seedCalled != tt.wantSeed {
				t.Errorf("seed called = %v, want %v", seedCalled, tt.wantSeed)
			}
			if tt.wantSeed && (gotDonor != tt.donor || gotDst != dbPath) {
				t.Errorf("seed called with (%q, %q), want (%q, %q)", gotDonor, gotDst, tt.donor, dbPath)
			}
			if tt.seedErr != nil {
				if warning == "" {
					t.Error("expected warning when seed fails")
				}
				if len(statuses) != 2 {
					t.Fatalf("expected start and failure status, got %v", statuses)
				}
			} else if warning != "" {
				t.Errorf("unexpected warning: %q", warning)
			}
		})
	}
}
