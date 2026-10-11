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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var snapshotter = cupaloy.New(
	cupaloy.EnvVariableName("UPDATE_SNAPSHOTS"),
	cupaloy.SnapshotSubdirectory(langSnapshotDirectory),
)

func fixturesPath(lang string) string {
	p, _ := filepath.Abs(filepath.Join("testdata", "fixtures", lang))
	return p
}

// startLangServer starts the historical semantic snapshot server with a
// reduced max chunk token size suitable for all-minilm's 512-token context
// window. The retrieval mode is explicit: these snapshots record semantic
// chunk identities and are meaningless under BM25 (see TestE2E_Catalog for
// the mode-neutral, source-judged relevance suite).
func startLangServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	requireSemantic(t)

	cmd := exec.Command(serverBinary, "stdio")
	cmd.Env = append(childEnv("semantic", t.TempDir()),
		"LUMEN_BACKEND=ollama",
		// all-minilm uses BERT WordPiece tokenisation which is ~4x denser than
		// our 4-chars-per-token estimate, so cap chunks at 100 "tokens" (400
		// chars) to stay within the model's 512-token context window.
		"LUMEN_MAX_CHUNK_TOKENS=100",
		// Lang tests need indexing to complete before the first search returns.
		// CPU embedding can still exceed the default 15s timeout on slower CI
		// runners, so allow up to 10m to avoid partially-built index results.
		"LUMEN_REINDEX_TIMEOUT=10m",
	)

	transport := &mcp.CommandTransport{Command: cmd}
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "e2e-lang-test-client",
		Version: "0.1.0",
	}, nil)

	ctx := context.Background()
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func langSearch(t *testing.T, session *mcp.ClientSession, lang, query string) []searchResultItem {
	t.Helper()
	dir := fixturesPath(lang)
	out := callSearch(t, session, map[string]any{
		"query":     query,
		"path":      dir,
		"limit":     10,
		"min_score": -1.0,
	})
	// Sort by (filePath, startLine) for deterministic snapshots across environments.
	slices.SortFunc(out.Results, func(a, b searchResultItem) int {
		if a.FilePath != b.FilePath {
			if a.FilePath < b.FilePath {
				return -1
			}
			return 1
		}
		return a.StartLine - b.StartLine
	})

	return out.Results
}

func formatLangSearchResults(results []searchResultItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "results: %d\n", len(results))
	for _, result := range results {
		fmt.Fprintf(&b, "%s:%d-%d  %s (%s)\n", result.FilePath, result.StartLine, result.EndLine, result.Symbol, result.Kind)
	}
	return b.String()
}

func compareLangSnapshotT(t *testing.T, results []searchResultItem) {
	t.Helper()
	formatted := formatLangSearchResults(results)

	if _, updating := os.LookupEnv("UPDATE_SNAPSHOTS"); updating {
		snapshotter.SnapshotT(t, formatted)
		return
	}

	snapshotName := strings.ReplaceAll(t.Name(), "/", "-")
	snapshotPath := filepath.Join(langSnapshotDirectory, snapshotName)
	expected, err := os.ReadFile(snapshotPath)
	if os.IsNotExist(err) {
		// Cupaloy owns snapshot creation as well as its intentional test failure.
		snapshotter.SnapshotT(t, formatted)
		return
	}
	if err != nil {
		t.Fatalf("failed to read language snapshot %s: %v", snapshotPath, err)
	}

	if err := compareLangSnapshot(string(expected), results); err != nil {
		t.Error(err)
	}
}

func runLangTest(t *testing.T, lang string, queries []string) {
	t.Helper()
	session := startLangServer(t)
	for _, q := range queries {
		q := q
		t.Run(strings.ReplaceAll(q, " ", "_"), func(t *testing.T) {
			results := langSearch(t, session, lang, q)
			compareLangSnapshotT(t, results)
		})
	}
}

