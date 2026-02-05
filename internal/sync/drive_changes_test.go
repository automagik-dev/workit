package sync

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDriveChangeOpString(t *testing.T) {
	tests := []struct {
		op   DriveChangeOp
		want string
	}{
		{DriveOpCreate, "create"},
		{DriveOpModify, "modify"},
		{DriveOpDelete, "delete"},
		{DriveChangeOp(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.op.String()
		if got != tt.want {
			t.Errorf("DriveChangeOp(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestDefaultPollInterval(t *testing.T) {
	// Clear any existing env var
	os.Unsetenv("GOG_SYNC_POLL_INTERVAL")

	interval := DefaultPollInterval()
	if interval != 5*time.Second {
		t.Errorf("DefaultPollInterval() = %v, want %v", interval, 5*time.Second)
	}
}

func TestDefaultPollIntervalFromEnv(t *testing.T) {
	os.Setenv("GOG_SYNC_POLL_INTERVAL", "10s")
	defer os.Unsetenv("GOG_SYNC_POLL_INTERVAL")

	interval := DefaultPollInterval()
	if interval != 10*time.Second {
		t.Errorf("DefaultPollInterval() = %v, want %v", interval, 10*time.Second)
	}
}

func TestDefaultPollIntervalInvalidEnv(t *testing.T) {
	os.Setenv("GOG_SYNC_POLL_INTERVAL", "invalid")
	defer os.Unsetenv("GOG_SYNC_POLL_INTERVAL")

	interval := DefaultPollInterval()
	// Should fall back to default
	if interval != 5*time.Second {
		t.Errorf("DefaultPollInterval() = %v, want %v (default on invalid)", interval, 5*time.Second)
	}
}

func TestNewDrivePoller(t *testing.T) {
	poller := NewDrivePoller(nil, nil, 1, "folder-id", 5*time.Second)

	if poller == nil {
		t.Fatal("NewDrivePoller returned nil")
	}
	if poller.folderID != "folder-id" {
		t.Errorf("folderID = %q, want %q", poller.folderID, "folder-id")
	}
	if poller.configID != 1 {
		t.Errorf("configID = %d, want %d", poller.configID, 1)
	}
	if poller.pollInterval != 5*time.Second {
		t.Errorf("pollInterval = %v, want %v", poller.pollInterval, 5*time.Second)
	}
}

func TestDrivePollerEventsChannel(t *testing.T) {
	poller := NewDrivePoller(nil, nil, 1, "folder-id", 5*time.Second)

	events := poller.Events()
	if events == nil {
		t.Error("Events() returned nil channel")
	}
}

func TestDrivePollerErrorsChannel(t *testing.T) {
	poller := NewDrivePoller(nil, nil, 1, "folder-id", 5*time.Second)

	errors := poller.Errors()
	if errors == nil {
		t.Error("Errors() returned nil channel")
	}
}

func TestDrivePollerContextCancellation(t *testing.T) {
	// Create a poller with nil service - Start should handle gracefully
	poller := NewDrivePoller(nil, nil, 1, "folder-id", 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Start should return quickly due to context cancellation
	done := make(chan error, 1)
	go func() {
		done <- poller.Start(ctx)
	}()

	select {
	case err := <-done:
		// Should return context.Canceled or nil
		if err != nil && err != context.Canceled {
			t.Errorf("Start() returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start() did not return after context cancellation")
	}
}

func TestDriveChangeFields(t *testing.T) {
	now := time.Now()
	change := DriveChange{
		FileID:    "file-123",
		FileName:  "test.txt",
		MimeType:  "text/plain",
		Op:        DriveOpCreate,
		Removed:   false,
		Timestamp: now,
	}

	if change.FileID != "file-123" {
		t.Errorf("FileID = %q, want %q", change.FileID, "file-123")
	}
	if change.FileName != "test.txt" {
		t.Errorf("FileName = %q, want %q", change.FileName, "test.txt")
	}
	if change.MimeType != "text/plain" {
		t.Errorf("MimeType = %q, want %q", change.MimeType, "text/plain")
	}
	if change.Op != DriveOpCreate {
		t.Errorf("Op = %v, want %v", change.Op, DriveOpCreate)
	}
	if change.Removed != false {
		t.Errorf("Removed = %v, want %v", change.Removed, false)
	}
	if !change.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", change.Timestamp, now)
	}
}

func TestIsInFolderWithParents(t *testing.T) {
	poller := NewDrivePoller(nil, nil, 1, "target-folder-id", 5*time.Second)

	tests := []struct {
		name     string
		parents  []string
		folderID string
		want     bool
	}{
		{
			name:     "file in target folder",
			parents:  []string{"target-folder-id"},
			folderID: "target-folder-id",
			want:     true,
		},
		{
			name:     "file not in target folder",
			parents:  []string{"other-folder-id"},
			folderID: "target-folder-id",
			want:     false,
		},
		{
			name:     "file with multiple parents including target",
			parents:  []string{"other-folder-id", "target-folder-id"},
			folderID: "target-folder-id",
			want:     true,
		},
		{
			name:     "file with no parents",
			parents:  []string{},
			folderID: "target-folder-id",
			want:     false,
		},
		{
			name:     "nil parents",
			parents:  nil,
			folderID: "target-folder-id",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := poller.isInFolderByParents(tt.parents)
			if got != tt.want {
				t.Errorf("isInFolderByParents(%v) = %v, want %v", tt.parents, got, tt.want)
			}
		})
	}
}
