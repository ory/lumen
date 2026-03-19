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
)

// MarkdownChunker splits Markdown/MDX files into chunks by ATX heading sections.
// Each heading (# / ## / ###) plus its body becomes one chunk.
// Content before the first heading is emitted as a "preamble" chunk if non-empty.
// Oversized chunks pass through to the existing split.go pipeline.
type MarkdownChunker struct{}

// NewMarkdownChunker returns a new MarkdownChunker.
func NewMarkdownChunker() *MarkdownChunker { return &MarkdownChunker{} }

// Chunk implements Chunker for Markdown/MDX files.
func (c *MarkdownChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	lines := strings.Split(string(content), "\n")
	var chunks []Chunk

	type section struct {
		symbol    string
		startLine int
		lines     []string
	}

	var current *section
	flush := func(endLine int) {
		if current == nil {
			return
		}
		body := strings.TrimSpace(strings.Join(current.lines, "\n"))
		if body == "" {
			current = nil
			return
		}
		chunks = append(chunks, makeChunk(filePath, current.symbol, "section", current.startLine, endLine, body))
		current = nil
	}

	for i, line := range lines {
		lineNum := i + 1
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			flush(lineNum - 1)
			heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
			current = &section{symbol: heading, startLine: lineNum, lines: []string{line}}
			continue
		}
		if current != nil {
			current.lines = append(current.lines, line)
		} else if strings.TrimSpace(line) != "" && len(chunks) == 0 {
			current = &section{symbol: "preamble", startLine: lineNum, lines: []string{line}}
		}
	}
	flush(len(lines))

	return chunks, nil
}
