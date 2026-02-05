// Package sync provides Google Drive sync functionality.
package sync

import (
	"time"
)

// ConflictStrategy defines how to handle sync conflicts.
type ConflictStrategy string

const (
	// ConflictRename keeps both versions by renaming the conflicting file.
	ConflictRename ConflictStrategy = "rename"
	// ConflictLocalWins overwrites the remote with the local version.
	ConflictLocalWins ConflictStrategy = "local-wins"
	// ConflictRemoteWins overwrites the local with the remote version.
	ConflictRemoteWins ConflictStrategy = "remote-wins"
)

// SyncState represents the state of a synced item.
type SyncState string

const (
	// StateSynced means the local and remote files are in sync.
	StateSynced SyncState = "synced"
	// StatePendingUpload means the local file needs to be uploaded.
	StatePendingUpload SyncState = "pending_upload"
	// StatePendingDownload means the remote file needs to be downloaded.
	StatePendingDownload SyncState = "pending_download"
	// StateConflict means both local and remote have changed.
	StateConflict SyncState = "conflict"
	// StateError means there was an error syncing this file.
	StateError SyncState = "error"
)

// SyncConfig represents a sync configuration between a local folder and a Drive folder.
type SyncConfig struct {
	ID            int64     `json:"id"`
	LocalPath     string    `json:"local_path"`
	DriveFolderID string    `json:"drive_folder_id"`
	DriveID       string    `json:"drive_id,omitempty"` // For shared drives
	CreatedAt     time.Time `json:"created_at"`
	LastSyncAt    time.Time `json:"last_sync_at,omitempty"`
	ChangeToken   string    `json:"change_token,omitempty"` // Drive changes page token
}

// SyncItem represents a tracked file/folder in a sync configuration.
type SyncItem struct {
	ID          int64     `json:"id"`
	ConfigID    int64     `json:"config_id"`
	LocalPath   string    `json:"local_path"`  // Relative to config's local_path
	DriveID     string    `json:"drive_id"`    // Drive file ID
	LocalMD5    string    `json:"local_md5"`   // MD5 hash of local file
	RemoteMD5   string    `json:"remote_md5"`  // MD5 hash from Drive
	LocalMtime  time.Time `json:"local_mtime"` // Local modification time
	RemoteMtime time.Time `json:"remote_mtime"`
	SyncState   SyncState `json:"sync_state"`
}

// SyncLogEntry represents an entry in the sync log.
type SyncLogEntry struct {
	ID        int64     `json:"id"`
	ConfigID  int64     `json:"config_id"`
	Action    string    `json:"action"` // upload, download, delete, conflict, error
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"` // JSON details
}

// SyncStatus represents the current status of a sync configuration.
type SyncStatus struct {
	Config        SyncConfig `json:"config"`
	TotalItems    int64      `json:"total_items"`
	SyncedItems   int64      `json:"synced_items"`
	PendingItems  int64      `json:"pending_items"`
	ConflictItems int64      `json:"conflict_items"`
	ErrorItems    int64      `json:"error_items"`
	DaemonRunning bool       `json:"daemon_running"`
}
