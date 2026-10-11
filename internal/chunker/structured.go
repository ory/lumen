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
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// StructuredChunker splits YAML and JSON files by their key hierarchy.
// It recursively descends the document tree: if a subtree fits within the
// token budget, it is emitted as one chunk with its dotted key path prepended.
// If a subtree is too large, the chunker recurses into its children.
// Chunk spans and content come from the source text (yaml.v3 node positions),
// never from re-serialized nodes, so line ranges cover the whole subtree
// including blank lines and comments.
// The splitOversizedChunks pipeline in index/split.go is the backstop for
// leaf nodes that cannot be subdivided further.
type StructuredChunker struct {
	maxChars int // maxTokens * 4 (1 token ≈ 4 chars)
}

// NewStructuredChunker returns a StructuredChunker. maxTokens is the token
// budget per chunk; use LUMEN_MAX_CHUNK_TOKENS (default 2048).
func NewStructuredChunker(maxTokens int) *StructuredChunker {
	return &StructuredChunker{maxChars: maxTokens * 4}
}

// Chunk implements Chunker for YAML and JSON files.
func (c *StructuredChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return nil, nil
	}
	src := newStructuredSource(content)

	// Fast path: small file fits as a single chunk.
	if len(trimmed) <= c.maxChars {
		return []Chunk{src.wholeDocumentChunk(filePath)}, nil
	}

	// Parse into a yaml.Node tree. yaml.v3 understands both YAML and JSON.
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var chunks []Chunk
	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err != nil {
			break // EOF or parse error: return what we have
		}
		if doc.Kind == 0 || len(doc.Content) == 0 {
			continue
		}
		// doc is always a DocumentNode; doc.Content[0] is the root.
		root := doc.Content[0]
		sp, docStart := src.rootSpan(root)
		chunks = append(chunks, c.recurse(filePath, src, root, "", sp, docStart)...)
	}

	// If parsing produced nothing (e.g. parse error on large file), fall back.
	if len(chunks) == 0 {
		return []Chunk{src.wholeDocumentChunk(filePath)}, nil
	}
	return chunks, nil
}

// recurse emits chunks for node, whose source extent is sp. If the source
// text fits within maxChars, it emits a single chunk. Otherwise it recurses
// into children, which may claim head comments from line floor onwards.
func (c *StructuredChunker) recurse(filePath string, src *structuredSource, node *yaml.Node, path string, sp srcSpan, floor int) []Chunk {
	symbol := normalizeSymbol(path)
	if len(src.text(sp, c.maxChars)) <= c.maxChars {
		return c.createNodeChunk(filePath, src, symbol, sp)
	}

	switch node.Kind {
	case yaml.MappingNode:
		return c.recurseMapping(filePath, src, node, path, sp, floor)
	case yaml.SequenceNode:
		return c.recurseSequence(filePath, src, node, path, sp, floor)
	default:
		return c.createNodeChunk(filePath, src, symbol, sp)
	}
}

func normalizeSymbol(path string) string {
	if path == "" {
		return "root"
	}
	return path
}

func (c *StructuredChunker) createNodeChunk(filePath string, src *structuredSource, symbol string, sp srcSpan) []Chunk {
	content := "# path: " + symbol + "\n" + src.text(sp, c.maxChars)
	return []Chunk{makeChunk(filePath, symbol, "section", sp.startLine, sp.endLine, content)}
}

func (c *StructuredChunker) recurseMapping(filePath string, src *structuredSource, node *yaml.Node, path string, sp srcSpan, floor int) []Chunk {
	flow := isFlow(node)
	var chunks []Chunk
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode, valNode := node.Content[i], node.Content[i+1]
		var pair srcSpan
		if flow {
			pair = src.flowSpan(keyNode, valNode)
		} else {
			next := 0
			if i+2 < len(node.Content) {
				next = node.Content[i+2].Line
			}
			pair = src.blockSpan(keyNode.Line, next, keyNode.Column-1, floor, sp.endLine)
			floor = pair.endLine + 1
		}
		chunks = append(chunks, c.recurse(filePath, src, valNode, joinKeyPath(path, keyNode.Value), pair, keyNode.Line+1)...)
	}
	return chunks
}

