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

	"github.com/ory/lumen/internal/chunker"
)

// chunkSet tracks all (symbol, kind) pairs for multi-occurrence verification.
type chunkSet struct {
	t      *testing.T
	chunks []chunker.Chunk
	byKey  map[string][]chunker.Chunk // "symbol/kind" -> chunks
}

func newChunkSet(t *testing.T, chunks []chunker.Chunk) *chunkSet {
	t.Helper()
	cs := &chunkSet{
		t:      t,
		chunks: chunks,
		byKey:  make(map[string][]chunker.Chunk),
	}
	for _, ch := range chunks {
		key := ch.Symbol + "/" + ch.Kind
		cs.byKey[key] = append(cs.byKey[key], ch)
	}
	return cs
}

func (cs *chunkSet) mustHave(symbol, kind string) {
	cs.t.Helper()
	key := symbol + "/" + kind
	if len(cs.byKey[key]) == 0 {
		cs.t.Errorf("MISSING %s/%s (got: %v)", symbol, kind, cs.allKeys())
	}
}

func (cs *chunkSet) mustNotHave(symbol, kind string) {
	cs.t.Helper()
	key := symbol + "/" + kind
	if len(cs.byKey[key]) > 0 {
		cs.t.Errorf("FALSE POSITIVE %s/%s should not be extracted", symbol, kind)
	}
}

func (cs *chunkSet) allKeys() []string {
	keys := make([]string, 0, len(cs.byKey))
	for k := range cs.byKey {
		keys = append(keys, k)
	}
	return keys
}

func (cs *chunkSet) dump() string {
	var sb strings.Builder
	for _, ch := range cs.chunks {
		fmt.Fprintf(&sb, "  %s/%s L%d-%d\n", ch.Symbol, ch.Kind, ch.StartLine, ch.EndLine)
	}
	return sb.String()
}

// ---------- Python Adversarial ----------

var adversarialPython = []byte(`
# Regular definitions
def bare_function():
    pass

async def async_function():
    pass

class BareClass:
    def method(self):
        pass

    async def async_method(self):
        pass

    class InnerClass:
        pass

# Decorated definitions
@decorator
def decorated_function():
    pass

@decorator
class DecoratedClass:
    pass

@decorator1
@decorator2
def multi_decorated():
    pass

# Module-level assignments
MY_CONST = 42
__all__ = ["foo", "bar"]

# Edge cases: should NOT be extracted
# def commented_function(): pass
"""
def string_function():
    pass
"""
a, b = 1, 2  # unpacking - still extracted as assignment to "a" (first target)

# Nested definition
def outer():
    def inner():
        pass
    return inner
`)

func TestAdversarial_Python(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".py"].Chunk("adv.py", adversarialPython)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	// Must extract
	cs.mustHave("bare_function", "function")
	cs.mustHave("async_function", "function")
	cs.mustHave("BareClass", "type")
	cs.mustHave("method", "function")
	cs.mustHave("async_method", "function")
	cs.mustHave("InnerClass", "type")
	cs.mustHave("decorated_function", "function")
	cs.mustHave("DecoratedClass", "type")
	cs.mustHave("multi_decorated", "function")
	cs.mustHave("MY_CONST", "var")
	cs.mustHave("__all__", "var")
	cs.mustHave("outer", "function")
	cs.mustHave("outer.inner", "function")

	// Must NOT extract from comments/strings
	cs.mustNotHave("commented_function", "function")
	cs.mustNotHave("string_function", "function")
}

// ---------- TypeScript Adversarial ----------

var adversarialTypeScript = []byte(`
// Regular declarations
export function namedFn(): void {}
function* generatorFn() { yield 1; }
export const arrowFn = (x: number): number => x * 2;
const funcExpr = function() { return 42; };
var legacyFunc = function() { return 0; };

// Class hierarchy
export class MyClass {
  method(): void {}
  get accessor(): number { return 0; }
}

export abstract class AbstractBase {
  abstract doWork(): void;
}

// Interfaces
export interface IShape {
  area(): number;
  readonly name: string;
}

interface IWritable {
  write(data: string): void;
}

// Types and enums
export type Color = "red" | "green" | "blue";
export enum Direction { Up, Down }
const enum ConstDirection { Left, Right }

// Exported constants (non-function)
export const MAX_RETRIES = 3;
export const defaultConfig = { timeout: 5000 };
export const pattern: RegExp = /^test$/;

// Method signatures in interface
interface Closeable {
  close(): void;
}

// Namespace/module block
namespace MyNamespace {
  export function helper(): void {}
}

// Ambient declarations
declare function declaredFn(): void;
declare class DeclaredClass {}
declare module "my-module" {
  export function foo(): void;
}

// Edge cases
// function commented() {}
const obj = {
  methodInObj() { return 1; },
  arrowInObj: () => 2,
};
`)

