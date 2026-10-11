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
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func openCrashIndexer(mode, dsn, root string) (*Indexer, error) {
	if mode == "bm25" {
		return NewLexicalIndexer(dsn, 512, root)
	}
	return NewIndexerForProject(dsn, &mockEmbedder{dims: 4, model: "test-model"}, 512, "int8", root)
}

// TestCrashHelperProcess is the child of TestIndexRecoversAfterKill. It
// indexes until the third file, reports on stdout and then blocks so the
// parent can SIGKILL it with earlier files committed.
func TestCrashHelperProcess(t *testing.T) {
	mode, dsn, root := os.Getenv("LUMEN_CRASH_MODE"), os.Getenv("LUMEN_CRASH_DSN"), os.Getenv("LUMEN_CRASH_ROOT")
	if mode == "" {
		t.Skip("helper process only")
	}
	idx, err := openCrashIndexer(mode, dsn, root)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(2)
	}
	_, _ = idx.Index(context.Background(), root, false, func(i, _ int, _ string) {
		if i == 2 {
			fmt.Println("committed")
			select {} // wait for SIGKILL
		}
	})
	fmt.Println("finished without reaching the third file")
	os.Exit(3)
}

func TestIndexRecoversAfterKill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGKILL semantics")
	}
	for _, mode := range []string{"bm25", "semantic"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			for i := range 6 {
				body := fmt.Sprintf("package p\n\n// Handler%d serves requests.\nfunc Handler%d() {}\n", i, i)
				if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("h%d.go", i)), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			dsn := filepath.Join(t.TempDir(), "index.db")
			cmd := exec.Command(os.Args[0], "-test.run=^TestCrashHelperProcess$", "-test.count=1")
			cmd.Env = append(os.Environ(), "LUMEN_CRASH_MODE="+mode, "LUMEN_CRASH_DSN="+dsn, "LUMEN_CRASH_ROOT="+root)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			line := make(chan string, 1)
			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					if text := scanner.Text(); text == "committed" || strings.HasPrefix(text, "error") || strings.HasPrefix(text, "finished") {
						line <- text
						return
					}
				}
				line <- "eof"
			}()
			select {
			case got := <-line:
				if got != "committed" {
					t.Fatalf("helper: %s", got)
				}
			case <-time.After(60 * time.Second): // failure guard, not ordering
				t.Fatal("helper never reached the third file")
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()

			idx, err := openCrashIndexer(mode, dsn, root)
			if err != nil {
				t.Fatalf("reopen after kill: %v", err)
			}
			defer func() { _ = idx.Close() }()
			if fresh, err := idx.IsFresh(root); err != nil || fresh {
				t.Fatalf("killed pass reads as fresh=%v err=%v", fresh, err)
			}
			if _, err := idx.Index(context.Background(), root, false, nil); err != nil {
				t.Fatalf("retry after kill: %v", err)
			}
			if fresh, err := idx.IsFresh(root); err != nil || !fresh {
				t.Fatalf("retry fresh=%v err=%v", fresh, err)
			}
			status, err := idx.Status(root)
			if err != nil {
				t.Fatal(err)
			}
			if status.IndexedFiles != 6 || status.TotalChunks != 6 {
				t.Fatalf("after retry files=%d chunks=%d, want 6/6 (no duplicates)", status.IndexedFiles, status.TotalChunks)
			}
		})
	}
}
