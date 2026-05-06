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

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestLauncherSpawnStdioHandshake spawns `node launcher.mjs stdio` via
// exec.Command (no shell) and runs the MCP initialize handshake. Validates
// that the launcher chain — Node -> launcher.mjs -> lumen binary -> MCP
// stdio server — produces a clean JSON-RPC stream with no leading garbage.
//
// This is the regression gate the proposal calls for: any byte-0 / shebang /
// shell-fragment regression that the polyglot kept hitting fails this test
// deterministically because exec.Command bypasses shell entirely.
func TestLauncherSpawnStdioHandshake(t *testing.T) {
	cmd := exec.Command(nodePath(t), launcherPath(t), "stdio")
	cmd.Env = sandboxEnv(t)

	transport := &mcp.CommandTransport{Command: cmd}
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "integration-spawn-test",
		Version: "0.1.0",
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("MCP initialize handshake failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
}

// TestLauncherSpawnHookSessionStart spawns the SessionStart hook subcommand
// via the launcher and verifies it produces a valid JSON object on stdout
// with a clean exit. Mirrors the invocation pattern in hooks/hooks.json and
// hooks/hooks-cursor.json after Phase 2 migrates them off run.cmd.
func TestLauncherSpawnHookSessionStart(t *testing.T) {
	for _, host := range []string{"claude", "cursor"} {
		t.Run(host, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx,
				nodePath(t), launcherPath(t),
				"hook", "session-start", "lumen", "--host", host,
			)
			cmd.Env = sandboxEnv(t)
			// Hook reads optional JSON from stdin ({"cwd":"..."}); empty input is fine —
			// the handler falls back to os.Getwd().
			cmd.Stdin = bytes.NewReader(nil)

			stdout, err := cmd.Output()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					t.Fatalf("hook session-start (host=%s) exited %d: stderr=%s", host, ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
				}
				t.Fatalf("hook session-start (host=%s) failed: %v", host, err)
			}

			// Output must be a single, parseable JSON object — proves no shell
			// echo / shebang noise leaked into stdout.
			trimmed := bytes.TrimSpace(stdout)
			var payload map[string]any
			if err := json.Unmarshal(trimmed, &payload); err != nil {
				t.Fatalf("hook session-start (host=%s) stdout is not valid JSON: %v\nstdout=%q", host, err, stdout)
			}
			if len(payload) == 0 {
				t.Fatalf("hook session-start (host=%s) returned empty JSON object", host)
			}
		})
	}
}

// TestLauncherSpawnHookPreToolUse exercises the second hook subcommand wired
// into hooks/hooks.json. Pre-tool-use exits cleanly without producing JSON
// when there's nothing to intercept (it reads tool_input from stdin).
func TestLauncherSpawnHookPreToolUse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		nodePath(t), launcherPath(t),
		"hook", "pre-tool-use", "lumen",
	)
	cmd.Env = sandboxEnv(t)
	cmd.Stdin = bytes.NewReader(nil)

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("hook pre-tool-use exited %d: stderr=%s", ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		t.Fatalf("hook pre-tool-use failed: %v", err)
	}
}
