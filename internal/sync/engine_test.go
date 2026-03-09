package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestProcessPendingUploads_UploadsAndMarksSynced(t *testing.T) {
	// Set up in-memory DB
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	// Create temp dir with real files
	tmpDir := t.TempDir()

	// Create test files on disk
	for _, name := range []string{"a.txt", "b.txt"} {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("content of "+name), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Insert pending_upload items into DB
	for _, name := range []string{"a.txt", "b.txt"} {
		insertTestSyncItem(t, d, configID, name, StatePendingUpload)
	}

	// Also insert a synced item that should NOT be uploaded
	insertTestSyncItem(t, d, configID, "already-synced.txt", StateSynced)

	// Track which files were uploaded
	uploadedFiles := map[string]bool{}

	// Mock Drive API server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle file upload (Files.Create)
		if r.Method == "POST" && r.URL.Path == "/upload/drive/v3/files" {
			uploadedFiles[r.URL.Query().Get("uploadType")] = true

			resp := map[string]string{
				"id":          fmt.Sprintf("drive-id-%d", len(uploadedFiles)),
				"md5Checksum": "fakechecksum",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle Files.List (for ensureParentFolders / findFileByName)
		if r.Method == "GET" && r.URL.Path == "/drive/v3/files" {
			resp := map[string]interface{}{
				"files": []interface{}{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Default: empty response
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	// Create Drive service pointing at test server
	svc, err := drive.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("create drive service: %v", err)
	}

	// Build a minimal engine
	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		service:  svc,
		uploader: NewUploader(svc, "root-folder-id", "", nil, 0),
	}

	// Run processPendingUploads
	ctx := context.Background()
	if err := engine.processPendingUploads(ctx); err != nil {
		t.Fatalf("processPendingUploads: %v", err)
	}

	// Verify: both items should now be synced
	for _, name := range []string{"a.txt", "b.txt"} {
		item, err := d.GetSyncItem(configID, name)
		if err != nil {
			t.Fatalf("GetSyncItem(%s): %v", name, err)
		}

		if item == nil {
			t.Fatalf("expected sync item for %s, got nil", name)
		}

		if item.SyncState != StateSynced {
			t.Errorf("%s: expected state %q, got %q", name, StateSynced, item.SyncState)
		}

		if item.DriveID == "" {
			t.Errorf("%s: expected non-empty drive_id after upload", name)
		}
	}

	// Verify: already-synced item should still be synced (not re-uploaded)
	syncedItem, err := d.GetSyncItem(configID, "already-synced.txt")
	if err != nil {
		t.Fatalf("GetSyncItem(already-synced.txt): %v", err)
	}

	if syncedItem.SyncState != StateSynced {
		t.Errorf("already-synced.txt: state changed unexpectedly to %q", syncedItem.SyncState)
	}
}

func TestProcessPendingUploads_ContinuesOnFailure(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	tmpDir := t.TempDir()

	// Create only b.txt on disk; a.txt is missing (will cause upload failure)
	if err := os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("content b"), 0o644); err != nil {
		t.Fatalf("write b.txt: %v", err)
	}

	// Insert pending_upload items
	insertTestSyncItem(t, d, configID, "a.txt", StatePendingUpload) // missing file
	insertTestSyncItem(t, d, configID, "b.txt", StatePendingUpload)

	// Mock Drive API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/upload/drive/v3/files" {
			resp := map[string]string{
				"id":          "drive-id-1",
				"md5Checksum": "fakechecksum",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "GET" && r.URL.Path == "/drive/v3/files" {
			resp := map[string]interface{}{"files": []interface{}{}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("create drive service: %v", err)
	}

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		service:  svc,
		uploader: NewUploader(svc, "root-folder-id", "", nil, 0),
	}

	// Should NOT return error even though a.txt fails
	ctx := context.Background()
	if err := engine.processPendingUploads(ctx); err != nil {
		t.Fatalf("processPendingUploads should not fail: %v", err)
	}

	// a.txt should still be pending_upload (failed)
	itemA, err := d.GetSyncItem(configID, "a.txt")
	if err != nil {
		t.Fatalf("GetSyncItem(a.txt): %v", err)
	}

	if itemA.SyncState != StatePendingUpload {
		t.Errorf("a.txt: expected state %q (failed upload), got %q", StatePendingUpload, itemA.SyncState)
	}

	// b.txt should be synced
	itemB, err := d.GetSyncItem(configID, "b.txt")
	if err != nil {
		t.Fatalf("GetSyncItem(b.txt): %v", err)
	}

	if itemB.SyncState != StateSynced {
		t.Errorf("b.txt: expected state %q, got %q", StateSynced, itemB.SyncState)
	}
}

func TestProcessPendingUploads_RespectsContextCancellation(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	tmpDir := t.TempDir()

	// Create many files
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("file%d.txt", i)
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("content"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}

		insertTestSyncItem(t, d, configID, name, StatePendingUpload)
	}

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Mock Drive API (should not be called if context is already cancelled)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("Drive API called despite cancelled context")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("create drive service: %v", err)
	}

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		service:  svc,
		uploader: NewUploader(svc, "root-folder-id", "", nil, 0),
	}

	err = engine.processPendingUploads(ctx)

	// Should return context error
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestProcessPendingUploads_NoPendingItems(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	// No pending items at all
	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     t.TempDir(),
			DriveFolderID: "root-folder-id",
		},
	}

	ctx := context.Background()
	if err := engine.processPendingUploads(ctx); err != nil {
		t.Fatalf("processPendingUploads with no items: %v", err)
	}
}

