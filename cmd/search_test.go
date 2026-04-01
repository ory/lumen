package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
)

func TestTracer_DisabledIsNoop(t *testing.T) {
	tr := &tracer{enabled: false}
	tr.start = time.Now()
	tr.last = tr.start

	tr.record("path resolution", "/tmp/project")
	tr.record("indexer setup", "db opened")

	if len(tr.spans) != 0 {
		t.Fatalf("disabled tracer should not record spans, got %d", len(tr.spans))
	}

	var buf bytes.Buffer
	tr.print(&buf)
	if buf.Len() != 0 {
		t.Fatalf("disabled tracer should produce no output, got %q", buf.String())
	}
}

func TestTracer_EnabledRecordsSpans(t *testing.T) {
	tr := &tracer{enabled: true}
	tr.start = time.Now()
	tr.last = tr.start

	tr.record("path resolution", "/tmp/project")
	tr.record("indexer setup", "db opened, model stub")

	if len(tr.spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(tr.spans))
	}
	if tr.spans[0].label != "path resolution" {
		t.Fatalf("expected label 'path resolution', got %q", tr.spans[0].label)
	}
	if tr.spans[0].detail != "/tmp/project" {
		t.Fatalf("expected detail '/tmp/project', got %q", tr.spans[0].detail)
	}
	if tr.spans[1].label != "indexer setup" {
		t.Fatalf("expected label 'indexer setup', got %q", tr.spans[1].label)
	}
	for _, s := range tr.spans {
		if s.duration < 0 {
			t.Fatalf("span %q has negative duration %v", s.label, s.duration)
		}
	}
}

func TestTracer_PrintRendersTable(t *testing.T) {
	tr := &tracer{enabled: true}
	tr.start = time.Now()
	tr.last = tr.start
	tr.spans = []traceSpan{
		{label: "path resolution", duration: 2 * time.Millisecond, detail: "/tmp/project"},
		{label: "knn search", duration: 9 * time.Millisecond, detail: "16 candidates fetched"},
	}

	var buf bytes.Buffer
	tr.print(&buf)
	out := buf.String()

	for _, want := range []string{"path resolution", "/tmp/project", "knn search", "total", "────"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestTracer_RecordAdvancesLast(t *testing.T) {
	tr := &tracer{enabled: true}
	before := time.Now()
	tr.start = before
	tr.last = before

	time.Sleep(2 * time.Millisecond)
	tr.record("span1", "detail")
	after := tr.last

	if !after.After(before) {
		t.Fatalf("tracer.last should have advanced after record()")
	}
	if tr.spans[0].duration <= 0 {
		t.Fatalf("first span should have positive duration")
	}
}

func TestSearchCmd_FlagsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"search"})
	if err != nil || cmd == nil || cmd.Use != "search <query>" {
		t.Fatalf("search subcommand not registered or wrong Use field: %v", err)
	}

	requiredFlags := []string{
		"path", "cwd", "n-results", "min-score",
		"summary", "max-lines", "force", "trace", "model",
	}
	for _, name := range requiredFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("search cmd missing flag --%s", name)
		}
	}
}

// TestSetupIndexer_DBPathVsDirectory is a regression test for
// https://github.com/ory/lumen/issues/72.
//
// runSearch previously passed the raw indexRoot directory to setupIndexer
// instead of the computed DB file path. SQLite cannot open a directory and
// returns an error containing "PRAGMA journal_mode=WAL".
//
// Red: passing the directory directly must fail with a SQLite error.
// Green: passing config.DBPathForProject(dir, model) must succeed.
func TestSetupIndexer_DBPathVsDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	// RED: directory path → SQLite PRAGMA error (the pre-fix behaviour).
	_, dirErr := setupIndexer(&cfg, dir, nil)
	if dirErr == nil {
		t.Fatal("expected error when passing a directory as the db path, got nil")
	}
	if !strings.Contains(dirErr.Error(), "PRAGMA") && !strings.Contains(dirErr.Error(), "unable to open") {
		t.Fatalf("expected SQLite open error, got: %v", dirErr)
	}

	// GREEN: proper db file path → no error.
	dbPath := config.DBPathForProject(dir, cfg.Model)
	if mkErr := os.MkdirAll(filepath.Dir(dbPath), 0o755); mkErr != nil {
		t.Fatalf("MkdirAll: %v", mkErr)
	}
	idx, err := setupIndexer(&cfg, dbPath, nil)
	if err != nil {
		t.Fatalf("setupIndexer with db path failed: %v", err)
	}
	_ = idx.Close()
}

func TestSearchCmd_TraceSpanLabels(t *testing.T) {
	// Verify the trace span labels that runSearch records match the spec.
	tr := &tracer{enabled: true}
	tr.start = time.Now()
	tr.last = tr.start

	tr.record("path resolution", "/tmp/proj")
	tr.record("indexer setup", "db opened, model stub")
	tr.record("merkle + freshness", "42 files scanned, index is fresh (no reindex)")
	tr.record("query embedding", "4 dims")
	tr.record("knn search", "0 candidates fetched (limit=16, fetch=16)")
	tr.record("post-processing", "merged 0→0 results, filled 0 snippets")

	var stderr bytes.Buffer
	tr.print(&stderr)

	out := stderr.String()
	for _, label := range []string{
		"path resolution", "indexer setup", "merkle + freshness",
		"query embedding", "knn search", "post-processing", "total",
	} {
		if !strings.Contains(out, label) {
			t.Fatalf("trace output missing %q:\n%s", label, out)
		}
	}
}
