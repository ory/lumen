//go:build repro

package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ory/lumen/internal/chunker"
	"github.com/ory/lumen/internal/embedder"
)

func TestReproContextLength(t *testing.T) {
	projectDir := "/Users/aeneas/workspace/go/agent-index-go"
	ch := chunker.NewMultiChunker(chunker.DefaultLanguages(100))

	var allChunks []chunker.Chunk
	_ = filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "testdata" || base == "vendor" || base == ".git" || base == ".claude" || base == ".worktrees" || base == "bin" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		valid := false
		for _, e := range chunker.SupportedExtensions() {
			if ext == e { valid = true; break }
		}
		if !valid {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, _ := filepath.Rel(projectDir, path)
		chunks, chunkErr := ch.Chunk(rel, content)
		if chunkErr != nil {
			return nil
		}
		allChunks = append(allChunks, chunks...)
		return nil
	})

	allChunks = splitOversizedChunks(allChunks, 100)
	allChunks = mergeUndersizedChunks(allChunks, minMergeTokens)
	allChunks = splitOversizedChunks(allChunks, 100) // re-split after merge

	t.Logf("Total chunks: %d", len(allChunks))

	// Find the largest chunk
	maxLen := 0
	for i, c := range allChunks {
		text := "// " + c.FilePath + "\n" + c.Content
		if len(text) > maxLen {
			maxLen = len(text)
		}
		if len(text) > 400 {
			t.Logf("Chunk %d exceeds 400 chars: file=%s symbol=%s kind=%s len=%d", i, c.FilePath, c.Symbol, c.Kind, len(text))
		}
	}
	t.Logf("Max chunk text length: %d chars", maxLen)

	emb, err := embedder.NewOllama("all-minilm", 384, 512, "http://localhost:11434")
	if err != nil {
		t.Fatal(err)
	}

	const batchSize = 32
	for i := 0; i < len(allChunks); i += batchSize {
		end := i + batchSize
		if end > len(allChunks) {
			end = len(allChunks)
		}
		batch := allChunks[i:end]
		texts := make([]string, len(batch))
		for j, c := range batch {
			texts[j] = "// " + c.FilePath + "\n" + c.Content
		}
		_, err := emb.Embed(context.Background(), texts)
		if err != nil {
			// Find which individual chunk fails
			for j, text := range texts {
				if _, singleErr := emb.Embed(context.Background(), []string{text}); singleErr != nil {
					t.Logf("Individual chunk %d fails: file=%s symbol=%s kind=%s len=%d", i+j, batch[j].FilePath, batch[j].Symbol, batch[j].Kind, len(text))
				}
			}
			t.Fatalf("Batch %d-%d failed: %v", i, end-1, err)
		}
	}
	t.Log("All batches succeeded!")
}
