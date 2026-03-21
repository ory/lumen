//go:build !windows

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
