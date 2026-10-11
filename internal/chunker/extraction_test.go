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

package chunker_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/chunker"
)

// wantDecl describes one declaration a chunker must emit: the exact span and
// kind for symbol, plus substrings its content must include.
type wantDecl struct {
	symbol     string
	kind       string
	start, end int
	contains   []string
}

// chunkSource chunks src with the default chunker for path's extension and
// verifies every chunk's content matches its source line range.
func chunkSource(t *testing.T, path, src string) []chunker.Chunk {
	t.Helper()
	c, ok := chunker.DefaultLanguages(512)[filepath.Ext(path)]
	if !ok {
		t.Fatalf("no chunker for %s", path)
	}
	chunks, err := c.Chunk(path, []byte(src))
	if err != nil {
		t.Fatalf("Chunk(%s): %v", path, err)
	}
	lines := strings.Split(src, "\n")
	for _, ch := range chunks {
		if ch.StartLine < 1 || ch.EndLine < ch.StartLine || ch.EndLine > len(lines) {
			t.Fatalf("%s: invalid span %d-%d", ch.Symbol, ch.StartLine, ch.EndLine)
		}
		// Tree-sitter chunks are byte ranges: they may start after the first
		// line's indentation and end before trailing text on the last line.
		got := strings.Split(ch.Content, "\n")
		want := lines[ch.StartLine-1 : ch.EndLine]
		ok := len(got) == len(want) && strings.HasSuffix(want[0], got[0]) && strings.HasPrefix(want[len(want)-1], got[len(got)-1])
		if len(got) == 1 && len(want) == 1 {
			ok = strings.Contains(want[0], got[0])
		}
		for i := 1; ok && i < len(got)-1; i++ {
			ok = got[i] == want[i]
		}
		if !ok {
			t.Errorf("%s %d-%d: content does not match its source range:\n%q\nwant\n%q", ch.Symbol, ch.StartLine, ch.EndLine, ch.Content, strings.Join(want, "\n"))
		}
	}
	return chunks
}

func describeChunks(chunks []chunker.Chunk) string {
	var b strings.Builder
	for _, ch := range chunks {
		fmt.Fprintf(&b, "\n  %s %s %d-%d", ch.Kind, ch.Symbol, ch.StartLine, ch.EndLine)
	}
	return b.String()
}

// assertDecls checks that each wanted declaration is emitted exactly once with
// the wanted kind and span.
func assertDecls(t *testing.T, chunks []chunker.Chunk, wants []wantDecl) {
	t.Helper()
	for _, w := range wants {
		var matches []chunker.Chunk
		for _, ch := range chunks {
			if ch.Symbol == w.symbol {
				matches = append(matches, ch)
			}
		}
		if len(matches) != 1 {
			t.Errorf("symbol %q: got %d chunks, want 1; chunks:%s", w.symbol, len(matches), describeChunks(chunks))
			continue
		}
		ch := matches[0]
		if ch.Kind != w.kind || ch.StartLine != w.start || ch.EndLine != w.end {
			t.Errorf("symbol %q: got %s %d-%d, want %s %d-%d", w.symbol, ch.Kind, ch.StartLine, ch.EndLine, w.kind, w.start, w.end)
		}
		for _, s := range w.contains {
			if !strings.Contains(ch.Content, s) {
				t.Errorf("symbol %q: content missing %q:\n%s", w.symbol, s, ch.Content)
			}
		}
	}
}

// assertNoSymbol checks that no chunk is emitted for any of symbols.
func assertNoSymbol(t *testing.T, chunks []chunker.Chunk, symbols ...string) {
	t.Helper()
	for _, s := range symbols {
		for _, ch := range chunks {
			if ch.Symbol == s {
				t.Errorf("unexpected chunk %s %s %d-%d", ch.Kind, ch.Symbol, ch.StartLine, ch.EndLine)
			}
		}
	}
}
