// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

//go:build worktree_e2e

// Package worktrees proves Lumen's worktree indexing efficiency through its
// production entry points. Real Lumen binaries (CLI, SessionStart hook with its
// detached background indexer, and the MCP stdio server) run against real Git
// worktrees that share one isolated XDG data home. Only the remote embedding
// provider is replaced, by a deterministic HTTP server that records every
// request in a ledger.
//
// Assertion messages carry stable "[A:<tag>]" markers. The curated production
// mutation gate (see mutants/) requires a mutant to fail its designated test
// with its designated marker; any other failure does not count as a kill.
package worktrees

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sqlitevec "github.com/ory/lumen/internal/sqlitevec"
	"github.com/ory/lumen/internal/store"
)

const testModel = "lumen-worktree-test-embed"

// settleDelay exceeds the scanner's two-second timestamp race window, after
// which unchanged files are eligible for metadata-based hash reuse.
const settleDelay = 2100 * time.Millisecond

var lumenBinary string

func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		// Windows conservatively rehashes content on every scan; this suite
		// asserts the macOS/Linux metadata reuse contract.
		fmt.Println("skipping worktree integration suite on windows")
		os.Exit(0)
	}
	if err := sqlitevec.Auto(); err != nil {
		fmt.Fprintf(os.Stderr, "register sqlite-vec: %v\n", err)
		os.Exit(1)
	}
	dir, err := os.MkdirTemp("", "lumen-worktree-e2e-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root, err := moduleRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Unique per run, so leftover detached children can be found and killed by
	// executable path without touching any other Lumen installation.
	lumenBinary = filepath.Join(dir, fmt.Sprintf("lumen-wt-%d-%d", os.Getpid(), time.Now().UnixNano()))
	// Same flags as `make build-local`: CGO for sqlite-vec, fts5 for search.
	build := exec.Command("go", "build", "-tags=fts5", "-o", lumenBinary, ".")
	build.Dir = root
	build.Env = append(os.Environ(), "CGO_ENABLED=1")
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build lumen from %s: %v\n", root, err)
		os.Exit(1)
	}
	code := m.Run()
	killStrayLumen()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func moduleRoot() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", err
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		return "", errors.New("not inside a Go module")
	}
	return filepath.Dir(gomod), nil
}

// killStrayLumen terminates detached children (e.g. SessionStart background
// indexers) still running from this run's unique binary.
func killStrayLumen() {
	for _, pid := range lumenPIDs("") {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
}

// lumenPIDs lists processes running this run's binary whose command line
// contains every word of filter.
func lumenPIDs(filter string) []int {
	out, err := exec.Command("ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != lumenBinary {
			continue
		}
		if filter != "" && !strings.Contains(strings.Join(fields[1:], " "), filter) {
			continue
		}
		if pid, err := strconv.Atoi(fields[0]); err == nil && pid != os.Getpid() {
			pids = append(pids, pid)
		}
	}
	return pids
}

// ---------------------------------------------------------------------------
// Deterministic embedding provider and request ledger
// ---------------------------------------------------------------------------

type embedRequest struct {
	Seq      int
	Endpoint string
	Inputs   []string
	Status   int
	Query    bool
}

type gate struct {
	match   func([]string) bool
	status  int // nonzero: respond immediately with this HTTP status
	arrived chan struct{}
	release chan struct{}
	once    sync.Once
	fired   bool
}

// Release lets a blocked request continue. Safe to call repeatedly.
func (g *gate) Release() { g.once.Do(func() { close(g.release) }) }

type provider struct {
	t      *testing.T
	srv    *httptest.Server
	dims   int
	mu     sync.Mutex
	reqs   []embedRequest
	query  map[string]bool
	gates  []*gate
	flight int
	peak   int
}

func newProvider(t *testing.T, dims int) *provider {
	p := &provider{t: t, dims: dims, query: map[string]bool{}}
	p.srv = httptest.NewServer(http.HandlerFunc(p.serve))
	t.Cleanup(func() {
		p.mu.Lock()
		for _, g := range p.gates {
			g.Release()
		}
		p.mu.Unlock()
		p.srv.Close()
	})
	return p
}

