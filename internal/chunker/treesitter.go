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

	sitter "github.com/smacker/go-tree-sitter"
)

// QueryDef defines a tree-sitter query pattern and the chunk kind it produces.
// Pattern must have captures: @decl (full declaration) and @name (identifier).
type QueryDef struct {
	Pattern string
	Kind    string
}

// LanguageDef bundles a tree-sitter language with its query patterns.
type LanguageDef struct {
	Language *sitter.Language
	Queries  []QueryDef
}

type compiledRule struct {
	query *sitter.Query
	kind  string
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
			query: q,
			kind:  qd.Kind,
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

	var chunks []Chunk

	for _, rule := range c.rules {
		qc := sitter.NewQueryCursor()
		qc.Exec(rule.query, root)

		for {
			m, ok := qc.NextMatch()
			if !ok {
				break
			}

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

			symbol := nameNode.Content(content)
			if parent := findEnclosingSymbol(declNode, content); parent != "" {
				symbol = parent + "." + symbol
			}
			startLine := int(declNode.StartPoint().Row) + 1
			startByte := declNode.StartByte()
			if comment := findLeadingComments(declNode); comment != nil {
				startLine = int(comment.StartPoint().Row) + 1
				startByte = comment.StartByte()
			}
			endLine := int(declNode.EndPoint().Row) + 1
			snippet := string(content[startByte:declNode.EndByte()])

			chunks = append(chunks, makeChunk(filePath, symbol, rule.kind, startLine, endLine, snippet))
		}
	}

	chunks = deduplicateByExactRange(chunks)
	return chunks, nil
}

// isCommentNode reports whether t is any comment node type.
// Tree-sitter grammars use "comment" (Python, TS, C, Ruby, PHP, C#),
// "line_comment" (Rust, Java), or "block_comment" (Rust, Java) depending
// on the grammar.
func isCommentNode(t string) bool {
	return t == "comment" || t == "line_comment" || t == "block_comment"
}

const maxLeadingCommentLines = 10

// findLeadingComments returns the earliest comment node immediately preceding
// node with no blank line between them, or nil if none exist.
// At most maxLeadingCommentLines lines of comments are captured.
func findLeadingComments(node *sitter.Node) *sitter.Node {
	var earliest *sitter.Node
	nextRow := node.StartPoint().Row
	commentLines := 0
	for sibling := node.PrevNamedSibling(); sibling != nil; sibling = sibling.PrevNamedSibling() {
		if !isCommentNode(sibling.Type()) {
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
		siblingLines := int(sibling.EndPoint().Row-sibling.StartPoint().Row) + 1
		if commentLines+siblingLines > maxLeadingCommentLines {
			break
		}
		commentLines += siblingLines
		earliest = sibling
		nextRow = sibling.StartPoint().Row
	}
	return earliest
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
		}
		current = current.Parent()
	}
	return ""
}
