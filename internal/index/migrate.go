// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/ory/lumen/internal/merkle"
	"github.com/ory/lumen/internal/store"
)

// PrepareLegacyMigration recovers vectors for unchanged chunks from a legacy
// per-worktree database. It is safe to ignore a missing source.
func (idx *Indexer) PrepareLegacyMigration(projectDir, legacyPath string) error {
	if legacyPath == "" || legacyPath == idx.dsn {
		return nil
	}
	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	db, err := sql.Open("sqlite3", legacyPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`PRAGMA query_only=ON`); err != nil {
		return err
	}

	legacyByChunk := make(map[string][]float32)
	rows, err := db.Query(`SELECT c.id, v.embedding FROM chunks c JOIN vec_chunks v ON v.id = c.id`)
	if err != nil {
		return fmt.Errorf("read legacy vectors: %w", err)
	}
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			_ = rows.Close()
			return err
		}
		vector, ok := decodeFloat32Vector(blob, idx.emb.Dimensions())
		if ok {
			legacyByChunk[id] = vector
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}

	fileRows, err := db.Query(`SELECT path, hash FROM files WHERE hash <> ''`)
	if err != nil {
		return fmt.Errorf("read legacy files: %w", err)
	}
	recovered := make(map[[32]byte][]float32)
	for fileRows.Next() {
		var relativePath, storedHash string
		if err := fileRows.Scan(&relativePath, &storedHash); err != nil {
			_ = fileRows.Close()
			return err
		}
		content, err := os.ReadFile(filepath.Join(projectDir, relativePath))
		if err != nil {
			continue
		}
		contentSum := sha256.Sum256(content)
		if hex.EncodeToString(contentSum[:]) != storedHash {
			continue
		}
		chunks, err := idx.chunker.Chunk(relativePath, content)
		if err != nil {
			continue
		}
		chunks = splitOversizedChunks(chunks, idx.maxChunkTokens)
		chunks = mergeUndersizedChunks(chunks)
		chunks = splitOversizedChunks(chunks, idx.maxChunkTokens)
		for _, chunk := range chunks {
			if vector, ok := legacyByChunk[chunk.ID]; ok {
				h := sha256.Sum256([]byte(store.EmbeddingInput(chunk)))
				recovered[h] = vector
			}
		}
	}
	if err := fileRows.Close(); err != nil {
		return err
	}
	idx.legacyVectors = recovered
	idx.legacySource = legacyPath
	return nil
}

func decodeFloat32Vector(blob []byte, dimensions int) ([]float32, bool) {
	if len(blob) != dimensions*4 {
		return nil, false
	}
	vector := make([]float32, dimensions)
	for i := range vector {
		bits := binary.LittleEndian.Uint32(blob[i*4 : i*4+4])
		vector[i] = math.Float32frombits(bits)
	}
	return vector, true
}

func (idx *Indexer) finishLegacyMigration(tree *merkle.Tree) {
	if idx.legacySource == "" {
		return
	}
	hashes, err := idx.store.GetFileHashes()
	if err != nil || len(hashes) != len(tree.Files) {
		return
	}
	for path, hash := range tree.Files {
		if hashes[path] != hash {
			return
		}
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(idx.legacySource + suffix)
	}
	_ = os.Remove(filepath.Dir(idx.legacySource))
	idx.legacySource = ""
	idx.legacyVectors = nil
}
