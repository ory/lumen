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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/evalcase"
)

const catalogTestdata = "../../testdata"

// copyCorpus copies a checked-in corpus so indexing never observes the
// repository's own Git topology or ignore files.
func copyCorpus(t testing.TB, corpus string) string {
	t.Helper()
	src := filepath.Join(catalogTestdata, filepath.FromSlash(corpus))
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// catalogLexicalSearch adapts SearchLexical results to evalcase results.
func catalogLexicalSearch(t testing.TB, idx *Indexer, root, query, path string) []evalcase.Result {
	t.Helper()
	hits, err := idx.SearchLexical(context.Background(), root, query, 10, filepath.FromSlash(path), true)
	if err != nil {
		t.Fatalf("search %q: %v", query, err)
	}
	out := make([]evalcase.Result, 0, len(hits))
	for _, h := range hits {
		out = append(out, evalcase.Result{File: filepath.ToSlash(h.FilePath), Symbol: h.Symbol, Kind: h.Kind, StartLine: h.StartLine, EndLine: h.EndLine, Content: h.Content, Related: h.RelatedTo != ""})
	}
	return out
}

// TestRetrievalCatalogBM25 runs the shared case catalog through real BM25
// indexing and search without any embedding service. Supported cases must
// meet their rank budget; diagnostic cases are reported only. Set
// LUMEN_EVAL_REPORT=<file> to write the Markdown report and
// LUMEN_EVAL_FILTER=<prefix> to run only case ids with that prefix.
func TestRetrievalCatalogBM25(t *testing.T) {
	cases, err := evalcase.Load(filepath.Join(catalogTestdata, "retrieval", "cases"))
	if err != nil {
		t.Fatal(err)
	}
	type corpusIndex struct {
		idx  *Indexer
		root string
	}
	indexes := map[string]corpusIndex{}
	open := func(corpus string) corpusIndex {
		if ci, ok := indexes[corpus]; ok {
			return ci
		}
		root := copyCorpus(t, corpus)
		idx, err := NewLexicalIndexer(filepath.Join(t.TempDir(), "index.db"), 512, root)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = idx.Close() })
		if _, err := idx.Index(context.Background(), root, false, nil); err != nil {
			t.Fatalf("index %s: %v", corpus, err)
		}
		indexes[corpus] = corpusIndex{idx, root}
		return indexes[corpus]
	}
	var outcomes []evalcase.Outcome
	for _, c := range cases {
		if !strings.HasPrefix(c.ID, os.Getenv("LUMEN_EVAL_FILTER")) {
			continue
		}
		ci := open(c.Corpus)
		var got []evalcase.Outcome
		if len(c.Steps) > 0 {
			var prior []evalcase.Result
			for i, s := range c.Steps {
				results := catalogLexicalSearch(t, ci.idx, ci.root, s.Query, s.Path)
				got = append(got, evalcase.JudgeStep(c, i, prior, results))
				prior = results
			}
		} else {
			got = append(got, evalcase.Judge(c, catalogLexicalSearch(t, ci.idx, ci.root, c.Query, c.Path)))
		}
		outcomes = append(outcomes, got...)
	}
	known, err := evalcase.LoadKnownMisses(filepath.Join(catalogTestdata, "retrieval"), "bm25")
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("LUMEN_EVAL_FILTER") != "" {
		for id := range known {
			if !strings.HasPrefix(id, os.Getenv("LUMEN_EVAL_FILTER")) {
				delete(known, id)
			}
		}
	}
	for _, problem := range evalcase.CheckKnownMisses(outcomes, known) {
		t.Error(problem)
	}
	report := evalcase.Report("bm25 (offline, lexical index)", outcomes)
	if path := os.Getenv("LUMEN_EVAL_REPORT"); path != "" {
		if err := os.WriteFile(path, []byte(report), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("\n" + evalcase.Table("category", evalcase.Summarize(outcomes, func(o evalcase.Outcome) string { return o.Category })))
}
