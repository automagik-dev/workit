package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConflictResolver handles sync conflict detection and resolution.
type ConflictResolver struct {
	strategy ConflictStrategy
	db       *DB
	configID int64
}

// NewConflictResolver creates a new conflict resolver.
func NewConflictResolver(db *DB, configID int64, strategy ConflictStrategy) *ConflictResolver {
	if strategy == "" {
		strategy = ConflictRename
	}

	return &ConflictResolver{
		strategy: strategy,
		db:       db,
		configID: configID,
	}
}

// ConflictInfo contains information about a detected conflict.
type ConflictInfo struct {
	LocalPath      string
	DriveID        string
	LocalMD5       string
	RemoteMD5      string
	LocalMtime     time.Time
	RemoteMtime    time.Time
	LastSyncMD5    string
	LastSyncMtime  time.Time
	LocalModified  bool
	RemoteModified bool
}

// DetectConflict checks if a file has a conflict.
func (r *ConflictResolver) DetectConflict(ctx context.Context, item *SyncItem, localPath string, remoteMD5 string, remoteMtime time.Time) (*ConflictInfo, error) {
	// Get current local file info
	absPath := localPath
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Local file was deleted - no conflict, just needs sync
			return nil, nil
		}

		return nil, fmt.Errorf("stat local file: %w", err)
	}

	// Compute current local MD5
	localMD5, err := computeMD5(absPath)
	if err != nil {
		return nil, fmt.Errorf("compute local md5: %w", err)
	}

	conflict := &ConflictInfo{
		LocalPath:     item.LocalPath,
		DriveID:       item.DriveID,
		LocalMD5:      localMD5,
		RemoteMD5:     remoteMD5,
		LocalMtime:    info.ModTime(),
		RemoteMtime:   remoteMtime,
		LastSyncMD5:   item.LocalMD5, // MD5 at last sync
		LastSyncMtime: item.LocalMtime,
	}

	// Check if local was modified since last sync
	conflict.LocalModified = localMD5 != item.LocalMD5

	// Check if remote was modified since last sync
	conflict.RemoteModified = remoteMD5 != item.RemoteMD5

	// Conflict exists if both local and remote were modified
	if conflict.LocalModified && conflict.RemoteModified {
		return conflict, nil
	}

	return nil, nil
}

// ResolveResult contains the result of conflict resolution.
type ResolveResult struct {
	Action         string // "rename", "local-wins", "remote-wins"
	RenamedPath    string // For rename strategy, the new path
	UploadLocal    bool   // True if local file should be uploaded
	DownloadRemote bool   // True if remote file should be downloaded
}

// Resolve resolves a conflict according to the configured strategy.
func (r *ConflictResolver) Resolve(ctx context.Context, conflict *ConflictInfo, localRoot string) (*ResolveResult, error) {
	result := &ResolveResult{
		Action: string(r.strategy),
	}

	switch r.strategy {
	case ConflictRename:
		// Keep both versions by renaming the local file
		absPath := filepath.Join(localRoot, conflict.LocalPath)
		renamedPath, err := r.renameWithTimestamp(absPath)
		if err != nil {
			return nil, fmt.Errorf("rename local file: %w", err)
		}
		result.RenamedPath = renamedPath
		result.DownloadRemote = true // Download remote to original path

		// Log the conflict
		_ = r.db.AddLogEntry(r.configID, "conflict", conflict.LocalPath, map[string]any{
			"strategy":   "rename",
			"renamed_to": renamedPath,
			"local_md5":  conflict.LocalMD5,
			"remote_md5": conflict.RemoteMD5,
		})

	case ConflictLocalWins:
		// Upload local version to overwrite remote
		result.UploadLocal = true

		// Log the conflict
		_ = r.db.AddLogEntry(r.configID, "conflict", conflict.LocalPath, map[string]any{
			"strategy":   "local-wins",
			"local_md5":  conflict.LocalMD5,
			"remote_md5": conflict.RemoteMD5,
		})

	case ConflictRemoteWins:
		// Download remote version to overwrite local
		result.DownloadRemote = true

		// Log the conflict
		_ = r.db.AddLogEntry(r.configID, "conflict", conflict.LocalPath, map[string]any{
			"strategy":   "remote-wins",
			"local_md5":  conflict.LocalMD5,
			"remote_md5": conflict.RemoteMD5,
		})

	default:
		return nil, fmt.Errorf("unknown conflict strategy: %s", r.strategy)
	}

	return result, nil
}

// renameWithTimestamp renames a file by appending a conflict timestamp.
// Example: file.txt -> file.conflict-2024-01-15-143022.txt
func (r *ConflictResolver) renameWithTimestamp(absPath string) (string, error) {
	dir := filepath.Dir(absPath)
	base := filepath.Base(absPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	timestamp := time.Now().Format("2006-01-02-150405")
	newName := fmt.Sprintf("%s.conflict-%s%s", name, timestamp, ext)
	newPath := filepath.Join(dir, newName)

	if err := os.Rename(absPath, newPath); err != nil {
		return "", fmt.Errorf("rename file: %w", err)
	}

	return newPath, nil
}

// ParseConflictStrategy parses a conflict strategy string.
func ParseConflictStrategy(s string) (ConflictStrategy, error) {
	switch strings.ToLower(s) {
	case "rename", "":
		return ConflictRename, nil
	case "local-wins", "local_wins", "localwins":
		return ConflictLocalWins, nil
	case "remote-wins", "remote_wins", "remotewins":
		return ConflictRemoteWins, nil
	default:
		return "", fmt.Errorf("unknown conflict strategy: %s (valid: rename, local-wins, remote-wins)", s)
	}
}
