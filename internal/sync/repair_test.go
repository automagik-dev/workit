package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestRepairReport_HasIssues(t *testing.T) {
	r := &RepairReport{}
	if r.HasIssues() {
		t.Error("empty report should not have issues")
	}

	r.DuplicateFolders = []DuplicateFolder{{Name: "dup"}}
	if !r.HasIssues() {
		t.Error("report with duplicate folders should have issues")
	}

	r = &RepairReport{StaleHashes: []StaleHash{{LocalPath: "f.txt"}}}
	if !r.HasIssues() {
		t.Error("report with stale hashes should have issues")
	}

	r = &RepairReport{OrphanedItems: []OrphanedItem{{LocalPath: "f.txt"}}}
	if !r.HasIssues() {
		t.Error("report with orphaned items should have issues")
	}
}

func TestNewRepairer(t *testing.T) {
	d := openTestDB(t)
	cfg := &SyncConfig{ID: 1, LocalPath: "/tmp/test", DriveFolderID: "root"}
	r := NewRepairer(d, cfg, nil)
	if r == nil {
		t.Fatal("expected non-nil repairer")
	}
	if r.db != d {
		t.Error("db mismatch")
	}
	if r.config != cfg {
		t.Error("config mismatch")
	}
}

// mockDriveState holds state for the mock Drive API server.
type mockDriveState struct {
	// folders maps parentID -> list of child folders
	folders map[string][]map[string]string
	// files maps fileID -> true (file exists on Drive)
	files map[string]bool
	// movedFiles tracks file moves: fileID -> new parentID
	movedFiles map[string]string
	// trashedFiles tracks trashed files
	trashedFiles map[string]bool
}

func newMockDriveState() *mockDriveState {
	return &mockDriveState{
		folders:      make(map[string][]map[string]string),
		files:        make(map[string]bool),
		movedFiles:   make(map[string]string),
		trashedFiles: make(map[string]bool),
	}
}

func (m *mockDriveState) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// The Drive API client sends requests to /files, /files/<id>, etc.
		// (the endpoint URL already acts as the base, no /drive/v3/ prefix)

		// Handle Files.List: GET /files
		if r.Method == "GET" && r.URL.Path == "/files" {
			q := r.URL.Query().Get("q")
			var files []map[string]string
			for parentID, children := range m.folders {
				if strings.Contains(q, "'"+parentID+"' in parents") {
					files = append(files, children...)
				}
			}
			resp := map[string]any{"files": files}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle Files.Get: GET /files/<id>
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/files/") {
			fileID := strings.TrimPrefix(r.URL.Path, "/files/")
			if m.files[fileID] {
				json.NewEncoder(w).Encode(map[string]string{"id": fileID})
				return
			}
			// Return 404 in googleapi.Error format
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    404,
					"message": "File not found",
					"errors":  []map[string]string{{"message": "File not found", "reason": "notFound"}},
				},
			})
			return
		}

		// Handle Files.Update: PATCH /files/<id>
		if r.Method == "PATCH" && strings.HasPrefix(r.URL.Path, "/files/") {
			fileID := strings.TrimPrefix(r.URL.Path, "/files/")
			addParents := r.URL.Query().Get("addParents")
			if addParents != "" {
				m.movedFiles[fileID] = addParents
			}

			// Check if it's a trash operation by reading the body
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				if trashed, ok := body["trashed"].(bool); ok && trashed {
					m.trashedFiles[fileID] = true
				}
			}

			json.NewEncoder(w).Encode(map[string]string{"id": fileID})
			return
		}

		fmt.Fprint(w, "{}")
	}
}

func createTestDriveService(t *testing.T, handler http.HandlerFunc) *drive.Service {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	svc, err := drive.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("create drive service: %v", err)
	}

	return svc
}

func TestScan_FindsDuplicateFolders(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	mock := newMockDriveState()
	// Two folders with the same name "docs" under the root
	mock.folders["folder-abc"] = []map[string]string{
		{"id": "folder-1", "name": "docs"},
		{"id": "folder-2", "name": "docs"},
		{"id": "folder-3", "name": "images"},
	}

	svc := createTestDriveService(t, mock.handler())
	repairer := NewRepairer(d, cfg, svc)

	report, err := repairer.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(report.DuplicateFolders) != 1 {
		t.Fatalf("expected 1 duplicate folder group, got %d", len(report.DuplicateFolders))
	}

	dup := report.DuplicateFolders[0]
	if dup.Name != "docs" {
		t.Errorf("expected duplicate name 'docs', got %q", dup.Name)
	}
	if len(dup.FolderIDs) != 2 {
		t.Errorf("expected 2 folder IDs, got %d", len(dup.FolderIDs))
	}
}

