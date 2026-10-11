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

//go:build e2e

package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ory/lumen/internal/evalcase"
)

// runCatalog sends every catalog case through the executable MCP server.
// Corpora are copied into isolated directories so indexing never sees this
// repository's Git topology.
func runCatalog(t *testing.T, session *mcp.ClientSession, extra map[string]any) []evalcase.Outcome {
	t.Helper()
	cases, err := evalcase.Load(filepath.Join("testdata", "retrieval", "cases"))
	if err != nil {
		t.Fatal(err)
	}
	roots := map[string]string{}
	corpusRoot := func(corpus string) string {
		if root, ok := roots[corpus]; ok {
			return root
		}
		root := t.TempDir()
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			root = resolved
		}
		copyDir(t, filepath.Join("testdata", filepath.FromSlash(corpus)), root)
		roots[corpus] = root
		return root
	}
	search := func(root, query, filter string) []evalcase.Result {
		searchPath := root
		if filter != "" {
			searchPath = filepath.Join(root, filepath.FromSlash(filter))
		}
		args := map[string]any{"query": query, "path": searchPath, "cwd": root, "limit": 10}
		for k, v := range extra {
			args[k] = v
		}
		out := callSearch(t, session, args)
		results := make([]evalcase.Result, 0, len(out.Results))
		for _, r := range out.Results {
			file := filepath.ToSlash(r.FilePath)
			if filepath.IsAbs(r.FilePath) {
				if rel, err := filepath.Rel(root, r.FilePath); err == nil {
					file = filepath.ToSlash(rel)
				}
			} else if filter != "" {
				file = path.Join(filter, file)
			}
			results = append(results, evalcase.Result{File: file, Symbol: r.Symbol, Kind: r.Kind, StartLine: r.StartLine, EndLine: r.EndLine, Content: r.Content, Related: r.RelatedTo != ""})
		}
		return results
	}
	var outcomes []evalcase.Outcome
	for _, c := range cases {
		if !strings.HasPrefix(c.ID, os.Getenv("LUMEN_EVAL_FILTER")) {
			continue
		}
		root := corpusRoot(c.Corpus)
		if len(c.Steps) == 0 {
			outcomes = append(outcomes, evalcase.Judge(c, search(root, c.Query, c.Path)))
			continue
		}
		var prior []evalcase.Result
		for i, s := range c.Steps {
			results := search(root, s.Query, s.Path)
			outcomes = append(outcomes, evalcase.JudgeStep(c, i, prior, results))
			prior = results
		}
	}
	return outcomes
}

func writeCatalogReport(t *testing.T, suffix, report string) {
	t.Helper()
	if p := os.Getenv("LUMEN_EVAL_REPORT"); p != "" {
		if err := os.WriteFile(strings.TrimSuffix(p, ".md")+"-"+suffix+".md", []byte(report), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// TestE2E_Catalog_BM25 is the executable counterpart of the offline
// TestRetrievalCatalogBM25: the same judgments, through MCP transport, text
// output parsing and the default BM25 mode, with no embedding service.
func TestE2E_Catalog_BM25(t *testing.T) {
	t.Parallel()
	session, trap, _ := startBM25Server(t)
	outcomes := runCatalog(t, session, nil)
	for _, o := range outcomes {
		if !o.Pass && o.Category == evalcase.Supported {
			t.Errorf("%s %q: %s", o.ID, o.Query, o.Detail)
		}
	}
	writeCatalogReport(t, "bm25-e2e", evalcase.Report("bm25 (executable MCP)", outcomes))
	t.Log("\n" + evalcase.Table("category", evalcase.Summarize(outcomes, func(o evalcase.Outcome) string { return o.Category })))
	trap.assertUnused(t)
}

// TestE2E_Catalog_Semantic measures the same catalog with real embeddings.
// It is diagnostic: semantic rank budgets were never part of the semantic
// contract, so it reports parity numbers instead of failing.
func TestE2E_Catalog_Semantic(t *testing.T) {
	requireSemantic(t)
	session := startServer(t)
	outcomes := runCatalog(t, session, map[string]any{"min_score": -1.0})
	writeCatalogReport(t, "semantic-e2e", evalcase.Report("semantic (all-minilm, executable MCP)", outcomes))
	t.Log("\n" + evalcase.Table("category", evalcase.Summarize(outcomes, func(o evalcase.Outcome) string { return o.Category })))
}
