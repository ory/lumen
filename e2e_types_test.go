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

// This file intentionally has no e2e build tag because the snapshot matcher
// unit tests share the black-box search result type without requiring Ollama.

package main

const langSnapshotDirectory = "testdata/snapshots"

// searchResultItem is shared by the black-box E2E tests and the snapshot
// matcher unit tests. It mirrors the semantic search tool's result shape while
// remaining decoupled from the production command types.
type searchResultItem struct {
	FilePath  string  `json:"file_path"`
	Symbol    string  `json:"symbol"`
	Kind      string  `json:"kind"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
	Content   string  `json:"content,omitempty"`
}
