package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
)

func TestExecuteAgentPersistsSuccessAndFailure(t *testing.T) {
	for _, tc := range []struct {
		name, output, tail string
		wantError          bool
	}{
		{"success", `{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":10}}`, "", false},
		{"exit", `{"type":"turn.completed","usage":{}}`, "exit 7", true},
		{"failed", `{"type":"turn.failed","error":{"message":"rate limit"}}`, "", true},
		{"missing", `{"type":"thread.started"}`, "", true},
		{"timeout", "", "sleep 30", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			script := "#!/bin/sh\ncat >/dev/null\ncat <<'EVENTS'\n" + tc.output + "\nEVENTS\n" + tc.tail + "\n"
			if err := os.WriteFile(filepath.Join(bin, "codex"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			cfg := &Config{Agent: "codex", Transport: "cli", Auth: "environment", Retrieval: "bm25", ResultsDir: t.TempDir()}
			timeout := time.Second
			if tc.name == "timeout" {
				timeout = 50 * time.Millisecond
			}
			m, err := ExecuteAgent(context.Background(), cfg, t.TempDir(), Baseline, "fix", "case", timeout, nil)
			if (err != nil) != tc.wantError {
				t.Fatalf("error %v, wantError %v", err, tc.wantError)
			}
			saved, loadErr := metrics.LoadFromFile(filepath.Join(cfg.ResultsDir, "case-metrics.json"))
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if saved.Agent != "codex" || saved.Runner != "cli" || (saved.Error != "") != tc.wantError {
				t.Fatalf("bad persisted metrics: %+v", saved)
			}
			if !tc.wantError && (m.InputTokens != 60 || m.CacheRead != 40) {
				t.Fatalf("bad usage: %+v", m)
			}
			for _, suffix := range []string{"raw.jsonl", "stderr.log"} {
				if _, err := os.Stat(filepath.Join(cfg.ResultsDir, "case-"+suffix)); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestRunRecordsFailedTaskTests(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "main.txt"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "--quiet"}, {"add", "."}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "--quiet", "-m", "fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, out)
		}
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repo
	commit, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\ncat >/dev/null\necho '{\"type\":\"turn.completed\",\"usage\":{}}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	cfg := &Config{Agent: "codex", Transport: "cli", Auth: "environment", Retrieval: "bm25", ResultsDir: t.TempDir(), TotalRuns: 1}
	result, err := Run(context.Background(), cfg, task.Task{ID: "fixture", Repo: repo, BaseCommit: strings.TrimSpace(string(commit)), TimeoutSeconds: 5, TestCommand: "echo regression-failed; exit 7"}, Baseline, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Metrics.TestStatus != "failed" {
		t.Fatalf("lost test failure: %+v", result.Metrics)
	}
	saved, err := metrics.LoadFromFile(filepath.Join(cfg.ResultsDir, "fixture-baseline-metrics.json"))
	if err != nil {
		t.Fatal(err)
	}
	if saved.TestStatus != "failed" {
		t.Fatal("failure not persisted")
	}
}
