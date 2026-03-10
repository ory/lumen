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

package config

import (
	"strings"
	"testing"
)

func TestEnvOrDefaultInt(t *testing.T) {
	t.Setenv("TEST_DIMS", "384")
	if got := EnvOrDefaultInt("TEST_DIMS", 1024); got != 384 {
		t.Fatalf("got %d, want 384", got)
	}
	if got := EnvOrDefaultInt("TEST_DIMS_UNSET", 1024); got != 1024 {
		t.Fatalf("got %d, want 1024", got)
	}
}

func TestDBPathForProject(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project", "model-a")
		p2 := DBPathForProject("/home/user/project", "model-a")
		if p1 != p2 {
			t.Fatalf("expected same path, got %q and %q", p1, p2)
		}
	})

	t.Run("different project paths produce different hashes", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project-a", "model-a")
		p2 := DBPathForProject("/home/user/project-b", "model-a")
		if p1 == p2 {
			t.Fatalf("expected different paths, got same: %q", p1)
		}
	})

	t.Run("different models produce different hashes", func(t *testing.T) {
		p1 := DBPathForProject("/home/user/project", "model-a")
		p2 := DBPathForProject("/home/user/project", "model-b")
		if p1 == p2 {
			t.Fatalf("expected different paths, got same: %q", p1)
		}
	})

	t.Run("uses IndexVersion not runtime state", func(t *testing.T) {
		// The path must be stable regardless of build-time variables.
		// We verify this by computing the path twice and confirming stability,
		// and by checking that IndexVersion is a non-empty hardcoded constant.
		if IndexVersion == "" {
			t.Fatal("IndexVersion must not be empty")
		}
		p1 := DBPathForProject("/some/path", "some-model")
		p2 := DBPathForProject("/some/path", "some-model")
		if p1 != p2 {
			t.Fatalf("path not stable: %q vs %q", p1, p2)
		}
	})

	t.Run("ends with index.db", func(t *testing.T) {
		p := DBPathForProject("/some/path", "model")
		if !strings.HasSuffix(p, "index.db") {
			t.Fatalf("expected path to end with index.db, got %q", p)
		}
	})
}
