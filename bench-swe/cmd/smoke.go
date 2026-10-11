package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/preflight"
	"github.com/aeneasr/lumen/bench-swe/internal/runner"
	"github.com/spf13/cobra"
)

func init() {
	var provider, transport, model, auth, results string
	var skipBuild bool
	cmd := &cobra.Command{Use: "smoke", Short: "Verify real agent fixes and successful Lumen MCP calls", RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		bench, err := findBenchDir()
		if err != nil {
			return err
		}
		root := filepath.Dir(bench)
		if results == "" {
			results = filepath.Join(root, "bench-results", "smoke-"+time.Now().Format("20060102-150405"))
		}
		results, err = filepath.Abs(results)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(results, 0o755); err != nil {
			return err
		}
		providers := []string{provider}
		if provider == "all" {
			providers = []string{"claude", "codex"}
		}
		for _, p := range providers {
			cfg := &runner.Config{Agent: p, Transport: transport, Auth: auth, Model: model, RepoRoot: root, ResultsDir: results, LumenBinary: filepath.Join(root, "bin", "lumen"), Retrieval: "bm25"}
			if p == "claude" && model == "" {
				cfg.Model = "haiku"
			}
			if err := cfg.AgentConfig().Validate(); err != nil {
				return err
			}
			if !skipBuild {
				if err := preflight.Build(ctx, root, cfg.LumenBinary); err != nil {
					return err
				}
				skipBuild = true
			}
			if err := preflight.Smoke(ctx, cfg); err != nil {
				return err
			}
		}
		fmt.Println("Smoke artifacts: " + results)
		return nil
	}}
	cmd.Flags().StringVar(&provider, "agent", "all", "Agent (claude|codex|all)")
	cmd.Flags().StringVar(&transport, "runner", "cli", "Runner (cli|sdk); CLI is the end-to-end gate")
	cmd.Flags().StringVar(&model, "model", "", "Model override (defaults to haiku for Claude, client default for Codex)")
	cmd.Flags().StringVar(&auth, "auth", "subscription", "Authentication (subscription|environment)")
	cmd.Flags().StringVar(&results, "results-dir", "", "Artifact directory")
	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Use the existing bin/lumen")
	rootCmd.AddCommand(cmd)
}