func (p *provider) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
		_, _ = fmt.Fprintf(w, `{"models":[{"name":%q,"model":%q}]}`, testModel, testModel)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
		_, _ = fmt.Fprintf(w, `{"data":[{"id":%q}]}`, testModel)
		return
	case r.Method == http.MethodPost && (r.URL.Path == "/api/embed" || r.URL.Path == "/v1/embeddings"):
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var body struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Model != testModel {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p.mu.Lock()
	queries := 0
	for _, in := range body.Input {
		if p.query[in] {
			queries++
		}
	}
	entry := embedRequest{Seq: len(p.reqs), Endpoint: r.URL.Path, Inputs: body.Input, Status: http.StatusOK, Query: queries > 0}
	if queries != 0 && queries != len(body.Input) {
		p.t.Errorf("[A:query-batched-with-documents] request mixed %d queries into %d inputs", queries, len(body.Input))
	}
	var g *gate
	for _, candidate := range p.gates {
		if !candidate.fired && candidate.match(body.Input) {
			candidate.fired = true
			g = candidate
			break
		}
	}
	if g != nil && g.status != 0 {
		entry.Status = g.status
	}
	p.reqs = append(p.reqs, entry)
	p.flight++
	p.peak = max(p.peak, p.flight)
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.flight--
		p.mu.Unlock()
	}()

	if g != nil {
		close(g.arrived)
		if g.status != 0 {
			w.WriteHeader(g.status)
			_, _ = fmt.Fprint(w, `{"error":"injected failure"}`)
			return
		}
		select {
		case <-g.release:
		case <-r.Context().Done():
			return
		}
	}
	vectors := make([][]float32, len(body.Input))
	for i, in := range body.Input {
		vectors[i] = deterministicVector(in, p.dims)
	}
	if r.URL.Path == "/api/embed" {
		_ = json.NewEncoder(w).Encode(map[string]any{"model": testModel, "embeddings": vectors})
		return
	}
	data := make([]map[string]any, len(vectors))
	for i, v := range vectors {
		data[i] = map[string]any{"index": i, "embedding": v}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func deterministicVector(input string, dims int) []float32 {
	out := make([]float32, dims)
	var norm float64
	for i := range out {
		h := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s", i, input)))
		v := float64(int32(binary.BigEndian.Uint32(h[:4]))) / math.MaxInt32
		out[i] = float32(v)
		norm += v * v
	}
	norm = math.Sqrt(norm)
	for i := range out {
		out[i] = float32(float64(out[i]) / norm)
	}
	return out
}

// registerQuery records a query string the harness is about to issue, so the
// ledger classifies requests by exact text rather than by request size.
func (p *provider) registerQuery(q string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.query[q] = true
}

// block arms a gate that holds the next matching embedding request open.
func (p *provider) block(match func([]string) bool) *gate {
	g := &gate{match: match, arrived: make(chan struct{}), release: make(chan struct{})}
	p.mu.Lock()
	p.gates = append(p.gates, g)
	p.mu.Unlock()
	return g
}

// fail arms a gate that answers the next matching request with status.
func (p *provider) fail(match func([]string) bool, status int) *gate {
	g := &gate{match: match, status: status, arrived: make(chan struct{}), release: make(chan struct{})}
	p.mu.Lock()
	p.gates = append(p.gates, g)
	p.mu.Unlock()
	return g
}

func documentsMentioning(substr string) func([]string) bool {
	return func(inputs []string) bool {
		for _, in := range inputs {
			if strings.HasPrefix(in, "// ") && strings.Contains(in, substr) {
				return true
			}
		}
		return false
	}
}

// mark returns a ledger position for later "since" queries.
func (p *provider) mark() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.reqs)
}

func (p *provider) since(mark int) []embedRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.reqs[mark:])
}

// documents returns successfully answered document inputs since mark, with
// multiplicity.
func (p *provider) documents(mark int) []string {
	var docs []string
	for _, r := range p.since(mark) {
		if !r.Query && r.Status == http.StatusOK {
			docs = append(docs, r.Inputs...)
		}
	}
	return docs
}

func (p *provider) queryRequests(mark int) int {
	n := 0
	for _, r := range p.since(mark) {
		if r.Query {
			n++
		}
	}
	return n
}

func (p *provider) peakInFlight() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.peak
}

func multiplicity(inputs []string) map[string]int {
	m := make(map[string]int, len(inputs))
	for _, in := range inputs {
		m[in]++
	}
	return m
}

