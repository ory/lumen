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

	sitter_svelte "github.com/alexaandru/go-sitter-forest/svelte"
	sitter "github.com/smacker/go-tree-sitter"
)

// SvelteChunker implements Chunker for .svelte files using a two-phase parse:
//  1. The outer Svelte grammar locates <script> elements.
//  2. Each script block's raw text is re-parsed with the TypeScript chunker
//     to extract named symbols (functions, classes, interfaces, etc.).
//
// Template syntax ({#if}, {#each}, bind:) lives in the outer parse and is
// intentionally not queried — only script-block symbols are indexed.
// $state()/$derived() parse fine as TypeScript call-expression initializers.
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

// Chunk parses a .svelte file and returns chunks for all named symbols found
// inside <script> blocks. Line numbers are file-relative (1-based).
func (s *SvelteChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	root, err := sitter.ParseCtx(context.Background(), content, s.svelteLanguage)
	if err != nil {
		return nil, fmt.Errorf("parse svelte %s: %w", filePath, err)
	}

	var chunks []Chunk
	var firstErr error
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "script_element" {
			sc, err := s.chunkScriptElement(filePath, content, node)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("chunk script element in %s: %w", filePath, err)
				}
				return
			}
			chunks = append(chunks, sc...)
			// Do not recurse further; script content is handled by the TS parser.
			return
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(root)

	return chunks, firstErr
}

// chunkScriptElement finds the raw_text child of a script_element, re-parses
// it with TypeScript, and adjusts all chunk line numbers to be file-relative.
func (s *SvelteChunker) chunkScriptElement(filePath string, content []byte, node *sitter.Node) ([]Chunk, error) {
	if s.tsChunker == nil {
		return nil, fmt.Errorf("chunk svelte %s: nil TypeScript chunker", filePath)
	}

	rawText := findRawText(node)
	if rawText == nil {
		return nil, nil
	}

	scriptContent := content[rawText.StartByte():rawText.EndByte()]
	// lineOffset is the 0-based row of the raw_text node in the file.
	// The TypeScript chunker produces 1-based lines relative to scriptContent,
	// so file_line = ts_line + lineOffset.
	lineOffset := int(rawText.StartPoint().Row)

	inner, err := s.tsChunker.Chunk(filePath, scriptContent)
	if err != nil {
		return nil, err
	}

	for i := range inner {
		inner[i].StartLine += lineOffset
		inner[i].EndLine += lineOffset
	}

	return inner, nil
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
