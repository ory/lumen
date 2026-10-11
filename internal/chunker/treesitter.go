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
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// QueryDef defines a tree-sitter query pattern and the chunk kind it produces.
// Pattern must have captures: @decl (full declaration) and @name (identifier).
// Patterns may use #eq? predicates.
type QueryDef struct {
	Pattern string
	Kind    string
	// SymbolFormat, when set, is a fmt format with one %s verb applied to the
	// @name text, for declarations whose name is not an identifier (operators,
	// indexers, finalizers).
	SymbolFormat string
	// OutsideFunctions drops matches nested in a function or method: local
	// assignments and declarations are not part of a file's declaration surface.
	OutsideFunctions bool
}

// LanguageDef bundles a tree-sitter language with its query patterns.
type LanguageDef struct {
	Language *sitter.Language
	Queries  []QueryDef
}

type compiledRule struct {
	query            *sitter.Query
	kind             string
	symbolFormat     string
	outsideFunctions bool
}

// TreeSitterChunker implements Chunker using tree-sitter.
type TreeSitterChunker struct {
	language *sitter.Language
	rules    []compiledRule
}

// NewTreeSitterChunker compiles the queries in def and returns a TreeSitterChunker.
func NewTreeSitterChunker(def LanguageDef) (*TreeSitterChunker, error) {
	rules := make([]compiledRule, 0, len(def.Queries))
	for _, qd := range def.Queries {
		q, err := sitter.NewQuery([]byte(qd.Pattern), def.Language)
		if err != nil {
			return nil, fmt.Errorf("compile query for kind %q: %w", qd.Kind, err)
		}
		rules = append(rules, compiledRule{
			query:            q,
			kind:             qd.Kind,
			symbolFormat:     qd.SymbolFormat,
			outsideFunctions: qd.OutsideFunctions,
		})
	}
	return &TreeSitterChunker{language: def.Language, rules: rules}, nil
}

// mustTreeSitterChunker panics if NewTreeSitterChunker returns an error.
// Use only for hardcoded query patterns.
func mustTreeSitterChunker(def LanguageDef) *TreeSitterChunker {
	c, err := NewTreeSitterChunker(def)
	if err != nil {
		panic(fmt.Sprintf("invalid tree-sitter query: %v", err))
	}
	return c
}

// Chunk parses content and returns semantic code chunks.
func (c *TreeSitterChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	root, err := sitter.ParseCtx(context.Background(), content, c.language)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	imports := treeImports(root, content)
	shadowed := treeShadowedNames(root, content)
	var chunks []Chunk

	for _, rule := range c.rules {
		qc := sitter.NewQueryCursor()
		qc.Exec(rule.query, root)

		for {
			m, ok := qc.NextMatch()
			if !ok {
				break
			}
			m = qc.FilterPredicates(m, content)

			var declNode, nameNode *sitter.Node
			for _, cap := range m.Captures {
				switch rule.query.CaptureNameForId(cap.Index) {
				case "decl":
					declNode = cap.Node
				case "name":
					nameNode = cap.Node
				}
			}

			if declNode == nil || nameNode == nil {
				continue
			}
			if rule.outsideFunctions && insideFunction(declNode) {
				continue
			}

			declNode = declarationNode(declNode)
			symbol := declarationName(nameNode, content)
			if rule.symbolFormat != "" {
				symbol = fmt.Sprintf(rule.symbolFormat, symbol)
			}
			if parent := findEnclosingSymbol(declNode, content); parent != "" {
				symbol = parent + "." + symbol
			}
			startLine := int(declNode.StartPoint().Row) + 1
			startByte := declNode.StartByte()
			leadingNode := declNode
			if p := declNode.Parent(); p != nil && p.Type() == "export_statement" {
				leadingNode = p
			}
			if comment := leadingBlock(leadingNode, content); comment != nil {
				startLine = int(comment.StartPoint().Row) + 1
				startByte = comment.StartByte()
			}
			endNode := declarationEnd(declNode)
			endLine := int(endNode.EndPoint().Row) + 1
			snippet := string(content[startByte:endNode.EndByte()])

			chunk := makeChunk(filePath, symbol, rule.kind, startLine, endLine, snippet)
			populateTreeMetadata(&chunk, declNode, content, startByte)
			chunk.Imports = imports
			filtered := chunk.References[:0]
			for _, ref := range chunk.References {
				if !shadowed[ref] {
					filtered = append(filtered, ref)
				}
			}
			chunk.References = filtered
			chunks = append(chunks, chunk)
		}
	}

	chunks = deduplicateByExactRange(chunks)
	return chunks, nil
}

