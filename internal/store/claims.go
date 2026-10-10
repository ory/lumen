// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const embeddingLease = 2 * time.Minute

// EmbedShared claims exact inputs before computing them. Completed results are
// handed to concurrent worktrees through a short-lived, bounded rendezvous
// cache; durable vector ownership still belongs to file revisions.
func (s *Store) EmbedShared(ctx context.Context, texts []string, compute func(context.Context, []string) ([][]float32, error)) ([][]float32, error) {
	owner := rand.Text()
	defer func() { _, _ = s.db.Exec(`DELETE FROM embedding_claims WHERE owner=? AND result IS NULL`, owner) }()
	output := make([][]float32, len(texts))
	positions := map[[32]byte][]int{}
	for i, text := range texts {
		h := sha256.Sum256([]byte(text))
		positions[h] = append(positions[h], i)
	}
	for len(positions) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var hashes [][32]byte
		var inputs []string
		for hash, indices := range positions {
			now := time.Now().UnixMilli()
			res, err := s.db.ExecContext(ctx, `INSERT INTO embedding_claims(input_hash,owner,expires_at) VALUES (?,?,?)
    ON CONFLICT(input_hash) DO UPDATE SET owner=excluded.owner,expires_at=excluded.expires_at,result=NULL
    WHERE embedding_claims.expires_at<=?`, hash[:], owner, now+embeddingLease.Milliseconds(), now)
			if err != nil {
				return nil, err
			}
			claimed, err := res.RowsAffected()
			if err != nil {
				return nil, err
			}
			if claimed > 0 {
				hashes = append(hashes, hash)
				inputs = append(inputs, texts[indices[0]])
				continue
			}
			var blob []byte
			err = s.reader().QueryRowContext(ctx, `SELECT result FROM embedding_claims WHERE input_hash=? AND result IS NOT NULL AND expires_at>?`, hash[:], now).Scan(&blob)
			if err == sql.ErrNoRows {
				continue
			}
			if err != nil {
				return nil, err
			}
			var vector []float32
			if err := json.Unmarshal(blob, &vector); err != nil {
				return nil, fmt.Errorf("decode shared embedding: %w", err)
			}
			if len(vector) != s.dimensions {
				return nil, fmt.Errorf("shared embedding dimensions mismatch")
			}
			for _, i := range indices {
				output[i] = vector
			}
			delete(positions, hash)
		}
		if len(inputs) > 0 {
			vectors, err := s.computeClaimed(ctx, owner, inputs, compute)
			if err != nil {
				return nil, err
			}
			if len(vectors) != len(inputs) {
				return nil, fmt.Errorf("embed returned %d vectors for %d inputs", len(vectors), len(inputs))
			}
			for i, hash := range hashes {
				if len(vectors[i]) != s.dimensions {
					return nil, fmt.Errorf("embedding dimensions mismatch")
				}
				blob, err := json.Marshal(vectors[i])
				if err != nil {
					return nil, err
				}
				now := time.Now().UnixMilli()
				res, err := s.db.ExecContext(ctx, `UPDATE embedding_claims SET result=?,expires_at=? WHERE input_hash=? AND owner=? AND expires_at>?`, blob, now+(10*time.Minute).Milliseconds(), hash[:], owner, now)
				if err != nil {
					return nil, err
				}
				n, err := res.RowsAffected()
				if err != nil {
					return nil, err
				}
				if n != 1 {
					return nil, fmt.Errorf("embedding lease expired before publication")
				}
				for _, position := range positions[hash] {
					output[position] = vectors[i]
				}
				delete(positions, hash)
			}
			if _, err := s.db.ExecContext(ctx, `DELETE FROM embedding_claims WHERE expires_at<=? OR input_hash IN (
    SELECT input_hash FROM (SELECT input_hash, sum(length(result)) OVER (ORDER BY expires_at DESC,input_hash) AS bytes
    FROM embedding_claims WHERE result IS NOT NULL) WHERE bytes>?)`, time.Now().UnixMilli(), 64<<20); err != nil {
				return nil, err
			}
		} else if len(positions) > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
	return output, nil
}

func (s *Store) computeClaimed(ctx context.Context, owner string, inputs []string, compute func(context.Context, []string) ([][]float32, error)) ([][]float32, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	stopped := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(embeddingLease / 4)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				stopped <- nil
				return
			case <-ctx.Done():
				stopped <- ctx.Err()
				return
			case <-ticker.C:
				now := time.Now().UnixMilli()
				if _, err := s.db.ExecContext(ctx, `UPDATE embedding_claims SET expires_at=? WHERE owner=? AND result IS NULL AND expires_at>?`, now+embeddingLease.Milliseconds(), owner, now); err != nil {
					cancel()
					stopped <- err
					return
				}
			}
		}
	}()
	vectors, err := compute(ctx, inputs)
	close(done)
	renewalErr := <-stopped
	if err != nil {
		return nil, err
	}
	if renewalErr != nil {
		return nil, renewalErr
	}
	return vectors, nil
}
