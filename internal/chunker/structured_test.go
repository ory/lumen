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

package chunker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructuredChunker_SmallYAML_SingleChunk(t *testing.T) {
	// Small file: must pass through as a single "document" chunk (fast path).
	input := []byte("name: foo\nversion: 1\n")
	c := NewStructuredChunker(2048)
	chunks, err := c.Chunk("test.yaml", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Kind != "document" {
		t.Errorf("Kind = %q, want document", chunks[0].Kind)
	}
	if chunks[0].Symbol != "root" {
		t.Errorf("Symbol = %q, want root", chunks[0].Symbol)
	}
}

func TestStructuredChunker_LargeYAML_SplitsAtTopLevelKeys(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("alpha:\n")
	for i := 0; i < 10; i++ {
		sb.WriteString("  key" + string(rune('a'+i)) + ": value\n")
	}
	sb.WriteString("beta:\n")
	for i := 0; i < 10; i++ {
		sb.WriteString("  key" + string(rune('a'+i)) + ": value\n")
	}

	c := NewStructuredChunker(2) // 2 tokens = 8 chars, forces splitting
	chunks, err := c.Chunk("test.yaml", []byte(sb.String()))
	if err != nil {
		t.Fatal(err)
	}

	if len(chunks) < 2 {
		t.Fatalf("expected >= 2 chunks, got %d", len(chunks))
	}
	for _, ch := range chunks {
		if ch.Kind != "section" {
			t.Errorf("chunk %q: Kind = %q, want section", ch.Symbol, ch.Kind)
		}
	}
	for _, ch := range chunks {
		if !strings.HasPrefix(ch.Content, "# path:") {
			t.Errorf("chunk %q: Content missing path prefix: %q", ch.Symbol, ch.Content[:min(40, len(ch.Content))])
		}
	}
	symbols := make(map[string]bool)
	for _, ch := range chunks {
		symbols[ch.Symbol] = true
	}
	if !symbols["alpha"] && !containsPrefixStr(symbols, "alpha.") {
		t.Errorf("no chunk for top-level key 'alpha'; symbols: %v", symbolKeysList(symbols))
	}
	if !symbols["beta"] && !containsPrefixStr(symbols, "beta.") {
		t.Errorf("no chunk for top-level key 'beta'; symbols: %v", symbolKeysList(symbols))
	}
}

func TestStructuredChunker_JSON_SmallFile(t *testing.T) {
	input := []byte(`{"name":"foo","version":"1"}`)
	c := NewStructuredChunker(2048)
	chunks, err := c.Chunk("test.json", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Kind != "document" {
		t.Errorf("Kind = %q, want document", chunks[0].Kind)
	}
}

func TestStructuredChunker_JSON_LargeFile_SplitsAtKeys(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"dependencies":{`)
	for i := 0; i < 20; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"pkg` + string(rune('a'+i)) + `":"1.0.0"`)
	}
	sb.WriteString(`},"devDependencies":{`)
	for i := 0; i < 20; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"dev` + string(rune('a'+i)) + `":"2.0.0"`)
	}
	sb.WriteString(`}}`)

	c := NewStructuredChunker(2)
	chunks, err := c.Chunk("test.json", []byte(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected >= 2 chunks, got %d", len(chunks))
	}
	for _, ch := range chunks {
		if !strings.HasPrefix(ch.Content, "# path:") {
			t.Errorf("chunk %q: missing path prefix", ch.Symbol)
		}
	}
}

func TestStructuredChunker_Empty(t *testing.T) {
	c := NewStructuredChunker(2048)
	chunks, err := c.Chunk("test.yaml", []byte("   "))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(chunks))
	}
}

func TestStructuredChunker_MultiDocYAML(t *testing.T) {
	input := []byte("name: alpha\n---\nname: beta\n")
	c := NewStructuredChunker(2048)
	chunks, err := c.Chunk("test.yaml", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk for multi-doc YAML")
	}
}

func TestStructuredChunker_PathPrefix_ContentEmbedded(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("grafana:\n")
	for i := 0; i < 30; i++ {
		sb.WriteString("  key" + string(rune('a'+i%26)) + ": value\n")
	}
	sb.WriteString("prometheus:\n")
	for i := 0; i < 30; i++ {
		sb.WriteString("  key" + string(rune('a'+i%26)) + ": value\n")
	}

	c := NewStructuredChunker(2)
	chunks, err := c.Chunk("values.yaml", []byte(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range chunks {
		if !strings.Contains(ch.Content, ch.Symbol) {
			t.Errorf("Content for chunk %q does not contain its own symbol", ch.Symbol)
		}
	}
}

func containsPrefixStr(m map[string]bool, prefix string) bool {
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

func symbolKeysList(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// sourceLines returns lines start..end (1-based, inclusive) of src.
func structuredSourceLines(src string, start, end int) string {
	lines := strings.Split(src, "\n")
	return strings.Join(lines[start-1:end], "\n")
}

func findStructuredChunk(t *testing.T, chunks []Chunk, symbol string) Chunk {
	t.Helper()
	for _, ch := range chunks {
		if ch.Symbol == symbol {
			return ch
		}
	}
	syms := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		syms = append(syms, ch.Symbol)
	}
	t.Fatalf("no chunk with symbol %q; have %v", symbol, syms)
	return Chunk{}
}

func assertStructuredSpan(t *testing.T, src string, ch Chunk, start, end int) {
	t.Helper()
	if ch.StartLine != start || ch.EndLine != end {
		t.Errorf("%s: span = %d-%d, want %d-%d", ch.Symbol, ch.StartLine, ch.EndLine, start, end)
	}
	want := "# path: " + ch.Symbol + "\n" + structuredSourceLines(src, start, end)
	if ch.Content != want {
		t.Errorf("%s: Content =\n%s\nwant\n%s", ch.Symbol, ch.Content, want)
	}
}

func TestStructuredChunker_JSON_NestedObjectSpansSourceLines(t *testing.T) {
	src := `{
  "paths": {
    "/pet/{petId}": {
      "get": {
        "summary": "Find pet by ID",
        "operationId": "getPetById",
        "responses": {
          "200": {
            "description": "successful operation"
          }
        }
      },
      "delete": {
        "summary": "Deletes a pet",
        "operationId": "deletePet"
      }
    }
  },
  "info": {
    "title": "Petstore"
  }
}
`
	c := NewStructuredChunker(60) // 240 chars: forces descent below "paths"
	chunks, err := c.Chunk("openapi.json", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	assertStructuredSpan(t, src, findStructuredChunk(t, chunks, "paths./pet/{petId}.get"), 4, 12)
	assertStructuredSpan(t, src, findStructuredChunk(t, chunks, "paths./pet/{petId}.delete"), 13, 16)
	assertStructuredSpan(t, src, findStructuredChunk(t, chunks, "info"), 19, 21)
}

func TestStructuredChunker_YAML_SpanIncludesBlankLinesAndComments(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("name: ci\n\njobs:\n  test:\n    runs-on: ubuntu-latest\n\n    # run the suite\n    steps:\n")
	for i := 0; i < 12; i++ {
		sb.WriteString("      - run: npm run step" + string(rune('a'+i)) + "\n")
	}
	sb.WriteString("\n      # coverage upload\n      - name: upload\n        with:\n          path: ./coverage/lcov.info\n\n")
	sb.WriteString("  lint:\n    runs-on: ubuntu-latest\n    steps:\n      - run: npm run lint\n")
	src := sb.String()

	c := NewStructuredChunker(130) // 520 chars: "jobs" splits, "jobs.test" fits
	chunks, err := c.Chunk("ci.yml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	// jobs.test starts at line 4 and ends at the lcov path line (25), not
	// at the re-serialized length.
	assertStructuredSpan(t, src, findStructuredChunk(t, chunks, "jobs.test"), 4, 25)
	assertStructuredSpan(t, src, findStructuredChunk(t, chunks, "jobs.lint"), 27, 30)
}

func TestStructuredChunker_YAML_MultiDocLineNumbers(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("# first\napiVersion: v1\nkind: Service\nspec:\n  ports:\n")
	for i := 0; i < 10; i++ {
		sb.WriteString("    - port: 80" + string(rune('0'+i)) + "\n")
	}
	sb.WriteString("---\napiVersion: policy/v1\nkind: PodDisruptionBudget\nspec:\n  minAvailable: 1\n\n  # labels\n")
	for i := 0; i < 10; i++ {
		sb.WriteString("  label" + string(rune('a'+i)) + ": value\n")
	}
	src := sb.String()

	c := NewStructuredChunker(50)
	chunks, err := c.Chunk("multi.yaml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var docTwoSpec Chunk
	for _, ch := range chunks {
		if ch.StartLine < 1 || ch.EndLine < ch.StartLine {
			t.Errorf("%s: invalid span %d-%d", ch.Symbol, ch.StartLine, ch.EndLine)
		}
		if ch.Symbol == "spec" && ch.StartLine > 15 {
			docTwoSpec = ch
		}
	}
	assertStructuredSpan(t, src, findStructuredChunk(t, chunks, "spec"), 4, 15)
	assertStructuredSpan(t, src, docTwoSpec, 19, 32)
}

func TestStructuredChunker_JSON_MinifiedKeepsSplitting(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"dependencies":{`)
	for i := 0; i < 20; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"pkg` + string(rune('a'+i)) + `":"1.0.0"`)
	}
	sb.WriteString(`}}`)
	c := NewStructuredChunker(10)
	chunks, err := c.Chunk("package.json", []byte(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	ch := findStructuredChunk(t, chunks, "dependencies.pkgc")
	if ch.StartLine != 1 || ch.EndLine != 1 {
		t.Errorf("span = %d-%d, want 1-1", ch.StartLine, ch.EndLine)
	}
	if ch.Content != "# path: dependencies.pkgc\n"+`"pkgc":"1.0.0"` {
		t.Errorf("Content = %q", ch.Content)
	}
}

func TestStructuredChunker_FixtureContentMatchesSourceSpan(t *testing.T) {
	var files []string
	for _, pattern := range []string{"../../testdata/fixtures/json/*.json", "../../testdata/fixtures/yaml/*.y*ml"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	if len(files) == 0 {
		t.Fatal("no fixtures found")
	}
	for _, maxTokens := range []int{16, 64, 512} {
		c := NewStructuredChunker(maxTokens)
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			src := string(data)
			lineCount := strings.Count(strings.TrimSuffix(src, "\n"), "\n") + 1
			chunks, err := c.Chunk(file, data)
			if err != nil {
				t.Fatal(err)
			}
			for _, ch := range chunks {
				if ch.StartLine < 1 || ch.EndLine < ch.StartLine || ch.EndLine > lineCount {
					t.Errorf("%s %s: invalid span %d-%d", file, ch.Symbol, ch.StartLine, ch.EndLine)
					continue
				}
				lines := structuredSourceLines(src, ch.StartLine, ch.EndLine)
				body := ch.Content
				if ch.Kind == "section" {
					body = strings.TrimPrefix(body, "# path: "+ch.Symbol+"\n")
				}
				if body != lines && (ch.StartLine != ch.EndLine || !strings.Contains(lines, body)) {
					t.Errorf("%s %s (%d-%d, max %d): content does not match source lines", file, ch.Symbol, ch.StartLine, ch.EndLine, maxTokens)
				}
			}
		}
	}
}

func TestStructuredChunker_YAML_HeadCommentsBelongToFollowingKey(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("# Default values\n\nalertmanager:\n  ## Configuration for the alertmanager service\n  ##\n  service:\n")
	for i := 0; i < 10; i++ {
		sb.WriteString("    annotation" + string(rune('a'+i)) + ": value\n")
	}
	sb.WriteString("\n  ## Port for the service\n\n  # enabled: false\n  port: 9093\n")
	src := sb.String()

	c := NewStructuredChunker(80) // 320 chars: "alertmanager" splits, "service" fits
	chunks, err := c.Chunk("values.yaml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	assertStructuredSpan(t, src, findStructuredChunk(t, chunks, "alertmanager.service"), 4, 16)
	assertStructuredSpan(t, src, findStructuredChunk(t, chunks, "alertmanager.port"), 18, 21)
}