func TestScan_FindsStaleHashes(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	tmpDir := t.TempDir()
	cfg.LocalPath = tmpDir

	// Create a file with known content
	testFile := filepath.Join(tmpDir, "modified.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Insert synced item with old MD5
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "modified.txt", "drive-id-1", "old-stale-md5", "remote-md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert sync item: %v", err)
	}

	// Also insert a non-synced item that should be skipped
	_, err = d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "pending.txt", "drive-id-2", "some-md5", "", time.Now(), time.Now(), StatePendingUpload,
	)
	if err != nil {
		t.Fatalf("insert pending item: %v", err)
	}

	mock := newMockDriveState()
	mock.files["drive-id-1"] = true
	mock.files["drive-id-2"] = true
	svc := createTestDriveService(t, mock.handler())

	repairer := NewRepairer(d, cfg, svc)
	report, err := repairer.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(report.StaleHashes) != 1 {
		t.Fatalf("expected 1 stale hash, got %d", len(report.StaleHashes))
	}

	stale := report.StaleHashes[0]
	if stale.LocalPath != "modified.txt" {
		t.Errorf("expected stale path 'modified.txt', got %q", stale.LocalPath)
	}
	if stale.StoredMD5 != "old-stale-md5" {
		t.Errorf("expected stored MD5 'old-stale-md5', got %q", stale.StoredMD5)
	}

	actualMD5, _ := computeMD5(testFile)
	if stale.ActualMD5 != actualMD5 {
		t.Errorf("expected actual MD5 %q, got %q", actualMD5, stale.ActualMD5)
	}
}

func TestScan_FindsOrphanedItems(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	tmpDir := t.TempDir()
	cfg.LocalPath = tmpDir

	// Create one file that exists locally
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Item with no local file and no remote file (both_missing)
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "gone.txt", "drive-gone", "md5", "md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}

	// Item with local file but no remote file (no_remote_file)
	_, err = d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "exists.txt", "drive-missing", "md5", "md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}

	// Item with no local file but remote file exists (no_local_file)
	_, err = d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "deleted-locally.txt", "drive-exists", "md5", "md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}

	mock := newMockDriveState()
	mock.files["drive-exists"] = true
	// drive-gone and drive-missing return 404
	svc := createTestDriveService(t, mock.handler())

	repairer := NewRepairer(d, cfg, svc)
	report, err := repairer.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(report.OrphanedItems) != 3 {
		t.Fatalf("expected 3 orphaned items, got %d", len(report.OrphanedItems))
	}

	// Check reasons
	reasons := make(map[string]string)
	for _, o := range report.OrphanedItems {
		reasons[o.LocalPath] = o.Reason
	}

	if reasons["gone.txt"] != "both_missing" {
		t.Errorf("gone.txt: expected reason 'both_missing', got %q", reasons["gone.txt"])
	}
	if reasons["exists.txt"] != "no_remote_file" {
		t.Errorf("exists.txt: expected reason 'no_remote_file', got %q", reasons["exists.txt"])
	}
	if reasons["deleted-locally.txt"] != "no_local_file" {
		t.Errorf("deleted-locally.txt: expected reason 'no_local_file', got %q", reasons["deleted-locally.txt"])
	}
}

func TestScan_NoIssues(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	tmpDir := t.TempDir()
	cfg.LocalPath = tmpDir

	// Create a file that exists locally
	testFile := filepath.Join(tmpDir, "good.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	md5Hash, _ := computeMD5(testFile)

	// Insert synced item with matching MD5
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "good.txt", "drive-good", md5Hash, md5Hash, time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}

	mock := newMockDriveState()
	mock.files["drive-good"] = true
	svc := createTestDriveService(t, mock.handler())

	repairer := NewRepairer(d, cfg, svc)
	report, err := repairer.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if report.HasIssues() {
		t.Errorf("expected no issues, got dupes=%d stale=%d orphans=%d",
			len(report.DuplicateFolders), len(report.StaleHashes), len(report.OrphanedItems))
	}
}

