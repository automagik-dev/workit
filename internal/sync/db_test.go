package sync

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// openTestDB opens an in-memory SQLite database with the sync schema.
func openTestDB(t *testing.T) *DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}

	d := &DB{db: sqlDB}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() { d.Close() })

	return d
}

// insertTestConfig inserts a sync config directly and returns its ID.
func insertTestConfig(t *testing.T, d *DB) int64 {
	t.Helper()

	result, err := d.db.Exec(
		`INSERT INTO sync_configs (local_path, drive_folder_id, drive_id, created_at)
		 VALUES (?, ?, ?, ?)`,
		"/tmp/test-sync", "folder-abc", "", time.Now(),
	)
	if err != nil {
		t.Fatalf("insert config: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	return id
}

// insertTestSyncItem inserts a sync item with a given state.
func insertTestSyncItem(t *testing.T, d *DB, configID int64, localPath string, state SyncState) {
	t.Helper()

	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, localPath, "", "abc123", "", time.Now(), time.Time{}, state,
	)
	if err != nil {
		t.Fatalf("insert sync item %q: %v", localPath, err)
	}
}

func TestListPendingUploads_OnlyReturnsPendingUpload(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	// Insert items with various states
	insertTestSyncItem(t, d, configID, "file1.txt", StatePendingUpload)
	insertTestSyncItem(t, d, configID, "file2.txt", StatePendingUpload)
	insertTestSyncItem(t, d, configID, "file3.txt", StateSynced)
	insertTestSyncItem(t, d, configID, "file4.txt", StateConflict)
	insertTestSyncItem(t, d, configID, "file5.txt", StateError)
	insertTestSyncItem(t, d, configID, "file6.txt", StatePendingDownload)

	items, err := d.ListPendingUploads(configID)
	if err != nil {
		t.Fatalf("ListPendingUploads: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 pending_upload items, got %d", len(items))
	}

	for _, item := range items {
		if item.SyncState != StatePendingUpload {
			t.Errorf("expected state %q, got %q for %s", StatePendingUpload, item.SyncState, item.LocalPath)
		}

		if item.ConfigID != configID {
			t.Errorf("expected config_id %d, got %d", configID, item.ConfigID)
		}
	}

	// Verify the returned items are file1.txt and file2.txt
	paths := map[string]bool{}
	for _, item := range items {
		paths[item.LocalPath] = true
	}

	if !paths["file1.txt"] {
		t.Error("expected file1.txt in results")
	}

	if !paths["file2.txt"] {
		t.Error("expected file2.txt in results")
	}
}

func TestListPendingUploads_EmptyResult(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	// Insert only non-pending items
	insertTestSyncItem(t, d, configID, "synced.txt", StateSynced)

	items, err := d.ListPendingUploads(configID)
	if err != nil {
		t.Fatalf("ListPendingUploads: %v", err)
	}

	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestListPendingUploads_FiltersByConfigID(t *testing.T) {
	d := openTestDB(t)
	configID1 := insertTestConfig(t, d)

	// Insert a second config
	result, err := d.db.Exec(
		`INSERT INTO sync_configs (local_path, drive_folder_id, drive_id, created_at)
		 VALUES (?, ?, ?, ?)`,
		"/tmp/test-sync-2", "folder-def", "", time.Now(),
	)
	if err != nil {
		t.Fatalf("insert config 2: %v", err)
	}

	configID2, _ := result.LastInsertId()

	// Insert pending_upload items for both configs
	insertTestSyncItem(t, d, configID1, "config1-file.txt", StatePendingUpload)
	insertTestSyncItem(t, d, configID2, "config2-file.txt", StatePendingUpload)

	// Query for config 1 only
	items, err := d.ListPendingUploads(configID1)
	if err != nil {
		t.Fatalf("ListPendingUploads: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item for config %d, got %d", configID1, len(items))
	}

	if items[0].LocalPath != "config1-file.txt" {
		t.Errorf("expected config1-file.txt, got %s", items[0].LocalPath)
	}
}
