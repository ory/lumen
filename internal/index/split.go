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
// limit pass through unchanged. Token count is estimated as len(content)/4.
func splitOversizedChunks(chunks []chunker.Chunk, maxTokens int) []chunker.Chunk {
	if maxTokens <= 0 {
		return chunks
	}

	maxChars := maxTokens * 4
	var result []chunker.Chunk
	for _, c := range chunks {
		if len(c.Content) <= maxChars {
			result = append(result, c)
			continue
		}
		subChunks := splitChunk(c, maxChars)
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

	return createSubChunks(c, parts)
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

// overlapLines is the number of lines from the end of the previous partition
// prepended to the next partition. This improves search recall for queries
// that match concepts spanning a split boundary.
const overlapLines = 5

func createSubChunks(c chunker.Chunk, parts [][]string) []chunker.Chunk {
	totalParts := len(parts)
	var result []chunker.Chunk
	lineOffset := 0

	for i, part := range parts {
		// Prepend overlap from the previous partition (except for the first).
		effective := part
		overlapCount := 0
		if i > 0 {
			prev := parts[i-1]
			n := min(overlapLines, len(prev))
			overlap := prev[len(prev)-n:]
			effective = make([]string, 0, n+len(part))
			effective = append(effective, overlap...)
			effective = append(effective, part...)
			overlapCount = n
		}

		content := strings.Join(effective, "")
		startLine := c.StartLine + lineOffset - overlapCount
		endLine := c.StartLine + lineOffset + len(part) - 1
		symbol := fmt.Sprintf("%s[%d/%d]", c.Symbol, i+1, totalParts)

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