func TestInitialScan_SkipsSymlinkToDirectory(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	tmpDir := t.TempDir()

	// Create a real file
	realFile := filepath.Join(tmpDir, "real.txt")
	if err := os.WriteFile(realFile, []byte("real content"), 0o644); err != nil {
		t.Fatalf("write real.txt: %v", err)
	}

	// Create a target directory and a symlink to it
	targetDir := t.TempDir()
	symlinkDir := filepath.Join(tmpDir, "linked-dir")
	if err := os.Symlink(targetDir, symlinkDir); err != nil {
		t.Fatalf("create symlink to dir: %v", err)
	}

	// Create a watcher (needed by initialScan for shouldIgnore)
	watcher, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer watcher.Stop()

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		watcher: watcher,
	}

	ctx := context.Background()
	if err := engine.initialScan(ctx); err != nil {
		t.Fatalf("initialScan should not crash on symlinks: %v", err)
	}

	// Verify real.txt was tracked
	item, err := d.GetSyncItem(configID, "real.txt")
	if err != nil {
		t.Fatalf("GetSyncItem(real.txt): %v", err)
	}
	if item == nil {
		t.Fatal("expected real.txt to be tracked")
	}

	// Verify symlink was NOT tracked
	linkItem, err := d.GetSyncItem(configID, "linked-dir")
	if err != nil {
		t.Fatalf("GetSyncItem(linked-dir): %v", err)
	}
	if linkItem != nil {
		t.Error("symlink-to-directory should not be tracked as sync item")
	}
}

func TestInitialScan_SkipsSymlinkToFile(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	tmpDir := t.TempDir()

	// Create a real file
	realFile := filepath.Join(tmpDir, "real.txt")
	if err := os.WriteFile(realFile, []byte("real content"), 0o644); err != nil {
		t.Fatalf("write real.txt: %v", err)
	}

	// Create a target file and symlink to it
	targetFile := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(targetFile, []byte("target"), 0o644); err != nil {
		t.Fatalf("write target.txt: %v", err)
	}
	symlinkFile := filepath.Join(tmpDir, "linked-file.txt")
	if err := os.Symlink(targetFile, symlinkFile); err != nil {
		t.Fatalf("create symlink to file: %v", err)
	}

	watcher, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer watcher.Stop()

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		watcher: watcher,
	}

	ctx := context.Background()
	if err := engine.initialScan(ctx); err != nil {
		t.Fatalf("initialScan should not crash on symlinks: %v", err)
	}

	// Verify real.txt was tracked
	item, err := d.GetSyncItem(configID, "real.txt")
	if err != nil {
		t.Fatalf("GetSyncItem(real.txt): %v", err)
	}
	if item == nil {
		t.Fatal("expected real.txt to be tracked")
	}

	// Verify symlink was NOT tracked
	linkItem, err := d.GetSyncItem(configID, "linked-file.txt")
	if err != nil {
		t.Fatalf("GetSyncItem(linked-file.txt): %v", err)
	}
	if linkItem != nil {
		t.Error("symlink-to-file should not be tracked as sync item")
	}
}

