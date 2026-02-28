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

import "strings"

// DataChunker emits the entire file as a single chunk, relying on the
// splitOversizedChunks pipeline to break it into line-boundary pieces.
type DataChunker struct{}

// NewDataChunker returns a new DataChunker.
func NewDataChunker() *DataChunker { return &DataChunker{} }

// Chunk implements Chunker. Emits the whole file as a single "document" chunk.
func (c *DataChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return nil, nil
	}
	lines := strings.Count(trimmed, "\n") + 1
	return []Chunk{makeChunk(filePath, "root", "document", 1, lines, trimmed)}, nil
}