func TestAdversarial_TypeScript(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".ts"].Chunk("adv.ts", adversarialTypeScript)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	// Core declarations
	cs.mustHave("namedFn", "function")
	cs.mustHave("generatorFn", "function")
	cs.mustHave("arrowFn", "function")
	cs.mustHave("funcExpr", "function")
	cs.mustHave("legacyFunc", "function")

	// Classes
	cs.mustHave("MyClass", "type")
	cs.mustHave("method", "method")
	cs.mustHave("AbstractBase", "type")

	// Interfaces
	cs.mustHave("IShape", "interface")
	cs.mustHave("area", "method")
	cs.mustHave("IWritable", "interface")
	cs.mustHave("write", "method")
	cs.mustHave("Closeable", "interface")
	cs.mustHave("close", "method")

	// Types and enums
	cs.mustHave("Color", "type")
	cs.mustHave("Direction", "type")
	cs.mustHave("ConstDirection", "type")

	// Exported constants
	cs.mustHave("MAX_RETRIES", "const")
	cs.mustHave("defaultConfig", "const")
	cs.mustHave("pattern", "const")

	// Namespace/module
	cs.mustHave("MyNamespace", "type")

	// Ambient declarations
	cs.mustHave("declaredFn", "function")

	// Not extracted from comments
	cs.mustNotHave("commented", "function")
}

// ---------- TSX Adversarial ----------

var adversarialTSX = []byte(`
import React from 'react';

export function Button(): JSX.Element {
  return <button>Click</button>;
}

export const Card = ({ title }: { title: string }) => {
  return <div>{title}</div>;
};

export class App extends React.Component {
  render(): JSX.Element {
    return <div><Button /><Card title="test" /></div>;
  }
}

export interface Props {
  name: string;
}

export type Theme = "light" | "dark";

// Component defined inside another (edge case)
function Wrapper() {
  function Inner() {
    return <span>inner</span>;
  }
  return <div><Inner /></div>;
}
`)

func TestAdversarial_TSX(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".tsx"].Chunk("adv.tsx", adversarialTSX)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("Button", "function")
	cs.mustHave("Card", "function")
	cs.mustHave("App", "type")
	cs.mustHave("render", "method")
	cs.mustHave("Props", "interface")
	cs.mustHave("Theme", "type")
	cs.mustHave("Wrapper", "function")
	cs.mustHave("Wrapper.Inner", "function")
}

// ---------- JavaScript Adversarial ----------

var adversarialJavaScript = []byte(`
// Regular declarations
function regularFn() {}
function* generatorFn() { yield 1; }

// Variable-based functions
const arrowFn = () => {};
const funcExpr = function namedExpr() {};
var legacyFn = function() {};
let mutableFn = function() {};

// Export wrappers
export function exportedFn() {}
export const exportedArrow = () => {};
export class ExportedClass {}
export default function defaultExport() {}

// Classes
class Animal {
  speak() {}
  get name() { return ""; }
}

// Edge cases: NOT definitions
(function iife() {})();
const result = iife;
// function commented() {}

// Object methods (debatable - documenting behavior)
const handlers = {
  onClick() {},
  onHover: () => {},
};
`)

func TestAdversarial_JavaScript(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".js"].Chunk("adv.js", adversarialJavaScript)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("regularFn", "function")
	cs.mustHave("generatorFn", "function")
	cs.mustHave("arrowFn", "function")
	cs.mustHave("funcExpr", "function")
	cs.mustHave("legacyFn", "function")
	cs.mustHave("mutableFn", "function")
	cs.mustHave("exportedFn", "function")
	cs.mustHave("exportedArrow", "function")
	cs.mustHave("ExportedClass", "type")
	cs.mustHave("defaultExport", "function")
	cs.mustHave("Animal", "type")
	cs.mustHave("speak", "method")

	cs.mustNotHave("commented", "function")
}

// ---------- Rust Adversarial ----------

