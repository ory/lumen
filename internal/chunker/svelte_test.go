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
	"slices"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/chunker"
)

// sampleSvelte is a representative SvelteKit component with:
//   - a <script lang="ts"> block containing TS symbols
//   - Svelte-specific rune syntax ($state, $derived) that parses as TS call expressions
//   - a template section that produces template regions
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

func TestSvelteChunker_NoChunkCases(t *testing.T) {
	c := chunker.DefaultLanguages(512)[".svelte"]

	for name, content := range map[string]string{
		"empty file":        "",
		"whitespace only":   "\n\n  \n",
		"empty script only": "<script>\n</script>\n",
	} {
		t.Run(name, func(t *testing.T) {
			chunks, err := c.Chunk("empty.svelte", []byte(content))
			if err != nil {
				t.Fatalf("Chunk: %v", err)
			}
			if len(chunks) != 0 {
				t.Errorf("expected 0 chunks, got %d: %v", len(chunks), svelteSummary(chunks))
			}
		})
	}
}

// svelteSummary renders chunks as "symbol(kind)@start-end" for failure output.
func svelteSummary(chunks []chunker.Chunk) []string {
	out := make([]string, len(chunks))
	for i, ch := range chunks {
		out[i] = fmt.Sprintf("%s(%s)@%d-%d", ch.Symbol, ch.Kind, ch.StartLine, ch.EndLine)
	}
	return out
}

func chunkSvelte(t *testing.T, path, src string) []chunker.Chunk {
	t.Helper()
	chunks, err := chunker.DefaultLanguages(512)[".svelte"].Chunk(path, []byte(src))
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}
	assertSvelteChunksValid(t, src, chunks)
	return chunks
}

// assertSvelteChunksValid checks file-relative spans (content lies within
// its lines, so hydration from the file reproduces it) and unique IDs.
func assertSvelteChunksValid(t *testing.T, src string, chunks []chunker.Chunk) {
	t.Helper()
	lines := strings.Split(src, "\n")
	ids := map[string]string{}
	for _, ch := range chunks {
		if ch.StartLine < 1 || ch.EndLine < ch.StartLine || ch.EndLine > len(lines) {
			t.Errorf("%s: invalid span %d-%d", ch.Symbol, ch.StartLine, ch.EndLine)
			continue
		}
		span := strings.Join(lines[ch.StartLine-1:ch.EndLine], "\n")
		if !strings.Contains(span, ch.Content) {
			t.Errorf("%s: content is not within lines %d-%d:\n%q", ch.Symbol, ch.StartLine, ch.EndLine, ch.Content)
		}
		// Regions own whole lines unless another construct shares the line.
		if idx := strings.Index(span, ch.Content); idx > 0 && (ch.Kind == "template" || ch.Kind == "style" || ch.Kind == "statement") &&
			strings.TrimSpace(span[:idx]) == "" {
			t.Errorf("%s: region does not start at its line start", ch.Symbol)
		}
		if prev, dup := ids[ch.ID]; dup {
			t.Errorf("duplicate chunk ID %s for %s and %s", ch.ID, prev, ch.Symbol)
		}
		ids[ch.ID] = ch.Symbol
	}
}

func svelteBySymbol(t *testing.T, chunks []chunker.Chunk, symbol string) chunker.Chunk {
	t.Helper()
	for _, ch := range chunks {
		if ch.Symbol == symbol {
			return ch
		}
	}
	t.Fatalf("no chunk %q in %v", symbol, svelteSummary(chunks))
	return chunker.Chunk{}
}

func TestSvelteChunker_TemplateOnly(t *testing.T) {
	src := "<h1>Static page</h1>\n\n<p>No script block here.</p>\n"
	chunks := chunkSvelte(t, "static.svelte", src)
	got := svelteSummary(chunks)
	want := []string{"template[1](template)@1-1", "template[2](template)@3-3"}
	if !slices.Equal(got, want) {
		t.Fatalf("chunks = %v, want %v", got, want)
	}
	if chunks[1].Content != "<p>No script block here.</p>" {
		t.Errorf("template[2] content = %q", chunks[1].Content)
	}
}

func TestSvelteChunker_StyleOnly(t *testing.T) {
	src := "<style lang=\"postcss\">\n\t.invite-error {\n\t\tborder-left: 3px solid red;\n\t}\n</style>\n"
	chunks := chunkSvelte(t, "Style.svelte", src)
	got := svelteSummary(chunks)
	if !slices.Equal(got, []string{"style[1](style)@1-5"}) {
		t.Fatalf("chunks = %v", got)
	}
	if want := strings.TrimSuffix(src, "\n"); chunks[0].Content != want {
		t.Errorf("style content = %q, want %q (attributes and delimiters included)", chunks[0].Content, want)
	}
}

