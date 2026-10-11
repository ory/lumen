package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/aeneasr/lumen/bench-swe/internal/judge"
	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
	"github.com/aeneasr/lumen/bench-swe/internal/preflight"
	"github.com/aeneasr/lumen/bench-swe/internal/report"
	"github.com/aeneasr/lumen/bench-swe/internal/runner"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
	"github.com/aeneasr/lumen/bench-swe/internal/tui"
)

var (
	flagAgent         string
	flagRunner        string
	flagModel         string
	flagAuth          string
	flagJudgeModel    string
	flagEmbedModel    string
	flagRetrieval     string
	flagClaudeModel   string
	flagLanguage      []string
	flagScenario      string
	flagParallel      int
	flagRuns          int
	flagResultsDir    string
	flagSkipJudge     bool
	flagSkipPreflight bool
	flagVerbose       bool
	flagOutput        string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run SWE-bench tasks across scenarios",
	RunE:  runBenchmarks,
}

func init() {
	runCmd.Flags().StringVar(&flagAgent, "agent", "claude", "Coding agent (claude|codex)")
	runCmd.Flags().StringVar(&flagRunner, "runner", "cli", "Execution transport (cli|sdk)")
	runCmd.Flags().StringVar(&flagModel, "model", "", "Agent model (Claude defaults to --claude-model; Codex uses client default)")
	runCmd.Flags().StringVar(&flagAuth, "auth", "subscription", "Authentication (subscription|environment)")
	runCmd.Flags().StringVar(&flagJudgeModel, "judge-model", "", "Judge model (same provider; Claude defaults to Sonnet)")
	runCmd.Flags().StringVar(&flagEmbedModel, "embed-model", "ordis/jina-embeddings-v2-base-code", "Embedding model")
	runCmd.Flags().StringVar(&flagRetrieval, "retrieval", "bm25", "Lumen retrieval mode (bm25|semantic)")
	runCmd.Flags().StringVar(&flagClaudeModel, "claude-model", "haiku", "Claude model for tasks")
	runCmd.Flags().StringArrayVar(&flagLanguage, "language", nil, "Filter by language (repeatable)")
	runCmd.Flags().StringVar(&flagScenario, "scenario", "all", "Scenario filter (baseline|with-lumen|all)")
	runCmd.Flags().IntVar(&flagParallel, "parallel", 3, "Max concurrent runs")
	runCmd.Flags().IntVar(&flagRuns, "runs", 1, "Number of runs per task×scenario")
	runCmd.Flags().StringVar(&flagResultsDir, "results-dir", "", "Override results directory")
	runCmd.Flags().BoolVar(&flagSkipJudge, "skip-judge", false, "Skip LLM judging phase")
	runCmd.Flags().BoolVar(&flagSkipPreflight, "skip-preflight", false, "Skip preflight checks")
	runCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Print overview table to stdout after completion")
	runCmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Write overview table to this file path")
}