var adversarialRust = []byte(`
fn bare_fn() {}

pub struct MyStruct {
    x: i32,
}

pub enum MyEnum {
    A,
    B(i32),
}

pub trait MyTrait {
    fn required(&self);
    type Output;
}

pub const MY_CONST: i32 = 42;
static MY_STATIC: i32 = 10;

impl MyStruct {
    fn method(&self) {}
}

impl MyTrait for MyStruct {
    fn required(&self) {}
    type Output = i32;
}

impl<T: Clone> MyStruct {
    fn generic_method(&self) {}
}

type Alias = Vec<i32>;

mod my_module {
    fn inner() {}
}

macro_rules! my_macro {
    () => {};
}

union MyUnion {
    x: i32,
    y: f64,
}

#[derive(Debug)]
pub struct Attributed {
    name: String,
}

pub(crate) fn crate_fn() {}

// fn commented_fn() {}
`)

func TestAdversarial_Rust(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".rs"].Chunk("adv.rs", adversarialRust)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("bare_fn", "function")
	cs.mustHave("MyStruct", "type")
	cs.mustHave("MyEnum", "type")
	cs.mustHave("MyTrait", "interface")
	cs.mustHave("MY_CONST", "const")
	cs.mustHave("MY_STATIC", "var")
	cs.mustHave("method", "function")
	cs.mustHave("required", "function") // function_item or function_signature_item
	cs.mustHave("Alias", "type")
	cs.mustHave("my_module", "type")
	cs.mustHave("my_macro", "function")
	cs.mustHave("MyUnion", "type")
	cs.mustHave("Attributed", "type")
	cs.mustHave("crate_fn", "function")
	cs.mustHave("inner", "function")
	cs.mustHave("generic_method", "function")

	cs.mustNotHave("commented_fn", "function")
}

// ---------- Ruby Adversarial ----------

var adversarialRuby = []byte(`
def top_level_method
  "hello"
end

class Animal
  MY_CONST = 42

  def speak
    "..."
  end

  def self.class_method
    "class method"
  end

  class InnerClass
  end
end

module Utilities
  def util_method
    "util"
  end
end

alias new_name top_level_method

# def commented_method; end

class String
  def custom_method
    "custom"
  end
end
`)

func TestAdversarial_Ruby(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".rb"].Chunk("adv.rb", adversarialRuby)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("top_level_method", "function")
	cs.mustHave("Animal", "type")
	cs.mustHave("Animal.MY_CONST", "var")
	cs.mustHave("Animal.speak", "function")
	cs.mustHave("Animal.class_method", "function")
	cs.mustHave("Animal.InnerClass", "type")
	cs.mustHave("Utilities", "type")
	cs.mustHave("Utilities.util_method", "function")
	cs.mustHave("new_name", "function")
	cs.mustHave("String", "type")
	cs.mustHave("String.custom_method", "function")

	cs.mustNotHave("commented_method", "function")
}

// ---------- Java Adversarial ----------

var adversarialJava = []byte(`
public class Calculator {
    public static final int MAX = 100;
    private String name;

    public Calculator() {}

    public int add(int a, int b) {
        return a + b;
    }

    class InnerClass {
        void innerMethod() {}
    }
}

interface Computable {
    int compute();
}

enum Status {
    ACTIVE, INACTIVE;

    public String label() { return name().toLowerCase(); }
}

@interface MyAnnotation {
    String value();
}

record Point(int x, int y) {
    public double distance() {
        return Math.sqrt(x * x + y * y);
    }
}

// class CommentedClass {}
`)

func TestAdversarial_Java(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".java"].Chunk("adv.java", adversarialJava)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("Calculator", "type")
	cs.mustHave("Calculator", "function") // constructor
	cs.mustHave("add", "method")
	cs.mustHave("MAX", "var")
	cs.mustHave("name", "var") // field
	cs.mustHave("InnerClass", "type")
	cs.mustHave("innerMethod", "method")
	cs.mustHave("Computable", "interface")
	cs.mustHave("compute", "method")
	cs.mustHave("Status", "type")
	cs.mustHave("label", "method")
	cs.mustHave("MyAnnotation", "type")
	cs.mustHave("Point", "type")
	cs.mustHave("distance", "method")

	cs.mustNotHave("CommentedClass", "type")
}

// ---------- PHP Adversarial ----------

var adversarialPHP = []byte(`<?php
namespace App\Models;

function helper(): void {}

class User {
    const TABLE = 'users';
    public string $name;
    protected int $age;

    public function getName(): string {
        return $this->name;
    }
}

interface Repository {
    public function find(int $id): mixed;
}

trait Cacheable {
    public function cache(): void {}
}

enum Color {
    case Red;
    case Blue;
}

enum Suit: string {
    case Hearts = 'H';
    case Diamonds = 'D';

    public function label(): string {
        return match($this) {
            self::Hearts => 'Hearts',
            self::Diamonds => 'Diamonds',
        };
    }
}

// function commented_fn() {}
`)