// langSuites are the original language-suite queries. Their source-judged
// relevance targets live in testdata/retrieval/cases as lang.<language>.* cases.
var langSuites = []struct {
	name, dir string
	queries   []string
}{
	{"Go", "go", []string{
		"HTTP request handler",
		"authentication token validation",
		"database connection pool",
		"time series storage and retrieval",
		"error handling and logging",
		"configuration loading from file",
	}},
	{"Java", "java", []string{
		"pet owner repository database",
		"REST controller request mapping",
		"JPA entity model fields",
		"Spring service dependency injection",
		"form input validation",
	}},
	{"Dart", "dart", []string{
		"HTTP request handler middleware",
		"server pipeline shelf handler",
		"response body status code",
		"cascade route matching",
		"logging middleware interceptor",
	}},
	{"PHP", "php", []string{
		"HTTP request handling middleware",
		"database query builder",
		"authentication guard session",
		"collection helper methods",
		"model relationships",
	}},
	{"JavaScript", "js", []string{
		"HTTP router middleware",
		"request response object",
		"event emitter listener",
		"error handling exception",
		"module exports function",
	}},
	{"TypeScript", "ts", []string{
		"event listener registration",
		"async operation with cancellation",
		"URI path manipulation",
		"lifecycle disposable resource",
		"platform detection operating system",
	}},
	{"Ruby", "ruby", []string{
		"route matching URL",
		"controller action rendering",
		"database record query scope",
		"authentication callback filter",
		"module include concern",
	}},
	{"Python", "python", []string{
		"HTTP route handler decorator",
		"database model query filter",
		"authentication login view",
		"request context middleware",
		"exception error handling",
	}},
	{"Rust", "rust", []string{
		"async runtime executor spawn",
		"file search pattern match",
		"TCP network listener accept",
		"mutex lock concurrent access",
		"error result propagation",
	}},
	{"YAML", "yaml", []string{
		"Kubernetes deployment replicas",
		"CI pipeline build steps",
		"service port configuration",
		"environment variables secrets",
	}},
	{"Markdown", "md", []string{
		"installation setup guide",
		"ownership borrowing memory",
		"error handling panic recover",
		"async concurrent threads",
	}},
	{"JSON", "json", []string{
		"build scripts commands",
		"TypeScript compiler options",
		"package dependencies versions",
		"API paths endpoints",
	}},
}

func TestLang_Go(t *testing.T)         { runLangSuite(t, "Go") }
func TestLang_Java(t *testing.T)       { runLangSuite(t, "Java") }
func TestLang_Dart(t *testing.T)       { runLangSuite(t, "Dart") }
func TestLang_PHP(t *testing.T)        { runLangSuite(t, "PHP") }
func TestLang_JavaScript(t *testing.T) { runLangSuite(t, "JavaScript") }
func TestLang_TypeScript(t *testing.T) { runLangSuite(t, "TypeScript") }
func TestLang_Ruby(t *testing.T)       { runLangSuite(t, "Ruby") }
func TestLang_Python(t *testing.T)     { runLangSuite(t, "Python") }
func TestLang_Rust(t *testing.T)       { runLangSuite(t, "Rust") }
func TestLang_YAML(t *testing.T)       { runLangSuite(t, "YAML") }
func TestLang_Markdown(t *testing.T)   { runLangSuite(t, "Markdown") }
func TestLang_JSON(t *testing.T)       { runLangSuite(t, "JSON") }

func runLangSuite(t *testing.T, name string) {
	t.Helper()
	for _, suite := range langSuites {
		if suite.name == name {
			runLangTest(t, suite.dir, suite.queries)
			return
		}
	}
	t.Fatalf("unknown language suite %s", name)
}

// TestLang_BM25SnapshotDiagnostics replays the unchanged historical semantic
// snapshots against BM25 output. Semantic snapshots record semantic chunk
// identities (merged/split names and spans), so mismatches are expected and
// are not relevance failures; this reports the compatibility gap and never
// fails. Relevance is judged by TestE2E_Catalog_BM25.
func TestLang_BM25SnapshotDiagnostics(t *testing.T) {
	session, trap, _ := startBM25Server(t)
	matched, total := 0, 0
	for _, suite := range langSuites {
		for _, q := range suite.queries {
			name := "TestLang_" + suite.name + "-" + strings.ReplaceAll(q, " ", "_")
			expected, err := os.ReadFile(filepath.Join(langSnapshotDirectory, name))
			if err != nil {
				t.Errorf("snapshot %s: %v", name, err)
				continue
			}
			total++
			results := langSearch(t, session, suite.dir, q)
			if err := compareLangSnapshot(string(expected), directSnapshotResults(results)); err != nil {
				t.Logf("%s/%s: snapshot mismatch (diagnostic): %v", suite.name, q, strings.SplitN(err.Error(), "\n", 2)[0])
				continue
			}
			matched++
		}
	}
	t.Logf("BM25 matched %d/%d historical semantic snapshots", matched, total)
	trap.assertUnused(t)
}

// directSnapshotResults drops BM25 related context, which semantic snapshots
// never contained.
func directSnapshotResults(results []searchResultItem) []searchResultItem {
	var direct []searchResultItem
	for _, r := range results {
		if r.RelatedTo == "" {
			direct = append(direct, r)
		}
	}
	return direct
}