func TestApply_FixesStaleHashes(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	tmpDir := t.TempDir()
	cfg.LocalPath = tmpDir

	testFile := filepath.Join(tmpDir, "stale.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	actualMD5, _ := computeMD5(testFile)

	// Insert synced item with old MD5
	result, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "stale.txt", "drive-id-1", "old-md5", "remote-md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	itemID, _ := result.LastInsertId()

	mock := newMockDriveState()
	svc := createTestDriveService(t, mock.handler())

	repairer := NewRepairer(d, cfg, svc)

	report := &RepairReport{
		StaleHashes: []StaleHash{
			{
				LocalPath: "stale.txt",
				StoredMD5: "old-md5",
				ActualMD5: actualMD5,
				ItemID:    itemID,
				DriveID:   "drive-id-1",
			},
		},
	}

	if err := repairer.Apply(context.Background(), report); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Verify the item was updated
	item, err := d.GetSyncItem(configID, "stale.txt")
	if err != nil {
		t.Fatalf("GetSyncItem: %v", err)
	}
	if item.LocalMD5 != actualMD5 {
		t.Errorf("expected local_md5 %q, got %q", actualMD5, item.LocalMD5)
	}
	if item.SyncState != StatePendingUpload {
		t.Errorf("expected state %q, got %q", StatePendingUpload, item.SyncState)
	}
}

func TestApply_RemovesOrphanedItems(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	// Insert an orphaned item
	result, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "orphan.txt", "drive-orphan", "md5", "md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	itemID, _ := result.LastInsertId()

	mock := newMockDriveState()
	svc := createTestDriveService(t, mock.handler())

	repairer := NewRepairer(d, cfg, svc)

	report := &RepairReport{
		OrphanedItems: []OrphanedItem{
			{
				LocalPath: "orphan.txt",
				DriveID:   "drive-orphan",
				Reason:    "both_missing",
				ItemID:    itemID,
				ConfigID:  configID,
			},
		},
	}

	if err := repairer.Apply(context.Background(), report); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Verify the item was deleted
	item, err := d.GetSyncItem(configID, "orphan.txt")
	if err != nil {
		t.Fatalf("GetSyncItem: %v", err)
	}
	if item != nil {
		t.Error("expected orphaned item to be deleted")
	}
}

func TestApply_MergesDuplicateFolders(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	mock := newMockDriveState()
	// The duplicate folder has one child (not a folder, so listed with no mimeType filter)
	mock.folders["folder-dup"] = []map[string]string{
		{"id": "child-1", "name": "file-in-dup.txt"},
	}
	// Empty keep folder
	mock.folders["folder-keep"] = []map[string]string{}
	// Root has no subfolders after merge (for rebuild)
	mock.folders["folder-abc"] = []map[string]string{}

	svc := createTestDriveService(t, mock.handler())

	repairer := NewRepairer(d, cfg, svc)

	report := &RepairReport{
		DuplicateFolders: []DuplicateFolder{
			{
				Name:      "docs",
				ParentID:  "folder-abc",
				FolderIDs: []string{"folder-keep", "folder-dup"},
			},
		},
	}

	if err := repairer.Apply(context.Background(), report); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Verify child was moved
	if newParent, ok := mock.movedFiles["child-1"]; !ok {
		t.Error("expected child-1 to be moved")
	} else if newParent != "folder-keep" {
		t.Errorf("expected child moved to folder-keep, got %s", newParent)
	}

	// Verify duplicate was trashed
	if !mock.trashedFiles["folder-dup"] {
		t.Error("expected folder-dup to be trashed")
	}
}

func TestScan_DryRunMakesNoChanges(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	tmpDir := t.TempDir()
	cfg.LocalPath = tmpDir

	// Create a stale file
	testFile := filepath.Join(tmpDir, "stale.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "stale.txt", "drive-id-1", "old-md5", "remote-md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}

	mock := newMockDriveState()
	mock.files["drive-id-1"] = true
	svc := createTestDriveService(t, mock.handler())

	repairer := NewRepairer(d, cfg, svc)

	// Scan only (dry run)
	report, err := repairer.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(report.StaleHashes) != 1 {
		t.Fatalf("expected 1 stale hash, got %d", len(report.StaleHashes))
	}

	// Verify the DB was NOT modified
	item, err := d.GetSyncItem(configID, "stale.txt")
	if err != nil {
		t.Fatalf("GetSyncItem: %v", err)
	}
	if item.LocalMD5 != "old-md5" {
		t.Errorf("scan should not modify DB; expected local_md5 'old-md5', got %q", item.LocalMD5)
	}
	if item.SyncState != StateSynced {
		t.Errorf("scan should not modify DB; expected state %q, got %q", StateSynced, item.SyncState)
	}
}

func TestListAllItems(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	insertTestSyncItem(t, d, configID, "a.txt", StateSynced)
	insertTestSyncItem(t, d, configID, "b.txt", StatePendingUpload)
	insertTestSyncItem(t, d, configID, "c.txt", StateError)

	items, err := d.ListAllItems(configID)
	if err != nil {
		t.Fatalf("ListAllItems: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestListSyncedItems(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	insertTestSyncItem(t, d, configID, "a.txt", StateSynced)
	insertTestSyncItem(t, d, configID, "b.txt", StatePendingUpload)
	insertTestSyncItem(t, d, configID, "c.txt", StateSynced)

	items, err := d.ListSyncedItems(configID)
	if err != nil {
		t.Fatalf("ListSyncedItems: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 synced items, got %d", len(items))
	}

	for _, item := range items {
		if item.SyncState != StateSynced {
			t.Errorf("expected state %q, got %q", StateSynced, item.SyncState)
		}
	}
}

func TestDeleteSyncFolders(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	// Create some folders
	if err := d.CreateSyncFolder(configID, "folder-1", "docs"); err != nil {
		t.Fatalf("CreateSyncFolder: %v", err)
	}
	if err := d.CreateSyncFolder(configID, "folder-2", "images"); err != nil {
		t.Fatalf("CreateSyncFolder: %v", err)
	}

	// Verify they exist
	ids, err := d.ListSyncFolderDriveIDs(configID)
	if err != nil {
		t.Fatalf("ListSyncFolderDriveIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(ids))
	}

	// Delete all
	if err := d.DeleteSyncFolders(configID); err != nil {
		t.Fatalf("DeleteSyncFolders: %v", err)
	}

	// Verify they're gone
	ids, err = d.ListSyncFolderDriveIDs(configID)
	if err != nil {
		t.Fatalf("ListSyncFolderDriveIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected 0 folders after delete, got %d", len(ids))
	}
}

func TestRemoveSyncItemByID(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	// Insert an item
	result, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "test.txt", "drive-1", "md5", "md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	itemID, _ := result.LastInsertId()

	// Remove by ID
	if err := d.RemoveSyncItemByID(itemID); err != nil {
		t.Fatalf("RemoveSyncItemByID: %v", err)
	}

	// Verify deleted
	item, err := d.GetSyncItem(configID, "test.txt")
	if err != nil {
		t.Fatalf("GetSyncItem: %v", err)
	}
	if item != nil {
		t.Error("expected item to be deleted")
	}
}

func TestScan_OrphanedItem_NoRemoteDueToEmptyDriveID(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	tmpDir := t.TempDir()
	cfg.LocalPath = tmpDir

	// Item with no local file and empty drive ID
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "never-uploaded.txt", "", "md5", "", time.Now(), time.Now(), StatePendingUpload,
	)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}

	mock := newMockDriveState()
	svc := createTestDriveService(t, mock.handler())

	repairer := NewRepairer(d, cfg, svc)
	report, err := repairer.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(report.OrphanedItems) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(report.OrphanedItems))
	}

	if report.OrphanedItems[0].Reason != "both_missing" {
		t.Errorf("expected reason 'both_missing', got %q", report.OrphanedItems[0].Reason)
	}
}

