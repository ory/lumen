package preflight

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/runner"
)

type Config struct {
	Agent       string
	Model       string
	Auth        string
	ResultsDir  string
	RepoRoot    string
	LumenBinary string
	Retrieval   string
	Backend     string
	EmbedModel  string
	OllamaHost  string
}

func Validate(ctx context.Context, cfg *Config) error {
	fmt.Println("Pre-flight checks...")

	if err := buildLumen(ctx, cfg); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	// BM25 is purely lexical; only semantic retrieval needs an embedding server.
	if cfg.Retrieval == "semantic" {
		if err := checkOllama(cfg); err != nil {
			return fmt.Errorf("ollama: %w", err)
		}
	}

	c := &runner.Config{Agent: cfg.Agent, Model: cfg.Model, Auth: cfg.Auth, Transport: "cli", RepoRoot: cfg.RepoRoot, ResultsDir: cfg.ResultsDir, LumenBinary: cfg.LumenBinary, Retrieval: cfg.Retrieval, Backend: cfg.Backend, EmbedModel: cfg.EmbedModel}
	if err := Smoke(ctx, c); err != nil {
		return err
	}

	fmt.Println("Pre-flight checks passed.")
	return nil
}

func buildLumen(ctx context.Context, cfg *Config) error {
	fmt.Print("  Building lumen... ")
	// BM25 needs SQLite FTS5, which go-sqlite3 only compiles with the fts5 tag.
	cmd := exec.CommandContext(ctx, "go", "build", "-tags", "fts5", "-o", cfg.LumenBinary, ".")
	cmd.Dir = cfg.RepoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	fmt.Println("ok")
	return nil
}

func checkOllama(cfg *Config) error {
	host := cfg.OllamaHost
	if host == "" {
		host = "http://localhost:11434"
	}

	fmt.Printf("  Checking Ollama at %s... ", host)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(host + "/api/tags")
	if err != nil {
		return fmt.Errorf("cannot reach Ollama at %s: %w", host, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ollama returned %d", resp.StatusCode)
	}
	fmt.Println("ok")
	return nil
}

func Build(ctx context.Context, repoRoot, binary string) error {
	return buildLumen(ctx, &Config{RepoRoot: repoRoot, LumenBinary: binary})
}
