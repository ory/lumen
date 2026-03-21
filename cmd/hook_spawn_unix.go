//go:build !windows

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

package cmd

import (
	"os"
	"os/exec"
	"syscall"
)

// spawnBackgroundIndexer launches "lumen index <projectPath>" as a fully
// detached background process (new session via Setsid). The spawned process
// acquires an advisory flock before indexing, so concurrent calls from
// multiple Claude terminals are safe — only one indexer runs at a time.
//
// Errors are silently ignored: background indexing is best-effort. If it
// fails, the MCP server falls back to its normal lazy EnsureFresh path.
func spawnBackgroundIndexer(projectPath string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "index", projectPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Start()
}