func requireEachOnce(t *testing.T, tag string, inputs []string) {
	t.Helper()
	for in, n := range multiplicity(inputs) {
		if n != 1 {
			t.Fatalf("[A:%s] document input embedded %d times: %q", tag, n, firstLine(in))
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}

// ---------------------------------------------------------------------------
// Isolated environment and production entry points
// ---------------------------------------------------------------------------

type env struct {
	t        *testing.T
	p        *provider
	dataHome string
	home     string
	extra    []string
	backend  string
}

type envOption func(*env)

func withEnv(kv ...string) envOption { return func(e *env) { e.extra = append(e.extra, kv...) } }

func withBackend(backend string) envOption { return func(e *env) { e.backend = backend } }

func withDims(dims int) envOption { return func(e *env) { e.p.dims = dims } }

// newEnv creates one isolated XDG data home shared by every Lumen process the
// test starts, plus isolated config and Git settings.
func newEnv(t *testing.T, opts ...envOption) *env {
	t.Helper()
	t.Cleanup(killStrayLumen)
	base := realDir(t, t.TempDir())
	e := &env{t: t, p: newProvider(t, 16), dataHome: filepath.Join(base, "data"), home: filepath.Join(base, "home"), backend: "ollama"}
	for _, opt := range opts {
		opt(e)
	}
	for _, dir := range []string{e.dataHome, e.home, filepath.Join(e.home, ".config")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	gitconfig := "[user]\n\tname = Lumen Test\n\temail = lumen@example.invalid\n[commit]\n\tgpgsign = false\n[tag]\n\tgpgsign = false\n[init]\n\tdefaultBranch = main\n"
	if err := os.WriteFile(filepath.Join(e.home, ".gitconfig"), []byte(gitconfig), 0o644); err != nil {
		t.Fatal(err)
	}
	// The MCP server performs daily maintenance on startup; a fresh stamp
	// keeps it from racing assertions. Maintenance is tested explicitly.
	if err := os.MkdirAll(filepath.Join(e.dataHome, "lumen"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.dataHome, "lumen", ".last-cleanup"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			if data, err := os.ReadFile(e.debugLogPath()); err == nil {
				lines := strings.Split(strings.TrimSpace(string(data)), "\n")
				t.Logf("debug.log tail:\n%s", strings.Join(lines[max(0, len(lines)-40):], "\n"))
			}
		}
	})
	return e
}

func (e *env) environ() []string {
	host := "OLLAMA_HOST=" + e.p.srv.URL
	if e.backend == "lmstudio" {
		host = "LM_STUDIO_HOST=" + e.p.srv.URL
	}
	vars := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + e.home,
		"XDG_DATA_HOME=" + e.dataHome,
		"XDG_CONFIG_HOME=" + filepath.Join(e.home, ".config"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + filepath.Join(e.home, ".gitconfig"),
		"GIT_TERMINAL_PROMPT=0",
		"LUMEN_BACKEND=" + e.backend,
		"LUMEN_EMBED_MODEL=" + testModel,
		"LUMEN_EMBED_DIMS=" + strconv.Itoa(e.p.dims),
		"LUMEN_LOG_LEVEL=info",
		host,
	}
	if tmp := os.Getenv("TMPDIR"); tmp != "" {
		vars = append(vars, "TMPDIR="+tmp)
	}
	return append(vars, e.extra...)
}

func (e *env) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, lumenBinary, args...)
	cmd.Env = e.environ()
	cmd.Dir = e.home
	return cmd
}

type result struct {
	stdout, stderr string
	err            error
	pid            int
}

