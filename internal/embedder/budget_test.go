// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package embedder

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func TestBudgetHonorsQueryIntentAndCancellation(t *testing.T) {
	dir := t.TempDir()
	intent := flock.New(filepath.Join(dir, "query.lock"))
	if ok, err := intent.TryRLock(); err != nil || !ok {
		t.Fatalf("intent: %v %v", ok, err)
	}
	defer func() { _ = intent.Close() }()
	ctx, cancel := context.WithTimeout(BackgroundContext(context.Background()), 40*time.Millisecond)
	defer cancel()
	if release, err := acquireBudget(ctx, dir); err == nil {
		release()
		t.Fatal("background request jumped ahead of query")
	}
	if err := intent.Unlock(); err != nil {
		t.Fatal(err)
	}
	release, err := acquireBudget(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel2()
	if other, err := acquireBudget(ctx2, dir); err == nil {
		other()
		t.Fatal("budget admitted overlapping requests")
	}
	release()
	next, err := acquireBudget(BackgroundContext(context.Background()), dir)
	if err != nil {
		t.Fatal(err)
	}
	next()
}
