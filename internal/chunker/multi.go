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

import "path/filepath"

// MultiChunker dispatches to per-extension Chunkers.
// Files with unrecognized extensions return nil, nil.
type MultiChunker struct {
	chunkers map[string]Chunker
}

// NewMultiChunker creates a MultiChunker from a map of extension → Chunker.
// Extensions must include the leading dot (e.g. ".go", ".py").
func NewMultiChunker(chunkers map[string]Chunker) *MultiChunker {
	return &MultiChunker{chunkers: chunkers}
}

// Chunk dispatches to the appropriate Chunker based on file extension.
// Returns nil, nil for unsupported extensions.
func (m *MultiChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	ext := filepath.Ext(filePath)
	c, ok := m.chunkers[ext]
	if !ok {
		return nil, nil
	}
	return c.Chunk(filePath, content)
}
