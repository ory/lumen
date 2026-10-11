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

package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuiltArtifactIndexesChunks runs only when CI supplies LUMEN_ARTIFACT.
// It executes the native release-shaped binary, including Windows, against a
// unavailable embedding endpoint (offline BM25 default) and catches distributions that scan files but
// persist zero chunks.
func TestBuiltArtifactIndexesChunks(t *testing.T) {
	artifact := os.Getenv("LUMEN_ARTIFACT")
	if artifact == "" {
		t.Skip("LUMEN_ARTIFACT is only set by the cross-platform build job")
	}
	absArtifact, err := filepath.Abs(artifact)
	if err != nil {
		t.Fatalf("resolve artifact: %v", err)
	}

	project := t.TempDir()
	source := `package smoke

// ValidateToken checks whether an authentication token is valid.
func ValidateToken(token string) bool { return token != "" }
`
	if err := os.WriteFile(filepath.Join(project, "auth.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := exec.Command(absArtifact, "search", "authentication token", "--path", project, "--min-score", "-1")
	dataHome := t.TempDir()
	cmd.Env = append(os.Environ(),
		"OLLAMA_HOST=http://127.0.0.1:1",
		"LM_STUDIO_HOST=http://127.0.0.1:1",
		"LUMEN_RETRIEVAL=bm25",
		"LUMEN_BACKEND=ollama",
		"LUMEN_EMBED_MODEL=artifact-smoke",
		"LUMEN_EMBED_DIMS=4",
		"XDG_DATA_HOME="+dataHome,
		"XDG_CONFIG_HOME="+dataHome,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("artifact search failed: %v\n%s", err, output)
	}
	text := string(output)
	if !strings.Contains(text, "ValidateToken") || !strings.Contains(text, "Found 1 result") {
		t.Fatalf("artifact did not index and search its chunk:\n%s", text)
	}
}
