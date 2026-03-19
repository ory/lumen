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

	"github.com/ory/lumen/internal/chunker"
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
	for i := range 100 {
		lines = append(lines, fmt.Sprintf("    line %d: some code content here\n", i))
	}
	content := strings.Join(lines, "")
	c := makeTestChunk("BigFunc", 10, 109, content)

	result := splitOversizedChunks([]chunker.Chunk{c}, 200)
	if len(result) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(result))
	}

	// All sub-chunks inherit the parent symbol unchanged.
	for i, r := range result {
		if r.Symbol != "BigFunc" {
			t.Errorf("chunk %d: expected symbol %q, got %q", i, "BigFunc", r.Symbol)
		}
		if r.Kind != "function" {
			t.Errorf("chunk %d: expected kind 'function', got %q", i, r.Kind)
		}
		if r.FilePath != "test.go" {
			t.Errorf("chunk %d: expected file 'test.go', got %q", i, r.FilePath)
		}
	}

	// Line ranges: first chunk starts at original start, last covers original end.
	// Adjacent chunks overlap by up to overlapLines.
	if result[0].StartLine != 10 {
		t.Errorf("first chunk should start at line 10, got %d", result[0].StartLine)
	}
	if result[len(result)-1].EndLine != 109 {
		t.Errorf("last chunk should end at line 109, got %d", result[len(result)-1].EndLine)
	}
	for i := 1; i < len(result); i++ {
		gap := result[i].StartLine - result[i-1].EndLine
		if gap > 1 {
			t.Errorf("gap between chunk %d (end %d) and %d (start %d)",
				i-1, result[i-1].EndLine, i, result[i].StartLine)
		}
		// Overlap means StartLine <= previous EndLine
		if gap > 0 {
			t.Logf("chunk %d-%d: no overlap (gap=%d)", i-1, i, gap)
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

	// All original content is present (each original line appears in at least one chunk)
	for _, origLine := range lines {
		found := false
		for _, r := range result {
			if strings.Contains(r.Content, origLine) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("original line missing from all chunks: %q", origLine[:20])
		}
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

func TestPartitionLines_SplitsAtBlankLine(t *testing.T) {
	// Build content where a blank line falls near the split boundary.
	// 20 lines of ~40 chars = ~800 chars; maxChars=500 forces one split.
	// The blank line at index 10 should be the preferred split point.
	var lines []string
	for i := range 20 {
		if i == 10 {
			lines = append(lines, "\n")
		} else {
			lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
		}
	}
	parts := partitionLines(lines, 500)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	// The blank line (index 10) should be the last line of part[0].
	last := strings.TrimSpace(parts[0][len(parts[0])-1])
	if last != "" {
		t.Errorf("expected part[0] to end at blank line, last line content: %q", last)
	}
}

func TestPartitionLines_SplitsAtClosingBrace(t *testing.T) {
	// Build content where a closing brace falls near the split boundary.
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
	}
	lines = append(lines, "}\n") // closing brace at index 10
	for i := range 9 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i+11))
	}
	parts := partitionLines(lines, 500)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	// The closing brace should be the last line of part[0].
	last := strings.TrimSpace(parts[0][len(parts[0])-1])
	if last != "}" {
		t.Errorf("expected part[0] to end at '}', last line content: %q", last)
	}
}

func TestPartitionLines_FallsBackWhenNoBoundary(t *testing.T) {
	// Content with no blank lines or closing braces — should still split.
	var lines []string
	for i := range 20 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
	}
	parts := partitionLines(lines, 500)
	if len(parts) < 2 {
		t.Fatalf("expected fallback split into at least 2 parts, got %d", len(parts))
	}
}

func TestSplitOversizedChunks_MixedSizes(t *testing.T) {
	small := makeTestChunk("Small", 1, 3, "small\n")
	var bigLines []string
	for i := range 50 {
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

func TestPartitionLines_SplitsAtClosingBraceComma(t *testing.T) {
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
	}
	lines = append(lines, "},\n") // trailing comma variant
	for i := range 9 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i+11))
	}
	parts := partitionLines(lines, 500)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	last := strings.TrimSpace(parts[0][len(parts[0])-1])
	if last != "}," {
		t.Errorf("expected part[0] to end at '},', got %q", last)
	}
}