func (c *StructuredChunker) recurseSequence(filePath string, src *structuredSource, node *yaml.Node, path string, sp srcSpan, floor int) []Chunk {
	flow := isFlow(node)
	var chunks []Chunk
	for i, item := range node.Content {
		var itemSpan srcSpan
		if flow {
			itemSpan = src.flowSpan(item, item)
		} else {
			next := 0
			if i+1 < len(node.Content) {
				next = node.Content[i+1].Line
			}
			itemSpan = src.blockSpan(item.Line, next, src.indent(item.Line), floor, sp.endLine)
			floor = itemSpan.endLine + 1
		}
		childPath := fmt.Sprintf("%s[%d]", path, i)
		chunks = append(chunks, c.recurse(filePath, src, item, childPath, itemSpan, item.Line)...)
	}
	return chunks
}

// joinKeyPath builds "parent.child"; if parent is empty returns child.
func joinKeyPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func isFlow(node *yaml.Node) bool {
	return node.Style&yaml.FlowStyle != 0
}

// srcSpan is a source extent: 1-based inclusive lines plus the exact byte
// range [start, end) inside them.
type srcSpan struct {
	startLine, endLine int
	start, end         int
}

// structuredSource indexes a YAML/JSON file by line so node positions can be
// mapped back to exact source text.
type structuredSource struct {
	data       []byte
	lineStarts []int // byte offset of each line; lineStarts[0] is line 1
	docEnds    []int // lines of "---" / "..." document markers
}

func newStructuredSource(data []byte) *structuredSource {
	s := &structuredSource{data: data, lineStarts: []int{0}}
	for i, b := range data {
		if b == '\n' && i+1 < len(data) {
			s.lineStarts = append(s.lineStarts, i+1)
		}
	}
	for n := 1; n <= s.lineCount(); n++ {
		line := s.line(n)
		if (strings.HasPrefix(line, "---") || strings.HasPrefix(line, "...")) &&
			(len(line) == 3 || line[3] == ' ' || line[3] == '\t' || line[3] == '\r') {
			s.docEnds = append(s.docEnds, n)
		}
	}
	return s
}

func (s *structuredSource) lineCount() int { return len(s.lineStarts) }

// lineEnd returns the byte offset just past line n, excluding its newline.
func (s *structuredSource) lineEnd(n int) int {
	if n < s.lineCount() {
		return s.lineStarts[n] - 1
	}
	return len(bytes.TrimSuffix(s.data, []byte("\n")))
}

func (s *structuredSource) line(n int) string {
	return string(s.data[s.lineStarts[n-1]:s.lineEnd(n)])
}

func (s *structuredSource) indent(n int) int {
	line := s.line(n)
	return len(line) - len(strings.TrimLeft(line, " "))
}

