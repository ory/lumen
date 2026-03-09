package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/aeneasr/lumen/bench-swe/internal/analysis"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
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

	tasks, err := task.LoadTasks(tasksDir, nil, "")
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	fmt.Printf("Analyzing %d tasks from %s...\n", len(tasks), resultsDir)
	return analysis.Analyze(resultsDir, benchDir, tasks)
}