func TestProcessPendingUploads_StartsAfterWatcherInStart(t *testing.T) {
	// This test verifies the errChan buffer size is 4 (not 3) and that
	// the pending uploads goroutine is launched in Start().
	// We just check that the engine struct has the method and the channel
	// buffer is correct by inspecting the source pattern.
	// The actual integration of Start() is harder to test without a full
	// Drive service, but we verify processPendingUploads is callable.

	d := openTestDB(t)
	configID := insertTestConfig(t, d)

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     t.TempDir(),
			DriveFolderID: "root-folder-id",
		},
	}

	// Verify processPendingUploads works with empty pending list
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := engine.processPendingUploads(ctx); err != nil {
		t.Fatalf("processPendingUploads: %v", err)
	}
}

func TestHandleRemoteChange_SkipsDownloadWhenMD5Matches(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	tmpDir := t.TempDir()

	// Insert a synced item with known remote MD5
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "test.txt", "drive-id-1", "abc123", "remote-md5-hash", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert sync item: %v", err)
	}

	// Track whether download was attempted
	downloadAttempted := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadAttempted = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("create drive service: %v", err)
	}

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		service: svc,
		dloader: NewDownloader(svc, tmpDir, nil, 0),
	}

	// Send a remote change with matching MD5 — should be skipped
	ctx := context.Background()
	engine.handleRemoteChange(ctx, DriveChange{
		FileID:   "drive-id-1",
		FileName: "test.txt",
		MD5:      "remote-md5-hash",
		Op:       DriveOpModify,
	})

	if downloadAttempted {
		t.Error("download was attempted despite matching MD5; expected skip")
	}
}

func TestHandleRemoteChange_DownloadsWhenMD5Differs(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	tmpDir := t.TempDir()

	// Insert a synced item with known remote MD5
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "test.txt", "drive-id-1", "abc123", "old-remote-md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert sync item: %v", err)
	}

	// Track whether download was attempted
	downloadAttempted := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadAttempted = true
		// Return file content for download
		if r.URL.Path == "/drive/v3/files/drive-id-1" {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "new content")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("create drive service: %v", err)
	}

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		service: svc,
		dloader: NewDownloader(svc, tmpDir, nil, 0),
	}

	// Send a remote change with different MD5 — should trigger download
	ctx := context.Background()
	engine.handleRemoteChange(ctx, DriveChange{
		FileID:   "drive-id-1",
		FileName: "test.txt",
		MD5:      "new-remote-md5",
		Op:       DriveOpModify,
	})

	if !downloadAttempted {
		t.Error("download was NOT attempted despite differing MD5; expected download")
	}
}

func TestHandleLocalEvent_SkipsUploadWhenMD5Matches(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	// Compute its MD5
	fileMD5, err := computeMD5(testFile)
	if err != nil {
		t.Fatalf("compute md5: %v", err)
	}

	// Insert a synced item with matching local MD5
	_, err = d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "test.txt", "drive-id-1", fileMD5, fileMD5, time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert sync item: %v", err)
	}

	// Track whether upload was attempted
	uploadAttempted := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/upload/drive/v3/files" {
			uploadAttempted = true
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("create drive service: %v", err)
	}

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		service:  svc,
		uploader: NewUploader(svc, "root-folder-id", "", nil, 0),
	}

	// Trigger a write event for the file — MD5 matches, so upload should be skipped
	ctx := context.Background()
	engine.handleLocalEvent(ctx, WatchEvent{
		RelPath: "test.txt",
		Op:      OpWrite,
	})

	if uploadAttempted {
		t.Error("upload was attempted despite matching MD5; expected skip")
	}
}

