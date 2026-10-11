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
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestIndexSurvivesMidScanChanges edits and deletes files after the Merkle
// scan but before they are read. The pass must index healthy files, never
// attach new content to the scanned hash, and not stamp the tree fresh; the
// next pass converges. Synchronization is the progress callback for the first
// file, which runs before any later file is read.
func TestIndexSurvivesMidScanChanges(t *testing.T) {
	for _, mode := range []string{"bm25", "semantic"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			write := func(name, body string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{"a", "b", "c", "d"} {
				write(name+".go", fmt.Sprintf("package p\n\nfunc %sOriginal() {}\n", strings.ToUpper(name)))
			}
			var idx *Indexer
			var err error
			if mode == "bm25" {
				idx, err = NewLexicalIndexer(filepath.Join(t.TempDir(), "index.db"), 512, root)
			} else {
				idx, err = NewIndexerForProject(filepath.Join(t.TempDir(), "index.db"), &mockEmbedder{dims: 4, model: "test-model"}, 512, "int8", root)
			}
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = idx.Close() })
			ctx := context.Background()
			if _, err := idx.Index(ctx, root, false, nil); err != nil {
				t.Fatal(err)
			}

			// Every file changes, so every file is in the next pass.
			for _, name := range []string{"a", "b", "c", "d"} {
				write(name+".go", fmt.Sprintf("package p\n\nfunc %sScanned() {}\n", strings.ToUpper(name)))
			}
			scannedB := fmt.Sprintf("%x", sha256.Sum256([]byte("package p\n\nfunc BScanned() {}\n")))
			var once sync.Once
			progress := func(_, _ int, msg string) {
				if strings.Contains(msg, "a.go") {
					once.Do(func() {
						write("b.go", "package p\n\nfunc BRacing() {}\n")
						if err := os.Remove(filepath.Join(root, "c.go")); err != nil {
							t.Error(err)
						}
					})
				}
			}
			if _, err := idx.Index(ctx, root, false, progress); err != nil {
				t.Fatalf("mid-scan change aborted the pass: %v", err)
			}
			hashes, err := idx.store.GetFileHashes()
			if err != nil {
				t.Fatal(err)
			}
			if hashes["b.go"] == scannedB {
				t.Error("b.go content read after the scan was stored under the scanned hash")
			}
			for _, healthy := range []string{"a.go", "d.go"} {
				data, _ := os.ReadFile(filepath.Join(root, healthy))
				if hashes[healthy] != fmt.Sprintf("%x", sha256.Sum256(data)) {
					t.Errorf("healthy %s was not indexed (hash %q)", healthy, hashes[healthy])
				}
			}
			if fresh, err := idx.IsFresh(root); err != nil || fresh {
				t.Fatalf("partial pass stamped fresh=%v err=%v", fresh, err)
			}

			if _, err := idx.Index(ctx, root, false, nil); err != nil {
				t.Fatal(err)
			}
			hashes, err = idx.store.GetFileHashes()
			if err != nil {
				t.Fatal(err)
			}
			data, _ := os.ReadFile(filepath.Join(root, "b.go"))
			if hashes["b.go"] != fmt.Sprintf("%x", sha256.Sum256(data)) {
				t.Errorf("retry did not index b.go's current content")
			}
			if _, ok := hashes["c.go"]; ok {
				t.Error("retry kept deleted c.go")
			}
			if fresh, err := idx.IsFresh(root); err != nil || !fresh {
				t.Fatalf("converged pass fresh=%v err=%v", fresh, err)
			}
		})
	}
}
