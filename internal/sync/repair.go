package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

// RepairReport contains the results of a repair scan.
type RepairReport struct {
	DuplicateFolders []DuplicateFolder `json:"duplicate_folders"`
	StaleHashes      []StaleHash       `json:"stale_hashes"`
	OrphanedItems    []OrphanedItem    `json:"orphaned_items"`
}

// DuplicateFolder represents folders with the same name under the same parent.
type DuplicateFolder struct {
	Name      string   `json:"name"`
	ParentID  string   `json:"parent_id"`
	FolderIDs []string `json:"folder_ids"` // All Drive IDs; first is the "keep" candidate (oldest)
}

// StaleHash represents a tracked file whose local MD5 doesn't match the DB.
type StaleHash struct {
	LocalPath string `json:"local_path"`
	StoredMD5 string `json:"stored_md5"`
	ActualMD5 string `json:"actual_md5"`
	ItemID    int64  `json:"item_id"`
	DriveID   string `json:"drive_id"`
}

// OrphanedItem represents a sync_item with no corresponding local or remote file.
type OrphanedItem struct {
	LocalPath string `json:"local_path"`
	DriveID   string `json:"drive_id"`
	Reason    string `json:"reason"` // "no_local_file", "no_remote_file", "both_missing"
	ItemID    int64  `json:"item_id"`
	ConfigID  int64  `json:"config_id"`
}

// HasIssues returns true if the report contains any issues.
func (r *RepairReport) HasIssues() bool {
	return len(r.DuplicateFolders) > 0 || len(r.StaleHashes) > 0 || len(r.OrphanedItems) > 0
}

// Repairer handles sync state repair operations.
type Repairer struct {
	db      *DB
	config  *SyncConfig
	service *drive.Service
}

// NewRepairer creates a new Repairer.
func NewRepairer(db *DB, config *SyncConfig, service *drive.Service) *Repairer {
	return &Repairer{db: db, config: config, service: service}
}

// Scan performs a dry-run scan and builds the repair report.
func (r *Repairer) Scan(ctx context.Context) (*RepairReport, error) {
	report := &RepairReport{}

	// 1. Find duplicate folders
	dupes, err := r.findDuplicateFolders(ctx, r.config.DriveFolderID)
	if err != nil {
		return nil, fmt.Errorf("scan duplicate folders: %w", err)
	}
	report.DuplicateFolders = dupes

	// 2. Find stale hashes
	stale, err := r.findStaleHashes(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan stale hashes: %w", err)
	}
	report.StaleHashes = stale

	// 3. Find orphaned items
	orphans, err := r.findOrphanedItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan orphaned items: %w", err)
	}
	report.OrphanedItems = orphans

	return report, nil
}

// Apply executes the fixes described in the report.
func (r *Repairer) Apply(ctx context.Context, report *RepairReport) error {
	// 1. Merge duplicate folders
	for _, dup := range report.DuplicateFolders {
		if err := r.mergeDuplicateFolder(ctx, dup); err != nil {
			return fmt.Errorf("merge duplicate folder %q: %w", dup.Name, err)
		}
	}

	// 2. Fix stale hashes
	for _, stale := range report.StaleHashes {
		if err := r.db.UpdateSyncItem(stale.ItemID, stale.DriveID, stale.ActualMD5, "", StatePendingUpload); err != nil {
			return fmt.Errorf("fix stale hash for %s: %w", stale.LocalPath, err)
		}
	}

	// 3. Remove orphaned items
	for _, orphan := range report.OrphanedItems {
		if err := r.db.RemoveSyncItemByID(orphan.ItemID); err != nil {
			return fmt.Errorf("remove orphan %s: %w", orphan.LocalPath, err)
		}
	}

	// 4. Rebuild folder registry after merging duplicates
	if len(report.DuplicateFolders) > 0 {
		if err := r.rebuildFolderRegistry(ctx); err != nil {
			return fmt.Errorf("rebuild folder registry: %w", err)
		}
	}

	return nil
}

// findDuplicateFolders walks the Drive folder tree and finds duplicate folders.
func (r *Repairer) findDuplicateFolders(ctx context.Context, parentID string) ([]DuplicateFolder, error) {
	var dupes []DuplicateFolder

	// List child folders of parentID
	children, err := r.listChildFolders(ctx, parentID)
	if err != nil {
		return nil, err
	}

	// Group by name
	byName := make(map[string][]string)
	for _, f := range children {
		byName[f.Name] = append(byName[f.Name], f.Id)
	}

	// Find duplicates
	for name, ids := range byName {
		if len(ids) > 1 {
			dupes = append(dupes, DuplicateFolder{
				Name:      name,
				ParentID:  parentID,
				FolderIDs: ids,
			})
		}
	}

	// Recurse into child folders (use unique names only, or first of each name)
	visited := make(map[string]bool)
	for _, f := range children {
		if visited[f.Id] {
			continue
		}
		visited[f.Id] = true

		subDupes, err := r.findDuplicateFolders(ctx, f.Id)
		if err != nil {
			// Log but continue with other folders
			continue
		}
		dupes = append(dupes, subDupes...)
	}

	return dupes, nil
}

// listChildFolders lists all child folders of a given parent, handling pagination.
func (r *Repairer) listChildFolders(ctx context.Context, parentID string) ([]*drive.File, error) {
	query := fmt.Sprintf("'%s' in parents and mimeType = 'application/vnd.google-apps.folder' and trashed = false", parentID)

	var allFiles []*drive.File
	var pageToken string

	for {
		call := r.service.Files.List().
			Context(ctx).
			Q(query).
			Fields("nextPageToken,files(id,name)").
			PageSize(100).
			SupportsAllDrives(true)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list folders in %s: %w", parentID, err)
		}

		allFiles = append(allFiles, resp.Files...)

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return allFiles, nil
}

