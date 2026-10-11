// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package chunker_test

import "testing"

func TestCPrototypesAndFunctionPointerTypedefs(t *testing.T) {
	src := `#include <stdlib.h>

/* Declared before use. */
void forward_declared(int x);
char *dup_name(const char *name);
typedef int (*CompareFn)(const void *, const void *);

void forward_declared(int x) {
    int local(int);
    (void)x;
}
`
	for _, path := range []string{"records.c", "records.h"} {
		t.Run(path, func(t *testing.T) {
			chunks := chunkSource(t, path, src)
			assertDecls(t, chunks, []wantDecl{
				{symbol: "dup_name", kind: "function", start: 5, end: 5},
				{symbol: "CompareFn", kind: "type", start: 6, end: 6},
			})
			var protos, defs int
			for _, ch := range chunks {
				if ch.Symbol == "forward_declared" {
					if ch.StartLine == 3 && ch.EndLine == 4 {
						protos++
					}
					if ch.StartLine == 8 && ch.EndLine == 11 {
						defs++
					}
				}
			}
			if protos != 1 || defs != 1 {
				t.Errorf("forward_declared: got %d prototypes and %d definitions, want 1 and 1:%s", protos, defs, describeChunks(chunks))
			}
			assertNoSymbol(t, chunks, "local", "forward_declared.local")
		})
	}
}

func TestCPPHeaderPrototypesInsideIncludeGuard(t *testing.T) {
	src := `#ifndef NET_H
#define NET_H

int open_socket(const char *host, int port);

#endif
`
	chunks := chunkSource(t, "net.h", src)
	assertDecls(t, chunks, []wantDecl{{symbol: "open_socket", kind: "function", start: 4, end: 4}})
}

func TestCPPOperatorsDestructorsAndQualifiedDefinitions(t *testing.T) {
	src := `struct Vec2 {
    bool operator==(const Vec2& other) const;
    float length() const;
    void reset() { x = 0; }
    ~Vec2();
};

bool Vec2::operator==(const Vec2& other) const {
    return x == other.x;
}

bool operator!=(const Vec2& a, const Vec2& b) {
    return !(a == b);
}

SocketHandle::~SocketHandle() {
    close(fd_);
}

std::string codec::hex::decode(const std::string& s) {
    return s;
}

const Header& Message::header() const {
    return header_;
}
`
	chunks := chunkSource(t, "vec2.cpp", src)
	// Out-of-line definitions keep the existing convention of naming the
	// declarator's final component; in-class members are qualified by their class.
	assertDecls(t, chunks, []wantDecl{
		{symbol: "Vec2.operator==", kind: "method", start: 2, end: 2},
		{symbol: "Vec2.length", kind: "method", start: 3, end: 3},
		{symbol: "Vec2.reset", kind: "method", start: 4, end: 4},
		{symbol: "Vec2.~Vec2", kind: "method", start: 5, end: 5},
		{symbol: "operator==", kind: "method", start: 8, end: 10, contains: []string{"return x == other.x;"}},
		{symbol: "operator!=", kind: "function", start: 12, end: 14},
		{symbol: "~SocketHandle", kind: "method", start: 16, end: 18},
		{symbol: "decode", kind: "method", start: 20, end: 22},
		{symbol: "header", kind: "method", start: 24, end: 26},
	})
}