func TestPartitionLines_SplitsAtJSClosingBrace(t *testing.T) {
	// JavaScript/TypeScript pattern: });
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
	}
	lines = append(lines, "});\n")
	for i := range 9 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i+11))
	}
	parts := partitionLines(lines, 500)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	last := strings.TrimSpace(parts[0][len(parts[0])-1])
	if last != "});" {
		t.Errorf("expected part[0] to end at '});', got %q", last)
	}
}

func TestPartitionLines_SplitsAtCppClosingBrace(t *testing.T) {
	// C++ pattern: };
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
	}
	lines = append(lines, "};\n")
	for i := range 9 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i+11))
	}
	parts := partitionLines(lines, 500)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	last := strings.TrimSpace(parts[0][len(parts[0])-1])
	if last != "};" {
		t.Errorf("expected part[0] to end at '};', got %q", last)
	}
}

func TestPartitionLines_SplitsAtRubyEnd(t *testing.T) {
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
	}
	lines = append(lines, "end\n")
	for i := range 9 {
		lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i+11))
	}
	parts := partitionLines(lines, 500)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	last := strings.TrimSpace(parts[0][len(parts[0])-1])
	if last != "end" {
		t.Errorf("expected part[0] to end at 'end', got %q", last)
	}
}

func TestPartitionLines_SplitsAtDedent(t *testing.T) {
	// Python-style: indented block followed by less-indented line
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("        line %02d: indented block\n", i))
	}
	// Dedent: new top-level definition
	lines = append(lines, "def next_function():\n")
	for i := range 9 {
		lines = append(lines, fmt.Sprintf("        line %02d: indented block\n", i+11))
	}
	parts := partitionLines(lines, 600)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	// The dedent line should be the first line of part[1] (split before it)
	first := strings.TrimSpace(parts[1][0])
	if first != "def next_function():" {
		t.Errorf("expected part[1] to start at dedented line, got %q", first)
	}
}

func TestPartitionLines_BoundaryAtLookbackEdge(t *testing.T) {
	// Place a blank line exactly at the start of the lookback window.
	// 30 lines total, maxChars forces split around line 25.
	// Blank at line 5 (outside window) should NOT be used;
	// blank at line 10 (at edge of 20-line lookback from ~25) should be used.
	var lines []string
	for i := range 30 {
		if i == 10 {
			lines = append(lines, "\n") // at lookback edge
		} else {
			lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
		}
	}
	parts := partitionLines(lines, 800)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	last := strings.TrimSpace(parts[0][len(parts[0])-1])
	if last != "" {
		t.Errorf("expected part[0] to end at blank line (lookback edge), got %q", last)
	}
}

func TestPartitionLines_MultipleSplits(t *testing.T) {
	// Content requiring 3+ splits with unevenly distributed boundaries.
	var lines []string
	for i := range 60 {
		if i == 8 || i == 35 || i == 50 {
			lines = append(lines, "}\n") // uneven boundary placement
		} else {
			lines = append(lines, fmt.Sprintf("    line %02d: some code content here\n", i))
		}
	}
	parts := partitionLines(lines, 600)
	if len(parts) < 3 {
		t.Fatalf("expected at least 3 parts, got %d", len(parts))
	}
	// Verify all original lines present
	allLines := make([]string, 0)
	for _, p := range parts {
		allLines = append(allLines, p...)
	}
	for _, orig := range lines {
		found := false
		for _, l := range allLines {
			if l == orig {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing line: %q", strings.TrimSpace(orig))
		}
	}
}

func TestCreateSubChunks_OverlapIncluded(t *testing.T) {
	// Verify that sub-chunks after the first include overlap lines from the previous partition.
	parts := [][]string{
		{"line1\n", "line2\n", "line3\n", "line4\n", "line5\n", "line6\n", "line7\n", "line8\n"},
		{"line9\n", "line10\n", "line11\n"},
	}
	c := makeTestChunk("Func", 1, 11, "")
	result := createSubChunks(c, parts, 0)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result))
	}
	// Second chunk should contain overlap from first partition's last N lines
	second := result[1].Content
	if !strings.Contains(second, "line4\n") {
		t.Error("expected second chunk to contain overlap line 'line4'")
	}
	if !strings.Contains(second, "line9\n") {
		t.Error("expected second chunk to contain 'line9'")
	}
	// First chunk should NOT have overlap
	first := result[0].Content
	if strings.Contains(first, "line9\n") {
		t.Error("first chunk should not contain lines from second partition")
	}
}

