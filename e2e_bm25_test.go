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
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// embeddingTrap is an HTTP server standing in for every embedding backend. BM25
// suites point OLLAMA_HOST and LM_STUDIO_HOST at it and assert it was never
// contacted.
type embeddingTrap struct {
	*httptest.Server
	requests atomic.Int64
}

func newEmbeddingTrap(t *testing.T) *embeddingTrap {
	t.Helper()
	trap := &embeddingTrap{}
	trap.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		trap.requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(trap.Close)
	return trap
}

func (e *embeddingTrap) assertUnused(t *testing.T) {
	t.Helper()
	if n := e.requests.Load(); n != 0 {
		t.Errorf("BM25 made %d embedding-service requests; want 0", n)
	}
}

// bm25Env returns an isolated BM25 environment whose embedding hosts point at
// the trap.
func bm25Env(dataHome string, trap *embeddingTrap, extra ...string) []string {
	env := childEnv("bm25", dataHome)
	env = slices.DeleteFunc(env, func(kv string) bool {
		return strings.HasPrefix(kv, "OLLAMA_HOST=") || strings.HasPrefix(kv, "LM_STUDIO_HOST=")
	})
	return append(append(env, "OLLAMA_HOST="+trap.URL, "LM_STUDIO_HOST="+trap.URL, "LUMEN_FRESHNESS_TTL=1s"), extra...)
}

func connectEnv(t *testing.T, env []string) *mcp.ClientSession {
	t.Helper()
	return connectEnvOpts(t, env, nil)
}

