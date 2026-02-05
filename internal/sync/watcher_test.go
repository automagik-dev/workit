package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	if w.root != tmpDir {
		t.Errorf("watcher.root = %v, want %v", w.root, tmpDir)
	}
}

func TestWatcher_CreateEvent(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watcher in background
	go func() {
		_ = w.Start(ctx)
	}()

	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Wait for debounce + processing time
	select {
	case event := <-w.Events():
		if event.Op != OpCreate && event.Op != OpWrite {
			t.Errorf("event.Op = %v, want OpCreate or OpWrite", event.Op)
		}
		if event.Path != testFile {
			t.Errorf("event.Path = %v, want %v", event.Path, testFile)
		}
		if event.RelPath != "test.txt" {
			t.Errorf("event.RelPath = %v, want %v", event.RelPath, "test.txt")
		}
	case err := <-w.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for create event")
	}
}

func TestWatcher_WriteEvent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file before watching
	testFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	select {
	case event := <-w.Events():
		if event.Op != OpWrite {
			t.Errorf("event.Op = %v, want OpWrite", event.Op)
		}
		if event.Path != testFile {
			t.Errorf("event.Path = %v, want %v", event.Path, testFile)
		}
	case err := <-w.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for write event")
	}
}

func TestWatcher_DeleteEvent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file before watching
	testFile := filepath.Join(tmpDir, "todelete.txt")
	if err := os.WriteFile(testFile, []byte("delete me"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Delete the file
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	select {
	case event := <-w.Events():
		if event.Op != OpDelete {
			t.Errorf("event.Op = %v, want OpDelete", event.Op)
		}
		if event.Path != testFile {
			t.Errorf("event.Path = %v, want %v", event.Path, testFile)
		}
	case err := <-w.Errors():
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for delete event")
	}
}

func TestWatcher_Debounce(t *testing.T) {
	tmpDir := t.TempDir()

	// Use longer debounce to ensure we can write multiple times
	w, err := NewWatcher(tmpDir, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Create file and write to it multiple times rapidly
	testFile := filepath.Join(tmpDir, "debounce.txt")
	for i := 0; i < 10; i++ {
		if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Should receive only one event after debounce period
	eventCount := 0
	timeout := time.After(1 * time.Second)

loop:
	for {
		select {
		case <-w.Events():
			eventCount++
		case <-timeout:
			break loop
		}
	}

	// With debouncing, we should get very few events (ideally 1-2)
	if eventCount > 3 {
		t.Errorf("got %d events, expected <= 3 due to debouncing", eventCount)
	}
	if eventCount == 0 {
		t.Error("got 0 events, expected at least 1")
	}
}

func TestWatcher_IgnoreGit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Create file in .git directory (should be ignored)
	gitFile := filepath.Join(gitDir, "config")
	if err := os.WriteFile(gitFile, []byte("git config"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Also create a normal file that should trigger event
	normalFile := filepath.Join(tmpDir, "normal.txt")
	if err := os.WriteFile(normalFile, []byte("normal content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Should only get event for normal.txt, not .git/config
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case event := <-w.Events():
			if filepath.Dir(event.Path) == gitDir || event.Path == gitFile {
				t.Errorf("received event for ignored .git path: %v", event.Path)
			}
			if event.Path == normalFile {
				return // success - got the normal file event
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-timeout:
			t.Fatal("timeout waiting for normal file event")
		}
	}
}

func TestWatcher_IgnoreHiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Create hidden file (should be ignored)
	hiddenFile := filepath.Join(tmpDir, ".hidden")
	if err := os.WriteFile(hiddenFile, []byte("hidden"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create normal file that should trigger event
	normalFile := filepath.Join(tmpDir, "visible.txt")
	if err := os.WriteFile(normalFile, []byte("visible"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case event := <-w.Events():
			if event.Path == hiddenFile {
				t.Errorf("received event for hidden file: %v", event.Path)
			}
			if event.Path == normalFile {
				return // success
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-timeout:
			t.Fatal("timeout waiting for visible file event")
		}
	}
}

func TestWatcher_IgnoreTempFiles(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Create temp file ending with ~ (should be ignored)
	tempFile := filepath.Join(tmpDir, "file.txt~")
	if err := os.WriteFile(tempFile, []byte("temp"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create normal file
	normalFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(normalFile, []byte("normal"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case event := <-w.Events():
			if event.Path == tempFile {
				t.Errorf("received event for temp file: %v", event.Path)
			}
			if event.Path == normalFile {
				return // success
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-timeout:
			t.Fatal("timeout waiting for normal file event")
		}
	}
}

func TestWatcher_RecursiveSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	// Give watcher time to add the new directory
	time.Sleep(100 * time.Millisecond)

	// Drain any directory creation events
drainLoop:
	for {
		select {
		case <-w.Events():
		default:
			break drainLoop
		}
	}

	// Create file in subdirectory
	subFile := filepath.Join(subDir, "subfile.txt")
	if err := os.WriteFile(subFile, []byte("sub content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Should receive event for file in subdirectory
	timeout := time.After(1 * time.Second)
	for {
		select {
		case event := <-w.Events():
			if event.Path == subFile {
				expectedRelPath := filepath.Join("subdir", "subfile.txt")
				if event.RelPath != expectedRelPath {
					t.Errorf("event.RelPath = %v, want %v", event.RelPath, expectedRelPath)
				}
				return // success
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-timeout:
			t.Fatal("timeout waiting for subdirectory file event")
		}
	}
}

func TestWatcher_Stop(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		_ = w.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Cancel should cause Start to return
	cancel()

	select {
	case <-done:
		// success - Start returned
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for watcher to stop")
	}

	// Explicit stop should also work
	if err := w.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestWatcher_IgnoreNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create node_modules directory
	nmDir := filepath.Join(tmpDir, "node_modules")
	if err := os.Mkdir(nmDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	w, err := NewWatcher(tmpDir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Create file in node_modules (should be ignored)
	nmFile := filepath.Join(nmDir, "package.json")
	if err := os.WriteFile(nmFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create normal file that should trigger event
	normalFile := filepath.Join(tmpDir, "index.js")
	if err := os.WriteFile(normalFile, []byte("console.log()"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case event := <-w.Events():
			if event.Path == nmFile {
				t.Errorf("received event for node_modules file: %v", event.Path)
			}
			if event.Path == normalFile {
				return // success
			}
		case err := <-w.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-timeout:
			t.Fatal("timeout waiting for normal file event")
		}
	}
}

func TestWatchOp_String(t *testing.T) {
	tests := []struct {
		op   WatchOp
		want string
	}{
		{OpCreate, "create"},
		{OpWrite, "write"},
		{OpDelete, "delete"},
		{OpRename, "rename"},
		{WatchOp(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.op.String(); got != tt.want {
				t.Errorf("WatchOp.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
