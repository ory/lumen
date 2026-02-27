package chunker_test

import (
	"testing"

	sitter_py "github.com/smacker/go-tree-sitter/python"
	sitter_rs "github.com/smacker/go-tree-sitter/rust"
	sitter_ts "github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/foobar/agent-index-go/internal/chunker"
)

var samplePython = []byte(`def greet(name):
    """Say hello."""
    return f"Hello, {name}"

class Animal:
    def speak(self):
        pass
`)

func TestTreeSitterChunker_Python(t *testing.T) {
	def := chunker.LanguageDef{
		Language: sitter_py.GetLanguage(),
		Queries: []chunker.QueryDef{
			{Pattern: `(function_definition name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(class_definition name: (identifier) @name) @decl`, Kind: "type"},
		},
	}
	c, err := chunker.NewTreeSitterChunker(def)
	if err != nil {
		t.Fatalf("NewTreeSitterChunker: %v", err)
	}

	chunks, err := c.Chunk("sample.py", samplePython)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbol := make(map[string]chunker.Chunk)
	for _, ch := range chunks {
		bySymbol[ch.Symbol] = ch
	}

	greet, ok := bySymbol["greet"]
	if !ok {
		t.Fatalf("expected chunk 'greet', got symbols: %v", symbolNames(chunks))
	}
	if greet.Kind != "function" {
		t.Errorf("greet.Kind = %q, want %q", greet.Kind, "function")
	}
	if greet.StartLine != 1 {
		t.Errorf("greet.StartLine = %d, want 1", greet.StartLine)
	}
	if greet.FilePath != "sample.py" {
		t.Errorf("greet.FilePath = %q", greet.FilePath)
	}
	if greet.ID == "" {
		t.Error("greet.ID is empty")
	}

	animal, ok := bySymbol["Animal"]
	if !ok {
		t.Fatalf("expected chunk 'Animal', got symbols: %v", symbolNames(chunks))
	}
	if animal.Kind != "type" {
		t.Errorf("animal.Kind = %q, want %q", animal.Kind, "type")
	}
	if animal.StartLine != 5 {
		t.Errorf("animal.StartLine = %d, want 5", animal.StartLine)
	}

	speak, ok := bySymbol["speak"]
	if !ok {
		t.Fatalf("expected chunk 'speak', got symbols: %v", symbolNames(chunks))
	}
	if speak.Kind != "function" {
		t.Errorf("speak.Kind = %q, want %q", speak.Kind, "function")
	}
}

var sampleTypeScript = []byte(`export function add(a: number, b: number): number {
  return a + b;
}

export class Calculator {
  multiply(x: number, y: number): number {
    return x * y;
  }
}

export interface Shape {
  area(): number;
}

export type Color = "red" | "green" | "blue";
`)

func TestTreeSitterChunker_TypeScript(t *testing.T) {
	def := chunker.LanguageDef{
		Language: sitter_ts.GetLanguage(),
		Queries: []chunker.QueryDef{
			{Pattern: `(function_declaration name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(class_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(interface_declaration name: (type_identifier) @name) @decl`, Kind: "interface"},
			{Pattern: `(type_alias_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(method_definition name: (property_identifier) @name) @decl`, Kind: "method"},
		},
	}
	c, err := chunker.NewTreeSitterChunker(def)
	if err != nil {
		t.Fatalf("NewTreeSitterChunker: %v", err)
	}

	chunks, err := c.Chunk("sample.ts", sampleTypeScript)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbol := make(map[string]chunker.Chunk)
	for _, ch := range chunks {
		bySymbol[ch.Symbol] = ch
	}

	check := func(symbol, kind string) {
		t.Helper()
		ch, ok := bySymbol[symbol]
		if !ok {
			t.Errorf("missing chunk %q (got: %v)", symbol, symbolNames(chunks))
			return
		}
		if ch.Kind != kind {
			t.Errorf("chunk %q kind = %q, want %q", symbol, ch.Kind, kind)
		}
	}

	check("add", "function")
	check("Calculator", "type")
	check("multiply", "method")
	check("Shape", "interface")
	check("Color", "type")
}

var sampleRust = []byte(`pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

pub struct Point {
    pub x: f64,
    pub y: f64,
}

pub trait Drawable {
    fn draw(&self);
}

pub const MAX_SIZE: usize = 1024;
`)

func TestTreeSitterChunker_Rust(t *testing.T) {
	def := chunker.LanguageDef{
		Language: sitter_rs.GetLanguage(),
		Queries: []chunker.QueryDef{
			{Pattern: `(function_item name: (identifier) @name) @decl`, Kind: "function"},
			{Pattern: `(struct_item name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(enum_item name: (type_identifier) @name) @decl`, Kind: "type"},
			{Pattern: `(trait_item name: (type_identifier) @name) @decl`, Kind: "interface"},
			{Pattern: `(const_item name: (identifier) @name) @decl`, Kind: "const"},
		},
	}
	c, err := chunker.NewTreeSitterChunker(def)
	if err != nil {
		t.Fatalf("NewTreeSitterChunker: %v", err)
	}

	chunks, err := c.Chunk("sample.rs", sampleRust)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbol := make(map[string]chunker.Chunk)
	for _, ch := range chunks {
		bySymbol[ch.Symbol] = ch
	}

	check := func(symbol, kind string) {
		t.Helper()
		ch, ok := bySymbol[symbol]
		if !ok {
			t.Errorf("missing chunk %q (got: %v)", symbol, symbolNames(chunks))
			return
		}
		if ch.Kind != kind {
			t.Errorf("chunk %q kind = %q, want %q", symbol, ch.Kind, kind)
		}
	}

	check("add", "function")
	check("Point", "type")
	check("Drawable", "interface")
	check("MAX_SIZE", "const")
}

func symbolNames(chunks []chunker.Chunk) []string {
	names := make([]string, len(chunks))
	for i, c := range chunks {
		names[i] = c.Symbol
	}
	return names
}

func TestMultiChunker_Dispatch(t *testing.T) {
	pyChunker := mustPyChunker(t)

	mc := chunker.NewMultiChunker(map[string]chunker.Chunker{
		".py": pyChunker,
	})

	// Known extension — returns chunks
	chunks, err := mc.Chunk("foo.py", samplePython)
	if err != nil {
		t.Fatalf("Chunk(.py): %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected chunks for .py, got none")
	}

	// Unknown extension — returns nil, nil
	chunks, err = mc.Chunk("foo.xyz", []byte("hello"))
	if err != nil {
		t.Fatalf("Chunk(.xyz): unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected no chunks for .xyz, got %d", len(chunks))
	}
}

func TestDefaultLanguages_AllExtensionsPresent(t *testing.T) {
	langs := chunker.DefaultLanguages()
	for _, ext := range chunker.SupportedExtensions() {
		if _, ok := langs[ext]; !ok {
			t.Errorf("DefaultLanguages() missing extension %q", ext)
		}
	}
}

// mustPyChunker creates a Python TreeSitterChunker for use in tests.
func mustPyChunker(t *testing.T) *chunker.TreeSitterChunker {
	t.Helper()
	def := chunker.LanguageDef{
		Language: sitter_py.GetLanguage(),
		Queries: []chunker.QueryDef{
			{Pattern: `(function_definition name: (identifier) @name) @decl`, Kind: "function"},
		},
	}
	c, err := chunker.NewTreeSitterChunker(def)
	if err != nil {
		t.Fatalf("NewTreeSitterChunker: %v", err)
	}
	return c
}
