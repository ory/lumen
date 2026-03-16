package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/aeneasr/lumen/bench-swe/internal/analysis"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
	"github.com/aeneasr/lumen/bench-swe/internal/tui"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze <results-dir>",
	Short: "Analyze benchmark conversations for chunker improvement insights",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyze,
}

func runAnalyze(_ *cobra.Command, args []string) error {
	resultsDir := args[0]

	benchDir, err := findBenchDir()
	if err != nil {
		return err
	}
	tasksDir := filepath.Join(benchDir, "tasks")

	tasks, err := task.LoadTasks(tasksDir, nil)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	p := tui.NewProgress(os.Stderr)
	p.Info(fmt.Sprintf("Analyzing %d tasks from %s", len(tasks), resultsDir))
	if err := analysis.Analyze(resultsDir, benchDir, tasks); err != nil {
		return err
	}
	p.Complete("Analysis complete.")
	return nil
}
