package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteAtomicExecutableWindows_NewFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "wk.exe")
	content := []byte("new-binary-content")

	if err := writeAtomicExecutableWindows(path, content); err != nil {
		t.Fatalf("writeAtomicExecutableWindows: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestWriteAtomicExecutableWindows_ReplaceExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "wk.exe")

	// Write an "old" binary first.
	if err := os.WriteFile(path, []byte("old-binary"), 0o644); err != nil {
		t.Fatal(err)
	}

	newContent := []byte("new-binary-content")
	if err := writeAtomicExecutableWindows(path, newContent); err != nil {
		t.Fatalf("writeAtomicExecutableWindows: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if string(got) != string(newContent) {
		t.Errorf("content = %q, want %q", got, newContent)
	}

	// .old should have been cleaned up.
	oldPath := path + ".old"
	if _, err := os.Stat(oldPath); err == nil {
		t.Errorf(".old file should have been removed: %s", oldPath)
	}
}

func TestWriteAtomicExecutable_NewFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "wk")
	content := []byte("binary-content")

	if err := writeAtomicExecutable(path, content); err != nil {
		t.Fatalf("writeAtomicExecutable: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		if info.Mode().Perm() != 0o755 {
			t.Errorf("permissions = %o, want 0755", info.Mode().Perm())
		}
	}
}

func TestWriteAtomicExecutable_ReplaceExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "wk")

	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	newContent := []byte("new-binary")

	if err := writeAtomicExecutable(path, newContent); err != nil {
		t.Fatalf("writeAtomicExecutable: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != string(newContent) {
		t.Errorf("content = %q, want %q", got, newContent)
	}
}

func TestWriteAtomicExecutableWindows_CleansLeftoverOld(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "wk.exe")
	oldPath := path + ".old"

	// Create a leftover .old from a previous run.
	if err := os.WriteFile(oldPath, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	newContent := []byte("fresh-binary")
	if err := writeAtomicExecutableWindows(path, newContent); err != nil {
		t.Fatalf("writeAtomicExecutableWindows: %v", err)
	}

	// .old should be gone (cleaned up at start).
	if _, err := os.Stat(oldPath); err == nil {
		t.Error(".old file should have been removed")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newContent) {
		t.Errorf("content = %q, want %q", got, newContent)
	}
}
