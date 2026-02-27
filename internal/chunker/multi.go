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
