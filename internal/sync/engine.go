package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/api/drive/v3"
)

// Engine orchestrates bidirectional sync between a local folder and Google Drive.
type Engine struct {
	db       *DB
	config   *SyncConfig
	service  *drive.Service
	watcher  *Watcher
	poller   *DrivePoller
	uploader *Uploader
	dloader  *Downloader

	mu      sync.Mutex
	running bool
}

// EngineOptions configures the sync engine.
type EngineOptions struct {
	DB           *DB
	Config       *SyncConfig
	DriveService *drive.Service
	Debounce     time.Duration
	PollInterval time.Duration
}

// NewEngine creates a new sync engine.
func NewEngine(opts EngineOptions) (*Engine, error) {
	if opts.Debounce == 0 {
		opts.Debounce = 500 * time.Millisecond
	}

	if opts.PollInterval == 0 {
		opts.PollInterval = DefaultPollInterval()
	}

	watcher, err := NewWatcher(opts.Config.LocalPath, opts.Debounce)
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	poller := NewDrivePoller(
		opts.DriveService,
		opts.DB,
		opts.Config.ID,
		opts.Config.DriveFolderID,
		opts.PollInterval,
	)

	uploader := NewUploader(opts.DriveService, opts.Config.DriveFolderID, opts.Config.DriveID)
	dloader := NewDownloader(opts.DriveService, opts.Config.LocalPath)

	return &Engine{
		db:       opts.DB,
		config:   opts.Config,
		service:  opts.DriveService,
		watcher:  watcher,
		poller:   poller,
		uploader: uploader,
		dloader:  dloader,
	}, nil
}

// Start begins the sync loop. Blocks until context is cancelled.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine already running")
	}
	e.running = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()

	// Start initial scan to populate sync_items
	if err := e.initialScan(ctx); err != nil {
		return fmt.Errorf("initial scan: %w", err)
	}

	// Create error channel to collect errors from goroutines
	errChan := make(chan error, 3)
	var wg sync.WaitGroup

	// Start the filesystem watcher
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := e.watcher.Start(ctx); err != nil && ctx.Err() == nil {
			errChan <- fmt.Errorf("watcher: %w", err)
		}
	}()

	// Start the Drive poller
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := e.poller.Start(ctx); err != nil && ctx.Err() == nil {
			errChan <- fmt.Errorf("poller: %w", err)
		}
	}()

	// Main event loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.eventLoop(ctx)
	}()

	// Wait for context cancellation or first error
	select {
	case <-ctx.Done():
		// Normal shutdown
	case err := <-errChan:
		return err
	}

	// Stop watcher
	if err := e.watcher.Stop(); err != nil {
		// Log but don't fail on cleanup errors
		_ = e.db.AddLogEntry(e.config.ID, "error", "", map[string]any{"error": err.Error()})
	}

	wg.Wait()

	return ctx.Err()
}

// eventLoop processes events from watcher and poller.
func (e *Engine) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event := <-e.watcher.Events():
			e.handleLocalEvent(ctx, event)

		case change := <-e.poller.Events():
			e.handleRemoteChange(ctx, change)

		case err := <-e.watcher.Errors():
			_ = e.db.AddLogEntry(e.config.ID, "error", "", map[string]any{
				"source": "watcher",
				"error":  err.Error(),
			})

		case err := <-e.poller.Errors():
			_ = e.db.AddLogEntry(e.config.ID, "error", "", map[string]any{
				"source": "poller",
				"error":  err.Error(),
			})
		}
	}
}

// handleLocalEvent processes a local filesystem change.
func (e *Engine) handleLocalEvent(ctx context.Context, event WatchEvent) {
	relPath := event.RelPath

	switch event.Op {
	case OpCreate, OpWrite:
		// Check if it's a file or directory
		absPath := filepath.Join(e.config.LocalPath, relPath)
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				// File was deleted before we could process it
				return
			}

			_ = e.db.AddLogEntry(e.config.ID, "error", relPath, map[string]any{"error": err.Error()})

			return
		}

		if info.IsDir() {
			// Create folder in Drive
			if err := e.uploader.CreateFolder(ctx, relPath); err != nil {
				_ = e.db.AddLogEntry(e.config.ID, "error", relPath, map[string]any{
					"action": "create_folder",
					"error":  err.Error(),
				})

				return
			}

			_ = e.db.AddLogEntry(e.config.ID, "upload", relPath, map[string]any{"type": "folder"})
		} else {
			// Upload file
			result, err := e.uploader.UploadFile(ctx, relPath, absPath)
			if err != nil {
				_ = e.db.AddLogEntry(e.config.ID, "error", relPath, map[string]any{
					"action": "upload",
					"error":  err.Error(),
				})

				return
			}

			// Update sync item
			if err := e.updateSyncItem(relPath, result); err != nil {
				_ = e.db.AddLogEntry(e.config.ID, "error", relPath, map[string]any{
					"action": "update_sync_item",
					"error":  err.Error(),
				})
			}

			_ = e.db.AddLogEntry(e.config.ID, "upload", relPath, map[string]any{
				"drive_id": result.DriveID,
				"md5":      result.MD5,
			})
		}

	case OpDelete:
		// Delete from Drive
		if err := e.uploader.Delete(ctx, relPath); err != nil {
			_ = e.db.AddLogEntry(e.config.ID, "error", relPath, map[string]any{
				"action": "delete",
				"error":  err.Error(),
			})

			return
		}

		// Remove sync item
		if err := e.removeSyncItem(relPath); err != nil {
			_ = e.db.AddLogEntry(e.config.ID, "error", relPath, map[string]any{
				"action": "remove_sync_item",
				"error":  err.Error(),
			})
		}

		_ = e.db.AddLogEntry(e.config.ID, "delete", relPath, nil)

	case OpRename:
		// Handle rename as delete + create
		// The create event will follow separately
		if err := e.uploader.Delete(ctx, relPath); err != nil {
			// Ignore not found errors for renames
			_ = e.db.AddLogEntry(e.config.ID, "error", relPath, map[string]any{
				"action": "rename_delete",
				"error":  err.Error(),
			})
		}
	}
}

