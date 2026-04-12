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
	"strings"
	"testing"

	sitter_c "github.com/smacker/go-tree-sitter/c"
	sitter_cs "github.com/smacker/go-tree-sitter/csharp"
	sitter_java "github.com/smacker/go-tree-sitter/java"
	sitter_php "github.com/smacker/go-tree-sitter/php"
	sitter_py "github.com/smacker/go-tree-sitter/python"
	sitter_rs "github.com/smacker/go-tree-sitter/rust"
	sitter_ts "github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/ory/lumen/internal/chunker"
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

	checkChunk(t, bySymbol, "greet", "function", 1, 3, "sample.py", "def greet")
	checkChunk(t, bySymbol, "Animal", "type", 5, 7, "sample.py", "")
	checkChunk(t, bySymbol, "speak", "function", 0, 0, "sample.py", "")
}

func checkChunk(t *testing.T, bySymbol map[string]chunker.Chunk, symbol, kind string, startLine, endLine int, filePath, contentContains string) {
	t.Helper()
	ch, ok := bySymbol[symbol]
	if !ok {
		t.Fatalf("expected chunk %q, got symbols: %v", symbol, getChunkSymbols(bySymbol))
	}
	checkChunkKind(t, symbol, ch.Kind, kind)
	checkChunkLines(t, symbol, ch.StartLine, ch.EndLine, startLine, endLine)
	checkChunkPath(t, symbol, ch.FilePath, filePath)
	checkChunkID(t, symbol, ch.ID)
	checkChunkContent(t, symbol, ch.Content, contentContains)
}

func checkChunkKind(t *testing.T, symbol, actual, expected string) {
	if actual != expected {
		t.Errorf("%s.Kind = %q, want %q", symbol, actual, expected)
	}
}

func checkChunkLines(t *testing.T, symbol string, actualStart, actualEnd, expectedStart, expectedEnd int) {
	if expectedStart > 0 && actualStart != expectedStart {
		t.Errorf("%s.StartLine = %d, want %d", symbol, actualStart, expectedStart)
	}
	if expectedEnd > 0 && actualEnd != expectedEnd {
		t.Errorf("%s.EndLine = %d, want %d", symbol, actualEnd, expectedEnd)
	}
}

func checkChunkPath(t *testing.T, symbol, actual, expected string) {
	if expected != "" && actual != expected {
		t.Errorf("%s.FilePath = %q, want %q", symbol, actual, expected)
	}
}

func checkChunkID(t *testing.T, symbol, id string) {
	if id == "" {
		t.Errorf("%s.ID is empty", symbol)
	}
}

func checkChunkContent(t *testing.T, symbol, content, contains string) {
	if contains != "" && !strings.Contains(content, contains) {
		t.Errorf("%s.Content does not contain %q", symbol, contains)
	}
}

func getChunkSymbols(bySymbol map[string]chunker.Chunk) []string {
	symbols := make([]string, 0, len(bySymbol))
	for s := range bySymbol {
		symbols = append(symbols, s)
	}
	return symbols
}