// isCommentNode reports whether t is any comment node type.
// Tree-sitter grammars use "comment" (Python, TS, C, Ruby, PHP, C#),
// "line_comment" (Rust, Java), "block_comment" (Rust, Java) or
// "documentation_comment" (Dart) depending on the grammar.
func isCommentNode(t string) bool {
	return t == "comment" || t == "line_comment" || t == "block_comment" || t == "documentation_comment"
}

// isAttributeNode reports whether t is an attribute or annotation that grammars
// emit as a sibling preceding the declaration it applies to.
func isAttributeNode(t string) bool {
	return t == "attribute_item" || t == "annotation"
}

// isDocComment reports whether comment documents the declaration that follows
// it: Rust outer doc comments, Dart documentation comments, and /// or /**
// comments in C-family languages.
func isDocComment(comment *sitter.Node, content []byte) bool {
	if comment.Type() == "documentation_comment" || comment.ChildByFieldName("outer") != nil {
		return true
	}
	if content == nil {
		return false
	}
	text := comment.Content(content)
	return (strings.HasPrefix(text, "///") && !strings.HasPrefix(text, "////")) ||
		(strings.HasPrefix(text, "/**") && !strings.HasPrefix(text, "/***") && text != "/**/")
}

// maxLeadingCommentLines caps how many lines of plain (non-doc) comments attach
// to a declaration, so a license header written directly above the first
// declaration is not absorbed into it. Doc comments and attributes are bounded
// only by contiguity; oversized chunks are split downstream by token budget.
const maxLeadingCommentLines = 10

// findLeadingComments is leadingBlock without source text: only doc comments
// recognizable by node type escape the plain-comment cap.
func findLeadingComments(node *sitter.Node) *sitter.Node {
	return leadingBlock(node, nil)
}

// leadingBlock returns the earliest node of the comment and attribute block
// immediately preceding node with no blank line in between, or nil if there is
// none. Inner doc comments (Rust //!) document the enclosing module and end the
// block. content is the source used to recognize doc comments by their text.
func leadingBlock(node *sitter.Node, content []byte) *sitter.Node {
	var earliest *sitter.Node
	nextRow := node.StartPoint().Row
	plainLines := 0
	for sibling := node.PrevNamedSibling(); sibling != nil; sibling = sibling.PrevNamedSibling() {
		comment := isCommentNode(sibling.Type())
		if !comment && !isAttributeNode(sibling.Type()) {
			break
		}
		if sibling.ChildByFieldName("inner") != nil {
			break
		}
		// Adjacency check: the comment must immediately precede nextRow with no
		// blank line in between. Two cases:
		//
		// 1. Normal comments (// or #): EndPoint.Column > 0, meaning the end
		//    position is mid-line. The comment is adjacent when it ends on the
		//    line directly before nextRow: EndPoint.Row+1 == nextRow.
		//
		// 2. Rust /// doc comments: tree-sitter places their EndPoint at column 0
		//    of the following line (the end byte includes the trailing newline).
		//    In this case EndPoint.Row already equals nextRow when truly adjacent.
		//    A blank line pushes the declaration further so EndPoint.Row < nextRow.
		endRow := sibling.EndPoint().Row
		endCol := sibling.EndPoint().Column
		adjacent := endRow == nextRow || // doc-comment end-at-col-0 case
			(endCol != 0 && endRow+1 == nextRow) // normal: end mid-line
		if !adjacent {
			break
		}
		if comment && !isDocComment(sibling, content) {
			lines := int(endRow - sibling.StartPoint().Row)
			if endCol != 0 {
				lines++
			}
			if plainLines+lines > maxLeadingCommentLines {
				break
			}
			plainLines += lines
		}
		earliest = sibling
		nextRow = sibling.StartPoint().Row
	}
	return earliest
}

// insideFunction reports whether node is nested in a function, method or
// lambda body.
func insideFunction(node *sitter.Node) bool {
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch p.Type() {
		case "function_definition", "function_declaration", "function_expression",
			"generator_function", "generator_function_declaration", "arrow_function",
			"method_definition", "lambda", "function_item", "method", "singleton_method":
			return true
		}
	}
	return false
}

