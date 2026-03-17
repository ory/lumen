package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/aeneasr/lumen/bench-swe/internal/task"
	"github.com/aeneasr/lumen/bench-swe/internal/tui"
)

const grepScoreThreshold = 0.5

var validateCmd = &cobra.Command{
	Use:   "validate [tasks-dir]",
	Short: "Validate benchmark tasks for quality issues",
	Long: `Checks all task JSON files for structural validity and greppability.

Greppability measures how many identifiers from the gold patch (file names,
function names) appear verbatim in the issue description. A high score means
Claude can locate the fix with grep alone, making the task a poor benchmark
for evaluating semantic search.

Tasks with a grep score above ` + fmt.Sprintf("%.0f%%", grepScoreThreshold*100) + ` are flagged as unsuitable for lumen benchmarking.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(_ *cobra.Command, args []string) error {
	tasksDir := "tasks"
	if len(args) > 0 {
		tasksDir = args[0]
	}
	baseDir, err := os.Getwd()
	if err != nil {
		return err
	}

	p := tui.NewProgress(os.Stderr)

	tasks, err := task.LoadTasks(tasksDir, nil)
	if err != nil {
		return err
	}

	p.Info(fmt.Sprintf("Validating %d tasks from %s", len(tasks), tasksDir))

	type result struct {
		t      task.Task
		score  float64
		leaked []string
		err    string
	}

	var results []result
	for _, t := range tasks {
		r := result{t: t}
		if err := t.Validate(baseDir); err != nil {
			r.err = err.Error()
			results = append(results, r)
			continue
		}
		patch, err := t.GoldPatch(baseDir)
		if err != nil {
			r.err = err.Error()
			results = append(results, r)
			continue
		}
		r.score, r.leaked = t.GrepScore(patch)
		results = append(results, r)
	}

	// Sort by score descending so worst offenders appear first.
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	var failed int
	for _, r := range results {
		if r.err != "" {
			p.Error(fmt.Sprintf("%s: %s", r.t.ID, r.err))
			failed++
			continue
		}
		msg := fmt.Sprintf("%s  grep_score=%.0f%%", r.t.ID, r.score*100)
		if len(r.leaked) > 0 {
			msg += fmt.Sprintf("  leaked=%v", r.leaked)
		}
		switch {
		case r.score >= grepScoreThreshold:
			p.Error("REJECT " + msg)
			failed++
		case r.score > 0:
			p.Warn("WARN   " + msg)
		default:
			p.Info("OK     " + msg)
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d task(s) failed validation", failed)
	}
	p.Complete(fmt.Sprintf("All %d tasks passed.", len(results)))
	return nil
}