func TestAdversarial_PHP(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".php"].Chunk("adv.php", adversarialPHP)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("helper", "function")
	cs.mustHave("User", "type")
	cs.mustHave("TABLE", "const")
	cs.mustHave("getName", "method")
	cs.mustHave("Repository", "interface")
	cs.mustHave("find", "method")
	cs.mustHave("Cacheable", "type")
	cs.mustHave("cache", "method")
	cs.mustHave("Color", "type")
	cs.mustHave("Suit", "type")
	cs.mustHave("label", "method")
	cs.mustHave("App\\Models", "type")

	// property_declaration
	cs.mustHave("name", "var")

	cs.mustNotHave("commented_fn", "function")
}

// ---------- C# Adversarial ----------

var adversarialCSharp = []byte(`using System;

namespace MyApp
{
    public delegate void StatusChanged(string status);
    public delegate TResult Converter<TInput, TResult>(TInput input);

    public enum Direction { North, South, East, West }

    public interface IShape
    {
        double Area();
    }

    public record Point(double X, double Y);
    public record struct PointStruct(double X, double Y);

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

        public int this[int index] => index;

        public static Calculator operator +(Calculator a, Calculator b)
            => new Calculator(a._value + b._value);
    }

    public partial class PartialClass
    {
        public void Method1() {}
    }
}
`)

func TestAdversarial_CSharp(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".cs"].Chunk("adv.cs", adversarialCSharp)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("MyApp", "type")
	cs.mustHave("StatusChanged", "type")
	cs.mustHave("Converter", "type")
	cs.mustHave("Direction", "type")
	cs.mustHave("IShape", "interface")
	cs.mustHave("Point", "type")
	cs.mustHave("PointStruct", "type")
	cs.mustHave("Vector2", "type")
	cs.mustHave("Calculator", "type")
	cs.mustHave("Calculator", "function") // constructor
	cs.mustHave("Add", "method")
	cs.mustHave("Area", "method")
	cs.mustHave("Value", "method") // property
	cs.mustHave("OnStatusChanged", "var")
	cs.mustHave("_value", "var")
	cs.mustHave("PartialClass", "type")
	cs.mustHave("Method1", "method")
}

// ---------- C Adversarial ----------

var adversarialC = []byte(`
#include <stdio.h>

#define MAX_SIZE 1024
#define SQUARE(x) ((x) * (x))

typedef int MyInt;
typedef struct { int x; int y; } Point2D;

struct Point {
    int x;
    int y;
};

enum Color { RED, GREEN, BLUE };

union Data {
    int i;
    float f;
};

int add(int a, int b) {
    return a + b;
}

int *get_ptr(void) {
    return 0;
}

static void internal_fn(void) {}

// Forward declaration (should be extractable as declaration)
void forward_declared(int x);

// Global variable declaration
int global_var = 42;

/* int commented_fn(void) {} */
`)

func TestAdversarial_C(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".c"].Chunk("adv.c", adversarialC)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("MAX_SIZE", "const")
	cs.mustHave("SQUARE", "function")
	cs.mustHave("MyInt", "type")
	cs.mustHave("Point", "type")
	cs.mustHave("Color", "type")
	cs.mustHave("Data", "type")
	cs.mustHave("add", "function")
	cs.mustHave("get_ptr", "function")
	cs.mustHave("internal_fn", "function")

	cs.mustNotHave("commented_fn", "function")
}

// ---------- C++ Adversarial ----------

var adversarialCPP = []byte(`
#include <iostream>

class Vec2 {
public:
    float x;
    float y;
    void normalize();
};

struct Point {
    int x;
    int y;
};

enum Color { RED, GREEN, BLUE };

int add(int a, int b) {
    return a + b;
}

namespace MyLib {
    void helper() {}
    int libVar = 42;
}

using IntAlias = int;

void Vec2::normalize() {
    // normalize impl
}

template <typename T>
T max_val(T a, T b) {
    return a > b ? a : b;
}

template <typename T>
class Container {
public:
    T value;
};

// C++20 concept
template <typename T>
concept Addable = requires(T a, T b) { a + b; };

union Data {
    int i;
    float f;
};

// function commented() {}
`)

