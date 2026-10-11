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
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	sitter_svelte "github.com/alexaandru/go-sitter-forest/svelte"
	sitter "github.com/smacker/go-tree-sitter"
)

// SvelteChunker implements Chunker for .svelte files using a two-phase parse:
//  1. The outer Svelte grammar splits the file into its immediate document
//     children: <script> elements, <style> elements and template nodes.
//  2. Each script block's raw text is re-parsed with the TypeScript chunker
//     to extract named symbols (functions, classes, interfaces, etc.).
//
// Besides script symbols it emits, in source order:
//   - "template" regions named template[n] (1-based, counted per file): one
//     per immediate template child (element, block, expression, text),
//     with inline callbacks, bindings and actions kept inside. Children that
//     share a source line, or a comment directly above a child, form one
//     region. Nested nodes are never emitted again as separate regions.
//   - "style" regions named style[n], including attributes and delimiters.
//   - Top-level script statements not covered by a symbol ($effect(...),
//     onMount(...), legacy `$:` labels, $props() destructuring), kind
//     "statement", named script[n].statement[m] (both 1-based; m counts the
//     statements emitted for the n-th script element). Uncovered variable
//     declarations with a single identifier (`let count = $state(0)`) keep
//     their real name with kind "var" or "const".
//
// Imports and comments alone are metadata, not chunks. Regions are bounded
// downstream by the indexers' token-budget splitting. Line numbers are always
// file-relative (1-based).
type SvelteChunker struct {
	svelteLanguage *sitter.Language
	tsChunker      *TreeSitterChunker
}

// NewSvelteChunker returns a SvelteChunker that extracts script-block symbols
// using tsChunker for the inner TypeScript parse.
func NewSvelteChunker(tsChunker *TreeSitterChunker) *SvelteChunker {
	return &SvelteChunker{
		svelteLanguage: sitter.NewLanguage(sitter_svelte.GetLanguage()),
		tsChunker:      tsChunker,
	}
}

// Chunk parses a .svelte file and returns script symbols and statements plus
// template and style regions.
func (s *SvelteChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	root, err := sitter.ParseCtx(context.Background(), content, s.svelteLanguage)
	if err != nil {
		return nil, fmt.Errorf("parse svelte %s: %w", filePath, err)
	}

	var chunks []Chunk
	var scripts, styles, templates int
	var group []*sitter.Node // pending template children of one region
	flushTemplate := func() {
		if len(group) == 0 {
			return
		}
		templates++
		name := fmt.Sprintf("template[%d]", templates)
		if c, ok := svelteRegion(filePath, content, name, "template", group[0].StartByte(), group[len(group)-1].EndByte()); ok {
			chunks = append(chunks, c)
		}
		group = nil
	}

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		switch child.Type() {
		case "script_element":
			flushTemplate()
			scripts++
			sc, err := s.chunkScriptElement(filePath, content, child, scripts)
			if err != nil {
				return chunks, fmt.Errorf("chunk script element in %s: %w", filePath, err)
			}
			chunks = append(chunks, sc...)
		case "style_element":
			flushTemplate()
			styles++
			if c, ok := svelteRegion(filePath, content, fmt.Sprintf("style[%d]", styles), "style", child.StartByte(), child.EndByte()); ok {
				chunks = append(chunks, c)
			}
		default:
			if len(bytes.TrimSpace(content[child.StartByte():child.EndByte()])) == 0 {
				continue
			}
			if len(group) > 0 && !joinsTemplateRegion(group[len(group)-1], child) {
				flushTemplate()
			}
			group = append(group, child)
		}
	}
	flushTemplate()

	return chunks, nil
}

// joinsTemplateRegion reports whether next belongs to the same template region
// as prev: it starts on the line prev ends on, or prev is a comment directly
// above it.
func joinsTemplateRegion(prev, next *sitter.Node) bool {
	prevEnd := prev.EndPoint().Row
	if prev.EndPoint().Column == 0 && prevEnd > prev.StartPoint().Row {
		prevEnd-- // node ends with a newline
	}
	return next.StartPoint().Row == prevEnd ||
		(prev.Type() == "comment" && next.StartPoint().Row == prevEnd+1)
}

