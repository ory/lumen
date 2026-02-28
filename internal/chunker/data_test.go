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

func TestDataChunker_YAML(t *testing.T) {
	c := NewDataChunker()
	input := []byte("name: foo\nversion: 1\n")
	chunks, err := c.Chunk("test.yaml", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Symbol != "root" {
		t.Errorf("Symbol = %q, want root", chunks[0].Symbol)
	}
	if chunks[0].Kind != "document" {
		t.Errorf("Kind = %q, want document", chunks[0].Kind)
	}
}

func TestDataChunker_JSON(t *testing.T) {
	c := NewDataChunker()
	input := []byte(`{"name":"foo","version":"1"}`)
	chunks, err := c.Chunk("test.json", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Symbol != "root" {
		t.Errorf("Symbol = %q, want root", chunks[0].Symbol)
	}
	if chunks[0].Kind != "document" {
		t.Errorf("Kind = %q, want document", chunks[0].Kind)
	}
}

func TestDataChunker_Empty(t *testing.T) {
	c := NewDataChunker()
	chunks, err := c.Chunk("test.yaml", []byte("   "))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(chunks))
	}
}
