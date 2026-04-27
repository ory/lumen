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

// sampleSwift is a representative Swift source file covering:
//   - top-level functions
//   - class with methods, properties, and static methods
//   - struct with computed properties
//   - enum with cases and methods
//   - protocol with function declarations and associated types
//   - actor with async methods
//   - extension with protocol conformance
//   - typealias
//   - top-level properties (let/var)
var sampleSwift = []byte(`import Foundation

/// A user account in the system.
public class UserAccount {
    var name: String
    let id: UUID

    func greet() -> String {
        return "Hello, \(name)"
    }

    static func create(name: String) -> UserAccount {
        return UserAccount(name: name)
    }
}

struct Point {
    var x: Double
    var y: Double

    func distance(to other: Point) -> Double {
        return sqrt(pow(x - other.x, 2) + pow(y - other.y, 2))
    }
}

enum Direction {
    case north
    case south
    case east
    case west

    func opposite() -> Direction {
        switch self {
        case .north: return .south
        case .south: return .north
        case .east: return .west
        case .west: return .east
        }
    }
}

protocol Drawable {
    var color: String { get set }
    func draw()
    associatedtype Canvas
}

actor Counter {
    var count: Int = 0

    func increment() {
        count += 1
    }
}

extension Point: Drawable {
    var color: String {
        get { return "red" }
        set { }
    }
    func draw() { }
    typealias Canvas = String
}

func topLevelFunction(x: Int) -> Int {
    return x * 2
}

let globalConstant = 42
var globalVariable = "hello"

typealias Callback = (Int) -> Void
`)

func TestSwiftChunker_Symbols(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".swift"]
	if !ok {
		t.Fatal("no chunker registered for .swift")
	}

	chunks, err := c.Chunk("Sources/App/UserAccount.swift", sampleSwift)
	if err != nil {
		t.Fatalf("Chunk: %v", err)
	}

	bySymbol := make(map[string]chunker.Chunk)
	for _, ch := range chunks {
		bySymbol[ch.Symbol] = ch
	}

	// Log all symbols for debugging.
	t.Logf("Found %d chunks:", len(chunks))
	for _, ch := range chunks {
		t.Logf("  %s (%s) [%d-%d]", ch.Symbol, ch.Kind, ch.StartLine, ch.EndLine)
	}

	// Types: class, struct, enum, actor.
	checkChunk(t, bySymbol, "UserAccount", "type", 0, 0, "Sources/App/UserAccount.swift", "class UserAccount")
	checkChunk(t, bySymbol, "Direction", "type", 0, 0, "Sources/App/UserAccount.swift", "enum Direction")
	checkChunk(t, bySymbol, "Counter", "type", 0, 0, "Sources/App/UserAccount.swift", "actor Counter")

	// Point appears twice (struct + extension), verify both exist.
	pointChunks := 0
	for _, ch := range chunks {
		if ch.Symbol == "Point" && ch.Kind == "type" {
			pointChunks++
		}
	}
	if pointChunks < 2 {
		t.Errorf("expected at least 2 Point chunks (struct + extension), got %d", pointChunks)
	}

	// Protocol.
	checkChunk(t, bySymbol, "Drawable", "interface", 0, 0, "Sources/App/UserAccount.swift", "protocol Drawable")

	// Functions (top-level).
	checkChunk(t, bySymbol, "topLevelFunction", "function", 0, 0, "Sources/App/UserAccount.swift", "func topLevelFunction")

	// Methods inside types (unqualified — same convention as TS/JS/Java).
	if _, ok := bySymbol["greet"]; !ok {
		t.Error("expected 'greet' in symbols")
	}
	if _, ok := bySymbol["distance"]; !ok {
		t.Error("expected 'distance' in symbols")
	}
	if _, ok := bySymbol["opposite"]; !ok {
		t.Error("expected 'opposite' in symbols")
	}
	if _, ok := bySymbol["increment"]; !ok {
		t.Error("expected 'increment' in symbols")
	}

	// Protocol function declarations.
	if _, ok := bySymbol["draw"]; !ok {
		t.Error("expected 'draw' in symbols (from protocol or extension)")
	}

	// Typealias (top-level).
	checkChunk(t, bySymbol, "Callback", "type", 0, 0, "Sources/App/UserAccount.swift", "typealias Callback")

	// Associated type in protocol.
	if _, ok := bySymbol["Canvas"]; !ok {
		t.Error("expected 'Canvas' in symbols")
	}

	// Enum cases.
	if _, ok := bySymbol["north"]; !ok {
		t.Error("expected 'north' enum case in symbols")
	}

	// Properties.
	if _, ok := bySymbol["name"]; !ok {
		t.Error("expected 'name' property in symbols")
	}
	if _, ok := bySymbol["globalConstant"]; !ok {
		t.Error("expected 'globalConstant' in symbols")
	}
	if _, ok := bySymbol["globalVariable"]; !ok {
		t.Error("expected 'globalVariable' in symbols")
	}

	// All chunks should have valid line numbers.
	for _, ch := range chunks {
		if ch.StartLine < 1 {
			t.Errorf("chunk %s: StartLine=%d should be >= 1", ch.Symbol, ch.StartLine)
		}
		if ch.EndLine < ch.StartLine {
			t.Errorf("chunk %s: EndLine=%d < StartLine=%d", ch.Symbol, ch.EndLine, ch.StartLine)
		}
	}
}

func TestSwiftChunker_NoSymbolsCases(t *testing.T) {
	langs := chunker.DefaultLanguages(512)
	c, ok := langs[".swift"]
	if !ok {
		t.Fatal("no chunker registered for .swift")
	}

	cases := []struct {
		name    string
		path    string
		content []byte
	}{
		{
			name:    "empty file",
			path:    "Empty.swift",
			content: []byte(``),
		},
		{
			name: "imports only",
			path: "ImportsOnly.swift",
			content: []byte(`import Foundation
import UIKit
`),
		},
		{
			name: "comments only",
			path: "Comments.swift",
			content: []byte(`// This is a comment
/* Block comment */
/// Doc comment
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
				for _, ch := range chunks {
					t.Logf("  unexpected: %s (%s)", ch.Symbol, ch.Kind)
				}
			}
		})
	}
}
