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
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/ory/lumen/internal/chunker"
)

// splitOversizedChunks splits chunks whose estimated token count exceeds
// maxTokens into smaller sub-chunks at line boundaries. Chunks under the
// limit pass through unchanged. Token count is estimated as len(content)/4,
// with additional budget reserved for the filepath prefix prepended during
// embedding ("// " + filePath + "\n").
func splitOversizedChunks(chunks []chunker.Chunk, maxTokens int) []chunker.Chunk {
	if maxTokens <= 0 {
		return chunks
	}

	maxChars := maxTokens * 4
	var result []chunker.Chunk
	for _, c := range chunks {
		// Reserve space for the embed prefix: "// " + filePath + "\n"
		overhead := 3 + len(c.FilePath) + 1
		budget := maxChars - overhead
		if budget < 1 {
			budget = 1
		}
		if len(c.Content) <= budget {
			result = append(result, c)
			continue
		}
		subChunks := splitChunk(c, budget)
		result = append(result, subChunks...)
	}
	return result
}

func splitChunk(c chunker.Chunk, maxChars int) []chunker.Chunk {
	lines := splitContentByLines(c.Content)
	parts := partitionLines(lines, maxChars)

	if len(parts) <= 1 {
		return []chunker.Chunk{c}
	}

	return createSubChunks(c, parts, maxChars)
}

func splitContentByLines(content string) []string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func partitionLines(lines []string, maxChars int) [][]string {
	var parts [][]string
	var current []string
	currentLen := 0
	for _, line := range lines {
		if currentLen+len(line) > maxChars && len(current) > 0 {
			splitAt := findSplitPoint(current)
			if splitAt > 0 && splitAt < len(current) {
				parts = append(parts, current[:splitAt])
				remaining := make([]string, len(current)-splitAt)
				copy(remaining, current[splitAt:])
				current = remaining
				currentLen = 0
				for _, l := range current {
					currentLen += len(l)
				}
			} else {
				parts = append(parts, current)
				current = nil
				currentLen = 0
			}
		}
		current = append(current, line)
		currentLen += len(line)
	}
	if len(current) > 0 {
		parts = append(parts, current)
	}
	return parts
}

// findSplitPoint scans backward through lines looking for a natural split
// boundary. It recognizes blank lines and block-ending patterns across
// language families:
//   - C-family: }, },  });  };
//   - Ruby/Elixir: end
//   - Python: lines with reduced indentation after a block (dedent heuristic)
//
// Returns the index at which to begin the next partition (i.e. the first part
// is lines[:idx]). Returns 0 if no suitable boundary is found within the
// lookback window.
func findSplitPoint(lines []string) int {
	const lookback = 20
	start := max(1, len(lines)-lookback)
	for i := len(lines) - 1; i >= start; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if isSplitBoundary(trimmed) {
			return i + 1
		}
		// Dedent heuristic: if this line is less indented than the next,
		// it likely starts a new block (works for Python, YAML, etc.).
		// Split before this line so it becomes the start of the next partition.
		if i+1 < len(lines) && trimmed != "" {
			thisIndent := countLeadingWhitespace(lines[i])
			nextIndent := countLeadingWhitespace(lines[i+1])
			if nextIndent > 0 && thisIndent < nextIndent {
				return i
			}
		}
	}
	return 0
}

func isSplitBoundary(trimmed string) bool {
	switch trimmed {
	case "", "}", "},", "});", "};", "end":
		return true
	}
	return false
}

func countLeadingWhitespace(s string) int {
	for i, c := range s {
		if c != ' ' && c != '\t' {
			return i
		}
	}
	return len(s)
}

// minMergeTokens is the threshold below which same-kind adjacent chunks are merged.
// Chunks smaller than this are vulnerable to keyword-density false positives.
const minMergeTokens = 50