var sampleTypeScript = []byte(`export function add(a: number, b: number): number {
  return a + b;
}

export class Calculator {
  multiply(x: number, y: number): number {
    return x * y;
  }
}

export abstract class Base {
  abstract doWork(): void;
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
			{Pattern: `(abstract_class_declaration name: (type_identifier) @name) @decl`, Kind: "type"},
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
	check("Base", "type")
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

var sampleJavaScript = []byte(`function greet(name) {
  return "hello " + name;
}

class Animal {
  speak() {
    return "...";
  }
}
`)

func TestTreeSitterChunker_JavaScript(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".js"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .js")
	}

	chunks, err := c.Chunk("sample.js", sampleJavaScript)
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

	check("greet", "function")
	check("Animal", "type")
	check("speak", "method")
}

var sampleModernTypeScript = []byte(`export const greet = (name: string): string => {
  return "hello " + name;
};

const handler = function() {
  return 42;
};

var legacy = function() {
  return 0;
};

export const gen = function*() {
  yield 1;
};

export function namedFn() {}

export enum Direction { Up, Down, Left, Right }

const enum Status { Active, Inactive }

export class MyClass {
  method() {}
}
`)

func TestTreeSitterChunker_TypeScript_ModernPatterns(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".ts"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .ts")
	}

	chunks, err := c.Chunk("sample.ts", sampleModernTypeScript)
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

	check("greet", "function")   // arrow function in const
	check("handler", "function") // function expression in const
	check("legacy", "function")  // function expression in var
	check("gen", "function")     // generator in const
	check("namedFn", "function") // regular function declaration
	check("Direction", "type")   // enum
	check("Status", "type")      // const enum
	check("MyClass", "type")     // class
}

var sampleExportedConsts = []byte(`export const domain: RegExp = /^[\w.-]+$/;
export const httpProtocol: RegExp = /^https?$/;
export const MAX_RETRIES = 3;
export const defaultConfig = { timeout: 5000, retries: 3 };
export const greetFn = (name: string) => "hello " + name;
`)

func TestTreeSitterChunker_TypeScript_ExportedConsts(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".ts"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .ts")
	}

	chunks, err := c.Chunk("regexes.ts", sampleExportedConsts)
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

	check("domain", "const")        // exported regex constant
	check("httpProtocol", "const")  // exported regex constant
	check("MAX_RETRIES", "const")   // exported numeric constant
	check("defaultConfig", "const") // exported object constant
	check("greetFn", "function")    // exported arrow function — should be "function" not "const"
}

var sampleModernJavaScript = []byte(`export const greet = (name) => {
  return "hello " + name;
};

const handler = function() {
  return 42;
};

var legacy = function() {
  return 0;
};

export const gen = function*() {
  yield 1;
};

export function namedFn() {}

export class Animal {
  speak() {}
}
`)

func TestTreeSitterChunker_JavaScript_ModernPatterns(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".js"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .js")
	}

	chunks, err := c.Chunk("sample.js", sampleModernJavaScript)
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

	check("greet", "function")   // arrow function in const
	check("handler", "function") // function expression in const
	check("legacy", "function")  // function expression in var
	check("gen", "function")     // generator in const
	check("namedFn", "function") // regular function declaration
	check("Animal", "type")      // class
}

var sampleTSX = []byte(`export function render(): JSX.Element {
  return <div />;
}

export class App {
  render(): JSX.Element {
    return <div />;
  }
}
`)

func TestTreeSitterChunker_TSX(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".tsx"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .tsx")
	}

	chunks, err := c.Chunk("sample.tsx", sampleTSX)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	// There will be two "render" chunks: one function and one method.
	// Use a multi-map to track all chunks by symbol.
	bySymbolKind := make(map[string]map[string]bool) // symbol -> set of kinds
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	if !bySymbolKind["render"]["function"] {
		t.Errorf("missing chunk render/function (got symbols: %v)", symbolNames(chunks))
	}
	if !bySymbolKind["render"]["method"] {
		t.Errorf("missing chunk render/method (got symbols: %v)", symbolNames(chunks))
	}
	if !bySymbolKind["App"]["type"] {
		t.Errorf("missing chunk App/type (got symbols: %v)", symbolNames(chunks))
	}
}

var sampleRuby = []byte(`def greet(name)
  "hello #{name}"
end

class Animal
  def speak
    "..."
  end
end
`)

func TestTreeSitterChunker_Ruby(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".rb"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .rb")
	}

	chunks, err := c.Chunk("sample.rb", sampleRuby)
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

	check("greet", "function")
	check("Animal", "type")
	check("Animal.speak", "function") // Ruby methods map to "function" kind
}

var sampleJava = []byte(`public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
`)

func TestTreeSitterChunker_Java(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".java"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .java")
	}

	chunks, err := c.Chunk("sample.java", sampleJava)
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

	check("Calculator", "type")
	check("add", "method")
}

var sampleC = []byte(`int add(int a, int b) {
    return a + b;
}

int *get_ptr(void) {
    return 0;
}

struct Point {
    int x;
    int y;
};
`)

func TestTreeSitterChunker_C(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".c"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .c")
	}

	chunks, err := c.Chunk("sample.c", sampleC)
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
	check("get_ptr", "function") // requires pointer-return fix
	check("Point", "type")
}

var sampleCPP = []byte(`class Vec2 {
    float x;
    float y;
};

int add(int a, int b) {
    return a + b;
}
`)

func TestTreeSitterChunker_CPP(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".cpp"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .cpp")
	}

	chunks, err := c.Chunk("sample.cpp", sampleCPP)
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

	check("Vec2", "type")
	check("add", "function")
}

var sampleCSharp = []byte(`using System;

namespace MyApp
{
    public delegate void StatusChanged(string status);

    public enum Direction { North, South, East, West }

    public interface IShape
    {
        double Area();
    }

    public record Point(double X, double Y);

    public struct Vector2
    {
        public double X;
        public double Y;
    }

    public class Calculator : IShape
    {
        private int _value;

        public event StatusChanged OnStatusChanged;

        public int Value { get => _value; set => _value = value; }

        public Calculator(int initial) { _value = initial; }

        ~Calculator() { }

        public int Add(int a, int b) => a + b;

        public double Area() => _value;
    }
}
`)

func TestTreeSitterChunker_CSharp(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".cs"]
	if !ok {
		t.Fatal("DefaultLanguages() missing .cs")
	}

	chunks, err := c.Chunk("sample.cs", sampleCSharp)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("Calculator", "type")     // class_declaration
	check("IShape", "interface")    // interface_declaration
	check("Vector2", "type")        // struct_declaration
	check("Direction", "type")      // enum_declaration
	check("Point", "type")          // record_declaration
	check("StatusChanged", "type")  // delegate_declaration
	check("Add", "method")          // method_declaration
	check("Value", "method")        // property_declaration
	check("Calculator", "function") // constructor_declaration
	check("OnStatusChanged", "var") // event_field_declaration
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
	// trivialSources maps file extensions to minimal source containing one named declaration.
	trivialSources := map[string][]byte{
		".go":   []byte("package main\nfunc Foo() {}"),
		".ts":   []byte("function foo() {}"),
		".tsx":  []byte("function foo() { return <div/>; }"),
		".js":   []byte("function foo() {}"),
		".jsx":  []byte("function foo() { return <div/>; }"),
		".mjs":  []byte("function foo() {}"),
		".py":   []byte("def foo():\n    pass"),
		".rs":   []byte("fn foo() {}"),
		".rb":   []byte("def foo\nend"),
		".java": []byte("class Foo {}"),
		".c":    []byte("void foo(void) {}"),
		".h":    []byte("void foo(void) {}"),
		".cpp":  []byte("void foo() {}"),
		".cc":   []byte("void foo() {}"),
		".cxx":  []byte("void foo() {}"),
		".hpp":  []byte("void foo() {}"),
		".php":  []byte("<?php\nfunction foo() {}"),
		".cs":   []byte("class Foo {}"),
		".dart":   []byte("void foo() {}"),
		".svelte": []byte("<script lang=\"ts\">\nfunction hello(): string { return 'hi'; }\n</script>\n<p>hi</p>\n"),
		".md":   []byte("# Introduction\nSome content here.\n"),
		".mdx":  []byte("# Introduction\nSome content here.\n"),
		".yaml": []byte("foo: bar\n"),
		".yml":  []byte("foo: bar\n"),
		".json": []byte(`{"foo": "bar"}`),
		".toml": []byte("[package]\nname = \"mymod\"\n"),
		".mod":  []byte("module example.com/mymod\n\ngo 1.26\n"),
	}

	langs := chunker.DefaultLanguages(512)

	for _, ext := range chunker.SupportedExtensions() {
		c, ok := langs[ext]
		if !ok {
			t.Errorf("DefaultLanguages() missing extension %q", ext)
			continue
		}
		src, ok := trivialSources[ext]
		if !ok {
			t.Errorf("trivialSources missing fixture for %q", ext)
			continue
		}
		chunks, err := c.Chunk("test"+ext, src)
		if err != nil {
			t.Errorf("Chunk(%q): %v", ext, err)
			continue
		}
		if len(chunks) == 0 {
			t.Errorf("Chunk(%q): expected at least 1 chunk, got 0", ext)
		}
	}
}

// --- Comprehensive tests for new query patterns ---

var samplePythonComprehensive = []byte(`
def bare_function():
    pass

class BareClass:
    pass

@decorator
def decorated_function():
    pass

@decorator
class DecoratedClass:
    pass

MY_VAR = 42
`)

func TestTreeSitterChunker_Python_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".py"]

	chunks, err := c.Chunk("sample.py", samplePythonComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("bare_function", "function")
	check("BareClass", "type")
	check("decorated_function", "function")
	check("DecoratedClass", "type")
	check("MY_VAR", "var")
}

var sampleRustComprehensive = []byte(`
fn bare_fn() {}

pub struct MyStruct {
    x: i32,
}

pub enum MyEnum {
    A,
    B,
}

pub trait MyTrait {
    fn required(&self);
}

pub const MY_CONST: i32 = 42;

impl MyStruct {
    fn method(&self) {}
}

impl<T> MyStruct {
    fn generic_method(&self) {}
}

type Alias = i32;

static MY_STATIC: i32 = 10;

mod my_module {}

macro_rules! my_macro {
    () => {};
}

union MyUnion {
    x: i32,
    y: f64,
}
`)

func TestTreeSitterChunker_Rust_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".rs"]

	chunks, err := c.Chunk("sample.rs", sampleRustComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("bare_fn", "function")
	check("MyStruct", "type")     // struct_item + impl_item
	check("MyEnum", "type")       // enum_item
	check("MyTrait", "interface") // trait_item
	check("MY_CONST", "const")    // const_item
	check("method", "function")   // function_item inside impl
	check("Alias", "type")        // type_item
	check("MY_STATIC", "var")     // static_item
	check("my_module", "type")    // mod_item
	check("my_macro", "function") // macro_definition
	check("MyUnion", "type")      // union_item
}

var sampleRubyComprehensive = []byte(`
def greet(name)
  "hello #{name}"
end

def self.class_method
  "class method"
end

class Animal
  def speak
    "..."
  end
end

module Utilities
end

alias new_greet greet
`)

func TestTreeSitterChunker_Ruby_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".rb"]

	chunks, err := c.Chunk("sample.rb", sampleRubyComprehensive)
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

	check("greet", "function")
	check("class_method", "function") // singleton_method
	check("Animal", "type")
	check("Animal.speak", "function")
	check("Utilities", "type")     // module
	check("new_greet", "function") // alias
}

var sampleJavaComprehensive = []byte(`
public class Calculator {
    public static final int MAX = 100;

    public int add(int a, int b) {
        return a + b;
    }

    public Calculator() {}
}

interface Computable {
    int compute();
}

enum Status { ACTIVE, INACTIVE }

@interface MyAnnotation {
    String value();
}

record Point(int x, int y) {}
`)

func TestTreeSitterChunker_Java_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".java"]

	chunks, err := c.Chunk("sample.java", sampleJavaComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("Calculator", "type")      // class
	check("Calculator", "function")  // constructor
	check("add", "method")           // method
	check("Computable", "interface") // interface
	check("Status", "type")          // enum
	check("MyAnnotation", "type")    // annotation_type
	check("Point", "type")           // record
	check("MAX", "var")              // field
}

var samplePHPComprehensive = []byte(`<?php
namespace App\Models;

function helper() {}

class User {
    const TABLE = 'users';

    public function getName() {}
}

interface Repository {
    public function find($id);
}

trait Cacheable {
    public function cache() {}
}

enum Color {
    case Red;
    case Blue;
}
`)

func TestTreeSitterChunker_PHP_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".php"]

	chunks, err := c.Chunk("sample.php", samplePHPComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("helper", "function")
	check("User", "type") // class_declaration — was missing!
	check("getName", "method")
	check("Repository", "interface")
	check("Cacheable", "type")   // trait
	check("Color", "type")       // enum
	check("App\\Models", "type") // namespace
	check("TABLE", "const")      // const_declaration
}

var sampleCSharpComprehensive = []byte(`using System;

namespace MyApp
{
    public delegate void StatusChanged(string status);

    public enum Direction { North, South, East, West }

    public interface IShape
    {
        double Area();
    }

    public record Point(double X, double Y);

    public struct Vector2
    {
        public double X;
        public double Y;
    }

    public class Calculator : IShape
    {
        private int _value;

        public event StatusChanged OnStatusChanged;

        public int Value { get => _value; set => _value = value; }

        public Calculator(int initial) { _value = initial; }

        ~Calculator() { }

        public int Add(int a, int b) => a + b;

        public double Area() => _value;
    }
}
`)

func TestTreeSitterChunker_CSharp_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".cs"]

	chunks, err := c.Chunk("sample.cs", sampleCSharpComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("MyApp", "type")          // namespace_declaration
	check("Calculator", "type")     // class_declaration
	check("IShape", "interface")    // interface_declaration
	check("Vector2", "type")        // struct_declaration
	check("Direction", "type")      // enum_declaration
	check("Point", "type")          // record_declaration
	check("StatusChanged", "type")  // delegate_declaration
	check("Add", "method")          // method_declaration
	check("Value", "method")        // property_declaration
	check("Calculator", "function") // constructor_declaration
	check("OnStatusChanged", "var") // event_field_declaration
	check("_value", "var")          // field_declaration
}

var sampleCComprehensive = []byte(`
int add(int a, int b) {
    return a + b;
}

int *get_ptr(void) {
    return 0;
}

struct Point {
    int x;
    int y;
};

enum Color { RED, GREEN, BLUE };

union Data {
    int i;
    float f;
};

typedef int MyInt;

#define MAX_SIZE 1024

#define SQUARE(x) ((x) * (x))
`)

func TestTreeSitterChunker_C_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".c"]

	chunks, err := c.Chunk("sample.c", sampleCComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("add", "function")
	check("get_ptr", "function") // pointer-return
	check("Point", "type")       // struct
	check("Color", "type")       // enum
	check("Data", "type")        // union
	check("MyInt", "type")       // typedef
	check("MAX_SIZE", "const")   // preproc_def
	check("SQUARE", "function")  // preproc_function_def
}

var sampleCPPComprehensive = []byte(`
class Vec2 {
    float x;
    float y;
};

struct Point {
    int x;
    int y;
};

int add(int a, int b) {
    return a + b;
}

enum Color { RED, GREEN, BLUE };

namespace MyLib {
    void helper() {}
}

using IntAlias = int;

void Vec2::normalize() {
    // ...
}

union Data {
    int i;
    float f;
};

typedef int MyInt;

#define MAX_SIZE 1024

#define SQUARE(x) ((x) * (x))
`)

func TestTreeSitterChunker_CPP_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".cpp"]

	chunks, err := c.Chunk("sample.cpp", sampleCPPComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("Vec2", "type")        // class_specifier
	check("Point", "type")       // struct_specifier
	check("add", "function")     // function_definition
	check("Color", "type")       // enum_specifier
	check("MyLib", "type")       // namespace_definition
	check("IntAlias", "type")    // alias_declaration
	check("normalize", "method") // qualified method definition
	check("Data", "type")        // union_specifier
	check("MyInt", "type")       // type_definition
	check("MAX_SIZE", "const")   // preproc_def
	check("SQUARE", "function")  // preproc_function_def
}

var sampleTSComprehensive = []byte(`
export function namedFn() {}

export function* genFn() { yield 1; }

export const arrowFn = (x: number) => x * 2;

export class MyClass {
  method() {}
}

export abstract class BaseClass {
  abstract doWork(): void;
}

export interface IShape {
  area(): number;
}

export type Color = "red" | "green" | "blue";

export enum Direction { Up, Down }

export const MAX = 100;

interface Writable {
  write(data: string): void;
}
`)

func TestTreeSitterChunker_TypeScript_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c := langs[".ts"]

	chunks, err := c.Chunk("sample.ts", sampleTSComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("namedFn", "function")
	check("genFn", "function")   // generator
	check("arrowFn", "function") // arrow function
	check("MyClass", "type")
	check("method", "method")
	check("BaseClass", "type") // abstract class
	check("IShape", "interface")
	check("Color", "type")     // type_alias
	check("Direction", "type") // enum
	check("MAX", "const")      // exported const
	check("write", "method")   // method_signature in interface
}

// findChunk returns the first chunk with the given symbol from chunks.
func findChunk(chunks []chunker.Chunk, symbol string) (chunker.Chunk, bool) {
	for _, c := range chunks {
		if c.Symbol == symbol {
			return c, true
		}
	}
	return chunker.Chunk{}, false
}

func TestTreeSitterChunker_LeadingComments(t *testing.T) {
	langs := chunker.DefaultLanguages(512)

	// commentCase exercises a single comment-capture scenario.
	type commentCase struct {
		name          string
		chunker       chunker.Chunker
		filePath      string
		src           string
		symbol        string
		wantStartLine int    // 1-based; 0 means "don't check"
		wantInContent string // must appear in Content
		wantMissing   string // must NOT appear in Content (empty = skip)
	}

	// Helpers that build ad-hoc chunkers for languages not exposed by DefaultLanguages
	// at the extension level, so we can control the exact query.
	mustC := func() chunker.Chunker {
		c, err := chunker.NewTreeSitterChunker(chunker.LanguageDef{
			Language: sitter_c.GetLanguage(),
			Queries: []chunker.QueryDef{
				{Pattern: `(function_definition declarator: (function_declarator declarator: (identifier) @name)) @decl`, Kind: "function"},
			},
		})
		if err != nil {
			t.Fatalf("C chunker: %v", err)
		}
		return c
	}()

	mustCS := func() chunker.Chunker {
		c, err := chunker.NewTreeSitterChunker(chunker.LanguageDef{
			Language: sitter_cs.GetLanguage(),
			Queries: []chunker.QueryDef{
				{Pattern: `(method_declaration name: (identifier) @name) @decl`, Kind: "method"},
			},
		})
		if err != nil {
			t.Fatalf("C# chunker: %v", err)
		}
		return c
	}()

	mustJava := func() chunker.Chunker {
		c, err := chunker.NewTreeSitterChunker(chunker.LanguageDef{
			Language: sitter_java.GetLanguage(),
			Queries: []chunker.QueryDef{
				{Pattern: `(method_declaration name: (identifier) @name) @decl`, Kind: "method"},
				{Pattern: `(class_declaration name: (identifier) @name) @decl`, Kind: "type"},
			},
		})
		if err != nil {
			t.Fatalf("Java chunker: %v", err)
		}
		return c
	}()

	mustPHP := func() chunker.Chunker {
		c, err := chunker.NewTreeSitterChunker(chunker.LanguageDef{
			Language: sitter_php.GetLanguage(),
			Queries: []chunker.QueryDef{
				{Pattern: `(function_definition name: (name) @name) @decl`, Kind: "function"},
			},
		})
		if err != nil {
			t.Fatalf("PHP chunker: %v", err)
		}
		return c
	}()

	mustRS := func() chunker.Chunker {
		c, err := chunker.NewTreeSitterChunker(chunker.LanguageDef{
			Language: sitter_rs.GetLanguage(),
			Queries: []chunker.QueryDef{
				{Pattern: `(function_item name: (identifier) @name) @decl`, Kind: "function"},
			},
		})
		if err != nil {
			t.Fatalf("Rust chunker: %v", err)
		}
		return c
	}()

	cases := []commentCase{
		// ── Ruby (comment node type: "comment") ──────────────────────────────
		{
			name:          "ruby: adjacent # comment captured",
			chunker:       langs[".rb"],
			filePath:      "f.rb",
			src:           "# greets the user\ndef greet(name)\n  \"hello #{name}\"\nend\n",
			symbol:        "greet",
			wantStartLine: 1,
			wantInContent: "# greets the user",
		},
		{
			name:          "ruby: blank line prevents capture",
			chunker:       langs[".rb"],
			filePath:      "f.rb",
			src:           "# unrelated\n\ndef other\nend\n",
			symbol:        "other",
			wantStartLine: 3,
			wantMissing:   "# unrelated",
		},
		{
			name:          "ruby: multiple consecutive comment lines all captured",
			chunker:       langs[".rb"],
			filePath:      "f.rb",
			src:           "# line one\n# line two\n# line three\ndef multi\nend\n",
			symbol:        "multi",
			wantStartLine: 1,
			wantInContent: "# line one",
		},
		{
			name:          "ruby: no preceding comment — StartLine at decl",
			chunker:       langs[".rb"],
			filePath:      "f.rb",
			src:           "def alone\nend\n",
			symbol:        "alone",
			wantStartLine: 1, // decl is on line 1
		},
		{
			name:          "ruby: comment on previous sibling not captured for next decl",
			chunker:       langs[".rb"],
			filePath:      "f.rb",
			src:           "# for greet\ndef greet\nend\ndef other\nend\n",
			symbol:        "other",
			wantStartLine: 4, // 'def other' is line 4
			wantMissing:   "# for greet",
		},

		// ── JavaScript (comment: "comment") ──────────────────────────────────
		{
			name:          "js: adjacent // comment captured",
			chunker:       langs[".js"],
			filePath:      "f.js",
			src:           "// adds two numbers\nfunction add(a, b) { return a + b; }\n",
			symbol:        "add",
			wantStartLine: 1,
			wantInContent: "// adds two numbers",
		},
		{
			name:          "js: adjacent /* block */ comment captured",
			chunker:       langs[".js"],
			filePath:      "f.js",
			src:           "/* multiplies */\nfunction mul(a, b) { return a * b; }\n",
			symbol:        "mul",
			wantStartLine: 1,
			wantInContent: "/* multiplies */",
		},
		{
			name:          "js: blank line prevents // capture",
			chunker:       langs[".js"],
			filePath:      "f.js",
			src:           "// old\n\nfunction sub(a, b) { return a - b; }\n",
			symbol:        "sub",
			wantStartLine: 3,
			wantMissing:   "// old",
		},

		// ── TypeScript (comment: "comment") ───────────────────────────────────
		{
			name:          "ts: adjacent // comment captured",
			chunker:       langs[".ts"],
			filePath:      "f.ts",
			src:           "// computes square\nfunction square(x: number): number { return x * x; }\n",
			symbol:        "square",
			wantStartLine: 1,
			wantInContent: "// computes square",
		},
		{
			name:          "ts: adjacent /** JSDoc */ captured",
			chunker:       langs[".ts"],
			filePath:      "f.ts",
			src:           "/** @returns the cube */\nfunction cube(x: number): number { return x ** 3; }\n",
			symbol:        "cube",
			wantStartLine: 1,
			wantInContent: "/** @returns the cube */",
		},

		// ── Python (comment: "comment") ───────────────────────────────────────
		{
			name:          "python: adjacent # comment captured",
			chunker:       langs[".py"],
			filePath:      "f.py",
			src:           "# returns greeting\ndef greet(name):\n    return f\"hi {name}\"\n",
			symbol:        "greet",
			wantStartLine: 1,
			wantInContent: "# returns greeting",
		},
		{
			name:          "python: blank line prevents capture",
			chunker:       langs[".py"],
			filePath:      "f.py",
			src:           "# unrelated\n\ndef other():\n    pass\n",
			symbol:        "other",
			wantStartLine: 3,
			wantMissing:   "# unrelated",
		},

		// ── Rust (comment: "line_comment" / "block_comment") ─────────────────
		{
			name:          "rust: adjacent // comment captured (line_comment)",
			chunker:       mustRS,
			filePath:      "f.rs",
			src:           "// adds two ints\nfn add(a: i32, b: i32) -> i32 { a + b }\n",
			symbol:        "add",
			wantStartLine: 1,
			wantInContent: "// adds two ints",
		},
		{
			name:          "rust: adjacent /// doc comment captured (line_comment)",
			chunker:       mustRS,
			filePath:      "f.rs",
			src:           "/// squares a number\nfn square(x: i32) -> i32 { x * x }\n",
			symbol:        "square",
			wantStartLine: 1,
			wantInContent: "/// squares a number",
		},
		{
			name:          "rust: adjacent /* block */ comment captured (block_comment)",
			chunker:       mustRS,
			filePath:      "f.rs",
			src:           "/* multiplies */\nfn mul(a: i32, b: i32) -> i32 { a * b }\n",
			symbol:        "mul",
			wantStartLine: 1,
			wantInContent: "/* multiplies */",
		},
		{
			name:          "rust: blank line prevents capture",
			chunker:       mustRS,
			filePath:      "f.rs",
			src:           "// old\n\nfn sub(a: i32, b: i32) -> i32 { a - b }\n",
			symbol:        "sub",
			wantStartLine: 3,
			wantMissing:   "// old",
		},

		// ── Java (comment: "line_comment" / "block_comment") ─────────────────
		// Java class_declaration is not in findEnclosingSymbol, so methods are
		// reported with their bare name (not class-qualified).
		{
			name:     "java: adjacent // comment captured (line_comment)",
			chunker:  mustJava,
			filePath: "f.java",
			src: "public class Foo {\n" +
				"    // computes sum\n" +
				"    public int add(int a, int b) { return a + b; }\n" +
				"}\n",
			symbol:        "add",
			wantStartLine: 2,
			wantInContent: "// computes sum",
		},
		{
			name:     "java: adjacent /** Javadoc */ captured (block_comment)",
			chunker:  mustJava,
			filePath: "f.java",
			src: "public class Bar {\n" +
				"    /** @return product */\n" +
				"    public int mul(int a, int b) { return a * b; }\n" +
				"}\n",
			symbol:        "mul",
			wantStartLine: 2,
			wantInContent: "/** @return product */",
		},

		// ── C (comment: "comment") ────────────────────────────────────────────
		{
			name:          "c: adjacent // comment captured",
			chunker:       mustC,
			filePath:      "f.c",
			src:           "// adds ints\nint add(int a, int b) { return a + b; }\n",
			symbol:        "add",
			wantStartLine: 1,
			wantInContent: "// adds ints",
		},
		{
			name:          "c: adjacent /* block */ comment captured",
			chunker:       mustC,
			filePath:      "f.c",
			src:           "/* multiplies */\nint mul(int a, int b) { return a * b; }\n",
			symbol:        "mul",
			wantStartLine: 1,
			wantInContent: "/* multiplies */",
		},

		// ── C# (comment: "comment") ───────────────────────────────────────────
		// Top-level functions in C# are local_function_statements; methods require
		// a class context to match (method_declaration).
		{
			name:     "csharp: adjacent // comment captured",
			chunker:  mustCS,
			filePath: "f.cs",
			src: "class Calc {\n" +
				"    // sums values\n" +
				"    void Add(int a, int b) {}\n" +
				"}\n",
			symbol:        "Add",
			wantStartLine: 2,
			wantInContent: "// sums values",
		},

		// ── PHP (comment: "comment") ──────────────────────────────────────────
		{
			name:          "php: adjacent // comment captured",
			chunker:       mustPHP,
			filePath:      "f.php",
			src:           "<?php\n// adds numbers\nfunction add($a, $b) { return $a + $b; }\n",
			symbol:        "add",
			wantStartLine: 2,
			wantInContent: "// adds numbers",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := tc.chunker.Chunk(tc.filePath, []byte(tc.src))
			if err != nil {
				t.Fatalf("Chunk: %v", err)
			}
			ch, ok := findChunk(chunks, tc.symbol)
			if !ok {
				var symbols []string
				for _, c := range chunks {
					symbols = append(symbols, c.Symbol)
				}
				t.Fatalf("missing chunk %q (got: %v)", tc.symbol, symbols)
			}
			if tc.wantStartLine != 0 && ch.StartLine != tc.wantStartLine {
				t.Errorf("StartLine = %d, want %d", ch.StartLine, tc.wantStartLine)
			}
			if tc.wantInContent != "" && !strings.Contains(ch.Content, tc.wantInContent) {
				t.Errorf("Content missing %q; got:\n%s", tc.wantInContent, ch.Content)
			}
			if tc.wantMissing != "" && strings.Contains(ch.Content, tc.wantMissing) {
				t.Errorf("Content must not contain %q; got:\n%s", tc.wantMissing, ch.Content)
			}
		})
	}
}

func TestTreeSitterChunker_LeadingCommentsCapped(t *testing.T) {
	var commentLines []string
	for i := 0; i < 20; i++ {
		commentLines = append(commentLines, fmt.Sprintf("# line %d", i+1))
	}
	src := strings.Join(commentLines, "\n") + "\ndef capped():\n    pass\n"

	c := mustPyChunker(t)
	chunks, err := c.Chunk("test.py", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	chunk := chunks[0]
	capturedLines := strings.Split(chunk.Content, "\n")
	commentCount := 0
	for _, line := range capturedLines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			commentCount++
		}
	}
	if commentCount > 10 {
		t.Errorf("captured %d comment lines, want ≤10", commentCount)
	}
}

var sampleDartComprehensive = []byte(`
void greet(String name) {
  print('Hello, $name');
}

class Animal {
  final String name;
  Animal(this.name);

  void speak() {
    print('...');
  }
}

mixin Swimming {
  void swim() {
    print('swimming');
  }
}

enum Color { red, green, blue }

extension StringHelper on String {
  bool get isBlank => trim().isEmpty;
}

typedef IntCallback = void Function(int);

abstract class Repository {
  Future<void> save(Object entity);
}
`)

func TestTreeSitterChunker_Dart_Comprehensive(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".dart"]
	if !ok {
		t.Fatal("missing chunker for .dart")
	}

	chunks, err := c.Chunk("sample.dart", sampleDartComprehensive)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbolKind := make(map[string]map[string]bool)
	for _, ch := range chunks {
		if bySymbolKind[ch.Symbol] == nil {
			bySymbolKind[ch.Symbol] = make(map[string]bool)
		}
		bySymbolKind[ch.Symbol][ch.Kind] = true
	}

	check := func(symbol, kind string) {
		t.Helper()
		kinds, ok := bySymbolKind[symbol]
		if !ok || !kinds[kind] {
			t.Errorf("missing chunk %q/%q (got: %v)", symbol, kind, symbolNames(chunks))
		}
	}

	check("greet", "function")
	check("Animal", "type")            // class_definition
	check("Animal.speak", "method")    // method_signature in class
	check("Swimming", "type")          // mixin_declaration
	check("Swimming.swim", "method")   // method_signature in mixin
	check("Color", "type")             // enum_declaration
	check("StringHelper", "type")      // extension_declaration
	check("StringHelper.isBlank", "method") // getter in extension
	check("IntCallback", "type")       // type_alias
	check("Repository", "type")        // abstract class
	check("Animal.Animal", "function") // constructor
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