const svelteDropdown = `<script lang="ts">
	import { onMount } from "svelte";
	import { clickOutside } from "$lib/actions/clickOutside";

	let { label }: { label: string } = $props();

	let open = $state(false);
	const limit = 10;

	function close() {
		open = false;
	}

	// Close on Escape while open.
	$effect(() => {
		if (!open) return;
		const onKeydown = (event: KeyboardEvent) => {
			if (event.key === "Escape") close();
		};
		window.addEventListener("keydown", onKeydown);
		return () => window.removeEventListener("keydown", onKeydown);
	});

	onMount(() => {
		console.log("mounted");
	});

	$: doubled = limit * 2;
</script>

<!-- Dropdown root -->
<div class="dropdown" use:clickOutside={close}>
	<button onclick={() => (open = !open)}>{label}</button>
	{#if open}
		<ul role="menu"><li>One</li></ul>
	{/if}
</div>

{#if open}
	<p>Open</p>
{/if}

<style>
	.dropdown {
		position: relative;
	}
</style>
`

func TestSvelteChunker_ScriptStatementsAndState(t *testing.T) {
	chunks := chunkSvelte(t, "Dropdown.svelte", svelteDropdown)

	effect := svelteBySymbol(t, chunks, "script[1].statement[2]")
	if effect.Kind != "statement" || effect.StartLine != 14 || effect.EndLine != 22 {
		t.Errorf("effect = %s(%s)@%d-%d, want statement@14-22", effect.Symbol, effect.Kind, effect.StartLine, effect.EndLine)
	}
	for _, want := range []string{"// Close on Escape", `window.addEventListener("keydown", onKeydown);`, `return () => window.removeEventListener("keydown", onKeydown);`} {
		if !strings.Contains(effect.Content, want) {
			t.Errorf("effect statement missing %q", want)
		}
	}

	props := svelteBySymbol(t, chunks, "script[1].statement[1]")
	if !strings.Contains(props.Content, "$props()") {
		t.Errorf("statement[1] = %q, want the $props destructuring", props.Content)
	}
	mount := svelteBySymbol(t, chunks, "script[1].statement[3]")
	if !strings.Contains(mount.Content, "onMount(() => {") || mount.StartLine != 24 || mount.EndLine != 26 {
		t.Errorf("statement[3] = %q@%d-%d, want onMount@24-26", mount.Content, mount.StartLine, mount.EndLine)
	}
	reactive := svelteBySymbol(t, chunks, "script[1].statement[4]")
	if reactive.Content != "\t$: doubled = limit * 2;" {
		t.Errorf("statement[4] = %q, want the $: label", reactive.Content)
	}

	open := svelteBySymbol(t, chunks, "open")
	if open.Kind != "var" || open.Content != "\tlet open = $state(false);" || open.StartLine != 7 {
		t.Errorf("open = %s@%d %q", open.Kind, open.StartLine, open.Content)
	}
	limit := svelteBySymbol(t, chunks, "limit")
	if limit.Kind != "const" {
		t.Errorf("limit.Kind = %q, want const", limit.Kind)
	}
	if closeFn := svelteBySymbol(t, chunks, "close"); closeFn.Kind != "function" {
		t.Errorf("close.Kind = %q", closeFn.Kind)
	}

	for _, ch := range chunks {
		if ch.Kind == "statement" && (strings.Contains(ch.Content, "function close") || strings.Contains(ch.Content, "import ")) {
			t.Errorf("%s duplicates a covered declaration or import: %q", ch.Symbol, ch.Content)
		}
		if strings.Contains(ch.Content, "<script") || strings.Contains(ch.Content, "</script>") {
			t.Errorf("%s(%s) wraps the whole script element", ch.Symbol, ch.Kind)
		}
	}
	if n := len(effect.Imports); n == 0 {
		t.Errorf("statement chunks should carry script import metadata")
	}
}