// lumen runs one CLI invocation to completion.
func (e *env) lumen(args ...string) result {
	e.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := e.command(ctx, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	r := result{stdout: stdout.String(), stderr: stderr.String(), err: err}
	if cmd.Process != nil {
		r.pid = cmd.Process.Pid
	}
	return r
}

func (e *env) mustLumen(args ...string) result {
	e.t.Helper()
	r := e.lumen(args...)
	if r.err != nil {
		e.t.Fatalf("lumen %s: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), r.err, r.stdout, r.stderr)
	}
	return r
}

type proc struct {
	cmd    *exec.Cmd
	done   chan struct{}
	stdout bytes.Buffer
	stderr bytes.Buffer
	err    error
}

// start launches a CLI invocation without waiting for it.
func (e *env) start(args ...string) *proc {
	e.t.Helper()
	p := &proc{done: make(chan struct{})}
	p.cmd = e.command(context.Background(), args...)
	p.cmd.Stdout, p.cmd.Stderr = &p.stdout, &p.stderr
	if err := p.cmd.Start(); err != nil {
		e.t.Fatal(err)
	}
	go func() { p.err = p.cmd.Wait(); close(p.done) }()
	e.t.Cleanup(func() { _ = p.cmd.Process.Kill(); <-p.done })
	return p
}

func (p *proc) wait(t *testing.T) error {
	t.Helper()
	select {
	case <-p.done:
		return p.err
	case <-time.After(3 * time.Minute):
		t.Fatalf("process %v did not exit", p.cmd.Args)
		return nil
	}
}

func (e *env) index(path string, extra ...string) result {
	e.t.Helper()
	return e.mustLumen(append(append([]string{"index"}, extra...), path)...)
}

// sessionStart runs the real SessionStart hook, which may spawn a detached
// background `lumen index` child from the same binary.
func (e *env) sessionStart(cwd string) (string, int) {
	e.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := e.command(ctx, "hook", "session-start", "lumen")
	cmd.Stdin = strings.NewReader(fmt.Sprintf(`{"cwd":%q}`, cwd))
	out, err := cmd.Output()
	if err != nil {
		e.t.Fatalf("session-start: %v", err)
	}
	var decoded struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		e.t.Fatalf("decode hook output %q: %v", out, err)
	}
	return decoded.HookSpecificOutput.AdditionalContext, cmd.Process.Pid
}

// waitNoIndexers waits until no `lumen index` process from this run remains.
func (e *env) waitNoIndexers() {
	e.t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for len(lumenPIDs(" index ")) > 0 {
		if time.Now().After(deadline) {
			e.t.Fatalf("background indexers still running: %v", lumenPIDs(" index "))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------------
// MCP stdio sessions
// ---------------------------------------------------------------------------

type mcpSession struct {
	e       *env
	session *mcp.ClientSession
	pid     int
}

func (e *env) mcp(extraEnv ...string) *mcpSession {
	e.t.Helper()
	cmd := e.command(context.Background(), "stdio")
	cmd.Env = append(cmd.Env, extraEnv...)
	client := mcp.NewClient(&mcp.Implementation{Name: "worktree-e2e", Version: "0.0.0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		e.t.Fatalf("connect MCP: %v", err)
	}
	s := &mcpSession{e: e, session: session, pid: cmd.Process.Pid}
	e.t.Cleanup(s.close)
	return s
}

func (s *mcpSession) close() {
	if s.session != nil {
		_ = s.session.Close()
		s.session = nil
	}
}

// search issues semantic_search exactly as an agent host would and returns
// the result file names. min_score=-1 returns every eligible chunk up to limit.
func (s *mcpSession) search(query, path, cwd string) ([]string, string) {
	s.e.t.Helper()
	s.e.p.registerQuery(query)
	ctx, cancel := contextWithTimeout(2 * time.Minute)
	defer cancel()
	res, err := s.session.CallTool(ctx, callParams(query, path, cwd))
	if err != nil {
		s.e.t.Fatalf("semantic_search %q: %v", path, err)
	}
	var text strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text.WriteString(tc.Text)
		}
	}
	if res.IsError {
		s.e.t.Fatalf("semantic_search %q returned error: %s", path, text.String())
	}
	var files []string
	for _, line := range strings.Split(text.String(), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, `<result:file filename="`); ok {
			if end := strings.IndexByte(rest, '"'); end >= 0 {
				files = append(files, rest[:end])
			}
		}
	}
	sort.Strings(files)
	return slices.Compact(files), text.String()
}

func callParams(query, path, cwd string) *mcp.CallToolParams {
	args := map[string]any{"query": query, "path": path, "limit": 200, "min_score": -1, "summary": true}
	if cwd != "" {
		args["cwd"] = cwd
	}
	return &mcp.CallToolParams{Name: "semantic_search", Arguments: args}
}

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// ---------------------------------------------------------------------------
// Production diagnostic events (debug.log)
// ---------------------------------------------------------------------------

type event map[string]any

func (ev event) msg() string         { s, _ := ev["msg"].(string); return s }
func (ev event) str(k string) string { s, _ := ev[k].(string); return s }
func (ev event) num(k string) int {
	f, _ := ev[k].(float64)
	return int(f)
}
func (ev event) pid() int { return ev.num("pid") }

func (e *env) debugLogPath() string { return filepath.Join(e.dataHome, "lumen", "debug.log") }

// logMark returns the current debug.log offset.
func (e *env) logMark() int64 {
	info, err := os.Stat(e.debugLogPath())
	if err != nil {
		return 0
	}
	return info.Size()
}

func (e *env) events(since int64, msg string) []event {
	e.t.Helper()
	f, err := os.Open(e.debugLogPath())
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Seek(since, io.SeekStart); err != nil {
		e.t.Fatal(err)
	}
	var out []event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var ev event
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue // a line being appended concurrently
		}
		if msg == "" || ev.msg() == msg {
			out = append(out, ev)
		}
	}
	return out
}

