package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseConflictStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected ConflictStrategy
		wantErr  bool
	}{
		{"rename", ConflictRename, false},
		{"RENAME", ConflictRename, false},
		{"", ConflictRename, false},
		{"local-wins", ConflictLocalWins, false},
		{"local_wins", ConflictLocalWins, false},
		{"localwins", ConflictLocalWins, false},
		{"remote-wins", ConflictRemoteWins, false},
		{"remote_wins", ConflictRemoteWins, false},
		{"remotewins", ConflictRemoteWins, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseConflictStrategy(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConflictResolver_RenameWithTimestamp(t *testing.T) {
	// Create a temp directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	resolver := &ConflictResolver{
		strategy: ConflictRename,
	}

	renamedPath, err := resolver.renameWithTimestamp(testFile)
	if err != nil {
		t.Fatalf("renameWithTimestamp failed: %v", err)
	}

	// Check that original file no longer exists
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("original file should not exist after rename")
	}

	// Check that renamed file exists
	if _, err := os.Stat(renamedPath); err != nil {
		t.Errorf("renamed file should exist: %v", err)
	}

	// Check that renamed file has correct pattern
	base := filepath.Base(renamedPath)
	if !containsConflictPattern(base) {
		t.Errorf("renamed file should contain .conflict- pattern: %s", base)
	}
}

func containsConflictPattern(name string) bool {
	// Check for pattern like test.conflict-2024-01-15-143022.txt
	return len(name) > 20 && // Must be long enough
		strings.Contains(name, ".conflict-")
}

func TestConflictResolver_DetectConflict(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create a test file
	if err := os.WriteFile(testFile, []byte("local content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	localMD5, _ := computeMD5(testFile)

	// Create a resolver (db can be nil for detection only)
	resolver := NewConflictResolver(nil, 1, ConflictRename)

	// Create a sync item representing the last synced state
	item := &SyncItem{
		LocalPath:  "test.txt",
		DriveID:    "drive-123",
		LocalMD5:   "old-md5-hash",
		RemoteMD5:  "old-md5-hash",
		LocalMtime: time.Now().Add(-1 * time.Hour),
	}

	// Test conflict detection when both local and remote changed
	conflict, err := resolver.DetectConflict(nil, item, testFile, "new-remote-md5", time.Now())
	if err != nil {
		t.Fatalf("DetectConflict failed: %v", err)
	}

	if conflict == nil {
		t.Error("expected conflict to be detected")
	} else {
		if !conflict.LocalModified {
			t.Error("expected LocalModified to be true")
		}

		if !conflict.RemoteModified {
			t.Error("expected RemoteModified to be true")
		}

		if conflict.LocalMD5 != localMD5 {
			t.Errorf("expected LocalMD5 %s, got %s", localMD5, conflict.LocalMD5)
		}
	}
}

func TestConflictResolver_NoConflict_OnlyLocalChanged(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create a test file
	if err := os.WriteFile(testFile, []byte("local content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	localMD5, _ := computeMD5(testFile)

	resolver := NewConflictResolver(nil, 1, ConflictRename)

	// Sync item where remote hasn't changed (remote MD5 matches last sync)
	item := &SyncItem{
		LocalPath:  "test.txt",
		DriveID:    "drive-123",
		LocalMD5:   "old-md5-hash",
		RemoteMD5:  "same-remote-md5",
		LocalMtime: time.Now().Add(-1 * time.Hour),
	}

	// Remote MD5 matches what we have stored - no remote change
	conflict, err := resolver.DetectConflict(nil, item, testFile, "same-remote-md5", time.Now())
	if err != nil {
		t.Fatalf("DetectConflict failed: %v", err)
	}

	if conflict != nil {
		t.Error("expected no conflict when only local changed")
	}

	_ = localMD5 // Silence unused variable warning
}

func TestConflictResolver_NoConflict_OnlyRemoteChanged(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create a test file with content that matches last sync
	if err := os.WriteFile(testFile, []byte("same content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	localMD5, _ := computeMD5(testFile)

	resolver := NewConflictResolver(nil, 1, ConflictRename)

	// Sync item where local hasn't changed (local MD5 matches last sync)
	item := &SyncItem{
		LocalPath:  "test.txt",
		DriveID:    "drive-123",
		LocalMD5:   localMD5, // Same as current local - no local change
		RemoteMD5:  "old-remote-md5",
		LocalMtime: time.Now().Add(-1 * time.Hour),
	}

	// Remote MD5 is different - remote changed
	conflict, err := resolver.DetectConflict(nil, item, testFile, "new-remote-md5", time.Now())
	if err != nil {
		t.Fatalf("DetectConflict failed: %v", err)
	}

	if conflict != nil {
		t.Error("expected no conflict when only remote changed")
	}
}
