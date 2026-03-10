package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/aeneasr/lumen/bench-swe/internal/task"
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

	tasks, err := task.LoadTasks(tasksDir, nil)
	if err != nil {
		return err
	}

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
			fmt.Printf("ERROR  %s: %s\n", r.t.ID, r.err)
			failed++
			continue
		}
		label := "OK    "
		if r.score >= grepScoreThreshold {
			label = "REJECT"
			failed++
		} else if r.score > 0 {
			label = "WARN  "
		}
		fmt.Printf("%s %s  grep_score=%.0f%%", label, r.t.ID, r.score*100)
		if len(r.leaked) > 0 {
			fmt.Printf("  leaked=%v", r.leaked)
		}
		fmt.Println()
	}

	if failed > 0 {
		return fmt.Errorf("%d task(s) failed validation", failed)
	}
	fmt.Printf("\nAll %d tasks passed.\n", len(results))
	return nil
}
