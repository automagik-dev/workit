//go:build !windows

package sync

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/automagik-dev/workit/internal/config"
)

// GetDaemonStatus checks if the daemon is running.
func GetDaemonStatus() (*DaemonStatus, error) {
	pidPath, err := PIDFilePath()
	if err != nil {
		return nil, fmt.Errorf("get PID file path: %w", err)
	}

	status := &DaemonStatus{}

	// Read PID file
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return status, nil // Not running
		}

		return nil, fmt.Errorf("read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		status.Error = "invalid PID in file"

		return status, nil
	}

	status.PID = pid

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		return status, nil // Process not found
	}

	// On Unix, FindProcess always succeeds. Check with signal 0.
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process is not running, clean up stale PID file
		_ = os.Remove(pidPath)

		return status, nil
	}

	status.Running = true

	return status, nil
}

// WritePIDFile writes the current process ID to the PID file.
func WritePIDFile() error {
	pidPath, err := PIDFilePath()
	if err != nil {
		return fmt.Errorf("get PID file path: %w", err)
	}

	// Ensure directory exists
	if _, err := config.EnsureDir(); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	pid := os.Getpid()

	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}

	return nil
}

// RemovePIDFile removes the PID file.
func RemovePIDFile() error {
	pidPath, err := PIDFilePath()
	if err != nil {
		return fmt.Errorf("get PID file path: %w", err)
	}

	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove PID file: %w", err)
	}

	return nil
}

// StopDaemon stops the running daemon by sending SIGTERM.
func StopDaemon() error {
	status, err := GetDaemonStatus()
	if err != nil {
		return err
	}

	if !status.Running {
		return fmt.Errorf("daemon is not running")
	}

	process, err := os.FindProcess(status.PID)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %w", err)
	}

	// Wait for process to exit (with timeout)
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)

		err := process.Signal(syscall.Signal(0))
		if err != nil {
			// Process has exited
			_ = RemovePIDFile()

			return nil
		}
	}

	// If still running after 3 seconds, send SIGKILL
	_ = process.Signal(syscall.SIGKILL)
	_ = RemovePIDFile()

	return nil
}

// CheckNotAlreadyRunning returns an error if the daemon is already running.
func CheckNotAlreadyRunning() error {
	status, err := GetDaemonStatus()
	if err != nil {
		return err
	}

	if status.Running {
		return fmt.Errorf("daemon already running with PID %d", status.PID)
	}

	return nil
}

// StartDaemon starts the sync daemon in the background.
// It re-executes the current binary with --internal-daemon flag.
func StartDaemon(localPath, account, conflict string) (int, error) {
	if err := CheckNotAlreadyRunning(); err != nil {
		return 0, err
	}

	// Get the current executable
	executable, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("get executable: %w", err)
	}

	// Get log file path
	logPath, err := LogFilePath()
	if err != nil {
		return 0, fmt.Errorf("get log file path: %w", err)
	}

	// Open log file for output
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	// Build command arguments
	args := []string{
		"sync", "start", localPath,
		"--internal-daemon",
		"--account", account,
		"--conflict", conflict,
	}

	// Create command
	cmd := exec.Command(executable, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil

	// Start in new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start daemon: %w", err)
	}

	pid := cmd.Process.Pid

	// Don't wait for the command to finish
	go func() {
		_ = cmd.Wait()
	}()

	return pid, nil
}
