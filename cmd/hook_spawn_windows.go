//go:build windows

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
	"path/filepath"
	"syscall"

	"github.com/ory/lumen/internal/config"
)

// spawnBackgroundIndexer launches "lumen index <projectPath>" as a detached
// background process on Windows using CREATE_NEW_PROCESS_GROUP and
// DETACHED_PROCESS flags. The spawned process acquires an advisory lock
// (via LockFileEx) before indexing, so concurrent calls are safe.
//
// Errors are silently ignored: background indexing is best-effort.
func spawnBackgroundIndexer(projectPath string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "index", projectPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008, // DETACHED_PROCESS
	}
	cmd.Stdout = nil

	logPath := filepath.Join(config.XDGDataDir(), "lumen", "debug.log")
	if f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		cmd.Stderr = f
		defer f.Close()
	}

	if err := cmd.Start(); err != nil {
		return
	}
	// Reap the child to avoid resource leaks.
	go func() { _ = cmd.Wait() }()
}
