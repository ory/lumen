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
