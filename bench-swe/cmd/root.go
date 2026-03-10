package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "bench-swe",
	Short: "SWE-bench benchmark pipeline for Lumen",
	Long:  "Runs SWE-bench style coding tasks with and without Lumen semantic search to measure token savings, time, cost, and quality.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(extractCmd)
}
