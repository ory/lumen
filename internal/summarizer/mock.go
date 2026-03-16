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

package summarizer

import (
	"context"
	"fmt"
)

// MockSummarizer returns deterministic, template-based summaries for testing.
// It is used when LUMEN_SUMMARY_MODEL="_mock" to avoid requiring a real LLM.
type MockSummarizer struct{}

// SummarizeChunk returns a deterministic summary for a code chunk.
func (m *MockSummarizer) SummarizeChunk(_ context.Context, chunk ChunkInfo) (string, error) {
	return fmt.Sprintf("This %s '%s' handles code operations.", chunk.Kind, chunk.Symbol), nil
}

// SummarizeChunks returns a deterministic summary for each chunk.
func (m *MockSummarizer) SummarizeChunks(ctx context.Context, chunks []ChunkInfo) ([]string, error) {
	return SummarizeChunksByOne(ctx, m, chunks)
}

// SummarizeFile returns a deterministic summary based on the number of chunk summaries.
func (m *MockSummarizer) SummarizeFile(_ context.Context, chunkSummaries []string) (string, error) {
	return fmt.Sprintf("This file contains %d code elements.", len(chunkSummaries)), nil
}