func TestCreateSubChunks_HeaderInjected(t *testing.T) {
	// Verify that sub-chunks after the first have the function signature header
	// (first headerLines lines of parts[0]) prepended before the overlap.
	header := []string{"func BigFunc(x int) {\n", "\t// doc comment\n", "\tvar a int\n", "\tvar b int\n", "\tvar c int\n"}
	rest := []string{"line6\n", "line7\n", "line8\n", "line9\n", "line10\n", "line11\n", "line12\n", "line13\n"}
	part0 := append(header, rest...)
	part1 := []string{"line14\n", "line15\n", "line16\n"}

	c := makeTestChunk("BigFunc", 1, 16, "")
	result := createSubChunks(c, [][]string{part0, part1}, 0)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result))
	}
	second := result[1].Content
	// Header lines must appear in the second chunk.
	for _, h := range header {
		if !strings.Contains(second, h) {
			t.Errorf("expected second chunk to contain header line %q", h)
		}
	}
	// Body line of second partition must also appear.
	if !strings.Contains(second, "line14\n") {
		t.Error("expected second chunk to contain 'line14'")
	}
	// First chunk must not contain lines from part1.
	if strings.Contains(result[0].Content, "line14\n") {
		t.Error("first chunk should not contain lines from second partition")
	}
}

func TestCreateSubChunks_OverlapLineNumbers(t *testing.T) {
	// Verify line numbers account for overlap correctly.
	// First partition has 20 lines so the 10-line overlap doesn't consume it entirely.
	parts := [][]string{
		{"a\n", "b\n", "c\n", "d\n", "e\n", "f\n", "g\n", "h\n", "i\n", "j\n",
			"k\n", "l\n", "m\n", "n\n", "o\n", "p\n", "q\n", "r\n", "s\n", "t\n"},
		{"u\n", "v\n"},
	}
	c := makeTestChunk("Func", 10, 31, "")
	result := createSubChunks(c, parts, 0)
	// First chunk: lines 10-29
	if result[0].StartLine != 10 || result[0].EndLine != 29 {
		t.Errorf("first chunk lines: got %d-%d, want 10-29", result[0].StartLine, result[0].EndLine)
	}
	// Second chunk with overlap: starts overlapLines=10 lines before partition boundary
	if result[1].StartLine != 20 {
		t.Errorf("second chunk start: got %d, want 20 (overlap of 10)", result[1].StartLine)
	}
	if result[1].EndLine != 31 {
		t.Errorf("second chunk end: got %d, want 31", result[1].EndLine)
	}
}

func makeVarChunk(symbol, filePath string, startLine, endLine int, content string) chunker.Chunk {
	return chunker.Chunk{
		ID:        "id-" + symbol,
		FilePath:  filePath,
		Symbol:    symbol,
		Kind:      "var",
		StartLine: startLine,
		EndLine:   endLine,
		Content:   content,
	}
}

func TestMergeUndersizedChunks_MergesSmallVarChunks(t *testing.T) {
	// Two small var chunks from the same file should be merged.
	c1 := makeVarChunk("Foo", "a.go", 1, 1, "var Foo = 1")
	c2 := makeVarChunk("Bar", "a.go", 2, 2, "var Bar = 2")
	result := mergeUndersizedChunks([]chunker.Chunk{c1, c2}, 50)
	if len(result) != 1 {
		t.Fatalf("expected 1 merged chunk, got %d", len(result))
	}
	m := result[0]
	if m.Symbol != "Foo+Bar" {
		t.Errorf("expected symbol 'Foo+Bar', got %q", m.Symbol)
	}
	if m.Kind != "var" {
		t.Errorf("expected kind 'var', got %q", m.Kind)
	}
	if m.StartLine != 1 || m.EndLine != 2 {
		t.Errorf("expected lines 1-2, got %d-%d", m.StartLine, m.EndLine)
	}
	if !strings.Contains(m.Content, "var Foo = 1") || !strings.Contains(m.Content, "var Bar = 2") {
		t.Errorf("merged content missing source chunks: %q", m.Content)
	}
}

func TestMergeUndersizedChunks_SkipsFunctions(t *testing.T) {
	// Function chunks should never be merged even if small.
	c1 := chunker.Chunk{ID: "f1", FilePath: "a.go", Symbol: "F1", Kind: "function", StartLine: 1, EndLine: 2, Content: "func F1() {}"}
	c2 := chunker.Chunk{ID: "f2", FilePath: "a.go", Symbol: "F2", Kind: "function", StartLine: 3, EndLine: 4, Content: "func F2() {}"}
	result := mergeUndersizedChunks([]chunker.Chunk{c1, c2}, 50)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks (functions untouched), got %d", len(result))
	}
}

