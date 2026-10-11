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
	"context"
	"os"
	"os/exec"
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
// open returns the session serving a corpus. In tolerant mode a search error
// is logged and judged as an empty result instead of failing the test.
// observe, when set, receives every raw search response.
func runCatalog(t *testing.T, open func(corpus string) *mcp.ClientSession, extra map[string]any, tolerant bool, observe func(text string)) []evalcase.Outcome {
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
	search := func(session *mcp.ClientSession, root, query, filter string) []evalcase.Result {
		searchPath := root
		if filter != "" {
			searchPath = filepath.Join(root, filepath.FromSlash(filter))
		}
		args := map[string]any{"query": query, "path": searchPath, "cwd": root, "limit": 10}
		for k, v := range extra {
			args[k] = v
		}
		res := callSearchRaw(t, session, args)
		text := getTextContent(t, res)
		if res.IsError {
			if !tolerant {
				t.Fatalf("semantic_search returned error: %s", text)
			}
			t.Logf("search %q failed (judged as no results): %s", query, text)
			return nil
		}
		if observe != nil {
			observe(text)
		}
		out := parseSearchText(t, text)
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
		session := open(c.Corpus)
		if len(c.Steps) == 0 {
			outcomes = append(outcomes, evalcase.Judge(c, search(session, root, c.Query, c.Path)))
			continue
		}
		var prior []evalcase.Result
		for i, s := range c.Steps {
			results := search(session, root, s.Query, s.Path)
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
// output parsing and the default BM25 mode, with no embedding service. Output
// is unlimited (max_tokens=-1) so judgments measure rank and full content;
// TestE2E_Catalog_BM25Budget measures what the default budget hides.
func TestE2E_Catalog_BM25(t *testing.T) {
	t.Parallel()
	session, trap, _ := startBM25Server(t)
	outcomes := runCatalog(t, func(string) *mcp.ClientSession { return session }, map[string]any{"max_tokens": -1}, false, nil)
	known, err := evalcase.LoadKnownMisses(filepath.Join("testdata", "retrieval"), "bm25")
	if err != nil {
		t.Fatal(err)
	}
	for id := range known {
		if !strings.HasPrefix(id, os.Getenv("LUMEN_EVAL_FILTER")) {
			delete(known, id)
		}
	}
	for _, problem := range evalcase.CheckKnownMisses(outcomes, known) {
		t.Error(problem)
	}
	writeCatalogReport(t, "bm25-e2e", evalcase.Report("bm25 (executable MCP)", outcomes))
	t.Log("\n" + evalcase.Table("category", evalcase.Summarize(outcomes, func(o evalcase.Outcome) string { return o.Category })))
	trap.assertUnused(t)
}

// TestE2E_Catalog_Semantic measures the same catalog with real embeddings.
// It is diagnostic: semantic rank budgets were never part of the semantic
// contract, so it reports parity numbers instead of failing. Embedding every
// corpus is slow and saturates a local Ollama, so it is opt-in
// (LUMEN_E2E_SEMANTIC_CATALOG=1, `make e2e-semantic-catalog`) and uses a fresh
// server per corpus so one overloaded corpus cannot poison the rest.
func TestE2E_Catalog_Semantic(t *testing.T) {
	if os.Getenv("LUMEN_E2E_SEMANTIC_CATALOG") != "1" {
		t.Skip("set LUMEN_E2E_SEMANTIC_CATALOG=1 to measure the catalog with real embeddings")
	}
	requireSemantic(t)
	sessions := map[string]*mcp.ClientSession{}
	open := func(corpus string) *mcp.ClientSession {
		if s, ok := sessions[corpus]; ok {
			return s
		}
		cmd := exec.Command(serverBinary, "stdio")
		cmd.Env = append(childEnv("semantic", t.TempDir()), "LUMEN_MAX_CHUNK_TOKENS=100", "LUMEN_REINDEX_TIMEOUT=30m")
		client := mcp.NewClient(&mcp.Implementation{Name: "e2e-catalog-semantic", Version: "0.1.0"}, nil)
		session, err := client.Connect(context.Background(), &mcp.CommandTransport{Command: cmd}, nil)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		t.Cleanup(func() { _ = session.Close() })
		sessions[corpus] = session
		return session
	}
	outcomes := runCatalog(t, open, map[string]any{"min_score": -1.0, "max_tokens": -1}, true, nil)
	writeCatalogReport(t, "semantic-e2e", evalcase.Report("semantic (all-minilm, executable MCP)", outcomes))
	t.Log("\n" + evalcase.Table("category", evalcase.Summarize(outcomes, func(o evalcase.Outcome) string { return o.Category })))
}
