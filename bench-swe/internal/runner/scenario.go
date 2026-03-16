package runner

import (
	"encoding/json"
	"fmt"
	"os"
)

type Scenario string

const (
	Baseline  Scenario = "baseline"
	WithLumen Scenario = "with-lumen"
)

func AllScenarios() []Scenario {
	return []Scenario{Baseline, WithLumen}
}

func ParseScenarios(filter string) ([]Scenario, error) {
	switch filter {
	case "", "all":
		return AllScenarios(), nil
	case "baseline":
		return []Scenario{Baseline}, nil
	case "with-lumen":
		return []Scenario{WithLumen}, nil
	default:
		return nil, fmt.Errorf("unknown scenario %q (valid: baseline, with-lumen, all)", filter)
	}
}

type mcpConfig struct {
	MCPServers map[string]mcpServer `json:"mcpServers"`
}

type mcpServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// WriteMCPConfig writes a temp MCP config JSON file.
// Returns path to the temp file and a cleanup function.
func WriteMCPConfig(s Scenario, cfg *Config) (string, func(), error) {
	var mcpCfg mcpConfig

	switch s {
	case Baseline:
		mcpCfg = mcpConfig{MCPServers: map[string]mcpServer{}}
	case WithLumen:
		env := map[string]string{
			"LUMEN_BACKEND":     cfg.Backend,
			"LUMEN_EMBED_MODEL": cfg.EmbedModel,
		}
		if cfg.Summaries {
			env["LUMEN_SUMMARIES"] = "true"
			env["LUMEN_SUMMARY_MODEL"] = cfg.SummaryModel
			env["LUMEN_SUMMARY_EMBED_MODEL"] = cfg.SummaryEmbedModel
		}
		mcpCfg = mcpConfig{
			MCPServers: map[string]mcpServer{
				"lumen": {
					Command: cfg.LumenBinary,
					Args:    []string{"stdio"},
					Env:     env,
				},
			},
		}
	default:
		return "", nil, fmt.Errorf("unknown scenario: %s", s)
	}

	data, err := json.Marshal(mcpCfg)
	if err != nil {
		return "", nil, err
	}

	f, err := os.CreateTemp("", fmt.Sprintf("bench-swe-mcp-%s-*.json", s))
	if err != nil {
		return "", nil, err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", nil, err
	}
	_ = f.Close()

	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

// ClaudeArgs returns the extra CLI arguments for a given scenario.
func ClaudeArgs(s Scenario, repoRoot string) []string {
	switch s {
	case WithLumen:
		return []string{
			"--plugin-dir", repoRoot,
		}
	default:
		return nil
	}
}
