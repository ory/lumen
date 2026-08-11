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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuiltArtifactIndexesChunks runs only when CI supplies LUMEN_ARTIFACT.
// It executes the native release-shaped binary, including Windows, against a
// deterministic Ollama stub and catches distributions that scan files but
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

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			_, _ = fmt.Fprint(w, `{"models":[{"name":"artifact-smoke"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/embed":
			var request struct {
				Input []string `json:"input"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			vectors := make([][]float32, len(request.Input))
			for i := range vectors {
				vectors[i] = []float32{1, 0, 0, 0}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": vectors})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()

	project := t.TempDir()
	source := `package smoke

// ValidateToken checks whether an authentication token is valid.
func ValidateToken(token string) bool { return token != "" }
`
	if err := os.WriteFile(filepath.Join(project, "auth.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := exec.Command(absArtifact, "search", "authentication token validation", "--path", project, "--min-score", "-1")
	dataHome := t.TempDir()
	cmd.Env = append(os.Environ(),
		"OLLAMA_HOST="+ollama.URL,
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
