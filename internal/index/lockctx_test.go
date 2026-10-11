// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package index

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A search that must switch the shared collection to another worktree waits
// for in-flight work on the current one. That wait must honor cancellation.
func TestSearchLexicalLockWaitHonorsCancellation(t *testing.T) {
	idx, root := lexicalFixture(t, map[string]string{"a.go": "package a\n\nfunc Checkout() {}\n"})
	other := t.TempDir()
	// Simulate an index pass in flight on root.
	release, err := idx.lockProject(root)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := idx.SearchLexical(canceled, other, "Checkout", 3, "", false); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled search err = %v, want context.Canceled", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := idx.SearchLexical(ctx, other, "Checkout", 3, "", false)
		done <- err
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("search canceled during lock wait err = %v", err)
		}
	case <-time.After(5 * time.Second): // failure guard, not ordering
		t.Fatal("search blocked on the project lock after cancellation")
	}
}