func TestApply_RebuildsFolderRegistryAfterMerge(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	cfg, _ := d.GetConfigByID(configID)

	// Pre-populate some folders
	if err := d.CreateSyncFolder(configID, "old-folder-1", "old-path"); err != nil {
		t.Fatalf("CreateSyncFolder: %v", err)
	}

	mock := newMockDriveState()
	// After merge, root has one subfolder
	mock.folders["folder-abc"] = []map[string]string{
		{"id": "new-folder-1", "name": "new-docs"},
	}
	mock.folders["new-folder-1"] = []map[string]string{}
	// Duplicate folder (keep+dup) - dup is empty
	mock.folders["folder-keep"] = []map[string]string{}
	mock.folders["folder-dup"] = []map[string]string{}

	svc := createTestDriveService(t, mock.handler())
	repairer := NewRepairer(d, cfg, svc)

	report := &RepairReport{
		DuplicateFolders: []DuplicateFolder{
			{
				Name:      "docs",
				ParentID:  "folder-abc",
				FolderIDs: []string{"folder-keep", "folder-dup"},
			},
		},
	}

	if err := repairer.Apply(context.Background(), report); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Old folder should be gone, new one should be present
	oldFolder, _ := d.GetSyncFolderByDriveID(configID, "old-folder-1")
	if oldFolder != nil {
		t.Error("expected old folder to be deleted during rebuild")
	}

	newFolder, _ := d.GetSyncFolderByDriveID(configID, "new-folder-1")
	if newFolder == nil {
		t.Error("expected new folder to be registered during rebuild")
	}
	if newFolder != nil && newFolder.LocalPath != "new-docs" {
		t.Errorf("expected path 'new-docs', got %q", newFolder.LocalPath)
	}
}
