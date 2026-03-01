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
	"fmt"
	"strings"
	"testing"

	"github.com/aeneasr/agent-index/internal/chunker"
)

func makeTestChunk(symbol string, startLine, endLine int, content string) chunker.Chunk {
	return chunker.Chunk{
		ID:        "original-id-1234",
		FilePath:  "test.go",
		Symbol:    symbol,
		Kind:      "function",
		StartLine: startLine,
		EndLine:   endLine,
		Content:   content,
	}
}

func TestSplitOversizedChunks_UnderLimit(t *testing.T) {
	c := makeTestChunk("SmallFunc", 1, 5, "func SmallFunc() {\n\treturn\n}\n")
	result := splitOversizedChunks([]chunker.Chunk{c}, 2048)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0].ID != c.ID {
		t.Fatalf("expected unchanged chunk, got different ID")
	}
}

func TestSplitOversizedChunks_SplitsLargeChunk(t *testing.T) {
	// Create a chunk with 100 lines, each ~40 chars = ~4000 chars total
	// With maxTokens=200 (800 chars), this should split into ~5 parts
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("    line %d: some code content here\n", i))
	}
	content := strings.Join(lines, "")
	c := makeTestChunk("BigFunc", 10, 109, content)

	result := splitOversizedChunks([]chunker.Chunk{c}, 200)
	if len(result) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(result))
	}

	// Check symbol format
	for i, r := range result {
		expected := fmt.Sprintf("BigFunc[%d/%d]", i+1, len(result))
		if r.Symbol != expected {
			t.Errorf("chunk %d: expected symbol %q, got %q", i, expected, r.Symbol)
		}
		if r.Kind != "function" {
			t.Errorf("chunk %d: expected kind 'function', got %q", i, r.Kind)
		}
		if r.FilePath != "test.go" {
			t.Errorf("chunk %d: expected file 'test.go', got %q", i, r.FilePath)
		}
	}

	// Line ranges are contiguous and cover original range
	if result[0].StartLine != 10 {
		t.Errorf("first chunk should start at line 10, got %d", result[0].StartLine)
	}
	if result[len(result)-1].EndLine != 109 {
		t.Errorf("last chunk should end at line 109, got %d", result[len(result)-1].EndLine)
	}
	for i := 1; i < len(result); i++ {
		if result[i].StartLine != result[i-1].EndLine+1 {
			t.Errorf("gap between chunk %d (end %d) and %d (start %d)",
				i-1, result[i-1].EndLine, i, result[i].StartLine)
		}
	}

	// IDs are unique
	seen := map[string]bool{}
	for _, r := range result {
		if seen[r.ID] {
			t.Errorf("duplicate ID: %s", r.ID)
		}
		seen[r.ID] = true
	}

	// Content reconstructs to original
	var reconstructed string
	for _, r := range result {
		reconstructed += r.Content
	}
	if reconstructed != content {
		t.Error("reconstructed content does not match original")
	}
}

func TestSplitOversizedChunks_SingleHugeLine(t *testing.T) {
	// One line exceeding maxChars — should pass through as one chunk (no infinite loop)
	content := strings.Repeat("x", 10000) + "\n"
	c := makeTestChunk("HugeLine", 1, 1, content)

	result := splitOversizedChunks([]chunker.Chunk{c}, 100)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk for single huge line, got %d", len(result))
	}
}

func TestSplitOversizedChunks_ZeroMaxTokens(t *testing.T) {
	c := makeTestChunk("Func", 1, 5, "content\n")
	result := splitOversizedChunks([]chunker.Chunk{c}, 0)
	if len(result) != 1 {
		t.Fatalf("expected passthrough with maxTokens=0, got %d chunks", len(result))
	}
}

func TestSplitOversizedChunks_MixedSizes(t *testing.T) {
	small := makeTestChunk("Small", 1, 3, "small\n")
	var bigLines []string
	for i := 0; i < 50; i++ {
		bigLines = append(bigLines, fmt.Sprintf("line %d content here\n", i))
	}
	big := makeTestChunk("Big", 10, 59, strings.Join(bigLines, ""))

	result := splitOversizedChunks([]chunker.Chunk{small, big}, 100)
	// First chunk should be the small one unchanged
	if result[0].Symbol != "Small" {
		t.Errorf("expected first chunk to be Small, got %s", result[0].Symbol)
	}
	// Remaining chunks should be splits of Big
	if len(result) < 3 {
		t.Fatalf("expected at least 3 chunks (1 small + 2+ splits), got %d", len(result))
	}
}
