package sync

import (
	"path/filepath"

	"github.com/automagik-dev/workit/internal/config"
)

const (
	// PIDFileName is the name of the PID file.
	PIDFileName = "sync.pid"
	// LogFileName is the name of the sync log file.
	LogFileName = "sync.log"
)

// PIDFilePath returns the path to the PID file.
func PIDFilePath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, PIDFileName), nil
}

// LogFilePath returns the path to the sync log file.
func LogFilePath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, LogFileName), nil
}

// DaemonStatus represents the status of the daemon.
type DaemonStatus struct {
	Running bool   `json:"running"`
	PID     int    `json:"pid,omitempty"`
	Error   string `json:"error,omitempty"`
}
