package runner

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aeneasr/lumen/bench-swe/internal/agent"
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

// MCPServerName is the MCP server name the benchmark registers lumen under.
// The plugin hooks (SessionStart nudge, PreToolUse suggestion) tell Claude to
// call mcp__plugin_lumen_lumen__semantic_search, the name a real plugin
// install produces. Registering the dynamic server as "plugin_lumen_lumen"
// yields the same tool name so the hooks point at a tool that exists.
const MCPServerName = "plugin_lumen_lumen"

type mcpConfig struct {
	MCPServers map[string]mcpServer `json:"mcpServers"`
}

type mcpServer = agent.Server

// ValidRetrieval reports whether mode is a retrieval mode lumen accepts.
func ValidRetrieval(mode string) bool {
	return mode == "bm25" || mode == "semantic"
}

// WriteMCPConfig writes a temp MCP config JSON file.
// Returns path to the temp file and a cleanup function.
func WriteMCPConfig(s Scenario, lumenBinary, retrieval, backend, model string) (string, func(), error) {
	cfg, err := scenarioConfig(s, lumenBinary, retrieval, backend, model)
	if err != nil {
		return "", nil, err
	}

	data, err := json.Marshal(cfg)
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

func scenarioConfig(s Scenario, lumenBinary, retrieval, backend, model string) (mcpConfig, error) {
	var cfg mcpConfig

	if !ValidRetrieval(retrieval) {
		return mcpConfig{}, fmt.Errorf("unknown retrieval mode %q (valid: bm25, semantic)", retrieval)
	}

	switch s {
	case Baseline:
		cfg = mcpConfig{MCPServers: map[string]mcpServer{}}
	case WithLumen:
		cfg = mcpConfig{
			MCPServers: map[string]mcpServer{
				MCPServerName: {
					Command: lumenBinary,
					Args:    []string{"stdio"},
					Env: map[string]string{
						"LUMEN_RETRIEVAL":   retrieval,
						"LUMEN_BACKEND":     backend,
						"LUMEN_EMBED_MODEL": model,
					},
				},
			},
		}
	default:
		return mcpConfig{}, fmt.Errorf("unknown scenario: %s", s)
	}

	return cfg, nil
}