// work sums production work counters for project since a log mark.
type work struct {
	scans, hashed, reused     int
	passes, parsed, attached  int
	failedPasses, forcedScans int
}

func (e *env) work(since int64, project string) work {
	e.t.Helper()
	var w work
	for _, ev := range e.events(since, "") {
		if project != "" && ev.str("project") != project {
			continue
		}
		switch ev.msg() {
		case "scan completed":
			w.scans++
			w.hashed += ev.num("files_hashed")
			w.reused += ev.num("files_reused")
			if b, _ := ev["forced"].(bool); b {
				w.forcedScans++
			}
		case "index pass completed":
			w.passes++
			w.parsed += ev.num("files_parsed")
			w.attached += ev.num("revisions_attached")
		case "index pass failed":
			w.failedPasses++
			w.parsed += ev.num("files_parsed")
			w.attached += ev.num("revisions_attached")
		}
	}
	return w
}

func (e *env) waitEvent(since int64, timeout time.Duration, pred func(event) bool) event {
	e.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		for _, ev := range e.events(since, "") {
			if pred(ev) {
				return ev
			}
		}
		if time.Now().After(deadline) {
			e.t.Fatal("timed out waiting for debug.log event")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitEventTagged is waitEvent whose timeout is a designated assertion: a
// production event that never happens is the defect, not harness noise.
func (e *env) waitEventTagged(tag string, since int64, timeout time.Duration, pred func(event) bool) event {
	e.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		for _, ev := range e.events(since, "") {
			if pred(ev) {
				return ev
			}
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("[A:%s] expected debug.log event never appeared", tag)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------------
// Git fixtures
// ---------------------------------------------------------------------------

func realDir(t *testing.T, dir string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func (e *env) git(dir string, args ...string) string {
	e.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = e.environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// repo creates and commits a Git repository below the test workspace.
func (e *env) repo(name string, files map[string]string) string {
	e.t.Helper()
	dir := filepath.Join(filepath.Dir(e.dataHome), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.t.Fatal(err)
	}
	e.git(dir, "init", "-q")
	writeFiles(e.t, dir, files)
	e.git(dir, "add", "-A")
	e.git(dir, "commit", "-q", "-m", "initial")
	return dir
}

func (e *env) worktree(repo, path string, args ...string) string {
	e.t.Helper()
	e.git(repo, append(append([]string{"worktree", "add", "-q"}, args...), path)...)
	return realDir(e.t, path)
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// goFile renders a Go file whose functions each become one chunk.
func goFile(pkg string, funcs ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n", pkg)
	for _, name := range funcs {
		fmt.Fprintf(&b, "\n// %s computes a value unique to %s.\nfunc %s(input int) int {\n\ttotal := input\n\tfor i := 0; i < %d; i++ {\n\t\ttotal += i * %d\n\t}\n\treturn total // %s\n}\n",
			name, name, name, len(name)+3, len(name)*7, name)
	}
	return b.String()
}

func baseFiles() map[string]string {
	return map[string]string{
		"alpha.go":     goFile("fixture", "AlphaOne", "AlphaTwo", "AlphaThree"),
		"beta.go":      goFile("fixture", "BetaOne", "BetaTwo", "BetaThree"),
		"pkg/gamma.go": goFile("pkg", "GammaOne", "GammaTwo", "GammaThree"),
		"pkg/delta.go": goFile("pkg", "DeltaOne", "DeltaTwo"),
		".gitignore":   "*.log\n",
	}
}

const baseFileCount = 4 // indexed source files in baseFiles

// ---------------------------------------------------------------------------
// Read-only collection inspection
// ---------------------------------------------------------------------------

// collections independently discovers collection databases on disk rather
// than recomputing Lumen's profile path.
func (e *env) collections() []string {
	e.t.Helper()
	matches, err := filepath.Glob(filepath.Join(e.dataHome, "lumen", "*", "index.db"))
	if err != nil {
		e.t.Fatal(err)
	}
	sort.Strings(matches)
	return matches
}

func (e *env) onlyCollection() string {
	e.t.Helper()
	dbs := e.collections()
	if len(dbs) != 1 {
		e.t.Fatalf("[A:one-collection] expected exactly one collection, found %d: %v", len(dbs), dbs)
	}
	return dbs[0]
}

type snapshot struct {
	projects        map[string]int64            // path -> id
	members         map[string]map[string]int64 // project path -> relative path -> revision id
	meta            map[string]map[string]string
	revisions       int
	retained        int
	retainedBytes   int64
	vectors         int
	vecRows         int
	chunkDefs       int
	vectorIDs       map[string]int64 // input hash -> vector id
	orphanVectors   int
	orphanRevisions int
	incomplete      int
	claimsPending   int
	claimsResults   int
	cacheBudget     int64
}

func openReadOnly(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro&_busy_timeout=10000")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	return db
}

func (e *env) inspect(dbPath string) snapshot {
	e.t.Helper()
	db := openReadOnly(e.t, dbPath)
	defer func() { _ = db.Close() }()
	q := func(dest any, query string, args ...any) {
		e.t.Helper()
		if err := db.QueryRow(query, args...).Scan(dest); err != nil {
			e.t.Fatalf("inspect %q: %v", query, err)
		}
	}
	s := snapshot{projects: map[string]int64{}, members: map[string]map[string]int64{}, meta: map[string]map[string]string{}, vectorIDs: map[string]int64{}}
	rows, err := db.Query(`SELECT id, path FROM projects`)
	if err != nil {
		e.t.Fatal(err)
	}
	ids := map[int64]string{}
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			e.t.Fatal(err)
		}
		s.projects[path], ids[id] = id, path
		s.members[path], s.meta[path] = map[string]int64{}, map[string]string{}
	}
	_ = rows.Close()
	rows, err = db.Query(`SELECT project_id, relative_path, file_revision_id FROM project_files`)
	if err != nil {
		e.t.Fatal(err)
	}
	for rows.Next() {
		var pid, rid int64
		var rel string
		if err := rows.Scan(&pid, &rel, &rid); err != nil {
			e.t.Fatal(err)
		}
		s.members[ids[pid]][rel] = rid
	}
	_ = rows.Close()
	rows, err = db.Query(`SELECT project_id, key, value FROM project_meta`)
	if err != nil {
		e.t.Fatal(err)
	}
	for rows.Next() {
		var pid int64
		var k, v string
		if err := rows.Scan(&pid, &k, &v); err != nil {
			e.t.Fatal(err)
		}
		s.meta[ids[pid]][k] = v
	}
	_ = rows.Close()
	rows, err = db.Query(`SELECT hex(input_hash), id FROM vector_keys`)
	if err != nil {
		e.t.Fatal(err)
	}
	for rows.Next() {
		var h string
		var id int64
		if err := rows.Scan(&h, &id); err != nil {
			e.t.Fatal(err)
		}
		s.vectorIDs[h] = id
	}
	_ = rows.Close()
	q(&s.revisions, `SELECT count(*) FROM file_revisions`)
	q(&s.retained, `SELECT count(*) FROM revision_cache`)
	q(&s.retainedBytes, `SELECT COALESCE(sum(bytes),0) FROM revision_cache`)
	q(&s.vectors, `SELECT count(*) FROM vector_keys`)
	q(&s.vecRows, `SELECT count(*) FROM vec_vectors`)
	q(&s.chunkDefs, `SELECT count(*) FROM chunk_defs`)
	q(&s.orphanVectors, `SELECT count(*) FROM vector_keys vk WHERE NOT EXISTS (SELECT 1 FROM chunk_defs cd WHERE cd.vector_id = vk.id)`)
	q(&s.orphanRevisions, `SELECT count(*) FROM file_revisions fr WHERE NOT EXISTS (SELECT 1 FROM project_files pf WHERE pf.file_revision_id = fr.id) AND NOT EXISTS (SELECT 1 FROM revision_cache rc WHERE rc.revision_id = fr.id)`)
	q(&s.incomplete, `SELECT count(*) FROM file_revisions WHERE complete = 0`)
	q(&s.claimsPending, `SELECT count(*) FROM embedding_claims WHERE result IS NULL`)
	q(&s.claimsResults, `SELECT count(*) FROM embedding_claims WHERE result IS NOT NULL`)
	var budget sql.NullString
	_ = db.QueryRow(`SELECT value FROM collection_meta WHERE key='revision_cache_bytes'`).Scan(&budget)
	if budget.Valid {
		s.cacheBudget, _ = strconv.ParseInt(budget.String, 10, 64)
	}
	return s
}

// requireNoGarbage asserts that no durable data is unreachable.
func requireNoGarbage(t *testing.T, tag string, s snapshot) {
	t.Helper()
	if s.orphanVectors != 0 || s.orphanRevisions != 0 || s.vecRows != s.vectors || s.incomplete != 0 {
		t.Fatalf("[A:%s] unreferenced durable data: orphanVectors=%d orphanRevisions=%d vecRows=%d vectors=%d incomplete=%d",
			tag, s.orphanVectors, s.orphanRevisions, s.vecRows, s.vectors, s.incomplete)
	}
}

func revisionSet(members map[string]int64) map[int64]bool {
	set := map[int64]bool{}
	for _, id := range members {
		set[id] = true
	}
	return set
}

// fixture writes deliberately alter collection state for setup (policy,
// aging); they never insert memberships, revisions, or vectors.
func (e *env) fixtureExec(dbPath, query string, args ...any) int64 {
	e.t.Helper()
	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=10000")
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	res, err := db.Exec(query, args...)
	if err != nil {
		e.t.Fatalf("fixture %q: %v", query, err)
	}
	n, _ := res.RowsAffected()
	return n
}

// setRetention applies a revision-cache policy through the production store.
func (e *env) setRetention(dbPath, project string, bytes int64, ttl time.Duration) {
	e.t.Helper()
	s, err := store.NewCollection(dbPath, e.p.dims, "int8", project)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if err := s.SetRevisionCachePolicy(bytes, ttl); err != nil {
		e.t.Fatal(err)
	}
}

// expireRendezvous makes the serialized result cache unusable so it cannot
// mask the durable-vector reuse path.
func (e *env) expireRendezvous(dbPath string) {
	e.fixtureExec(dbPath, `UPDATE embedding_claims SET expires_at = 1`)
}

// scanFiles lists completed scan sidecars next to a collection.
func scanFiles(t *testing.T, dbPath string) []string {
	t.Helper()
	matches, err := filepath.Glob(dbPath + ".scan-*.json")
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

// physicalBytes sums DB, WAL and scan sidecars. Call only after every Lumen
// process and inspector has closed.
func physicalBytes(t *testing.T, dbPath string) int64 {
	t.Helper()
	var total int64
	for _, path := range append([]string{dbPath, dbPath + "-wal", dbPath + "-shm"}, scanFiles(t, dbPath)...) {
		if info, err := os.Stat(path); err == nil {
			total += info.Size()
		}
	}
	return total
}

// queryIntentHeld reports whether a query process holds the shared compute
// intent lock in this data home.
func (e *env) queryIntentHeld() bool {
	lock := flock.New(filepath.Join(e.dataHome, "lumen", "compute", "query.lock"))
	defer func() { _ = lock.Close() }()
	ok, err := lock.TryLock()
	if err != nil {
		return false
	}
	if ok {
		_ = lock.Unlock()
	}
	return !ok
}

func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitGate(t *testing.T, g *gate, what string) {
	t.Helper()
	select {
	case <-g.arrived:
	case <-time.After(2 * time.Minute):
		t.Fatalf("timed out waiting for %s", what)
	}
}