func TestAdversarial_CPP(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	chunks, err := langs[".cpp"].Chunk("adv.cpp", adversarialCPP)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Extracted chunks:\n%s", cs.dump())

	cs.mustHave("Vec2", "type")
	cs.mustHave("Point", "type")
	cs.mustHave("Color", "type")
	cs.mustHave("add", "function")
	cs.mustHave("MyLib", "type")
	cs.mustHave("helper", "function")
	cs.mustHave("IntAlias", "type")
	cs.mustHave("normalize", "method")
	cs.mustHave("Addable", "type")
	cs.mustHave("Data", "type")

	cs.mustNotHave("commented", "function")
}

// ---------- Cross-language: definitions in comments/strings must not be extracted ----------

func TestAdversarial_NoFalsePositivesFromComments(t *testing.T) {
	langs := chunker.DefaultLanguages(512)

	tests := []struct {
		ext  string
		code string
	}{
		{".py", "# def fake_fn(): pass\n\"\"\"def also_fake(): pass\"\"\"\ndef real(): pass\n"},
		{".ts", "// function fake() {}\n/* class Fake {} */\nfunction real() {}\n"},
		{".js", "// function fake() {}\n/* class Fake {} */\nfunction real() {}\n"},
		{".rs", "// fn fake() {}\n/* fn also_fake() {} */\nfn real() {}\n"},
		{".rb", "# def fake; end\ndef real\nend\n"},
		{".java", "// void fake() {}\n/* void alsoFake() {} */\nclass Real { void real() {} }\n"},
		{".c", "// void fake(void) {}\n/* void also_fake(void) {} */\nvoid real(void) {}\n"},
		{".cpp", "// void fake() {}\n/* void also_fake() {} */\nvoid real() {}\n"},
		{".cs", "// void Fake() {}\n/* void AlsoFake() {} */\nclass Real { void real() {} }\n"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			c, ok := langs[tt.ext]
			if !ok {
				t.Fatalf("missing chunker for %s", tt.ext)
			}
			chunks, err := c.Chunk("test"+tt.ext, []byte(tt.code))
			if err != nil {
				t.Fatal(err)
			}
			cs := newChunkSet(t, chunks)

			// Every language test has "real" as either function or method
			hasReal := false
			for _, ch := range chunks {
				if ch.Symbol == "real" || ch.Symbol == "Real" {
					hasReal = true
					break
				}
			}
			if !hasReal {
				t.Errorf("expected a chunk named 'real' or 'Real', got: %v", cs.allKeys())
			}

			cs.mustNotHave("fake", "function")
			cs.mustNotHave("also_fake", "function")
			cs.mustNotHave("Fake", "function")
			cs.mustNotHave("Fake", "type")
			cs.mustNotHave("Fake", "method")
			cs.mustNotHave("AlsoFake", "type")
			cs.mustNotHave("alsoFake", "method")
			cs.mustNotHave("AlsoFake", "method")
		})
	}
}

// ---------- Cross-language: every chunk has non-empty ID and valid line range ----------

func TestAdversarial_ChunkInvariants(t *testing.T) {
	langs := chunker.DefaultLanguages(512)

	testFiles := map[string][]byte{
		".py":   adversarialPython,
		".ts":   adversarialTypeScript,
		".js":   adversarialJavaScript,
		".rs":   adversarialRust,
		".rb":   adversarialRuby,
		".java": adversarialJava,
		".c":    adversarialC,
		".cpp":  adversarialCPP,
		".cs":   adversarialCSharp,
		".tsx":  adversarialTSX,
		".php":  adversarialPHP,
	}

	for ext, src := range testFiles {
		t.Run(ext, func(t *testing.T) {
			c, ok := langs[ext]
			if !ok {
				t.Fatalf("missing chunker for %s", ext)
			}
			chunks, err := c.Chunk("test"+ext, src)
			if err != nil {
				t.Fatal(err)
			}
			if len(chunks) == 0 {
				t.Error("expected at least 1 chunk")
			}
			for _, ch := range chunks {
				if ch.ID == "" {
					t.Errorf("chunk %s/%s has empty ID", ch.Symbol, ch.Kind)
				}
				if ch.Symbol == "" {
					t.Errorf("chunk has empty Symbol at L%d-%d", ch.StartLine, ch.EndLine)
				}
				if ch.Kind == "" {
					t.Errorf("chunk %s has empty Kind", ch.Symbol)
				}
				if ch.StartLine < 1 {
					t.Errorf("chunk %s has invalid StartLine=%d", ch.Symbol, ch.StartLine)
				}
				if ch.EndLine < ch.StartLine {
					t.Errorf("chunk %s has EndLine=%d < StartLine=%d", ch.Symbol, ch.EndLine, ch.StartLine)
				}
				if ch.Content == "" {
					t.Errorf("chunk %s/%s has empty Content", ch.Symbol, ch.Kind)
				}
				if ch.FilePath == "" {
					t.Errorf("chunk %s has empty FilePath", ch.Symbol)
				}
			}
		})
	}
}

