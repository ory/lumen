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

func TestTreeSitterChunker_NoDuplicateChunks(t *testing.T) {
	c1 := Chunk{FilePath: "f.go", StartLine: 1, EndLine: 5, Kind: "function", Symbol: "A"}
	c2 := Chunk{FilePath: "f.go", StartLine: 1, EndLine: 5, Kind: "method", Symbol: "A"}
	result := deduplicateByExactRange([]Chunk{c1, c2})
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk after dedup, got %d", len(result))
	}
	if result[0].Kind != "method" {
		t.Errorf("expected kind=method (later wins), got %s", result[0].Kind)
	}
}
