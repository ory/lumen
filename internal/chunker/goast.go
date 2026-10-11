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
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	pathpkg "path"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// GoAST is a Chunker that uses go/ast to split Go source files into
// semantically meaningful chunks (functions, methods, types, interfaces,
// consts, vars, and the package declaration).
type GoAST struct{}

// NewGoAST returns a new GoAST chunker.
func NewGoAST() *GoAST {
	return &GoAST{}
}

// Chunk parses a Go source file and returns one Chunk per top-level
// declaration, including the package doc chunk.
func (g *GoAST) Chunk(filePath string, content []byte) ([]Chunk, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	var chunks []Chunk

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			chunks = append(chunks, chunkFuncDecl(fset, filePath, content, d))
		case *ast.GenDecl:
			chunks = append(chunks, chunkGenDecl(fset, filePath, content, d)...)
		}
	}

	var imports []string
	aliases := map[string]bool{}
	for _, spec := range file.Imports {
		if path, err := strconv.Unquote(spec.Path.Value); err == nil {
			imports = append(imports, path)
			alias := assumedPackageName(path)
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			if alias != "." && alias != "_" && alias != "" {
				aliases[alias] = true
				imports = append(imports, alias+"="+path)
			}
		}
	}
	imports = uniqueNames(imports)

	// Collect package-qualified calls and comments once, in source order, then
	// assign them to chunks by line range. Rescanning the file per chunk is
	// quadratic in the number of declarations.
	type span struct {
		start, end int
		text       string
	}
	var calls []span
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		base, ok := selector.X.(*ast.Ident)
		if ok && base.Obj == nil && aliases[base.Name] {
			calls = append(calls, span{fset.Position(call.Pos()).Line, fset.Position(call.End()).Line, base.Name + "." + selector.Sel.Name})
		}
		return true
	})
	slices.SortStableFunc(calls, func(a, b span) int { return a.start - b.start })
	comments := make([]span, 0, len(file.Comments))
	for _, comment := range file.Comments {
		comments = append(comments, span{fset.Position(comment.Pos()).Line, fset.Position(comment.End()).Line, comment.Text()})
	}
	within := func(spans []span, startLine, endLine int) []string {
		var out []string
		i, _ := slices.BinarySearchFunc(spans, startLine, func(s span, line int) int { return s.start - line })
		for ; i < len(spans) && spans[i].start <= endLine; i++ {
			if spans[i].end <= endLine {
				out = append(out, spans[i].text)
			}
		}
		return out
	}
	for i := range chunks {
		chunks[i].Exported = ast.IsExported(goDeclName(chunks[i].Symbol))
		chunks[i].Imports = imports
		chunks[i].References = uniqueNames(append(chunks[i].References, within(calls, chunks[i].StartLine, chunks[i].EndLine)...))
		chunks[i].Comments = strings.TrimSpace(strings.Join(within(comments, chunks[i].StartLine, chunks[i].EndLine), "\n"))
	}
	return chunks, nil
}

func chunkFuncDecl(fset *token.FileSet, filePath string, content []byte, d *ast.FuncDecl) Chunk {
	kind := "function"
	symbol := d.Name.Name

	if d.Recv != nil && len(d.Recv.List) > 0 {
		kind = "method"
		recvType := receiverTypeName(d.Recv.List[0].Type)
		symbol = recvType + "." + d.Name.Name
	}

	start, end := declRange(fset, d.Doc, d.Pos(), d.End())
	c := makeChunk(filePath, symbol, kind, start.Line, end.Line,
		sliceContent(content, start.Offset, end.Offset))
	endSignature := d.End()
	if d.Body != nil {
		endSignature = d.Body.Pos()
	}
	c.Signature = strings.TrimSpace(sliceContent(content, fset.Position(d.Pos()).Offset, fset.Position(endSignature).Offset))
	c.References = goReferences(d)
	return c
}

func chunkGenDecl(fset *token.FileSet, filePath string, content []byte, d *ast.GenDecl) []Chunk {
	var chunks []Chunk

	// Skip import declarations.
	if d.Tok == token.IMPORT {
		return nil
	}

	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			chunks = append(chunks, chunkTypeSpec(fset, filePath, content, d, s))

		case *ast.ValueSpec:
			chunks = append(chunks, chunkValueSpec(fset, filePath, content, d, s))
		}
	}

	return chunks
}