// handleRemoteChange processes a Drive change.
func (e *Engine) handleRemoteChange(ctx context.Context, change DriveChange) {
	switch change.Op {
	case DriveOpDelete:
		// Find the local path for this file
		item, err := e.db.GetSyncItemByDriveID(e.config.ID, change.FileID)
		if err != nil || item == nil {
			return // File not tracked
		}

		// Delete locally
		absPath := filepath.Join(e.config.LocalPath, item.LocalPath)
		if err := os.RemoveAll(absPath); err != nil && !os.IsNotExist(err) {
			_ = e.db.AddLogEntry(e.config.ID, "error", item.LocalPath, map[string]any{
				"action": "local_delete",
				"error":  err.Error(),
			})

			return
		}

		// Remove sync item
		if err := e.removeSyncItem(item.LocalPath); err != nil {
			_ = e.db.AddLogEntry(e.config.ID, "error", item.LocalPath, map[string]any{
				"action": "remove_sync_item",
				"error":  err.Error(),
			})
		}

		_ = e.db.AddLogEntry(e.config.ID, "download_delete", item.LocalPath, nil)

	case DriveOpCreate, DriveOpModify:
		// Check if we already have this file
		item, err := e.db.GetSyncItemByDriveID(e.config.ID, change.FileID)
		if err != nil {
			return
		}

		if item != nil && item.RemoteMD5 == change.FileID {
			// No change needed
			return
		}

		// Download the file
		result, err := e.dloader.DownloadFile(ctx, change.FileID, change.FileName)
		if err != nil {
			_ = e.db.AddLogEntry(e.config.ID, "error", change.FileName, map[string]any{
				"action":   "download",
				"error":    err.Error(),
				"drive_id": change.FileID,
			})

			return
		}

		// Update sync item
		if err := e.updateSyncItemFromDownload(result.LocalPath, change.FileID, result.MD5); err != nil {
			_ = e.db.AddLogEntry(e.config.ID, "error", result.LocalPath, map[string]any{
				"action": "update_sync_item",
				"error":  err.Error(),
			})
		}

		_ = e.db.AddLogEntry(e.config.ID, "download", result.LocalPath, map[string]any{
			"drive_id": change.FileID,
			"md5":      result.MD5,
		})
	}
}

// initialScan scans the local directory and populates sync_items.
func (e *Engine) initialScan(ctx context.Context) error {
	return filepath.WalkDir(e.config.LocalPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Skip the root
		if path == e.config.LocalPath {
			return nil
		}

		relPath, err := filepath.Rel(e.config.LocalPath, path)
		if err != nil {
			return err
		}

		// Skip ignored paths
		if e.watcher.shouldIgnore(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if d.IsDir() {
			// Track directory but don't upload yet
			return nil
		}

		// Check if already tracked
		item, err := e.db.GetSyncItem(e.config.ID, relPath)
		if err != nil {
			return err
		}

		if item != nil {
			// Already tracked
			return nil
		}

		// Compute MD5
		md5, err := computeMD5(path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// Add as pending upload
		return e.db.CreateSyncItem(e.config.ID, relPath, "", md5, "", info.ModTime(), time.Time{})
	})
}

// updateSyncItem updates or creates a sync item after upload.
func (e *Engine) updateSyncItem(relPath string, result *UploadResult) error {
	item, err := e.db.GetSyncItem(e.config.ID, relPath)
	if err != nil {
		return err
	}

	if item == nil {
		return e.db.CreateSyncItem(
			e.config.ID,
			relPath,
			result.DriveID,
			result.MD5,
			result.MD5,
			result.ModTime,
			time.Now(),
		)
	}

	return e.db.UpdateSyncItem(item.ID, result.DriveID, result.MD5, result.MD5, StateSynced)
}

// updateSyncItemFromDownload updates or creates a sync item after download.
func (e *Engine) updateSyncItemFromDownload(relPath, driveID, md5 string) error {
	item, err := e.db.GetSyncItem(e.config.ID, relPath)
	if err != nil {
		return err
	}

	if item == nil {
		return e.db.CreateSyncItem(
			e.config.ID,
			relPath,
			driveID,
			md5,
			md5,
			time.Now(),
			time.Now(),
		)
	}

	return e.db.UpdateSyncItem(item.ID, driveID, md5, md5, StateSynced)
}

// removeSyncItem removes a sync item.
func (e *Engine) removeSyncItem(relPath string) error {
	return e.db.RemoveSyncItem(e.config.ID, relPath)
}