// lineOf returns the 1-based line containing byte offset off.
func (s *structuredSource) lineOf(off int) int {
	lo, hi := 0, len(s.lineStarts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if s.lineStarts[mid] <= off {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}

// offset converts a yaml.v3 position (1-based line, 1-based rune column)
// into a byte offset.
func (s *structuredSource) offset(line, col int) int {
	if line < 1 {
		return 0
	}
	if line > s.lineCount() {
		return len(s.data)
	}
	off, end := s.lineStarts[line-1], s.lineEnd(line)
	for i := 1; i < col && off < end; i++ {
		_, w := utf8.DecodeRune(s.data[off:end])
		off += w
	}
	return off
}

// lines returns a span covering whole lines start..end.
func (s *structuredSource) lines(start, end int) srcSpan {
	return srcSpan{startLine: start, endLine: end, start: s.lineStarts[start-1], end: s.lineEnd(end)}
}

// text returns the source lines of sp. A single line longer than maxChars is
// shared with siblings (minified JSON), so only the node's own bytes are used.
func (s *structuredSource) text(sp srcSpan, maxChars int) string {
	if sp.startLine == sp.endLine && s.lineEnd(sp.endLine)-s.lineStarts[sp.startLine-1] > maxChars {
		return string(s.data[sp.start:sp.end])
	}
	return string(s.data[s.lineStarts[sp.startLine-1]:s.lineEnd(sp.endLine)])
}

func (s *structuredSource) isBlankOrComment(n, maxIndent int) bool {
	trimmed := strings.TrimSpace(s.line(n))
	return trimmed == "" || (strings.HasPrefix(trimmed, "#") && s.indent(n) <= maxIndent)
}

// blockSpan is the extent of a block-style entry starting at line start:
// from the comments directly above it (no higher than floor) up to the line
// before the next sibling (or the parent's last line), excluding trailing
// blank lines and comments at or left of the entry's indentation, which are
// head comments of whatever follows.
func (s *structuredSource) blockSpan(start, next, indent, floor, parentEnd int) srcSpan {
	if start < 1 {
		start = 1
	}
	head := start
	for head-1 >= max(floor, 1) && s.isBlankOrComment(head-1, len(s.line(head-1))) {
		head--
	}
	for head < start && strings.TrimSpace(s.line(head)) == "" {
		head++
	}
	end := parentEnd
	if next > start {
		end = min(end, next-1)
	}
	for end > start && s.isBlankOrComment(end, indent) {
		end--
	}
	return s.lines(head, max(start, end))
}

// rootSpan is the extent of a document's root node and the first line of
// its document.
func (s *structuredSource) rootSpan(root *yaml.Node) (srcSpan, int) {
	start, end := 1, s.lineCount()
	for _, marker := range s.docEnds {
		if marker < root.Line {
			start = marker + 1
		} else if marker > root.Line {
			end = marker - 1
			break
		}
	}
	if isFlow(root) {
		return s.flowSpan(root, root), start
	}
	return s.blockSpan(root.Line, 0, root.Column-1, start, end), start
}

// flowSpan is the exact extent from first's position to the end of last,
// for nodes in flow context (JSON objects, [a, b], {k: v}).
func (s *structuredSource) flowSpan(first, last *yaml.Node) srcSpan {
	start := s.offset(first.Line, first.Column)
	end := s.flowEnd(last)
	if end < start {
		end = start
	}
	return srcSpan{startLine: s.lineOf(start), endLine: s.lineOf(max(start, end-1)), start: start, end: end}
}

// flowEnd returns the byte offset just past node in flow context.
func (s *structuredSource) flowEnd(node *yaml.Node) int {
	off := s.offset(node.Line, node.Column)
	data := s.data
	switch {
	case node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode:
		open := byte('{')
		if node.Kind == yaml.SequenceNode {
			open = '['
		}
		if i := bytes.IndexByte(data[off:], open); i >= 0 {
			return matchBracket(data, off+i)
		}
		return len(data)
	case node.Style&yaml.DoubleQuotedStyle != 0 && off < len(data) && data[off] == '"':
		return skipQuoted(data, off)
	case node.Style&yaml.SingleQuotedStyle != 0 && off < len(data) && data[off] == '\'':
		return skipQuoted(data, off)
	default:
		end := off
		for end < len(data) && !bytes.ContainsRune([]byte(",]}\n"), rune(data[end])) {
			end++
		}
		for end > off && (data[end-1] == ' ' || data[end-1] == '\t' || data[end-1] == '\r') {
			end--
		}
		return end
	}
}

// skipQuoted returns the offset just past the quoted scalar starting at off.
func skipQuoted(data []byte, off int) int {
	quote := data[off]
	for i := off + 1; i < len(data); i++ {
		switch {
		case quote == '"' && data[i] == '\\':
			i++
		case data[i] == quote:
			if quote == '\'' && i+1 < len(data) && data[i+1] == '\'' {
				i++
				continue
			}
			return i + 1
		}
	}
	return len(data)
}

// matchBracket returns the offset just past the bracket matching the one at
// off, skipping quoted strings and comments.
func matchBracket(data []byte, off int) int {
	depth := 0
	prev := byte(' ')
	for i := off; i < len(data); i++ {
		b := data[i]
		switch {
		case b == '"' || (b == '\'' && bytes.IndexByte([]byte(" \t\n{[,:"), prev) >= 0):
			i = skipQuoted(data, i) - 1
		case b == '#' && (prev == ' ' || prev == '\t' || prev == '\n'):
			for i < len(data) && data[i] != '\n' {
				i++
			}
		case b == '{' || b == '[':
			depth++
		case b == '}' || b == ']':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		if i < len(data) {
			prev = data[i]
		}
	}
	return len(data)
}

// wholeDocumentChunk covers the file from its first to last non-blank line.
func (s *structuredSource) wholeDocumentChunk(filePath string) Chunk {
	first, last := 1, s.lineCount()
	for first < last && strings.TrimSpace(s.line(first)) == "" {
		first++
	}
	for last > first && strings.TrimSpace(s.line(last)) == "" {
		last--
	}
	sp := s.lines(first, last)
	return makeChunk(filePath, "root", "document", first, last, string(s.data[sp.start:sp.end]))
}