func connectEnvOpts(t *testing.T, env []string, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	cmd := exec.Command(serverBinary, "stdio")
	cmd.Env = env
	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-bm25-client", Version: "0.1.0"}, opts)
	session, err := client.Connect(context.Background(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// startBM25Server starts an offline BM25 MCP server. The returned trap counts
// any embedding-service traffic.
func startBM25Server(t *testing.T, extra ...string) (*mcp.ClientSession, *embeddingTrap, string) {
	t.Helper()
	trap := newEmbeddingTrap(t)
	dataHome := t.TempDir()
	return connectEnv(t, bm25Env(dataHome, trap, extra...)), trap, dataHome
}

// startBM25ServerWithOpts starts an offline BM25 server whose embedding hosts
// point at trap, mirroring the semantic startServerWithOpts settings.
func startBM25ServerWithOpts(t *testing.T, trap *embeddingTrap, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	return connectEnvOpts(t, bm25Env(t.TempDir(), trap), opts)
}

func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func directResults(results []searchResultItem) []searchResultItem {
	return slices.DeleteFunc(slices.Clone(results), func(r searchResultItem) bool { return r.RelatedTo != "" })
}

func relatedResults(results []searchResultItem) []searchResultItem {
	return slices.DeleteFunc(slices.Clone(results), func(r searchResultItem) bool { return r.RelatedTo == "" })
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: mustJSON(t, args)})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return res
}

var bm25StatusRe = regexp.MustCompile(`Files: (\d+) \| Chunks: (\d+) .*\nLast indexed: (.*) \| Stale: (\w+)`)

type bm25Status struct {
	files, chunks int
	stale         bool
	text          string
}

func callBM25Status(t *testing.T, session *mcp.ClientSession, path string) bm25Status {
	t.Helper()
	res := callTool(t, session, "index_status", map[string]any{"path": path})
	text := getTextContent(t, res)
	if res.IsError {
		t.Fatalf("index_status error: %s", text)
	}
	m := bm25StatusRe.FindStringSubmatch(text)
	if m == nil || !strings.HasPrefix(text, "BM25 index: ") {
		t.Fatalf("unexpected BM25 status text: %q", text)
	}
	files, _ := strconv.Atoi(m[1])
	chunks, _ := strconv.Atoi(m[2])
	return bm25Status{files: files, chunks: chunks, stale: m[4] == "true", text: text}
}

func TestE2E_BM25_OfflineDiscoveryAndHealth(t *testing.T) {
	t.Parallel()
	session, trap, _ := startBM25Server(t)
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	for _, want := range []string{"semantic_search", "index_status", "health_check"} {
		if !slices.Contains(names, want) {
			t.Errorf("tool %s missing from %v", want, names)
		}
	}
	res := callTool(t, session, "health_check", map[string]any{})
	text := getTextContent(t, res)
	if res.IsError || !strings.Contains(text, "Model: bm25") || !strings.Contains(text, "Status: OK") {
		t.Fatalf("health_check = %q (error=%v)", text, res.IsError)
	}
	out := callSearch(t, session, map[string]any{"query": "ValidateToken", "path": sampleProjectPath(t)})
	if out.Retrieval != "bm25" || len(out.Results) == 0 {
		t.Fatalf("search = %+v", out)
	}
	trap.assertUnused(t)
}

func TestE2E_BM25_SampleProjectLifecycle(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	copyDir(t, sampleProjectPath(t), project)
	trap := newEmbeddingTrap(t)
	dataHome := t.TempDir()
	session := connectEnv(t, bm25Env(dataHome, trap))

	out := callSearch(t, session, map[string]any{"query": "authentication token validation", "path": project, "limit": 5})
	if !out.Reindexed || out.IndexedFiles != 7 {
		t.Errorf("first search: reindexed=%v indexed=%d, want true/7", out.Reindexed, out.IndexedFiles)
	}
	direct := directResults(out.Results)
	if len(direct) == 0 || len(direct) > 5 {
		t.Fatalf("direct results = %d, want 1..5", len(direct))
	}
	for i, r := range out.Results {
		if r.FilePath == "" || r.Symbol == "" || !validChunkKinds[r.Kind] || r.StartLine <= 0 || r.EndLine < r.StartLine || r.Content == "" {
			t.Errorf("result[%d] malformed: %+v", i, r)
		}
		if r.RelatedTo == "" && (!r.HasScore || r.Score <= 0) {
			t.Errorf("direct result[%d] needs a positive BM25 score: %+v", i, r)
		}
	}
	for i := 1; i < len(direct); i++ {
		if direct[i].Score > direct[i-1].Score {
			t.Errorf("direct results not in descending score order at %d: %v > %v", i, direct[i].Score, direct[i-1].Score)
		}
	}
	vt := findResult(direct, "ValidateToken")
	if vt == nil || !strings.Contains(vt.Content, "func ValidateToken") {
		t.Fatalf("ValidateToken missing from %v", resultSymbols(direct))
	}
	status := callBM25Status(t, session, project)
	if status.files != 7 || status.chunks == 0 || status.stale {
		t.Fatalf("status = %+v", status)
	}
	_ = session.Close()

	// A restarted server reuses the persisted index without reindexing.
	session = connectEnv(t, bm25Env(dataHome, trap))
	if again := callBM25Status(t, session, project); again.files != status.files || again.chunks != status.chunks || again.stale {
		t.Fatalf("restarted status = %+v, want %+v", again, status)
	}
	out = callSearch(t, session, map[string]any{"query": "ValidateToken", "path": project})
	if out.Reindexed {
		t.Error("restarted search reindexed an unchanged project")
	}
	if d := directResults(out.Results); len(d) == 0 || d[0].Symbol != "ValidateToken" {
		t.Errorf("exact identifier should rank first, got %v", resultSymbols(d))
	}
	trap.assertUnused(t)
}

func TestE2E_BM25_IncrementalAddEditDelete(t *testing.T) {
	t.Parallel()
	// A negligible freshness TTL makes every search re-check the tree, so the
	// test needs no sleeps to observe edits.
	session, trap, _ := startBM25Server(t, "LUMEN_FRESHNESS_TTL=1ns")
	project := t.TempDir()
	copyDir(t, sampleProjectPath(t), project)
	search := func(q string) semanticSearchOutput {
		return callSearch(t, session, map[string]any{"query": q, "path": project, "lexical_only": true})
	}
	if out := search("authentication"); !out.Reindexed {
		t.Fatal("first search should index")
	}
	if out := search("authentication"); out.Reindexed {
		t.Fatal("unchanged search should not reindex")
	}
	shutdown := filepath.Join(project, "shutdown.go")
	if err := os.WriteFile(shutdown, []byte("package project\n\n// GracefulShutdown drains active connections.\nfunc GracefulShutdown(timeout int) error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := search("GracefulShutdown"); findResult(out.Results, "GracefulShutdown") == nil {
		t.Fatalf("added symbol not found: %v", resultSymbols(out.Results))
	}
	if err := os.WriteFile(shutdown, []byte("package project\n\n// ForcefulShutdown kills active connections.\nfunc ForcefulShutdown() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := search("GracefulShutdown ForcefulShutdown")
	if findResult(out.Results, "ForcefulShutdown") == nil || findResult(out.Results, "GracefulShutdown") != nil {
		t.Fatalf("edit not reflected: %v", resultSymbols(out.Results))
	}
	if err := os.Remove(shutdown); err != nil {
		t.Fatal(err)
	}
	if out := search("ForcefulShutdown"); findResult(out.Results, "ForcefulShutdown") != nil {
		t.Fatalf("deleted symbol still returned: %v", resultSymbols(out.Results))
	}
	trap.assertUnused(t)
}

const relatedProject = `import { reserveSlot as claim } from './reservation';
// Prevent duplicate payment processing.
export function processPayment(id: string) { return claim(id); }
`

func TestE2E_BM25_LimitsExpansionAndMinScore(t *testing.T) {
	t.Parallel()
	session, trap, _ := startBM25Server(t)
	files := map[string]string{
		"billing/payment.ts":     relatedProject,
		"billing/reservation.ts": "export function reserveSlot(key: string) { return ledger.insertIfAbsent(key); }\n",
	}
	for i := range 6 {
		files["billing/payment_note_"+strconv.Itoa(i)+".ts"] = "// payment processing note " + strconv.Itoa(i) + "\nexport function note" + strconv.Itoa(i) + "() { return 'payment'; }\n"
	}
	project := writeProject(t, files)

	out := callSearch(t, session, map[string]any{"query": "processPayment payment", "path": project, "limit": 3})
	if d := directResults(out.Results); len(d) != 3 {
		t.Fatalf("limit=3 returned %d direct results", len(d))
	}
	related := relatedResults(out.Results)
	if len(related) == 0 || len(related) > 2 {
		t.Fatalf("related = %d, want 1..2: %+v", len(related), out.Results)
	}
	bytes := 0
	for _, r := range related {
		bytes += len(r.Content)
		if r.HasScore || r.Relation != "direct_call" || !strings.HasPrefix(r.RelatedTo, "billing/payment.ts:") {
			t.Errorf("related snippet provenance = %+v", r)
		}
	}
	if bytes > 6000 {
		t.Errorf("related context %d bytes > 6000", bytes)
	}
	if r := findResult(related, "reserveSlot"); r == nil || !strings.Contains(r.Content, "insertIfAbsent") {
		t.Errorf("reserveSlot related context missing: %+v", related)
	}

	out = callSearch(t, session, map[string]any{"query": "processPayment payment", "path": project, "limit": 3, "lexical_only": true})
	if len(relatedResults(out.Results)) != 0 || len(out.Results) != 3 {
		t.Errorf("lexical_only must return only 3 direct results: %+v", out.Results)
	}
	for _, minScore := range []any{nil, -1.0, 0.0} {
		args := map[string]any{"query": "processPayment", "path": project}
		if minScore != nil {
			args["min_score"] = minScore
		}
		if out := callSearch(t, session, args); len(directResults(out.Results)) == 0 {
			t.Errorf("min_score=%v returned no results", minScore)
		}
	}
	res := callSearchRaw(t, session, map[string]any{"query": "processPayment", "path": project, "min_score": 0.5})
	if !res.IsError || !strings.Contains(getTextContent(t, res), "min_score") {
		t.Errorf("positive min_score must be rejected in BM25, got error=%v %q", res.IsError, getTextContent(t, res))
	}
	trap.assertUnused(t)
}

func TestE2E_BM25_RelatedProvenanceAndShadowing(t *testing.T) {
	t.Parallel()
	session, trap, _ := startBM25Server(t)
	project := writeProject(t, map[string]string{
		"billing/payment.ts":     relatedProject,
		"billing/reservation.ts": "export function reserveSlot(key: string) { return ledger.insertIfAbsent(key); }\n",
		// A destructured local binding shadows the imported alias: claim here is
		// not reserveSlot and must not be expanded.
		"billing/refund.ts": "import { reserveSlot as claim } from './reservation';\nexport function processRefund(id: string, deps: any) { const { claim } = deps; return claim(id); }\n",
	})
	out := callSearch(t, session, map[string]any{"query": "processPayment", "path": project})
	related := relatedResults(out.Results)
	if len(related) != 1 || related[0].Symbol != "reserveSlot" || related[0].RelatedTo != "billing/payment.ts:processPayment" {
		t.Fatalf("processPayment related = %+v", related)
	}
	out = callSearch(t, session, map[string]any{"query": "processRefund", "path": project})
	if d := directResults(out.Results); len(d) == 0 || d[0].Symbol != "processRefund" {
		t.Fatalf("processRefund direct = %v", resultSymbols(d))
	}
	// processPayment may also rank for "processRefund" and legitimately
	// expand; only expansion attributed to processRefund would be wrong.
	for _, r := range relatedResults(out.Results) {
		if strings.HasPrefix(r.RelatedTo, "billing/refund.ts:") {
			t.Errorf("shadowed local claim must not expand to reserveSlot: %+v", r)
		}
	}
	trap.assertUnused(t)
}

func TestE2E_BM25_InvalidEmbeddingConfigDoesNotBlock(t *testing.T) {
	t.Parallel()
	trap := newEmbeddingTrap(t)
	dataHome := t.TempDir()
	configDir := filepath.Join(dataHome, "config", "lumen")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	// Embedding settings that would fail validation or probing in semantic
	// mode must not matter for BM25.
	if err := os.WriteFile(configPath, []byte("retrieval: bm25\nmodel: definitely-not-a-registered-model\nservers:\n  - backend: ollama\n    host: "+trap.URL+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := slices.DeleteFunc(bm25Env(dataHome, trap), func(kv string) bool { return strings.HasPrefix(kv, "LUMEN_RETRIEVAL=") })
	session := connectEnv(t, env)
	project := writeProject(t, map[string]string{"a.go": "package a\n\n// Checkout reserves stock.\nfunc Checkout() {}\n"})
	out := callSearch(t, session, map[string]any{"query": "Checkout", "path": project})
	if out.Retrieval != "bm25" || findResult(out.Results, "Checkout") == nil {
		t.Fatalf("BM25 search with invalid embedding config = %+v", out)
	}

	// Retrieval is pinned at startup: a reload that selects semantic (or is
	// malformed) must not switch the running server away from BM25.
	if err := os.WriteFile(configPath, []byte("retrieval: semantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := callSearch(t, session, map[string]any{"query": "Checkout", "path": project}); out.Retrieval != "bm25" {
		t.Errorf("retrieval changed after reload: %q", out.Retrieval)
	}
	if err := os.WriteFile(configPath, []byte("retrieval: [unterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := callSearch(t, session, map[string]any{"query": "Checkout", "path": project}); out.Retrieval != "bm25" || findResult(out.Results, "Checkout") == nil {
		t.Errorf("malformed reload broke BM25: %+v", out)
	}
	trap.assertUnused(t)
}

func TestE2E_BM25_CLIOutputForms(t *testing.T) {
	t.Parallel()
	project := gitSampleProject(t)
	dataHome := t.TempDir()
	trap := newEmbeddingTrap(t)
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(serverBinary, args...)
		cmd.Env = bm25Env(dataHome, trap)
		out, err := cmd.Output()
		if err != nil {
			var stderr string
			if ee, ok := err.(*exec.ExitError); ok {
				stderr = string(ee.Stderr)
			}
			t.Fatalf("lumen %v: %v\n%s", args, err, stderr)
		}
		return string(out)
	}
	run("index", project)
	full, err := parseSearchOutput(run("search", "--path", project, "ValidateToken"))
	if err != nil {
		t.Fatal(err)
	}
	if full.Retrieval != "bm25" || len(full.Results) == 0 || full.Results[0].Symbol != "ValidateToken" {
		t.Fatalf("full = %+v", full)
	}
	summary, err := parseSearchOutput(run("search", "--path", project, "--summary", "ValidateToken"))
	if err != nil {
		t.Fatal(err)
	}
	limited, err := parseSearchOutput(run("search", "--path", project, "--max-lines", "2", "ValidateToken"))
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Results) != len(full.Results) || len(limited.Results) != len(full.Results) {
		t.Fatalf("result counts differ: full=%d summary=%d limited=%d", len(full.Results), len(summary.Results), len(limited.Results))
	}
	for i := range full.Results {
		f, s, l := full.Results[i], summary.Results[i], limited.Results[i]
		if f.FilePath != s.FilePath || f.StartLine != s.StartLine || f.Symbol != l.Symbol || f.Score != s.Score {
			t.Errorf("result %d differs across forms: %+v / %+v / %+v", i, f, s, l)
		}
		if s.Content != "" {
			t.Errorf("summary result %d has content", i)
		}
		if lines := strings.Split(l.Content, "\n"); len(lines) > 3 || !strings.HasPrefix(f.Content, strings.TrimSuffix(strings.Join(lines[:min(2, len(lines))], "\n"), "...")) {
			t.Errorf("max-lines result %d = %q (full %q)", i, l.Content, f.Content)
		}
	}
	if _, _, err := runCLIMode(t, "bm25", dataHome, "search", "--path", project, "--min-score", "0.4", "ValidateToken"); err == nil {
		t.Error("CLI positive --min-score must fail in BM25")
	}
	trap.assertUnused(t)
}