// chunkScriptElement finds the raw_text child of a script_element, re-parses
// it with TypeScript, and adjusts all chunk line numbers to be file-relative.
// scriptIndex is the 1-based index of the script element in the file.
func (s *SvelteChunker) chunkScriptElement(filePath string, content []byte, node *sitter.Node, scriptIndex int) ([]Chunk, error) {
	if s.tsChunker == nil {
		return nil, fmt.Errorf("chunk svelte %s: nil TypeScript chunker", filePath)
	}

	rawText := findRawText(node)
	if rawText == nil {
		return nil, nil
	}

	scriptStart := rawText.StartByte()
	scriptContent := content[scriptStart:rawText.EndByte()]
	// lineOffset is the 0-based row of the raw_text node in the file.
	// The TypeScript chunker produces 1-based lines relative to scriptContent,
	// so file_line = ts_line + lineOffset.
	lineOffset := int(rawText.StartPoint().Row)

	inner, err := s.tsChunker.Chunk(filePath, scriptContent)
	if err != nil {
		return nil, err
	}

	covered := symbolByteSpans(scriptContent, inner)
	for i := range inner {
		inner[i].StartLine += lineOffset
		inner[i].EndLine += lineOffset
	}

	tsRoot, err := sitter.ParseCtx(context.Background(), scriptContent, s.tsChunker.language)
	if err != nil {
		return nil, fmt.Errorf("parse script %s: %w", filePath, err)
	}
	imports := treeImports(tsRoot, scriptContent)
	shadowed := treeShadowedNames(tsRoot, scriptContent)

	statements := 0
	for i := 0; i < int(tsRoot.NamedChildCount()); i++ {
		stmt := tsRoot.NamedChild(i)
		if isCommentNode(stmt.Type()) || stmt.Type() == "import_statement" || isCoveredStatement(stmt, covered) {
			continue
		}
		symbol, kind := uncoveredDeclarationName(stmt, scriptContent)
		if symbol == "" {
			statements++
			symbol, kind = fmt.Sprintf("script[%d].statement[%d]", scriptIndex, statements), "statement"
		}
		start := stmt.StartByte()
		if comment := findLeadingComments(stmt); comment != nil {
			start = comment.StartByte()
		}
		c, ok := svelteRegion(filePath, content, symbol, kind, scriptStart+start, scriptStart+stmt.EndByte())
		if !ok {
			continue
		}
		populateTreeMetadata(&c, stmt, scriptContent, start)
		c.Imports = imports
		refs := c.References[:0]
		for _, ref := range c.References {
			if !shadowed[ref] {
				refs = append(refs, ref)
			}
		}
		c.References = refs
		inner = append(inner, c)
	}

	sort.SliceStable(inner, func(a, b int) bool { return inner[a].StartLine < inner[b].StartLine })
	return inner, nil
}

// byteSpan is a half-open byte range.
type byteSpan struct{ start, end uint32 }

// symbolByteSpans locates each chunk's content in src, starting at the
// chunk's first line, so coverage is decided by actual spans rather than
// lines that several constructs may share.
func symbolByteSpans(src []byte, chunks []Chunk) []byteSpan {
	lineStarts := []int{0}
	for i, b := range src {
		if b == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	spans := make([]byteSpan, 0, len(chunks))
	for _, c := range chunks {
		if c.StartLine < 1 || c.StartLine > len(lineStarts) {
			continue
		}
		off := lineStarts[c.StartLine-1]
		if idx := bytes.Index(src[off:], []byte(c.Content)); idx >= 0 {
			start := uint32(off + idx)
			spans = append(spans, byteSpan{start, start + uint32(len(c.Content))})
		}
	}
	return spans
}

// isCoveredStatement reports whether a symbol chunk already contains stmt,
// or the declaration it exports.
func isCoveredStatement(stmt *sitter.Node, covered []byteSpan) bool {
	nodes := []*sitter.Node{stmt}
	if decl := stmt.ChildByFieldName("declaration"); decl != nil {
		nodes = append(nodes, decl)
	}
	for _, n := range nodes {
		for _, sp := range covered {
			if sp.start <= n.StartByte() && n.EndByte() <= sp.end {
				return true
			}
		}
	}
	return false
}

// uncoveredDeclarationName names a variable declaration that declares exactly
// one identifier, such as `let count = $state(0)`. Destructuring patterns and
// other statements return "".
func uncoveredDeclarationName(stmt *sitter.Node, src []byte) (symbol, kind string) {
	if stmt.Type() != "lexical_declaration" && stmt.Type() != "variable_declaration" {
		return "", ""
	}
	var name *sitter.Node
	for i := 0; i < int(stmt.NamedChildCount()); i++ {
		decl := stmt.NamedChild(i)
		if decl.Type() != "variable_declarator" {
			continue
		}
		if name != nil {
			return "", ""
		}
		name = decl.ChildByFieldName("name")
	}
	if name == nil || name.Type() != "identifier" {
		return "", ""
	}
	kind = "var"
	if stmt.ChildCount() > 0 && stmt.Child(0).Type() == "const" {
		kind = "const"
	}
	return name.Content(src), kind
}

// svelteRegion builds a chunk for content[start:end] with surrounding
// whitespace trimmed. The span is widened to whole lines when only
// whitespace separates it from the line boundaries, so Content equals the
// source lines unless another construct shares a line. The ID also hashes
// the symbol, keeping regions with identical text distinct.
func svelteRegion(filePath string, content []byte, symbol, kind string, start, end uint32) (Chunk, bool) {
	s, e := int(start), int(end)
	for s < e && isSpaceByte(content[s]) {
		s++
	}
	for e > s && isSpaceByte(content[e-1]) {
		e--
	}
	if s == e {
		return Chunk{}, false
	}
	lineStart := bytes.LastIndexByte(content[:s], '\n') + 1
	if len(bytes.TrimSpace(content[lineStart:s])) == 0 {
		s = lineStart
	}
	lineEnd := len(content)
	if i := bytes.IndexByte(content[e:], '\n'); i >= 0 {
		lineEnd = e + i
	}
	if len(bytes.TrimSpace(content[e:lineEnd])) == 0 {
		e = lineEnd
	}
	text := string(content[s:e])
	startLine := bytes.Count(content[:s], []byte{'\n'}) + 1
	endLine := startLine + bytes.Count(content[s:e], []byte{'\n'})
	c := makeChunk(filePath, symbol, kind, startLine, endLine, text)
	h := sha256.Sum256([]byte(filePath + ":" + symbol + ":" + text))
	c.ID = fmt.Sprintf("%x", h)[:16]
	return c, true
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}

// findRawText returns the first raw_text child of node, or nil.
func findRawText(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		if child := node.Child(i); child.Type() == "raw_text" {
			return child
		}
	}
	return nil
}
