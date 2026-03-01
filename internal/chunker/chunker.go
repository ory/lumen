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

// Package chunker splits source files into semantically meaningful code chunks.
package chunker

// Chunk represents a semantically meaningful piece of source code.
type Chunk struct {
	ID        string // deterministic: sha256(filePath + symbol + startLine)[:16]
	FilePath  string // relative to project root
	Symbol    string // "FuncName", "TypeName.MethodName"
	Kind      string // "function", "method", "type", "interface", "const", "var", "package"
	StartLine int
	EndLine   int
	Content   string // raw source text, used for embedding
}

// Chunker splits source files into semantically meaningful chunks.
type Chunker interface {
	Chunk(filePath string, content []byte) ([]Chunk, error)
}