func TestMergeUndersizedChunks_BreaksAcrossFiles(t *testing.T) {
	// Small var chunks from different files must NOT be merged.
	c1 := makeVarChunk("X", "a.go", 1, 1, "var X = 1")
	c2 := makeVarChunk("Y", "b.go", 1, 1, "var Y = 2")
	result := mergeUndersizedChunks([]chunker.Chunk{c1, c2}, 50)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks (different files), got %d", len(result))
	}
}

func TestMergeUndersizedChunks_LargeChunkBreaksRun(t *testing.T) {
	// A large chunk in the middle breaks the merge run.
	small1 := makeVarChunk("A", "a.go", 1, 1, "var A = 1")
	large := makeVarChunk("Big", "a.go", 2, 2, strings.Repeat("x", 300)) // >= 50*4=200 chars
	small2 := makeVarChunk("B", "a.go", 3, 3, "var B = 3")
	result := mergeUndersizedChunks([]chunker.Chunk{small1, large, small2}, 50)
	// small1 alone (no neighbor to merge with), large passes through, small2 alone
	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	if result[0].Symbol != "A" {
		t.Errorf("expected first chunk symbol 'A', got %q", result[0].Symbol)
	}
}

func TestMergeUndersizedChunks_SingleChunkNoMerge(t *testing.T) {
	c := makeVarChunk("Solo", "a.go", 1, 1, "var Solo = 1")
	result := mergeUndersizedChunks([]chunker.Chunk{c}, 50)
	if len(result) != 1 || result[0].Symbol != "Solo" {
		t.Fatalf("expected single chunk unchanged, got %v", result)
	}
}

func TestMergeUndersizedChunks_MixedKindsNotMerged(t *testing.T) {
	// var and const of the same file should NOT merge (different kinds).
	varChunk := makeVarChunk("V", "a.go", 1, 1, "var V = 1")
	constChunk := chunker.Chunk{ID: "c1", FilePath: "a.go", Symbol: "C", Kind: "const", StartLine: 2, EndLine: 2, Content: "const C = 2"}
	result := mergeUndersizedChunks([]chunker.Chunk{varChunk, constChunk}, 50)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks (different kinds not merged), got %d", len(result))
	}
}

func TestMergeUndersizedChunks_UniqueID(t *testing.T) {
	// Merged chunks must have a unique, stable ID.
	c1 := makeVarChunk("X", "a.go", 1, 1, "var X = 1")
	c2 := makeVarChunk("Y", "a.go", 2, 2, "var Y = 2")
	r1 := mergeUndersizedChunks([]chunker.Chunk{c1, c2}, 50)
	r2 := mergeUndersizedChunks([]chunker.Chunk{c1, c2}, 50)
	if r1[0].ID != r2[0].ID {
		t.Errorf("ID not stable across calls: %q vs %q", r1[0].ID, r2[0].ID)
	}
	if r1[0].ID == c1.ID || r1[0].ID == c2.ID {
		t.Errorf("merged ID collides with source ID: %q", r1[0].ID)
	}
}

func TestFindSplitPoint_ExactBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		wantIdx int
	}{
		{
			name:    "closing brace",
			lines:   []string{"a\n", "b\n", "}\n", "c\n"},
			wantIdx: 3,
		},
		{
			name:    "closing brace comma",
			lines:   []string{"a\n", "b\n", "},\n", "c\n"},
			wantIdx: 3,
		},
		{
			name:    "JS callback close",
			lines:   []string{"a\n", "b\n", "});\n", "c\n"},
			wantIdx: 3,
		},
		{
			name:    "C++ class close",
			lines:   []string{"a\n", "b\n", "};\n", "c\n"},
			wantIdx: 3,
		},
		{
			name:    "Ruby end",
			lines:   []string{"a\n", "b\n", "end\n", "c\n"},
			wantIdx: 3,
		},
		{
			name:    "blank line",
			lines:   []string{"a\n", "\n", "b\n", "c\n"},
			wantIdx: 2, // scans backward from 3, finds blank at idx 1, returns 2
		},
		{
			name:    "no boundary",
			lines:   []string{"a\n", "b\n", "c\n"},
			wantIdx: 0,
		},
		{
			name:    "single line",
			lines:   []string{"a\n"},
			wantIdx: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSplitPoint(tt.lines)
			if got != tt.wantIdx {
				t.Errorf("findSplitPoint() = %d, want %d", got, tt.wantIdx)
			}
		})
	}
}