// findStaleHashes compares local file MD5s against the DB for synced items.
func (r *Repairer) findStaleHashes(_ context.Context) ([]StaleHash, error) {
	items, err := r.db.ListSyncedItems(r.config.ID)
	if err != nil {
		return nil, err
	}

	var stale []StaleHash
	for _, item := range items {
		absPath := filepath.Join(r.config.LocalPath, item.LocalPath)

		actualMD5, err := computeMD5(absPath)
		if err != nil {
			// File may not exist; skip (orphan detection handles that)
			continue
		}

		if actualMD5 != item.LocalMD5 {
			stale = append(stale, StaleHash{
				LocalPath: item.LocalPath,
				StoredMD5: item.LocalMD5,
				ActualMD5: actualMD5,
				ItemID:    item.ID,
				DriveID:   item.DriveID,
			})
		}
	}

	return stale, nil
}

// findOrphanedItems checks all sync items for missing local or remote files.
func (r *Repairer) findOrphanedItems(ctx context.Context) ([]OrphanedItem, error) {
	items, err := r.db.ListAllItems(r.config.ID)
	if err != nil {
		return nil, err
	}

	var orphans []OrphanedItem
	for _, item := range items {
		absPath := filepath.Join(r.config.LocalPath, item.LocalPath)

		// Check local file
		_, localErr := os.Stat(absPath)
		localExists := localErr == nil

		// Check remote file
		remoteExists := true
		if item.DriveID != "" {
			_, remoteErr := r.service.Files.Get(item.DriveID).
				Context(ctx).
				Fields("id").
				SupportsAllDrives(true).
				Do()
			if remoteErr != nil {
				var gErr *googleapi.Error
				if errors.As(remoteErr, &gErr) && gErr.Code == 404 {
					remoteExists = false
				}
				// For other errors, assume the file exists (don't orphan on transient failures)
			}
		} else {
			// No drive ID means it was never uploaded
			remoteExists = false
		}

		if !localExists && !remoteExists {
			orphans = append(orphans, OrphanedItem{
				LocalPath: item.LocalPath,
				DriveID:   item.DriveID,
				Reason:    "both_missing",
				ItemID:    item.ID,
				ConfigID:  item.ConfigID,
			})
		} else if !localExists {
			orphans = append(orphans, OrphanedItem{
				LocalPath: item.LocalPath,
				DriveID:   item.DriveID,
				Reason:    "no_local_file",
				ItemID:    item.ID,
				ConfigID:  item.ConfigID,
			})
		} else if !remoteExists {
			orphans = append(orphans, OrphanedItem{
				LocalPath: item.LocalPath,
				DriveID:   item.DriveID,
				Reason:    "no_remote_file",
				ItemID:    item.ID,
				ConfigID:  item.ConfigID,
			})
		}
	}

	return orphans, nil
}

// mergeDuplicateFolder merges duplicate folders by moving children to the first (keep) folder.
func (r *Repairer) mergeDuplicateFolder(ctx context.Context, dup DuplicateFolder) error {
	if len(dup.FolderIDs) < 2 {
		return nil
	}

	keepID := dup.FolderIDs[0]

	for _, dupID := range dup.FolderIDs[1:] {
		// List all children of the duplicate folder
		children, err := r.listAllChildren(ctx, dupID)
		if err != nil {
			return fmt.Errorf("list children of %s: %w", dupID, err)
		}

		// Move each child to the keep folder
		for _, child := range children {
			_, err := r.service.Files.Update(child.Id, nil).
				AddParents(keepID).
				RemoveParents(dupID).
				SupportsAllDrives(true).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("move file %s: %w", child.Id, err)
			}
		}

		// Trash the now-empty duplicate folder
		_, err = r.service.Files.Update(dupID, &drive.File{Trashed: true}).
			SupportsAllDrives(true).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("trash duplicate folder %s: %w", dupID, err)
		}
	}

	return nil
}

// listAllChildren lists all children (files and folders) of a parent folder.
func (r *Repairer) listAllChildren(ctx context.Context, parentID string) ([]*drive.File, error) {
	query := fmt.Sprintf("'%s' in parents and trashed = false", parentID)

	var allFiles []*drive.File
	var pageToken string

	for {
		call := r.service.Files.List().
			Context(ctx).
			Q(query).
			Fields("nextPageToken,files(id,name)").
			PageSize(100).
			SupportsAllDrives(true)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list children of %s: %w", parentID, err)
		}

		allFiles = append(allFiles, resp.Files...)

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return allFiles, nil
}

// rebuildFolderRegistry clears and rebuilds the sync_folders table.
func (r *Repairer) rebuildFolderRegistry(ctx context.Context) error {
	if err := r.db.DeleteSyncFolders(r.config.ID); err != nil {
		return fmt.Errorf("delete sync folders: %w", err)
	}

	return r.walkAndRegisterFolders(ctx, r.config.DriveFolderID, "")
}

// walkAndRegisterFolders recursively walks Drive folders and registers them.
func (r *Repairer) walkAndRegisterFolders(ctx context.Context, parentID, parentPath string) error {
	children, err := r.listChildFolders(ctx, parentID)
	if err != nil {
		return err
	}

	for _, f := range children {
		folderPath := f.Name
		if parentPath != "" {
			folderPath = filepath.Join(parentPath, f.Name)
		}

		if err := r.db.CreateSyncFolder(r.config.ID, f.Id, folderPath); err != nil {
			// Log but continue
			continue
		}

		// Recurse
		if err := r.walkAndRegisterFolders(ctx, f.Id, folderPath); err != nil {
			// Log but continue
			continue
		}
	}

	return nil
}