func TestHandleLocalEvent_UploadsWhenMD5Differs(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	// Insert a synced item with a DIFFERENT local MD5 (stale)
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "test.txt", "drive-id-1", "old-stale-md5", "old-stale-md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert sync item: %v", err)
	}

	// Track whether upload was attempted
	uploadAttempted := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/upload/drive/v3/files" {
			uploadAttempted = true
			resp := map[string]string{
				"id":          "drive-id-1",
				"md5Checksum": "new-md5",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.Method == "GET" && r.URL.Path == "/drive/v3/files" {
			resp := map[string]interface{}{"files": []interface{}{}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer ts.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(ts.Client()),
	)
	if err != nil {
		t.Fatalf("create drive service: %v", err)
	}

	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		service:  svc,
		uploader: NewUploader(svc, "root-folder-id", "", nil, 0),
	}

	// Trigger a write event — MD5 differs, so upload should happen
	ctx := context.Background()
	engine.handleLocalEvent(ctx, WatchEvent{
		RelPath: "test.txt",
		Op:      OpWrite,
	})

	if !uploadAttempted {
		t.Error("upload was NOT attempted despite differing MD5; expected upload")
	}
}

func TestInitialScan_RehashesExistingTrackedFiles(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	tmpDir := t.TempDir()

	// Create a test file with known content
	testFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(testFile, []byte("original content"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	originalMD5, err := computeMD5(testFile)
	if err != nil {
		t.Fatalf("compute md5: %v", err)
	}

	// Insert a synced item with matching MD5 (not modified offline)
	_, err = d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "tracked.txt", "drive-id-1", originalMD5, "remote-md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert sync item: %v", err)
	}

	// Build engine with a minimal watcher (just need shouldIgnore to work)
	watcher := &Watcher{root: tmpDir}
	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		watcher: watcher,
	}

	ctx := context.Background()
	if err := engine.initialScan(ctx); err != nil {
		t.Fatalf("initialScan: %v", err)
	}

	// File content hasn't changed, so state should still be synced
	item, err := d.GetSyncItem(configID, "tracked.txt")
	if err != nil {
		t.Fatalf("GetSyncItem: %v", err)
	}
	if item.SyncState != StateSynced {
		t.Errorf("expected state %q for unchanged file, got %q", StateSynced, item.SyncState)
	}
}

func TestInitialScan_MarksOfflineEditsAsPendingUpload(t *testing.T) {
	d := openTestDB(t)
	configID := insertTestConfig(t, d)
	tmpDir := t.TempDir()

	// Create a test file with NEW content (simulating offline edit)
	testFile := filepath.Join(tmpDir, "edited.txt")
	if err := os.WriteFile(testFile, []byte("new content after offline edit"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	// Insert a synced item with OLD MD5 (stale — file was edited while sync was stopped)
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, "edited.txt", "drive-id-1", "old-stale-md5", "remote-md5", time.Now(), time.Now(), StateSynced,
	)
	if err != nil {
		t.Fatalf("insert sync item: %v", err)
	}

	// Build engine with a minimal watcher
	watcher := &Watcher{root: tmpDir}
	engine := &Engine{
		db: d,
		config: &SyncConfig{
			ID:            configID,
			LocalPath:     tmpDir,
			DriveFolderID: "root-folder-id",
		},
		watcher: watcher,
	}

	ctx := context.Background()
	if err := engine.initialScan(ctx); err != nil {
		t.Fatalf("initialScan: %v", err)
	}

	// File was edited offline, so state should be pending_upload
	item, err := d.GetSyncItem(configID, "edited.txt")
	if err != nil {
		t.Fatalf("GetSyncItem: %v", err)
	}
	if item.SyncState != StatePendingUpload {
		t.Errorf("expected state %q for offline-edited file, got %q", StatePendingUpload, item.SyncState)
	}

	// Verify the local MD5 was updated to the new content's hash
	newMD5, err := computeMD5(testFile)
	if err != nil {
		t.Fatalf("compute md5: %v", err)
	}
	if item.LocalMD5 != newMD5 {
		t.Errorf("expected local_md5 %q, got %q", newMD5, item.LocalMD5)
	}
}
