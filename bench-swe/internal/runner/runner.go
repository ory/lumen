package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
)

type Config struct {
	LumenBinary string
	RepoRoot    string
	ResultsDir  string
	Backend     string
	EmbedModel  string
	ClaudeModel string
	TotalRuns   int
}

type RunResult struct {
	TaskID     string
	Scenario   Scenario
	RunIndex   int
	Metrics    *metrics.Metrics
	PatchPath  string
	RawPath    string
	TestOutput string
}

// Slug returns the file prefix for a given task, scenario, and run.
// When totalRuns == 1, no run suffix is added for backwards compatibility.
func Slug(taskID string, s Scenario, runIndex, totalRuns int) string {
	slug := taskID + "-" + string(s)
	if totalRuns > 1 {
		slug += fmt.Sprintf("-run%d", runIndex)
	}
	return slug
}

func Run(ctx context.Context, cfg *Config, t task.Task, s Scenario, runIndex int) (*RunResult, error) {

	workdir, err := os.MkdirTemp("", "bench-swe-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(workdir) }()
	// Resolve symlinks so the DB path computed during indexing matches the CWD
	// that Claude reports via os.Getwd() (e.g. /var/... vs /private/var/... on macOS).
	if resolved, err := filepath.EvalSymlinks(workdir); err == nil {
		workdir = resolved
	}
	repoDir := filepath.Join(workdir, "repo")

	// Shallow clone: try fetching the single base commit directly first.
	// If the remote rejects the SHA (GitHub doesn't support allowReachableSHA1InWant),
	// fall back to a full clone and checkout the commit.
	cloneCtx, cloneCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cloneCancel()
	for _, args := range [][]string{
		{"init", "--quiet", repoDir},
		{"-C", repoDir, "remote", "add", "origin", t.Repo},
	} {
		if out, err := exec.CommandContext(cloneCtx, "git", args...).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git %s: %w\n%s", args[0], err, out)
		}
	}

	// Try direct SHA fetch first (fastest, works on some Git servers)
	shallowArgs := []string{"-C", repoDir, "fetch", "--quiet", "--depth=1", "--filter=blob:none", "origin", t.BaseCommit}
	if _, err := exec.CommandContext(cloneCtx, "git", shallowArgs...).CombinedOutput(); err != nil {
		// Direct SHA fetch failed — remove partial repo and do a full clone instead.
		// Partial clone config left behind by --filter=blob:none would cause checkout
		// to lazily fetch objects by SHA, which GitHub also rejects.
		fmt.Printf("  %-20s direct SHA fetch failed, falling back to full clone\n", t.ID)
		_ = os.RemoveAll(repoDir)
		cloneArgs := []string{"clone", "--quiet", t.Repo, repoDir}
		if out, err := exec.CommandContext(cloneCtx, "git", cloneArgs...).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git clone fallback: %w\n%s", err, out)
		}
	}

	if out, err := exec.CommandContext(cloneCtx, "git", "-C", repoDir, "checkout", "--quiet", t.BaseCommit).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git checkout: %w\n%s", err, out)
	}

	// For Python tasks, create an isolated venv to avoid system/conda package conflicts
	var taskEnv []string
	if t.Language == "python" {
		venvDir := filepath.Join(workdir, "venv")
		venvCmd := exec.CommandContext(ctx, "python3", "-m", "venv", venvDir)
		venvCmd.Dir = repoDir
		if out, err := venvCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("creating python venv: %w\n%s", err, out)
		}
		venvBin := filepath.Join(venvDir, "bin")
		taskEnv = []string{
			"VIRTUAL_ENV=" + venvDir,
			"PATH=" + venvBin + ":" + os.Getenv("PATH"),
		}
	}

	// Run setup commands
	for _, setupCmd := range t.SetupCommands {
		cmd := exec.CommandContext(ctx, "sh", "-c", setupCmd)
		cmd.Dir = repoDir
		if len(taskEnv) > 0 {
			cmd.Env = append(os.Environ(), taskEnv...)
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("setup command %q: %w\n%s", setupCmd, err, out)
		}
	}

	// Index if lumen scenario
	if s == WithLumen {
		fmt.Printf("  %-20s %-10s indexing...\n", t.ID, string(s))
		indexStart := time.Now()
		cmd := exec.CommandContext(ctx, cfg.LumenBinary, "index", repoDir)
		cmd.Env = append(os.Environ(),
			"LUMEN_BACKEND="+cfg.Backend,
			"LUMEN_EMBED_MODEL="+cfg.EmbedModel,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("  %-20s %-10s index FAILED (%v): %s\n", t.ID, string(s), err, out)
		} else {
			fmt.Printf("  %-20s %-10s indexed    [%.1fs]\n", t.ID, string(s), time.Since(indexStart).Seconds())
		}
	}

	// Write MCP config
	mcpPath, cleanup, err := WriteMCPConfig(s, cfg.LumenBinary, cfg.Backend, cfg.EmbedModel)
	if err != nil {
		return nil, fmt.Errorf("writing MCP config: %w", err)
	}
	defer cleanup()

	// Build Claude args
	slug := Slug(t.ID, s, runIndex, cfg.TotalRuns)
	rawPath := filepath.Join(cfg.ResultsDir, slug+"-raw.jsonl")

	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--model", cfg.ClaudeModel,
		"--effort", "high",
		"--strict-mcp-config",
		"--dangerously-skip-permissions",
		"--mcp-config", mcpPath,
	}
	args = append(args, ClaudeArgs(s, cfg.RepoRoot)...)

	prompt := fmt.Sprintf("You are working in this repository. Fix the following issue.\nMake only the minimal changes needed to resolve it. Do not refactor unrelated code.\n\nIssue: %s\n\n%s", t.IssueTitle, t.IssueBody)
	args = append(args, "--", prompt)

	// Run Claude with timeout
	timeout := time.Duration(t.TimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	claudeCmd := exec.CommandContext(runCtx, "claude", args...)
	claudeCmd.Dir = repoDir
	claudeCmd.Env = append(cleanEnvForClaude(),
		"LUMEN_BACKEND="+cfg.Backend,
		"LUMEN_EMBED_MODEL="+cfg.EmbedModel,
	)
	rawFile, err := os.Create(rawPath)
	if err != nil {
		return nil, fmt.Errorf("creating raw file: %w", err)
	}
	stderrPath := filepath.Join(cfg.ResultsDir, slug+"-stderr.log")
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		return nil, fmt.Errorf("creating stderr file: %w", err)
	}
	defer func() { _ = stderrFile.Close() }()

	claudeCmd.Stdout = rawFile
	claudeCmd.Stderr = stderrFile

	_ = claudeCmd.Run() // non-zero exit is OK
	_ = rawFile.Close()

	// Capture patch
	_ = exec.CommandContext(ctx, "git", "-C", repoDir, "add", "-A").Run()
	patchPath := filepath.Join(cfg.ResultsDir, slug+"-patch.diff")
	diffOut, _ := exec.CommandContext(ctx, "git", "-C", repoDir, "diff", t.BaseCommit).Output()
	_ = os.WriteFile(patchPath, diffOut, 0o644)

	// Extract metrics
	m, err := metrics.ExtractFromJSONL(rawPath)
	if err != nil {
		return &RunResult{
			TaskID:   t.ID,
			Scenario: s,
			RunIndex: runIndex,
			RawPath:  rawPath,
		}, nil
	}

	metricsPath := filepath.Join(cfg.ResultsDir, slug+"-metrics.json")
	_ = m.SaveToFile(metricsPath)

	// Run tests if defined
	var testOutput string
	if t.TestCommand != "" {
		testCmd := exec.CommandContext(ctx, "sh", "-c", t.TestCommand)
		testCmd.Dir = repoDir
		if len(taskEnv) > 0 {
			testCmd.Env = append(os.Environ(), taskEnv...)
		}
		out, _ := testCmd.CombinedOutput()
		testOutput = string(out)
		testPath := filepath.Join(cfg.ResultsDir, slug+"-tests.txt")
		_ = os.WriteFile(testPath, out, 0o644)
	}

	// Strip PII
	stripPII(rawPath)

	return &RunResult{
		TaskID:     t.ID,
		Scenario:   s,
		RunIndex:   runIndex,
		Metrics:    m,
		PatchPath:  patchPath,
		RawPath:    rawPath,
		TestOutput: testOutput,
	}, nil
}

// cleanEnvForClaude returns os.Environ() without CLAUDECODE so that
// claude can be spawned from inside a Claude Code session.
func cleanEnvForClaude() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			env = append(env, e)
		}
	}
	return env
}

func stripPII(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	s := string(data)
	home := os.Getenv("HOME")
	if home != "" {
		s = strings.ReplaceAll(s, home, "~")
	}
	// Don't replace bare USER — it's already covered by HOME replacement
	// since HOME typically contains the username (e.g., /Users/dan -> ~)
	_ = os.WriteFile(path, []byte(s), 0o644)
}
