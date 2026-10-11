package cmd

import "testing"

func TestResultsSlug(t *testing.T) {
	tests := []struct {
		retrieval, backend, model, want string
	}{
		{"bm25", "ollama", "ordis/jina-embeddings-v2-base-code", "bm25"},
		{"semantic", "ollama", "ordis/jina-embeddings-v2-base-code", "ollama-jina-embeddings-v2-base-code"},
	}
	for _, tt := range tests {
		if got := resultsSlug(tt.retrieval, tt.backend, tt.model); got != tt.want {
			t.Errorf("resultsSlug(%q) = %q, want %q", tt.retrieval, got, tt.want)
		}
	}
}
