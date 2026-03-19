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
	"testing"
)

func TestMarkdownChunker_Basic(t *testing.T) {
	src := `# Introduction
This is intro text.

## Installation

Run this command.

### Advanced
More details.
`
	c := NewMarkdownChunker()
	chunks, err := c.Chunk("docs/readme.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if chunks[0].Symbol != "Introduction" {
		t.Errorf("chunk[0].Symbol = %q, want Introduction", chunks[0].Symbol)
	}
	if chunks[0].Kind != "section" {
		t.Errorf("chunk[0].Kind = %q, want section", chunks[0].Kind)
	}
	if chunks[1].Symbol != "Installation" {
		t.Errorf("chunk[1].Symbol = %q, want Installation", chunks[1].Symbol)
	}
	if chunks[2].Symbol != "Advanced" {
		t.Errorf("chunk[2].Symbol = %q, want Advanced", chunks[2].Symbol)
	}
}

func TestMarkdownChunker_Empty(t *testing.T) {
	c := NewMarkdownChunker()
	chunks, err := c.Chunk("empty.md", []byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(chunks))
	}
}

func TestMarkdownChunker_NoHeadings(t *testing.T) {
	src := `Just some plain text
without any headings.
`
	c := NewMarkdownChunker()
	chunks, err := c.Chunk("notes.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	// Preamble chunk
	if len(chunks) != 1 {
		t.Fatalf("expected 1 preamble chunk, got %d", len(chunks))
	}
	if chunks[0].Symbol != "preamble" {
		t.Errorf("chunk[0].Symbol = %q, want preamble", chunks[0].Symbol)
	}
}
