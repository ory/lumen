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
	"testing"

	"github.com/ory/lumen/internal/chunker"
)

// sampleSvelte is a representative SvelteKit component with:
//   - a <script lang="ts"> block containing TS symbols
//   - Svelte-specific rune syntax ($state, $derived) that parses as TS call expressions
//   - a template section that must NOT produce chunks
var sampleSvelte = []byte(`<script lang="ts">
	import { onMount } from 'svelte';

	export let title: string = 'Hello';

	interface Props {
		name: string;
	}

	function greet(name: string): string {
		return ` + "`" + `Hello, ${name}!` + "`" + `;
	}

	const count = $state(0);
	const doubled = $derived(count * 2);

	class Counter {
		increment() { return 1; }
	}
</script>

<h1>{title}</h1>
<p>{greet('world')}</p>
`)

func TestSvelteChunker_ScriptSymbols(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".svelte"]
	if !ok {
		t.Fatal("no chunker registered for .svelte")
	}

	chunks, err := c.Chunk("src/routes/+page.svelte", sampleSvelte)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbol := make(map[string]chunker.Chunk)
	for _, ch := range chunks {
		bySymbol[ch.Symbol] = ch
	}

	// Functions and types from the TS script block
	checkChunk(t, bySymbol, "greet", "function", 0, 0, "src/routes/+page.svelte", "function greet")
	checkChunk(t, bySymbol, "Props", "interface", 0, 0, "src/routes/+page.svelte", "interface Props")
	checkChunk(t, bySymbol, "Counter", "type", 0, 0, "src/routes/+page.svelte", "class Counter")

	// Line numbers must be file-relative (1-based), not script-block-relative.
	// greet is on line 10 of the file (first line of script is line 1, content
	// starts on line 2, greet starts on the 10th line of the file).
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk from script symbols")
	}
	if chunks[0].StartLine < 2 {
		t.Errorf("expected file-relative line numbers (>= 2), got StartLine=%d for first chunk", chunks[0].StartLine)
	}
}

func TestSvelteChunker_NoSymbolsCases(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".svelte"]

	cases := []struct {
		name    string
		path    string
		content []byte
	}{
		{
			name: "empty script block",
			path: "empty.svelte",
			content: []byte(`<script>
</script>
<p>hello</p>
`),
		},
		{
			name: "no script block",
			path: "static.svelte",
			content: []byte(`<h1>Static page</h1>
<p>No script block here.</p>
`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := c.Chunk(tc.path, tc.content)
			if err != nil {
				t.Fatalf("Chunk: %v", err)
			}
			if len(chunks) != 0 {
				t.Errorf("expected 0 chunks, got %d", len(chunks))
			}
		})
	}
}
