package release_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDistributionManifestVersionsStayAligned(t *testing.T) {
	t.Parallel()

	manifestVersion := readVersionMap(t, "../../.release-please-manifest.json")["."]
	if manifestVersion == "" {
		t.Fatal("missing root version in .release-please-manifest.json")
	}

	for _, path := range []string{
		"../../.claude-plugin/plugin.json",
		"../../.cursor-plugin/plugin.json",
		"../../plugins/lumen/.codex-plugin/plugin.json",
		"../../plugins/lumen/plugin.json",
		"../../package.json",
	} {
		if got := readVersionField(t, path); got != manifestVersion {
			t.Fatalf("%s version = %q, want %q", path, got, manifestVersion)
		}
	}
}

func TestCodexPluginUsesNativeMCPConfiguration(t *testing.T) {
	t.Parallel()

	var manifest struct {
		Name       string `json:"name"`
		Skills     string `json:"skills"`
		MCPServers string `json:"mcpServers"`
	}
	readJSON(t, "../../plugins/lumen/.codex-plugin/plugin.json", &manifest)
	if manifest.Name != "lumen" {
		t.Fatalf("plugin name = %q, want %q", manifest.Name, "lumen")
	}
	if manifest.Skills != "./skills/" {
		t.Fatalf("skills path = %q, want %q", manifest.Skills, "./skills/")
	}
	if manifest.MCPServers != "./.mcp.json" {
		t.Fatalf("MCP path = %q, want %q", manifest.MCPServers, "./.mcp.json")
	}

	var agentManifest struct {
		Schema string `json:"$schema"`
		Name   string `json:"name"`
	}
	readJSON(t, "../../plugins/lumen/plugin.json", &agentManifest)
	if agentManifest.Schema != "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json" {
		t.Fatalf("Agent Plugins schema = %q", agentManifest.Schema)
	}
	if agentManifest.Name != "lumen" {
		t.Fatalf("Agent Plugins name = %q, want %q", agentManifest.Name, "lumen")
	}

	assertNativeMCPServer(t, "../../plugins/lumen/.mcp.json", "")
	assertNativeMCPServer(t, "../../plugins/lumen/mcp.json", "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json")

	if _, err := os.Stat("../../.codex-plugin/plugin.json"); !os.IsNotExist(err) {
		t.Fatalf("repo-root .codex-plugin/plugin.json must not exist; stat error = %v", err)
	}
}

func assertNativeMCPServer(t *testing.T, path, wantSchema string) {
	t.Helper()

	var mcp struct {
		Schema  string `json:"$schema"`
		Servers map[string]struct {
			Type    string   `json:"type"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	readJSON(t, path, &mcp)
	if mcp.Schema != wantSchema {
		t.Fatalf("%s schema = %q, want %q", path, mcp.Schema, wantSchema)
	}
	lumen, ok := mcp.Servers["lumen"]
	if !ok {
		t.Fatalf("%s is missing the lumen server", path)
	}
	if lumen.Type != "stdio" {
		t.Fatalf("MCP type = %q, want %q", lumen.Type, "stdio")
	}
	if lumen.Command != "./scripts/run" {
		t.Fatalf("MCP command = %q, want %q", lumen.Command, "./scripts/run")
	}
	if strings.Contains(lumen.Command, "${") {
		t.Fatalf("MCP command must not contain a variable placeholder: %q", lumen.Command)
	}
	if !reflect.DeepEqual(lumen.Args, []string{"stdio"}) {
		t.Fatalf("MCP args = %#v, want %#v", lumen.Args, []string{"stdio"})
	}
}

func TestCodexPackageAssetsStayInSync(t *testing.T) {
	t.Parallel()

	for _, paths := range [][2]string{
		{"../../scripts/run", "../../plugins/lumen/scripts/run"},
		{"../../scripts/run.cmd", "../../plugins/lumen/scripts/run.cmd"},
		{"../../skills/doctor/SKILL.md", "../../plugins/lumen/skills/doctor/SKILL.md"},
		{"../../skills/reindex/SKILL.md", "../../plugins/lumen/skills/reindex/SKILL.md"},
	} {
		root := readFile(t, paths[0])
		packaged := readFile(t, paths[1])
		if !bytes.Equal(root, packaged) {
			t.Errorf("%s differs from %s", paths[1], paths[0])
		}
	}
}

func TestRepoLocalCodexMarketplace(t *testing.T) {
	t.Parallel()

	var marketplace struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name   string `json:"name"`
			Source struct {
				Source string `json:"source"`
				Path   string `json:"path"`
			} `json:"source"`
			Policy struct {
				Products []string `json:"products"`
			} `json:"policy"`
		} `json:"plugins"`
	}
	readJSON(t, "../../.agents/plugins/marketplace.json", &marketplace)
	if marketplace.Name != "lumen-local" {
		t.Fatalf("marketplace name = %q, want %q", marketplace.Name, "lumen-local")
	}
	if len(marketplace.Plugins) != 1 {
		t.Fatalf("marketplace plugin count = %d, want 1", len(marketplace.Plugins))
	}
	plugin := marketplace.Plugins[0]
	if plugin.Name != "lumen" || plugin.Source.Source != "local" || plugin.Source.Path != "./plugins/lumen" {
		t.Fatalf("unexpected marketplace plugin: %#v", plugin)
	}
	if !reflect.DeepEqual(plugin.Policy.Products, []string{"CODEX"}) {
		t.Fatalf("marketplace products = %#v, want %#v", plugin.Policy.Products, []string{"CODEX"})
	}
}

func readVersionMap(t *testing.T, path string) map[string]string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}

	out := map[string]string{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal(%q): %v", path, err)
	}
	return out
}

func readJSON(t *testing.T, path string, out any) {
	t.Helper()

	data := readFile(t, path)
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("Unmarshal(%q): %v", path, err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return data
}

func readVersionField(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}

	var out struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal(%q): %v", path, err)
	}
	return out.Version
}
