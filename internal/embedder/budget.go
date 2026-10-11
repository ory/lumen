// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package embedder

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

type backgroundKey struct{}

// BackgroundContext marks indexing batches as lower priority than queries.
func BackgroundContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, backgroundKey{}, true)
}

// acquireBudget serializes backend requests across processes for this user.
// Query intent is a shared OS lock, so process death cannot strand priority.
// Background callers recheck intent after acquiring capacity to avoid jumping
// ahead of queries queued behind an already-running request.
func acquireBudget(ctx context.Context, dir string) (func(), error) {
	if dir == "" {
		return func() {}, nil
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	capacity := flock.New(filepath.Join(dir, "compute.lock"))
	intent := flock.New(filepath.Join(dir, "query.lock"))
	closeAll := func() { _ = capacity.Close(); _ = intent.Close() }
	background, _ := ctx.Value(backgroundKey{}).(bool)
	if !background {
		ok, err := intent.TryRLockContext(ctx, 10*time.Millisecond)
		if err != nil || !ok {
			closeAll()
			if err == nil {
				err = ctx.Err()
			}
			return nil, err
		}
		ok, err = capacity.TryLockContext(ctx, 10*time.Millisecond)
		if err != nil || !ok {
			closeAll()
			if err == nil {
				err = ctx.Err()
			}
			return nil, err
		}
		return closeAll, nil
	}
	for {
		if err := ctx.Err(); err != nil {
			closeAll()
			return nil, err
		}
		available, err := intent.TryLock()
		if err != nil {
			closeAll()
			return nil, err
		}
		if available {
			_ = intent.Unlock()
			acquired, err := capacity.TryLock()
			if err != nil {
				closeAll()
				return nil, err
			}
			if acquired {
				available, err = intent.TryLock()
				if err != nil {
					closeAll()
					return nil, err
				}
				if available {
					_ = intent.Unlock()
					return closeAll, nil
				}
				_ = capacity.Unlock()
			}
		}
		select {
		case <-ctx.Done():
			closeAll()
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// SetComputeBudget enables a shared on-disk compute budget for this embedder.
// Configure it before use. An empty path disables coordination for tests.
func (f *FailoverEmbedder) SetComputeBudget(dir string) { f.budgetDir = dir }
