package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/aeneasr/lumen/bench-swe/internal/judge"
	"github.com/aeneasr/lumen/bench-swe/internal/preflight"
	"github.com/aeneasr/lumen/bench-swe/internal/report"
	"github.com/aeneasr/lumen/bench-swe/internal/runner"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
)

var (
	flagEmbedModel    string
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
	runCmd.Flags().StringVar(&flagEmbedModel, "embed-model", "ordis/jina-embeddings-v2-base-code", "Embedding model")
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

	// Resolve paths
	benchDir, err := findBenchDir()
	if err != nil {
		return err
	}
	repoRoot := filepath.Dir(benchDir)
	lumenBinary := filepath.Join(repoRoot, "bin", "lumen")
	tasksDir := filepath.Join(benchDir, "tasks")

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
		modelSlug := filepath.Base(flagEmbedModel)
		ts := time.Now().Format("20060102-150405")
		resultsDir = filepath.Join(repoRoot, "bench-results",
			fmt.Sprintf("swe-%s-%s-%s", ts, backend, modelSlug))
	}
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return fmt.Errorf("creating results dir: %w", err)
	}

	// Preflight
	if !flagSkipPreflight {
		pfCfg := &preflight.Config{
			RepoRoot:    repoRoot,
			LumenBinary: lumenBinary,
			Backend:     backend,
			EmbedModel:  flagEmbedModel,
			OllamaHost:  os.Getenv("OLLAMA_HOST"),
		}
		if err := preflight.Validate(ctx, pfCfg); err != nil {
			return fmt.Errorf("preflight failed: %w", err)
		}
	}

	// Load tasks
	tasks, err := task.LoadTasks(tasksDir, flagLanguage)
	if err != nil {
		return err
	}

	// Validate tasks
	for _, t := range tasks {
		if err := t.Validate(benchDir); err != nil {
			return fmt.Errorf("task validation: %w", err)
		}
	}

	// Parse scenarios
	scenarios, err := runner.ParseScenarios(flagScenario)
	if err != nil {
		return err
	}

	totalRuns := max(flagRuns, 1)

	if totalRuns > 1 {
		fmt.Printf("\nRunning %d tasks x %d scenarios x %d runs (parallel=%d)\n\n",
			len(tasks), len(scenarios), totalRuns, flagParallel)
	} else {
		fmt.Printf("\nRunning %d tasks x %d scenarios (parallel=%d)\n\n",
			len(tasks), len(scenarios), flagParallel)
	}

	// Run tasks
	runCfg := &runner.Config{
		LumenBinary: lumenBinary,
		RepoRoot:    repoRoot,
		ResultsDir:  resultsDir,
		Backend:     backend,
		EmbedModel:  flagEmbedModel,
		ClaudeModel: flagClaudeModel,
		TotalRuns:   totalRuns,
	}

	var mu sync.Mutex
	var results []runner.RunResult

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(flagParallel)

	for _, t := range tasks {
		g.Go(func() error {
			var lines []string
			var taskResults []runner.RunResult
			for _, s := range scenarios {
				for run := 1; run <= totalRuns; run++ {
					result, err := runner.Run(gCtx, runCfg, t, s, run)
					runLabel := fmt.Sprintf("%-10s", s)
					if totalRuns > 1 {
						runLabel = fmt.Sprintf("%-10s run%d", s, run)
					}
					var line string
					if err != nil {
						line = fmt.Sprintf("  %-20s %s ERROR: %v\n", t.ID, runLabel, err)
					} else if result != nil && result.Metrics != nil {
						m := result.Metrics
						durS := float64(m.DurationMS) / 1000.0
						line = fmt.Sprintf("  %-20s %s done  [%5.1fs  $%.4f  in=%d+%dcr  out=%d]\n",
							t.ID, runLabel, durS, m.CostUSD, m.InputTokens, m.CacheRead, m.OutputTokens)
					} else if result != nil {
						line = fmt.Sprintf("  %-20s %s done  (no metrics)\n", t.ID, runLabel)
					}
					lines = append(lines, line)
					if result != nil {
						taskResults = append(taskResults, *result)
					}
				}
			}
			mu.Lock()
			defer mu.Unlock()
			for _, l := range lines {
				fmt.Print(l)
			}
			results = append(results, taskResults...)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Judge (fresh context so a canceled run phase doesn't block judging)
	if !flagSkipJudge {
		fmt.Println("\nJudging results...")
		judgeCtx, judgeCancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer judgeCancel()
		var judgeMu sync.Mutex
		judgeG, judgeCtx := errgroup.WithContext(judgeCtx)
		judgeG.SetLimit(flagParallel)

		for _, t := range tasks {
			judgeG.Go(func() error {
				var lines []string
				for _, s := range scenarios {
					for run := 1; run <= totalRuns; run++ {
						slug := runner.Slug(t.ID, s, run, totalRuns)
						result, err := judgeTask(judgeCtx, benchDir, runCfg, t, s, slug)
						runLabel := fmt.Sprintf("%-10s", s)
						if totalRuns > 1 {
							runLabel = fmt.Sprintf("%-10s run%d", s, run)
						}
						var line string
						if err != nil {
							line = fmt.Sprintf("  %-20s %s error: %v\n", t.ID, runLabel, err)
						} else if result != nil {
							line = fmt.Sprintf("  %-20s %s %s\n", t.ID, runLabel, result.Rating)
						}
						lines = append(lines, line)
					}
				}
				judgeMu.Lock()
				defer judgeMu.Unlock()
				for _, l := range lines {
					fmt.Print(l)
				}
				return nil
			})
		}
		if err := judgeG.Wait(); err != nil {
			fmt.Printf("  Judge error: %v\n", err)
		}
	}

	// Reports
	fmt.Println("\nGenerating reports...")
	rptCfg := &report.Config{
		ResultsDir:  resultsDir,
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

	fmt.Printf("\nResults: %s\n", resultsDir)
	return nil
}

func judgeTask(ctx context.Context, benchDir string, cfg *runner.Config, t task.Task, s runner.Scenario, slug string) (*judge.JudgeResult, error) {
	patchPath := filepath.Join(cfg.ResultsDir, slug+"-patch.diff")

	candidatePatch, err := os.ReadFile(patchPath)
	if err != nil {
		return nil, nil // no patch = skip
	}

	goldPatch, err := t.GoldPatch(benchDir)
	if err != nil {
		return nil, fmt.Errorf("loading gold patch for %s: %w", t.ID, err)
	}

	result, err := judge.Judge(ctx, t.IssueTitle, t.IssueBody, goldPatch, string(candidatePatch), string(s))
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
