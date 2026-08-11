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
	"strings"
	"testing"

	"github.com/ory/lumen/internal/config"
)

func TestRunIndex_RefusesUnindexableRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lumenignore"), []byte("**\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := runIndex(indexCmd, []string{dir})
	if err == nil {
		t.Fatal("expected runIndex to refuse an un-indexable root, got nil error")
	}
	if !strings.Contains(err.Error(), ".lumenignore catch-all") {
		t.Fatalf("expected error to mention the .lumenignore catch-all reason, got %q", err.Error())
	}
}

func TestConfiguredDBPathUsesExplicitDimensionsForKnownModel(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	const model = "ordis/jina-embeddings-v2-base-code"
	cmd := flagsTestHarness(t, `
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
    dims: 1024
`)
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	got := configuredDBPath(cfg, project, model)
	want := config.DBPathForProjectProfile(project, model, 1024, cfg.VectorStorage(), cfg.MaxChunkTokens())
	registryPath := config.DBPathForProjectProfile(project, model, 768, cfg.VectorStorage(), cfg.MaxChunkTokens())
	if got != want {
		t.Fatalf("configuredDBPath = %q, want explicit-dimension path %q", got, want)
	}
	if got == registryPath {
		t.Fatal("explicit dimensions must not reuse the built-in model profile")
	}
}
