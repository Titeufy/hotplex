//go:build !windows

package sys

import (
	"os"
	"os/exec"
	"syscall"
)

// SetupCmdSysProcAttr configures the command to run in its own process group (Unix).
// Returns zero handle (unused on Unix).
func SetupCmdSysProcAttr(cmd *exec.Cmd) (uintptr, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return 0, nil
}

// KillProcessGroup terminates the entire process tree using the negative PID (Unix).
// The jobHandle parameter is ignored on Unix (only used on Windows).
func KillProcessGroup(cmd *exec.Cmd, jobHandle uintptr) {
	if cmd != nil && cmd.Process != nil {
		// We set Setpgid = true in SetupCmdSysProcAttr, so negate the PID to kill the group.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck
	}
}

// AssignProcessToJob is a no-op on Unix (only used on Windows).
func AssignProcessToJob(jobHandle uintptr, process *os.Process) error {
	return nil
}

// CloseJobHandle is a no-op on Unix (only used on Windows).
func CloseJobHandle(jobHandle uintptr) {
	// No-op on Unix
}

// IsProcessAlive checks if the process is still running using Signal(0) (Unix).
func IsProcessAlive(process *os.Process) bool {
	if process == nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