func TestSvelteChunker_TemplateAndStyleRegions(t *testing.T) {
	chunks := chunkSvelte(t, "Dropdown.svelte", svelteDropdown)

	var regions []string
	for _, ch := range chunks {
		if ch.Kind == "template" || ch.Kind == "style" {
			regions = append(regions, fmt.Sprintf("%s(%s)@%d-%d", ch.Symbol, ch.Kind, ch.StartLine, ch.EndLine))
		}
	}
	want := []string{"template[1](template)@31-37", "template[2](template)@39-41", "style[1](style)@43-47"}
	if !slices.Equal(regions, want) {
		t.Fatalf("regions = %v, want %v", regions, want)
	}
	root := svelteBySymbol(t, chunks, "template[1]")
	for _, want := range []string{"<!-- Dropdown root -->", "use:clickOutside={close}", "onclick={() => (open = !open)}", "<li>One</li>"} {
		if !strings.Contains(root.Content, want) {
			t.Errorf("template[1] missing %q", want)
		}
	}
}

func TestSvelteChunker_ModuleAndInstanceScripts(t *testing.T) {
	for _, attr := range []string{`context="module"`, `module`} {
		t.Run(attr, func(t *testing.T) {
			src := "<script " + attr + " lang=\"ts\">\n\texport const prerender = true;\n\tregisterRoute(\"/x\");\n</script>\n\n<script lang=\"ts\">\n\tlet count = $state(0);\n\t$effect(() => console.log(count));\n\tfunction inc() {\n\t\tcount++;\n\t}\n</script>\n\n<button onclick={inc}>{count}</button>\n"
			chunks := chunkSvelte(t, "Counter.svelte", src)
			got := svelteSummary(chunks)
			for _, want := range []string{
				"prerender(const)@2-2",
				"script[1].statement[1](statement)@3-3",
				"count(var)@7-7",
				"script[2].statement[1](statement)@8-8",
				"inc(function)@9-11",
				"template[1](template)@14-14",
			} {
				if !slices.Contains(got, want) {
					t.Errorf("missing %s in %v", want, got)
				}
			}
		})
	}
}

func TestSvelteChunker_SameLineNodes(t *testing.T) {
	src := "<script>\n\tlet a = $state(0); $effect(() => console.log(a));\n</script>\n<h1>A</h1><p>B</p>\n<br/><br/>\n<div>C</div>\n"
	chunks := chunkSvelte(t, "Same.svelte", src)
	got := svelteSummary(chunks)
	want := []string{
		"a(var)@2-2",
		"script[1].statement[1](statement)@2-2",
		"template[1](template)@4-4",
		"template[2](template)@5-5",
		"template[3](template)@6-6",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("chunks = %v, want %v", got, want)
	}
	if a := svelteBySymbol(t, chunks, "a"); a.Content != "\tlet a = $state(0);" {
		t.Errorf("a content = %q", a.Content)
	}
	if st := svelteBySymbol(t, chunks, "script[1].statement[1]"); st.Content != "$effect(() => console.log(a));" {
		t.Errorf("statement content = %q", st.Content)
	}
	if tpl := svelteBySymbol(t, chunks, "template[1]"); tpl.Content != "<h1>A</h1><p>B</p>" {
		t.Errorf("same-line template children must form one region, got %q", tpl.Content)
	}
}

func TestSvelteChunker_LargeComponentNoWrapperDuplication(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("<script lang=\"ts\">\n")
	for i := range 40 {
		fmt.Fprintf(&sb, "\tfunction handler%d(e: Event) {\n\t\tconsole.log(%d, e);\n\t}\n", i, i)
	}
	sb.WriteString("</script>\n\n<section class=\"page\">\n")
	for i := range 200 {
		fmt.Fprintf(&sb, "\t<div class=\"row\">\n\t\t<button onclick={handler%d}>Row %d</button>\n\t\t{#if rows[%d]}<span>{rows[%d]}</span>{/if}\n\t</div>\n", i%40, i, i, i)
	}
	sb.WriteString("</section>\n")
	src := sb.String()

	chunks := chunkSvelte(t, "Large.svelte", src)
	var templates, functions int
	for _, ch := range chunks {
		switch ch.Kind {
		case "template":
			templates++
			if !strings.HasPrefix(ch.Content, "<section") || !strings.HasSuffix(ch.Content, "</section>") {
				t.Errorf("template region should be the whole top-level element, got %d-%d", ch.StartLine, ch.EndLine)
			}
		case "function":
			functions++
			if strings.Count(ch.Content, "\n") != 2 {
				t.Errorf("%s not retained whole: %q", ch.Symbol, ch.Content)
			}
		case "statement":
			t.Errorf("unexpected statement %s", ch.Symbol)
		}
	}
	if templates != 1 {
		t.Errorf("template regions = %d, want 1 (nested elements are not re-emitted)", templates)
	}
	if functions != 40 {
		t.Errorf("functions = %d, want 40", functions)
	}
}
