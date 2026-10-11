//go:build !unix

package agent

import "os/exec"

func manageProcess(cmd *exec.Cmd) {}
