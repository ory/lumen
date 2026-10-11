package preflight

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/analysis"
	"github.com/aeneasr/lumen/bench-swe/internal/runner"
)

// Smoke exercises the actual client, MCP transport, retrieval, edit and test
// loop on a deterministic offline fixture. It is deliberately not an SDK mock.
func Smoke(ctx context.Context, cfg *runner.Config) error {
	for _, scenario := range runner.AllScenarios() {
		if err := smokeScenario(ctx, cfg, scenario); err != nil {
			return fmt.Errorf("%s/%s/%s smoke: %w", cfg.Agent, cfg.Transport, scenario, err)
		}
	}
	return nil
}

func smokeScenario(ctx context.Context, cfg *runner.Config, scenario runner.Scenario) error {
	dir, err := os.MkdirTemp("", "bench-smoke-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	files := map[string]string{
		"go.mod":    "module example.com/retrybudget\n\ngo 1.22\n",
		"budget.go": "package retrybudget\n\n// RemainingRetries returns the remaining retry budget after used attempts.\nfunc RemainingRetries(limit, used int) int { return limit - used }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	for _, args := range [][]string{{"init", "--quiet"}, {"add", "."}, {"-c", "user.name=Benchmark", "-c", "user.email=benchmark@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "--quiet", "-m", "fixture"}} {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("fixture git: %w: %s", err, out)
		}
	}
	if scenario == runner.WithLumen {
		cmd := exec.CommandContext(ctx, cfg.LumenBinary, "index", dir)
		cmd.Env = append(os.Environ(), "LUMEN_RETRIEVAL="+cfg.Retrieval, "LUMEN_BACKEND="+cfg.Backend, "LUMEN_EMBED_MODEL="+cfg.EmbedModel)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("fixture index: %w: %s", err, out)
		}
	}
	prompt := "Fix the remaining retry budget calculation: when used attempts exceed the limit, return zero, never a negative number. Preserve the existing behavior otherwise. Edit only budget.go."
	if scenario == runner.WithLumen {
		prompt = "First call the Lumen semantic_search MCP tool with query 'remaining retry budget after used attempts' and path " + dir + ". Then " + prompt
	}
	slug := "smoke-" + cfg.Agent + "-" + cfg.Transport + "-" + string(scenario)
	if _, err := runner.ExecuteAgent(ctx, cfg, dir, scenario, prompt, slug, 3*time.Minute, nil); err != nil {
		return err
	}
	rawPath := filepath.Join(cfg.ResultsDir, slug+"-raw.jsonl")
	calls, err := analysis.ExtractToolCalls(rawPath)
	if err != nil {
		return err
	}
	succeeded := false
	for _, call := range calls {
		if strings.HasPrefix(call.Name, "mcp__") && scenario == runner.Baseline {
			return fmt.Errorf("baseline leaked MCP tool %s", call.Name)
		}
		if analysis.IsLumenSearchTool(call.Name) && call.Completed && !call.Failed && strings.Contains(call.Result, "budget.go") {
			succeeded = true
		}
	}
	if scenario == runner.WithLumen && !succeeded {
		return fmt.Errorf("no successful Lumen search returning budget.go in %s", rawPath)
	}
	diff := exec.CommandContext(ctx, "git", "diff", "--", "budget.go")
	diff.Dir = dir
	patch, err := diff.Output()
	if err != nil {
		return err
	}
	if len(patch) == 0 {
		return fmt.Errorf("agent produced no fix")
	}
	if err := os.WriteFile(filepath.Join(cfg.ResultsDir, slug+"-patch.diff"), patch, 0o644); err != nil {
		return err
	}
	// Install the trusted test AFTER the agent exits, so editing tests cannot
	// produce a false pass. Testing this fixture needs no network dependencies.
	test := `package retrybudget
import "testing"
func TestRemainingRetries(t *testing.T) {
 for _, c := range [][3]int{{5,2,3},{5,5,0},{5,7,0},{0,1,0}} {
  if got := RemainingRetries(c[0], c[1]); got != c[2] { t.Errorf("RemainingRetries(%d,%d)=%d want %d",c[0],c[1],got,c[2]) }
 }
}
`
	if err := os.WriteFile(filepath.Join(dir, "budget_test.go"), []byte(test), 0o644); err != nil {
		return err
	}
	testCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(testCtx, "go", "test", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if writeErr := os.WriteFile(filepath.Join(cfg.ResultsDir, slug+"-tests.txt"), out, 0o644); writeErr != nil {
		return writeErr
	}
	if err != nil {
		return fmt.Errorf("fix failed fixture tests: %w: %s", err, out)
	}
	fmt.Printf("  %s: passed (fix verified, Lumen search=%v)\n", slug, succeeded)
	return nil
}
