//go:build windows

package sync

import "fmt"

// GetDaemonStatus checks if the daemon is running.
// Not supported on Windows.
func GetDaemonStatus() (*DaemonStatus, error) {
	return nil, fmt.Errorf("sync daemon is not supported on Windows")
}

// WritePIDFile writes the current process ID to the PID file.
// Not supported on Windows.
func WritePIDFile() error {
	return fmt.Errorf("sync daemon is not supported on Windows")
}

// RemovePIDFile removes the PID file.
// Not supported on Windows.
func RemovePIDFile() error {
	return fmt.Errorf("sync daemon is not supported on Windows")
}

// StopDaemon stops the running daemon.
// Not supported on Windows.
func StopDaemon() error {
	return fmt.Errorf("sync daemon is not supported on Windows")
}

// CheckNotAlreadyRunning returns an error if the daemon is already running.
// Not supported on Windows.
func CheckNotAlreadyRunning() error {
	return fmt.Errorf("sync daemon is not supported on Windows")
}

// StartDaemon starts the sync daemon in the background.
// Not supported on Windows.
func StartDaemon(localPath, account, conflict string) (int, error) {
	return 0, fmt.Errorf("sync daemon is not supported on Windows")
}
