//go:build unix

package agent

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// A timeout must stop the SDK, CLI, shell tools and MCP children together.
func manageProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = 5 * time.Second
}
