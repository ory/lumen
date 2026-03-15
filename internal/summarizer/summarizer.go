// Package summarizer generates natural-language summaries of code chunks and
// files using LLM chat completion APIs (Ollama and LM Studio).
package summarizer

import (
	"context"
	"fmt"
	"strings"
)

// ChunkInfo carries the fields needed to summarize a code chunk.
// It is intentionally decoupled from chunker.Chunk to keep the package
// dependency graph clean.
type ChunkInfo struct {
	Kind    string
	Symbol  string
	Content string
}

// Summarizer generates natural-language summaries of code.
type Summarizer interface {
	SummarizeChunk(ctx context.Context, chunk ChunkInfo) (string, error)
	SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error)
}

// chunkPrompt returns the LLM prompt for summarizing a single chunk.
func chunkPrompt(chunk ChunkInfo) string {
	return fmt.Sprintf(
		"Summarize what this %s '%s' does in 2-3 sentences, focusing on its purpose and behavior:\n\n%s",
		chunk.Kind, chunk.Symbol, chunk.Content,
	)
}

// filePrompt returns the LLM prompt for summarizing a file from its chunk summaries.
func filePrompt(chunkSummaries []string) string {
	return fmt.Sprintf(
		"Summarize what this file does in 3-5 sentences, covering its main purpose, key types/functions, and role in the codebase:\n\n%s",
		strings.Join(chunkSummaries, "\n"),
	)
}
