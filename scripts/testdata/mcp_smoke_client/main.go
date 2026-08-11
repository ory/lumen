// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command mcp_smoke_client verifies an installed Lumen MCP command end to end.
// It performs initialize, tools/list, health_check, semantic_search, and
// index_status against a real fixture instead of merely inspecting host config.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type codexConfig struct {
	Transport struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
	} `json:"transport"`
}

func main() {
	var args stringList
	var envVars stringList
	command := flag.String("command", "", "installed MCP command to execute")
	codexConfigPath := flag.String("codex-config", "", "JSON emitted by `codex mcp get lumen --json`")
	project := flag.String("project", "", "fixture project to index and search")
	query := flag.String("query", "authentication token validation", "semantic search query")
	expect := flag.String("expect", "ValidateToken", "text expected in semantic search output")
	flag.Var(&args, "arg", "MCP command argument (repeatable)")
	flag.Var(&envVars, "env", "MCP environment entry KEY=VALUE (repeatable)")
	flag.Parse()

	if *project == "" {
		fatalf("--project is required")
	}
	absProject, err := filepath.Abs(*project)
	if err != nil {
		fatalf("resolve project: %v", err)
	}

	transportEnv := map[string]string{}
	if *codexConfigPath != "" {
		data, err := os.ReadFile(*codexConfigPath)
		if err != nil {
			fatalf("read Codex MCP config: %v", err)
		}
		var cfg codexConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			fatalf("decode Codex MCP config: %v", err)
		}
		*command = cfg.Transport.Command
		args = append(args, cfg.Transport.Args...)
		for key, value := range cfg.Transport.Env {
			transportEnv[key] = value
		}
	}
	if *command == "" {
		fatalf("--command or --codex-config is required")
	}
	if !filepath.IsAbs(*command) {
		fatalf("resolved MCP command is not absolute: %s", *command)
	}
	if strings.Contains(*command, "${") {
		fatalf("resolved MCP command contains an unexpanded variable: %s", *command)
	}
	for _, item := range envVars {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			fatalf("invalid --env %q; want KEY=VALUE", item)
		}
		transportEnv[key] = value
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, *command, args...)
	cmd.Dir = absProject
	cmd.Env = os.Environ()
	for key, value := range transportEnv {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "lumen-install-smoke", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		fatalf("initialize installed MCP server: %v", err)
	}
	defer func() { _ = session.Close() }()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		fatalf("tools/list: %v", err)
	}
	discovered := map[string]bool{}
	for _, tool := range tools.Tools {
		discovered[tool.Name] = true
	}
	for _, name := range []string{"semantic_search", "index_status", "health_check"} {
		if !discovered[name] {
			fatalf("tools/list did not expose %q (got %v)", name, discovered)
		}
	}

	health := call(ctx, session, "health_check", map[string]any{})
	if health.IsError || !strings.Contains(textContent(health), "Status: OK") {
		fatalf("health_check did not report a ready model: %s", textContent(health))
	}

	search := call(ctx, session, "semantic_search", map[string]any{
		"query": *query, "path": absProject, "cwd": absProject, "limit": 10, "min_score": -1,
	})
	searchText := textContent(search)
	if search.IsError || !strings.Contains(searchText, *expect) {
		fatalf("semantic_search did not return %q: %s", *expect, searchText)
	}

	status := call(ctx, session, "index_status", map[string]any{"path": absProject})
	statusText := textContent(status)
	match := regexp.MustCompile(`Chunks: ([1-9][0-9]*)`).FindStringSubmatch(statusText)
	if status.IsError || match == nil {
		fatalf("index_status did not report non-zero chunks: %s", statusText)
	}

	fmt.Printf("installed MCP smoke passed: tools/list, health_check, semantic_search, index_status (%s)\n", match[0])
}

func call(ctx context.Context, session *mcp.ClientSession, name string, arguments any) *mcp.CallToolResult {
	data, err := json.Marshal(arguments)
	if err != nil {
		fatalf("encode %s arguments: %v", name, err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: json.RawMessage(data)})
	if err != nil {
		fatalf("call %s: %v", name, err)
	}
	return result
}

func textContent(result *mcp.CallToolResult) string {
	var texts []string
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			texts = append(texts, text.Text)
		}
	}
	return strings.Join(texts, "\n")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "mcp smoke: "+format+"\n", args...)
	os.Exit(1)
}