func chunkTypeSpec(fset *token.FileSet, filePath string, content []byte, d *ast.GenDecl, s *ast.TypeSpec) Chunk {
	kind := "type"
	if _, ok := s.Type.(*ast.InterfaceType); ok {
		kind = "interface"
	}
	doc := s.Doc
	if doc == nil {
		doc = d.Doc
	}
	var start, end token.Position
	if len(d.Specs) == 1 {
		start, end = declRange(fset, doc, d.Pos(), d.End())
	} else {
		start, end = declRange(fset, s.Doc, s.Pos(), s.End())
	}
	c := makeChunk(filePath, s.Name.Name, kind, start.Line, end.Line,
		sliceContent(content, start.Offset, end.Offset))
	c.Signature = "type " + s.Name.Name
	c.References = goReferences(s)
	return c
}

func chunkValueSpec(fset *token.FileSet, filePath string, content []byte, d *ast.GenDecl, s *ast.ValueSpec) Chunk {
	kind := "var"
	if d.Tok == token.CONST {
		kind = "const"
	}
	doc := s.Doc
	if doc == nil {
		doc = d.Doc
	}
	var start, end token.Position
	if len(d.Specs) == 1 {
		start, end = declRange(fset, doc, d.Pos(), d.End())
	} else {
		start, end = declRange(fset, s.Doc, s.Pos(), s.End())
	}
	c := makeChunk(filePath, s.Names[0].Name, kind, start.Line, end.Line,
		sliceContent(content, start.Offset, end.Offset))
	c.Signature = kind + " " + s.Names[0].Name
	if s.Type != nil {
		c.Signature += " " + sliceContent(content, fset.Position(s.Type.Pos()).Offset, fset.Position(s.Type.End()).Offset)
	}
	c.References = goReferences(s)
	return c
}

func declRange(fset *token.FileSet, doc *ast.CommentGroup, pos, end token.Pos) (token.Position, token.Position) {
	startPos := fset.Position(pos)
	if doc != nil {
		startPos = fset.Position(doc.Pos())
	}
	endPos := fset.Position(end)
	return startPos, endPos
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	}
	return "unknown"
}

func sliceContent(content []byte, startOffset, endOffset int) string {
	if startOffset < 0 {
		startOffset = 0
	}
	if endOffset > len(content) {
		endOffset = len(content)
	}
	return string(content[startOffset:endOffset])
}

func makeChunk(filePath, symbol, kind string, startLine, endLine int, content string) Chunk {
	// Hash filePath+content so chunks are unique even in minified files where
	// multiple symbols share the same line number.
	h := sha256.New()
	h.Write([]byte(filePath))
	h.Write([]byte{':'})
	h.Write([]byte(content))
	id := fmt.Sprintf("%x", h.Sum(nil))[:16]
	return Chunk{
		ID:        id,
		FilePath:  filePath,
		Symbol:    symbol,
		Kind:      kind,
		StartLine: startLine,
		EndLine:   endLine,
		Content:   content,
	}
}

// goReferences deliberately excludes selector calls and locally bound variables:
// resolving these requires type information, not just matching their spelling.
func goReferences(node ast.Node) []string {
	var refs []string
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if ok && (id.Obj == nil || id.Obj.Kind == ast.Fun) {
			refs = append(refs, id.Name)
		}
		return true
	})
	return uniqueNames(refs)
}

// goDeclName returns the declared identifier of a symbol. Methods are recorded
// as "Type.Method"; their export status depends on the method name only.
func goDeclName(symbol string) string {
	if dot := strings.LastIndexByte(symbol, '.'); dot >= 0 {
		return symbol[dot+1:]
	}
	return symbol
}

// assumedPackageName guesses the package name of an unaliased import the same
// way goimports does: skip a major-version suffix, drop a "go-" prefix and cut
// at the first character that cannot appear in an identifier.
func assumedPackageName(importPath string) string {
	base := pathpkg.Base(importPath)
	if strings.HasPrefix(base, "v") {
		if _, err := strconv.Atoi(base[1:]); err == nil {
			if dir := pathpkg.Dir(importPath); dir != "." {
				base = pathpkg.Base(dir)
			}
		}
	}
	base = strings.TrimPrefix(base, "go-")
	if i := strings.IndexFunc(base, func(r rune) bool { return r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) }); i >= 0 {
		base = base[:i]
	}
	return base
}