// mergeUndersizedChunks combines consecutive same-file, same-kind chunks that
// fall below minMergeTokens into a single chunk. Only "var", "const", and "type"
// kinds are eligible; functions and methods are always left intact.
// Token count is estimated as len(content)/4.
func mergeUndersizedChunks(chunks []chunker.Chunk, minTokens int) []chunker.Chunk {
	minChars := minTokens * 4
	result := make([]chunker.Chunk, 0, len(chunks))
	i := 0
	for i < len(chunks) {
		c := chunks[i]
		if !isMergeable(c.Kind) || len(c.Content) >= minChars {
			result = append(result, c)
			i++
			continue
		}
		// Collect a run of consecutive undersized same-file same-kind chunks.
		group := []chunker.Chunk{c}
		for i+len(group) < len(chunks) {
			next := chunks[i+len(group)]
			if next.FilePath != c.FilePath || next.Kind != c.Kind || len(next.Content) >= minChars {
				break
			}
			group = append(group, next)
		}
		if len(group) == 1 {
			result = append(result, c)
			i++
			continue
		}
		result = append(result, mergeChunkGroup(group))
		i += len(group)
	}
	return result
}

func isMergeable(kind string) bool {
	return kind == "var" || kind == "const" || kind == "type"
}

func mergeChunkGroup(group []chunker.Chunk) chunker.Chunk {
	var sb strings.Builder
	symbols := make([]string, len(group))
	for i, c := range group {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(c.Content)
		symbols[i] = c.Symbol
	}
	content := sb.String()
	first, last := group[0], group[len(group)-1]

	h := sha256.New()
	h.Write([]byte(first.FilePath))
	h.Write([]byte{':'})
	h.Write([]byte(content))
	id := fmt.Sprintf("%x", h.Sum(nil))[:16]

	return chunker.Chunk{
		ID:        id,
		FilePath:  first.FilePath,
		Symbol:    strings.Join(symbols, "+"),
		Kind:      first.Kind,
		StartLine: first.StartLine,
		EndLine:   last.EndLine,
		Content:   content,
	}
}

// overlapLines is the number of lines from the end of the previous partition
// prepended to the next partition. This improves search recall for queries
// that match concepts spanning a split boundary.
const overlapLines = 10

// headerLines is the number of lines taken from the start of the first
// partition and injected into all subsequent sub-chunks. This gives the
// embedding model the function signature / doc-comment context needed to
// associate algorithmic body code with its parent declaration.
const headerLines = 5

// linesLen returns the total byte length of all lines joined.
func linesLen(lines []string) int {
	n := 0
	for _, l := range lines {
		n += len(l)
	}
	return n
}

// createSubChunks builds sub-chunks from the partitioned lines.
// maxChars is the per-chunk content budget (0 = unlimited). Header and overlap
// lines are prepended to sub-chunks after the first only when they fit within
// the budget; if they would overflow, the sub-chunk uses just its own lines.
func createSubChunks(c chunker.Chunk, parts [][]string, maxChars int) []chunker.Chunk {
	var result []chunker.Chunk
	lineOffset := 0

	// Extract function signature / doc-comment header from the first partition.
	header := parts[0]
	if len(header) > headerLines {
		header = header[:headerLines]
	}

	for i, part := range parts {
		// Prepend overlap from the previous partition (except for the first).
		effective := part
		overlapCount := 0
		if i > 0 {
			prev := parts[i-1]
			n := min(overlapLines, len(prev))
			overlap := prev[len(prev)-n:]
			extended := make([]string, 0, len(header)+n+len(part))
			extended = append(extended, header...)  // function signature context
			extended = append(extended, overlap...) // cross-boundary overlap
			extended = append(extended, part...)
			// Only use the extended version if it fits within the content budget.
			// If maxChars <= 0 there is no limit, so always extend.
			if maxChars <= 0 || linesLen(extended) <= maxChars {
				effective = extended
				overlapCount = n
			}
		}

		content := strings.Join(effective, "")
		startLine := c.StartLine + lineOffset - overlapCount
		endLine := c.StartLine + lineOffset + len(part) - 1
		symbol := c.Symbol

		h := sha256.New()
		h.Write([]byte(c.FilePath))
		h.Write([]byte{':'})
		h.Write([]byte(content))
		id := fmt.Sprintf("%x", h.Sum(nil))[:16]

		result = append(result, chunker.Chunk{
			ID:        id,
			FilePath:  c.FilePath,
			Symbol:    symbol,
			Kind:      c.Kind,
			StartLine: startLine,
			EndLine:   endLine,
			Content:   content,
		})

		lineOffset += len(part)
	}
	return result
}