// declarationNode widens a matched node to the syntax node that owns the whole
// declaration: Python decorators, C++ template headers, and the Dart
// method_signature or declaration wrapping a signature.
func declarationNode(node *sitter.Node) *sitter.Node {
	p := node.Parent()
	if p == nil {
		return node
	}
	switch p.Type() {
	case "decorated_definition", "template_declaration":
		return p
	case "method_signature", "declaration":
		switch node.Type() {
		case "function_signature", "getter_signature", "setter_signature", "constructor_signature":
			return p
		}
	}
	return node
}

// declarationEnd returns the node where the declaration ends. Dart grammars
// emit a function's body as a function_body sibling following its signature.
func declarationEnd(node *sitter.Node) *sitter.Node {
	if next := node.NextNamedSibling(); next != nil && next.Type() == "function_body" {
		return next
	}
	return node
}

// declarationName returns the symbol text for a @name capture. C++ qualified
// names keep the existing convention of naming only the final component
// (Vec2::operator== -> operator==); member expressions drop layout whitespace.
func declarationName(name *sitter.Node, content []byte) string {
	for name.Type() == "qualified_identifier" {
		next := name.ChildByFieldName("name")
		if next == nil {
			break
		}
		name = next
	}
	text := name.Content(content)
	if name.Type() == "member_expression" {
		text = strings.Join(strings.Fields(text), "")
	}
	return text
}

// deduplicateByExactRange removes chunks with identical (StartLine, EndLine) ranges.
// When two chunks share the same range, the later one wins because
// queries are ordered from general to specific.
func deduplicateByExactRange(chunks []Chunk) []Chunk {
	if len(chunks) <= 1 {
		return chunks
	}
	type lineRange struct{ start, end int }
	best := make(map[lineRange]int, len(chunks))
	for i, c := range chunks {
		best[lineRange{c.StartLine, c.EndLine}] = i
	}
	seen := make(map[int]bool, len(best))
	for _, idx := range best {
		seen[idx] = true
	}
	result := make([]Chunk, 0, len(best))
	for i, c := range chunks {
		if seen[i] {
			result = append(result, c)
		}
	}
	return result
}

// findEnclosingSymbol walks up the AST from node and returns the name of the
// nearest enclosing named function or method, or "" if none is found.
func findEnclosingSymbol(node *sitter.Node, content []byte) string {
	current := node.Parent()
	for current != nil {
		switch current.Type() {
		case "function_declaration", "function_definition":
			if n := current.ChildByFieldName("name"); n != nil {
				return n.Content(content)
			}
		case "variable_declarator":
			if v := current.ChildByFieldName("value"); v != nil {
				switch v.Type() {
				case "arrow_function", "function_expression", "generator_function":
					if n := current.ChildByFieldName("name"); n != nil {
						return n.Content(content)
					}
				}
			}
		case "method_definition", "method_declaration":
			if n := current.ChildByFieldName("name"); n != nil {
				return n.Content(content)
			}
		case "method", "singleton_method", "class", "module":
			if n := current.ChildByFieldName("name"); n != nil {
				return n.Content(content)
			}
		case "field_declaration_list":
			// C++: members declared inside a class, struct or union body.
			if p := current.Parent(); p != nil {
				switch p.Type() {
				case "class_specifier", "struct_specifier", "union_specifier":
					if n := p.ChildByFieldName("name"); n != nil {
						return n.Content(content)
					}
				}
			}
		case "class_body", "extension_body":
			// Dart: the body node sits between the method and the enclosing type.
			// Only qualify for Dart-specific parent types to avoid affecting TS/JS/Java
			// which also use class_body but with class_declaration (not class_definition).
			if p := current.Parent(); p != nil {
				switch p.Type() {
				case "class_definition", "extension_declaration", "enum_declaration":
					if n := p.ChildByFieldName("name"); n != nil {
						return n.Content(content)
					}
				case "mixin_declaration":
					for i := 0; i < int(p.ChildCount()); i++ {
						if c := p.Child(i); c.Type() == "identifier" {
							return c.Content(content)
						}
					}
				}
			}
		}
		current = current.Parent()
	}
	return ""
}
