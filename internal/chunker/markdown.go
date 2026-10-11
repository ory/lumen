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
// Each heading (# through ######) plus its body becomes one chunk whose symbol
// is the heading text without the marker or optional closing sequence.
// Content before the first heading is emitted as a "preamble" chunk if non-empty.
// Lines inside fenced code blocks (``` or ~~~) are never headings, so a shell
// comment such as "# install" stays inside its section. Chunk content is the
// exact source lines of the section, with surrounding blank lines dropped.
// Oversized chunks pass through to the existing split.go pipeline.
type MarkdownChunker struct{}

// NewMarkdownChunker returns a new MarkdownChunker.
func NewMarkdownChunker() *MarkdownChunker { return &MarkdownChunker{} }

// Chunk implements Chunker for Markdown/MDX files.
func (c *MarkdownChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	lines := strings.Split(string(content), "\n")
	var chunks []Chunk

	symbol := ""
	start := 0 // 1-based start line of the open section; 0 when none is open
	flush := func(endLine int) {
		if start == 0 {
			return
		}
		first, last := start, endLine
		for first <= last && strings.TrimSpace(lines[first-1]) == "" {
			first++
		}
		for last >= first && strings.TrimSpace(lines[last-1]) == "" {
			last--
		}
		if first <= last {
			body := strings.Join(lines[first-1:last], "\n")
			chunks = append(chunks, makeChunk(filePath, symbol, "section", first, last, body))
		}
		start = 0
	}

	var fence markdownFence
	for i, line := range lines {
		lineNum := i + 1
		if fence.char != 0 {
			if fence.closedBy(line) {
				fence = markdownFence{}
			}
		} else if f, ok := openMarkdownFence(line); ok {
			fence = f
		} else if heading, ok := markdownHeading(line); ok {
			flush(lineNum - 1)
			symbol, start = heading, lineNum
			continue
		}
		if start == 0 && len(chunks) == 0 && strings.TrimSpace(line) != "" {
			symbol, start = "preamble", lineNum
		}
	}
	flush(len(lines))

	return chunks, nil
}

// markdownFence is an open CommonMark fenced code block.
type markdownFence struct {
	char byte // '`' or '~'; 0 when no fence is open
	n    int  // length of the opening fence run
}

// openMarkdownFence reports whether line opens a fenced code block: up to three
// spaces of indentation and at least three backticks or tildes. A backtick
// fence's info string may not contain backticks.
func openMarkdownFence(line string) (markdownFence, bool) {
	rest, ok := trimMarkdownIndent(line)
	if !ok || rest == "" || (rest[0] != '`' && rest[0] != '~') {
		return markdownFence{}, false
	}
	ch := rest[0]
	n := 0
	for n < len(rest) && rest[n] == ch {
		n++
	}
	if n < 3 || (ch == '`' && strings.IndexByte(rest[n:], '`') >= 0) {
		return markdownFence{}, false
	}
	return markdownFence{char: ch, n: n}, true
}

// closedBy reports whether line closes f: same fence character, at least as
// long as the opening run, up to three spaces of indentation and nothing but
// whitespace after it.
func (f markdownFence) closedBy(line string) bool {
	rest, ok := trimMarkdownIndent(line)
	if !ok {
		return false
	}
	n := 0
	for n < len(rest) && rest[n] == f.char {
		n++
	}
	return n >= f.n && strings.TrimSpace(rest[n:]) == ""
}

// markdownHeading returns the text of an ATX heading line (levels 1-6), with
// the optional closing # sequence removed. Headings with no text are ignored.
func markdownHeading(line string) (string, bool) {
	rest, ok := trimMarkdownIndent(line)
	if !ok {
		return "", false
	}
	n := 0
	for n < len(rest) && rest[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || (n < len(rest) && rest[n] != ' ' && rest[n] != '\t') {
		return "", false
	}
	text := strings.TrimSpace(rest[n:])
	if closing := strings.TrimRight(text, "#"); closing != text {
		if closing == "" {
			text = ""
		} else if c := closing[len(closing)-1]; c == ' ' || c == '\t' {
			text = strings.TrimSpace(closing)
		}
	}
	return text, text != ""
}

// trimMarkdownIndent strips up to three leading spaces; four or more make the
// line an indented code block, which cannot start a heading or fence.
func trimMarkdownIndent(line string) (string, bool) {
	line = strings.TrimSuffix(line, "\r")
	i := 0
	for i < len(line) && i < 4 && line[i] == ' ' {
		i++
	}
	if i == 4 || (i < len(line) && line[i] == '\t') {
		return "", false
	}
	return line[i:], true
}