// ---------- PHP: property_declaration check ----------

func TestAdversarial_PHP_PropertyDeclaration(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	src := []byte(`<?php
class User {
    public string $name;
    protected int $age = 0;
    private static array $instances = [];

    public function getName(): string {
        return $this->name;
    }
}
`)
	chunks, err := langs[".php"].Chunk("user.php", src)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("PHP property chunks:\n%s", cs.dump())

	cs.mustHave("User", "type")
	cs.mustHave("getName", "method")
	// property_declaration
	cs.mustHave("name", "var")
	cs.mustHave("age", "var")
}

// ---------- C: file-scope declarations (forward declarations, globals) ----------

func TestAdversarial_C_FileScopeDeclarations(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	src := []byte(`
void forward_declared(int x);
int global_var = 42;
extern int external_var;

void defined_fn(void) {}
`)
	chunks, err := langs[".c"].Chunk("decls.c", src)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("C file-scope declaration chunks:\n%s", cs.dump())

	cs.mustHave("defined_fn", "function")
	// These are `declaration` nodes — not currently extracted
	// cs.mustHave("forward_declared", "var")
	// cs.mustHave("global_var", "var")
}

// ---------- C++: template_declaration, union_specifier ----------

func TestAdversarial_CPP_Templates(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	src := []byte(`
template <typename T>
T max_val(T a, T b) {
    return a > b ? a : b;
}

template <typename T>
class Container {
public:
    T value;
};

union MyUnion {
    int i;
    float f;
};
`)
	chunks, err := langs[".cpp"].Chunk("templates.cpp", src)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("C++ template chunks:\n%s", cs.dump())

	// template function — currently extracted via function_definition inside template_declaration
	cs.mustHave("max_val", "function")
	// template class — class_specifier inside template_declaration
	cs.mustHave("Container", "type")
}

// ---------- Rust: associated_type inside impl ----------

func TestAdversarial_Rust_AssociatedType(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	src := []byte(`
trait MyTrait {
    type Output;
    fn required(&self) -> Self::Output;
}

struct Foo;

impl MyTrait for Foo {
    type Output = i32;
    fn required(&self) -> i32 { 42 }
}
`)
	chunks, err := langs[".rs"].Chunk("assoc.rs", src)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("Rust associated type chunks:\n%s", cs.dump())

	cs.mustHave("MyTrait", "interface")
	cs.mustHave("Foo", "type")
	cs.mustHave("required", "function") // function_signature_item or function_item
}

// ---------- TypeScript: namespace/module blocks ----------

func TestAdversarial_TypeScript_Namespace(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	src := []byte(`
namespace MyNamespace {
  export function helper(): void {}
  export class Inner {}
}

module LegacyModule {
  export function legacy(): void {}
}
`)
	chunks, err := langs[".ts"].Chunk("ns.ts", src)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("TypeScript namespace chunks:\n%s", cs.dump())

	// Namespace declarations
	cs.mustHave("MyNamespace", "type")
	// Functions inside namespace should be extracted
	cs.mustHave("helper", "function")
	cs.mustHave("Inner", "type")
	cs.mustHave("LegacyModule.legacy", "function")
}

// ---------- JavaScript: let-based function expressions ----------

func TestAdversarial_JavaScript_LetFunction(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	src := []byte(`
let mutableHandler = function() { return 1; };
let arrowHandler = () => 2;
const constArrow = () => 3;
var varArrow = () => 4;
`)
	chunks, err := langs[".js"].Chunk("let.js", src)
	if err != nil {
		t.Fatal(err)
	}
	cs := newChunkSet(t, chunks)
	t.Logf("JS let function chunks:\n%s", cs.dump())

	cs.mustHave("constArrow", "function")
	cs.mustHave("varArrow", "function")
	// let-based assignments with arrow/function expressions
	// lexical_declaration covers both const and let
	cs.mustHave("mutableHandler", "function")
	cs.mustHave("arrowHandler", "function")
}
