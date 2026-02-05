package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	gosync "sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchEvent represents a filesystem change event.
type WatchEvent struct {
	Path      string    // Absolute path
	RelPath   string    // Relative to watched root
	Op        WatchOp   // Operation type
	Timestamp time.Time // When the event occurred
}

// WatchOp represents the type of filesystem operation.
type WatchOp int

const (
	// OpCreate indicates a file or directory was created.
	OpCreate WatchOp = iota
	// OpWrite indicates a file was modified.
	OpWrite
	// OpDelete indicates a file or directory was deleted.
	OpDelete
	// OpRename indicates a file or directory was renamed.
	OpRename
)

// String returns a string representation of the operation.
func (o WatchOp) String() string {
	switch o {
	case OpCreate:
		return "create"
	case OpWrite:
		return "write"
	case OpDelete:
		return "delete"
	case OpRename:
		return "rename"
	default:
		return "unknown"
	}
}

// Watcher watches a directory for filesystem changes.
type Watcher struct {
	root     string
	watcher  *fsnotify.Watcher
	events   chan WatchEvent
	errors   chan error
	debounce time.Duration

	// Debouncing state
	mu      gosync.Mutex
	pending map[string]*debounceEntry
}

type debounceEntry struct {
	event WatchEvent
	timer *time.Timer
}

// NewWatcher creates a new filesystem watcher.
// debounce specifies how long to wait after the last event before emitting.
func NewWatcher(root string, debounce time.Duration) (*Watcher, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("absolute path: %w", err)
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	w := &Watcher{
		root:     absRoot,
		watcher:  fsWatcher,
		events:   make(chan WatchEvent, 100),
		errors:   make(chan error, 10),
		debounce: debounce,
		pending:  make(map[string]*debounceEntry),
	}

	// Add root directory and all subdirectories
	if err := w.addRecursive(absRoot); err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("add recursive watches: %w", err)
	}

	return w, nil
}

// Events returns the channel of watch events.
func (w *Watcher) Events() <-chan WatchEvent {
	return w.events
}

// Errors returns the channel of errors.
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

// Start begins watching. Blocks until context is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			select {
			case w.errors <- err:
			default:
				// errors channel full, drop
			}
		}
	}
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	// Cancel all pending timers
	w.mu.Lock()
	for _, entry := range w.pending {
		entry.timer.Stop()
	}
	w.pending = make(map[string]*debounceEntry)
	w.mu.Unlock()

	return w.watcher.Close()
}

// handleEvent processes a raw fsnotify event.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// Check if this path should be ignored
	if w.shouldIgnore(path) {
		return
	}

	// Convert to our event type
	var op WatchOp
	switch {
	case event.Has(fsnotify.Create):
		op = OpCreate
		// If a directory was created, add it to watch
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			if err := w.addRecursive(path); err != nil {
				select {
				case w.errors <- fmt.Errorf("add watch for new dir %s: %w", path, err):
				default:
				}
			}
		}
	case event.Has(fsnotify.Write):
		op = OpWrite
	case event.Has(fsnotify.Remove):
		op = OpDelete
	case event.Has(fsnotify.Rename):
		op = OpRename
	default:
		// chmod or other event we don't care about
		return
	}

	// Calculate relative path
	relPath, err := filepath.Rel(w.root, path)
	if err != nil {
		relPath = path
	}

	watchEvent := WatchEvent{
		Path:      path,
		RelPath:   relPath,
		Op:        op,
		Timestamp: time.Now(),
	}

	// Debounce the event
	w.debounceEvent(watchEvent)
}

// debounceEvent schedules an event to be emitted after the debounce period.
func (w *Watcher) debounceEvent(event WatchEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// If there's an existing pending event for this path, cancel its timer
	if entry, exists := w.pending[event.Path]; exists {
		entry.timer.Stop()
		// Update the event (use the latest operation)
		entry.event = event
		entry.timer = time.AfterFunc(w.debounce, func() {
			w.emitEvent(event.Path)
		})
	} else {
		// Create new pending entry
		timer := time.AfterFunc(w.debounce, func() {
			w.emitEvent(event.Path)
		})
		w.pending[event.Path] = &debounceEntry{
			event: event,
			timer: timer,
		}
	}
}

// emitEvent sends the pending event for a path and removes it from pending.
func (w *Watcher) emitEvent(path string) {
	w.mu.Lock()
	entry, exists := w.pending[path]
	if exists {
		delete(w.pending, path)
	}
	w.mu.Unlock()

	if !exists {
		return
	}

	// Non-blocking send
	select {
	case w.events <- entry.event:
	default:
		// Channel full, drop oldest and try again
		select {
		case <-w.events:
		default:
		}
		select {
		case w.events <- entry.event:
		default:
		}
	}
}

// shouldIgnore returns true if the path should be ignored.
func (w *Watcher) shouldIgnore(path string) bool {
	// Get relative path for pattern matching
	relPath, err := filepath.Rel(w.root, path)
	if err != nil {
		relPath = path
	}

	// Split path into components
	parts := strings.Split(relPath, string(filepath.Separator))

	for _, part := range parts {
		// Ignore .git directory
		if part == ".git" {
			return true
		}

		// Ignore .gog-sync directory
		if part == ".gog-sync" {
			return true
		}

		// Ignore node_modules directory
		if part == "node_modules" {
			return true
		}

		// Ignore __pycache__ directory
		if part == "__pycache__" {
			return true
		}

		// Ignore hidden files/directories (starting with .)
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}

	// Get the base name for file-specific checks
	base := filepath.Base(path)

	// Ignore files ending with ~ (temp files)
	if strings.HasSuffix(base, "~") {
		return true
	}

	// Ignore .DS_Store
	if base == ".DS_Store" {
		return true
	}

	return false
}

// addRecursive adds a directory and all subdirectories to watch.
func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// If we can't access a path, skip it
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}

		// Only add directories
		if !d.IsDir() {
			return nil
		}

		// Skip ignored directories
		if w.shouldIgnore(path) {
			return filepath.SkipDir
		}

		// Add to watcher
		if err := w.watcher.Add(path); err != nil {
			// Ignore errors for directories we can't watch
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return fmt.Errorf("add watch for %s: %w", path, err)
		}

		return nil
	})
}
