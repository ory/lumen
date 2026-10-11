// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmbeddingClaimsShareConcurrentWork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	a, err := NewCollection(path, 4, "int8", "/a")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	b, err := NewCollection(path, 4, "int8", "/b")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var calls atomic.Int32
	started := make(chan struct{})
	unblock := make(chan struct{})
	compute := func(ctx context.Context, texts []string) ([][]float32, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		select {
		case <-unblock:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return [][]float32{{1, 0, 0, 0}}, nil
	}
	done := make(chan error, 2)
	run := func(s *Store) {
		vectors, err := s.EmbedShared(ctx, []string{"same", "same"}, compute)
		if err == nil && (len(vectors) != 2 || vectors[1][0] != 1) {
			err = errors.New("incorrect duplicate results")
		}
		done <- err
	}
	go run(a)
	<-started
	go run(b)
	close(unblock)
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("computed %d times", calls.Load())
	}
}

func TestEmbeddingClaimsRecoverAndCancel(t *testing.T) {
	s, err := NewCollection(":memory:", 4, "float32", "/a")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	hash := sha256.Sum256([]byte("input"))
	if _, err := s.db.Exec(`INSERT INTO embedding_claims(input_hash,owner,expires_at) VALUES (?,?,?)`, hash[:], "dead", time.Now().Add(-time.Second).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("backend failed")
	_, err = s.EmbedShared(context.Background(), []string{"input"}, func(context.Context, []string) ([][]float32, error) { return nil, failure })
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM embedding_claims`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed owner stranded its claim")
	}
	if _, err := s.db.Exec(`INSERT INTO embedding_claims(input_hash,owner,expires_at) VALUES (?,?,?)`, hash[:], "busy", time.Now().Add(time.Minute).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = s.EmbedShared(ctx, []string{"input"}, func(context.Context, []string) ([][]float32, error) {
		t.Fatal("computed another owner's input")
		return nil, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancellation: %v", err)
	}
}
