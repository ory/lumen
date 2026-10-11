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
	"slices"
	"strings"
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

// assertMarkdownExactLines checks that each chunk's Content equals the exact source
// lines of its 1-based inclusive span.
func assertMarkdownExactLines(t *testing.T, src string, chunks []Chunk) {
	t.Helper()
	lines := strings.Split(src, "\n")
	for _, ch := range chunks {
		if ch.StartLine < 1 || ch.EndLine < ch.StartLine || ch.EndLine > len(lines) {
			t.Errorf("chunk %q has invalid span %d-%d", ch.Symbol, ch.StartLine, ch.EndLine)
			continue
		}
		want := strings.Join(lines[ch.StartLine-1:ch.EndLine], "\n")
		if ch.Content != want {
			t.Errorf("chunk %q content does not match lines %d-%d:\n got %q\nwant %q", ch.Symbol, ch.StartLine, ch.EndLine, ch.Content, want)
		}
	}
}

func markdownSymbols(chunks []Chunk) []string {
	out := make([]string, len(chunks))
	for i, ch := range chunks {
		out[i] = ch.Symbol
	}
	return out
}

func TestMarkdownChunker_FencedCommentIsNotHeading(t *testing.T) {
	src := "# Tool\n\n## Installation\n\nInstall it.\n\n```sh\n# install tool production\nnpm install tool\n```\n\nDone.\n\n## Usage\n\nRun it.\n"
	chunks, err := NewMarkdownChunker().Chunk("docs/tool.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got := markdownSymbols(chunks)
	want := []string{"Tool", "Installation", "Usage"}
	if !slices.Equal(got, want) {
		t.Fatalf("symbols = %q, want %q", got, want)
	}
	inst := chunks[1]
	if !strings.Contains(inst.Content, "# install tool production") || !strings.Contains(inst.Content, "Done.") {
		t.Errorf("Installation section lost its fenced block: %q", inst.Content)
	}
	if inst.StartLine != 3 || inst.EndLine != 12 {
		t.Errorf("Installation span = %d-%d, want 3-12", inst.StartLine, inst.EndLine)
	}
	assertMarkdownExactLines(t, src, chunks)
}

func TestMarkdownChunker_TildeAndLongFences(t *testing.T) {
	src := "# A\n\n~~~python\n# not a heading\n```\n# still not a heading\n~~~\n\n# B\n\n````md\n```\n# nested fence content\n```\n````\n\n# C\n"
	chunks, err := NewMarkdownChunker().Chunk("x.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got := markdownSymbols(chunks)
	want := []string{"A", "B", "C"}
	if !slices.Equal(got, want) {
		t.Fatalf("symbols = %q, want %q", got, want)
	}
	assertMarkdownExactLines(t, src, chunks)
}

func TestMarkdownChunker_UnclosedFenceRunsToEOF(t *testing.T) {
	src := "# Setup\n\n```bash\n# configure\nmake\n\n# Build\nmake all\n"
	chunks, err := NewMarkdownChunker().Chunk("x.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got := markdownSymbols(chunks)
	if !slices.Equal(got, []string{"Setup"}) {
		t.Fatalf("symbols = %q, want [Setup]", got)
	}
	if !strings.Contains(chunks[0].Content, "make all") {
		t.Errorf("unclosed fence content missing: %q", chunks[0].Content)
	}
	assertMarkdownExactLines(t, src, chunks)
}

func TestMarkdownChunker_AllHeadingLevels(t *testing.T) {
	src := "# One\na\n## Two\nb\n### Three\nc\n#### Four\nd\n##### Five ####\ne\n###### Six\nf\n####### Seven is text\ng\n#NoSpace is text\n"
	chunks, err := NewMarkdownChunker().Chunk("x.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got := markdownSymbols(chunks)
	want := []string{"One", "Two", "Three", "Four", "Five", "Six"}
	if !slices.Equal(got, want) {
		t.Fatalf("symbols = %q, want %q", got, want)
	}
	if !strings.Contains(chunks[5].Content, "#NoSpace is text") || !strings.Contains(chunks[5].Content, "####### Seven") {
		t.Errorf("non-headings should stay in the H6 section: %q", chunks[5].Content)
	}
	assertMarkdownExactLines(t, src, chunks)
}

func TestMarkdownChunker_MDX(t *testing.T) {
	src := "import { Chart } from './chart'\n\nexport const meta = { title: 'Docs' }\n\n# Charts\n\n<Chart data={data} />\n\n```jsx\n# not heading\n```\n"
	chunks, err := NewMarkdownChunker().Chunk("docs/charts.mdx", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	got := markdownSymbols(chunks)
	if !slices.Equal(got, []string{"preamble", "Charts"}) {
		t.Fatalf("symbols = %q, want [preamble Charts]", got)
	}
	assertMarkdownExactLines(t, src, chunks)
}