func runBenchmarks(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	p := tui.NewProgress(os.Stderr)

	// Resolve paths
	benchDir, err := findBenchDir()
	if err != nil {
		return err
	}
	repoRoot := filepath.Dir(benchDir)
	lumenBinary := filepath.Join(repoRoot, "bin", "lumen")
	tasksDir := filepath.Join(benchDir, "tasks")

	if !runner.ValidRetrieval(flagRetrieval) {
		return fmt.Errorf("unknown retrieval mode %q (valid: bm25, semantic)", flagRetrieval)
	}

	model := flagModel
	if flagAgent == "claude" && model == "" {
		model = flagClaudeModel
	}
	agentCfg := (&runner.Config{Agent: flagAgent, Transport: flagRunner, Auth: flagAuth, Model: model}).AgentConfig()
	if err := agentCfg.Validate(); err != nil {
		return err
	}
	if flagParallel < 1 || flagRuns < 1 {
		return fmt.Errorf("--parallel and --runs must be positive")
	}

	// Parse scenarios
	scenarios, err := runner.ParseScenarios(flagScenario)
	if err != nil {
		return err
	}

	if flagRunner == "sdk" {
		pkg := "@openai/codex-sdk"
		if flagAgent == "claude" {
			pkg = "@anthropic-ai/claude-agent-sdk"
		}
		probe := exec.CommandContext(ctx, "node", "--input-type=module", "-e", "await import("+fmt.Sprintf("%q", pkg)+")")
		probe.Dir = benchDir
		if err := probe.Run(); err != nil {
			return fmt.Errorf("SDK unavailable; run npm ci in bench-swe first: %w", err)
		}
	}

	// Detect backend
	backend := os.Getenv("LUMEN_BACKEND")
	if backend == "" {
		backend = "ollama"
		if strings.HasPrefix(flagEmbedModel, "nomic-ai/") {
			backend = "lmstudio"
		}
	}

	// Results directory
	resultsDir := flagResultsDir
	if resultsDir == "" {
		ts := time.Now().Format("20060102-150405")
		resultsDir = filepath.Join(repoRoot, "bench-results",
			fmt.Sprintf("swe-%s-%s-%s-%s", ts, flagAgent, flagRunner, resultsSlug(flagRetrieval, backend, flagEmbedModel)))
	}
	resultsDir, err = filepath.Abs(resultsDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return fmt.Errorf("creating results dir: %w", err)
	}

	// A manifest prevents provider/transport runs from silently overwriting each other.
	manifestPath := filepath.Join(resultsDir, "run-config.json")
	manifest, err := os.OpenFile(manifestPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("results directory already has a run or is not writable: %w", err)
	}
	err = json.NewEncoder(manifest).Encode(struct {
		Agent          any    `json:"agent"`
		Retrieval      string `json:"retrieval"`
		EmbeddingModel string `json:"embedding_model"`
	}{agentCfg, flagRetrieval, flagEmbedModel})
	closeErr := manifest.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}

	// Preflight
	if !flagSkipPreflight {
		p.StartSpinner("Running preflight checks...")
		pfCfg := &preflight.Config{
			Agent: flagAgent, Model: model, Auth: flagAuth, ResultsDir: resultsDir,
			RepoRoot:    repoRoot,
			LumenBinary: lumenBinary,
			Retrieval:   flagRetrieval,
			Backend:     backend,
			EmbedModel:  flagEmbedModel,
			OllamaHost:  os.Getenv("OLLAMA_HOST"),
		}
		pfErr := preflight.Validate(ctx, pfCfg)
		p.StopSpinner()
		if pfErr != nil {
			return fmt.Errorf("preflight failed: %w", pfErr)
		}
	}

	// Load tasks
	p.StartSpinner("Loading tasks...")
	tasks, err := task.LoadTasks(tasksDir, flagLanguage)
	p.StopSpinner()
	if err != nil {
		return err
	}

	// Validate tasks
	for _, t := range tasks {
		if err := t.Validate(benchDir); err != nil {
			return fmt.Errorf("task validation: %w", err)
		}
	}

	totalRuns := max(flagRuns, 1)
	total := len(tasks) * len(scenarios) * totalRuns

	if totalRuns > 1 {
		p.Info(fmt.Sprintf("Running %d tasks × %d scenarios × %d runs (parallel=%d)",
			len(tasks), len(scenarios), totalRuns, flagParallel))
	} else {
		p.Info(fmt.Sprintf("Running %d tasks × %d scenarios (parallel=%d)",
			len(tasks), len(scenarios), flagParallel))
	}

	// Run tasks
	runCfg := &runner.Config{
		Agent: flagAgent, Transport: flagRunner, Model: model, Auth: flagAuth, JudgeModel: flagJudgeModel,
		LumenBinary: lumenBinary,
		RepoRoot:    repoRoot,
		ResultsDir:  resultsDir,
		Retrieval:   flagRetrieval,
		Backend:     backend,
		EmbedModel:  flagEmbedModel,
		ClaudeModel: flagClaudeModel,
		TotalRuns:   totalRuns,
	}

	var mu sync.Mutex
	var results []runner.RunResult
	var runRows [][]string
	completed := 0
	failures := 0

	p.Start("Running", total)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(flagParallel)

	for _, t := range tasks {
		g.Go(func() error {
			var taskResults []runner.RunResult
			var taskRows [][]string
			taskFailures := 0
			for _, s := range scenarios {
				for run := 1; run <= totalRuns; run++ {
					result, err := runner.Run(gCtx, runCfg, t, s, run)
					runLabel := string(s)
					if totalRuns > 1 {
						runLabel = fmt.Sprintf("%s run%d", s, run)
					}
					var row []string
					if err != nil {
						taskFailures++
						row = []string{t.ID, runLabel, "—", "—", "—", "ERROR: " + err.Error()}
					} else if result != nil && result.Metrics != nil {
						m := result.Metrics
						status := "done"
						if m.TestStatus != "" {
							status += "; tests " + m.TestStatus
						}
						row = []string{
							t.ID,
							runLabel,
							fmt.Sprintf("%.1fs", float64(m.DurationMS)/1000.0),
							m.CostLabel(),
							fmt.Sprintf("%d+%dcr/%d", m.InputTokens, m.CacheRead, m.OutputTokens),
							status,
						}
					} else {
						row = []string{t.ID, runLabel, "—", "—", "—", "done (no metrics)"}
					}
					taskRows = append(taskRows, row)
					if result != nil {
						taskResults = append(taskResults, *result)
					}
				}
			}
			mu.Lock()
			completed += len(taskRows)
			failures += taskFailures
			p.Update(completed, t.ID)
			runRows = append(runRows, taskRows...)
			results = append(results, taskResults...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		p.Stop()
		return err
	}
	p.Stop()

	p.PrintTable([]string{"Task", "Scenario", "Time", "Cost", "Tokens (in+cr/out)", "Status"}, runRows)

	// Judge (fresh context so a canceled run phase doesn't block judging)
	if !flagSkipJudge {
		p.Info("Judging results...")
		judgeCtx, judgeCancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer judgeCancel()

		var judgeMu sync.Mutex
		var judgeRows [][]string
		judgeCompleted := 0
		judgeFailures := 0
		judgeTotal := len(tasks) * len(scenarios) * totalRuns

		p.Start("Judging", judgeTotal)

		judgeG, judgeCtx := errgroup.WithContext(judgeCtx)
		judgeG.SetLimit(flagParallel)

		for _, t := range tasks {
			judgeG.Go(func() error {
				var taskRows [][]string
				taskFailures := 0
				for _, s := range scenarios {
					for run := 1; run <= totalRuns; run++ {
						slug := runner.Slug(t.ID, s, run, totalRuns)
						result, err := judgeTask(judgeCtx, benchDir, runCfg, t, s, slug)
						runLabel := string(s)
						if totalRuns > 1 {
							runLabel = fmt.Sprintf("%s run%d", s, run)
						}
						var row []string
						if err != nil {
							taskFailures++
							row = []string{t.ID, runLabel, "ERROR: " + err.Error()}
						} else if result != nil {
							row = []string{t.ID, runLabel, string(result.Rating)}
						} else {
							row = []string{t.ID, runLabel, "—"}
						}
						taskRows = append(taskRows, row)
					}
				}
				judgeMu.Lock()
				judgeCompleted += len(taskRows)
				judgeFailures += taskFailures
				p.Update(judgeCompleted, t.ID)
				judgeRows = append(judgeRows, taskRows...)
				judgeMu.Unlock()
				return nil
			})
		}

		if err := judgeG.Wait(); err != nil {
			p.Stop()
			p.Error(fmt.Sprintf("Judge error: %v", err))
		} else {
			p.Stop()
		}
		p.PrintTable([]string{"Task", "Scenario", "Rating"}, judgeRows)
		failures += judgeFailures
	}

	// Reports
	p.Info("Generating reports...")
	rptCfg := &report.Config{
		Agent: flagAgent, Runner: flagRunner, Model: model,
		ResultsDir:  resultsDir,
		Retrieval:   flagRetrieval,
		EmbedModel:  flagEmbedModel,
		ClaudeModel: flagClaudeModel,
		Tasks:       tasks,
		Scenarios:   scenarios,
		Runs:        totalRuns,
	}
	if err := report.Generate(rptCfg); err != nil {
		return err
	}
	rptCfg.Verbose = flagVerbose
	rptCfg.OutputPath = flagOutput
	if err := report.GenerateOverview(rptCfg); err != nil {
		return err
	}

	if failures > 0 {
		return fmt.Errorf("%d run/judge failure(s); see artifacts in %s", failures, resultsDir)
	}
	p.Complete("Results: " + resultsDir)
	return nil
}

func judgeTask(ctx context.Context, benchDir string, cfg *runner.Config, t task.Task, s runner.Scenario, slug string) (*judge.JudgeResult, error) {
	m, err := metrics.LoadFromFile(filepath.Join(cfg.ResultsDir, slug+"-metrics.json"))
	if err != nil || m.Error != "" {
		return nil, nil
	}
	patchPath := filepath.Join(cfg.ResultsDir, slug+"-patch.diff")

	candidatePatch, err := os.ReadFile(patchPath)
	if err != nil {
		return nil, nil // no patch = skip
	}

	goldPatch, err := t.GoldPatch(benchDir)
	if err != nil {
		return nil, fmt.Errorf("loading gold patch for %s: %w", t.ID, err)
	}

	judgeCfg := cfg.AgentConfig()
	judgeCfg.Model = cfg.JudgeModel
	if judgeCfg.Model == "" {
		if judgeCfg.Provider == "claude" {
			judgeCfg.Model = "claude-sonnet-4-6"
		} else {
			judgeCfg.Model = cfg.Model
		}
	}
	result, err := judge.JudgeWithAgent(ctx, judgeCfg, t.IssueTitle, t.IssueBody, goldPatch, string(candidatePatch), string(s))
	if err != nil {
		return nil, err
	}

	jsonPath := filepath.Join(cfg.ResultsDir, slug+"-judge.json")
	mdPath := filepath.Join(cfg.ResultsDir, slug+"-judge.md")
	if err := judge.SaveResult(result, jsonPath, mdPath); err != nil {
		return nil, err
	}

	return result, nil
}

func findBenchDir() (string, error) {
	// Try current directory first
	if _, err := os.Stat("tasks"); err == nil {
		abs, _ := filepath.Abs(".")
		return abs, nil
	}
	// Try bench-swe subdirectory
	if _, err := os.Stat("bench-swe/tasks"); err == nil {
		abs, _ := filepath.Abs("bench-swe")
		return abs, nil
	}
	// Try relative to executable
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "tasks")); err == nil {
		return dir, nil
	}
	return "", fmt.Errorf("cannot find bench-swe directory (looked for tasks/ in cwd, bench-swe/, and exe dir)")
}

// resultsSlug names a results directory after what produced the index: the
// embedding backend and model for semantic retrieval, or just "bm25".
func resultsSlug(retrieval, backend, embedModel string) string {
	if retrieval == "bm25" {
		return "bm25"
	}
	return backend + "-" + filepath.Base(embedModel)
}
