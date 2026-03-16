// Package summarizer generates natural-language summaries of code chunks and
// files using LLM chat completion APIs (Ollama and LM Studio).
package summarizer

import (
	"context"
	"encoding/json"
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
	// SummarizeChunk returns a 2-3 sentence summary of a single code chunk.
	SummarizeChunk(ctx context.Context, chunk ChunkInfo) (string, error)
	// SummarizeChunks returns one summary per input chunk in the same order.
	// Implementations should batch the LLM call where possible.
	SummarizeChunks(ctx context.Context, chunks []ChunkInfo) ([]string, error)
	// SummarizeFile returns a 3-5 sentence summary of a file from its chunk summaries.
	SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error)
}

// SummarizeChunksByOne implements SummarizeChunks via sequential SummarizeChunk
// calls. Used as a fallback when a batch LLM call fails or returns the wrong
// number of results.
func SummarizeChunksByOne(ctx context.Context, s Summarizer, chunks []ChunkInfo) ([]string, error) {
	out := make([]string, len(chunks))
	for i, c := range chunks {
		summary, err := s.SummarizeChunk(ctx, c)
		if err != nil {
			return nil, err
		}
		out[i] = summary
	}
	return out, nil
}

// chunkPrompt returns the LLM prompt for summarizing a single chunk.
func chunkPrompt(chunk ChunkInfo) string {
	return fmt.Sprintf(
		"Summarize what this %s '%s' does in 2-3 sentences, focusing on its purpose and behavior:\n\n%s",
		chunk.Kind, chunk.Symbol, chunk.Content,
	)
}

// batchChunkPrompt returns a prompt asking the LLM to summarize all chunks at
// once and respond with {"summaries":["...","...",...]}. Using JSON format
// (enforced by the backend) ensures a machine-readable response.
func batchChunkPrompt(chunks []ChunkInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b,
		`Summarize each of the following %d code chunks.
Return ONLY a JSON object: {"summaries":["summary1","summary2",...]} with exactly %d strings in the same order.
Each summary must be 2-3 sentences focusing on purpose and behavior. No extra text outside the JSON.`,
		len(chunks), len(chunks))
	for i, c := range chunks {
		fmt.Fprintf(&b, "\n\n### Chunk %d (%s '%s')\n%s", i+1, c.Kind, c.Symbol, c.Content)
	}
	return b.String()
}

// parseBatchSummaries parses the {"summaries":[...]} JSON produced by a batch
// prompt. Returns nil if the response is malformed or has the wrong count.
func parseBatchSummaries(raw string, wantCount int) []string {
	raw = strings.TrimSpace(raw)
	// Trim common model preamble/postamble outside the JSON object.
	if i := strings.Index(raw, "{"); i > 0 {
		raw = raw[i:]
	}
	if i := strings.LastIndex(raw, "}"); i >= 0 && i < len(raw)-1 {
		raw = raw[:i+1]
	}
	var resp struct {
		Summaries []string `json:"summaries"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil
	}
	if len(resp.Summaries) != wantCount {
		return nil
	}
	return resp.Summaries
}

// filePrompt returns the LLM prompt for summarizing a file from its chunk summaries.
func filePrompt(chunkSummaries []string) string {
	return fmt.Sprintf(
		"Summarize what this file does in 3-5 sentences, covering its main purpose, key types/functions, and role in the codebase:\n\n%s",
		strings.Join(chunkSummaries, "\n"),
	)
}
