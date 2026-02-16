package input

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helperChdirTemp creates a temp dir, changes CWD into it, and returns a
// cleanup function that restores the previous CWD.
func helperChdirTemp(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chdir(orig) })

	return tmp
}

func TestResolveFileInput_LiteralPassthrough(t *testing.T) {
	helperChdirTemp(t)

	val, err := ResolveFileInput("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != "hello world" {
		t.Fatalf("expected literal passthrough, got %q", val)
	}
}

func TestResolveFileInput_EmptyString(t *testing.T) {
	helperChdirTemp(t)

	val, err := ResolveFileInput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != "" {
		t.Fatalf("expected empty string, got %q", val)
	}
}

func TestResolveFileInput_ReadSuccess(t *testing.T) {
	tmp := helperChdirTemp(t)

	content := "email body content\nwith newlines"
	if err := os.WriteFile(filepath.Join(tmp, "test.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	val, err := ResolveFileInput("file://test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != content {
		t.Fatalf("expected %q, got %q", content, val)
	}
}

func TestResolveFileInput_ReadSuccessAbsolutePath(t *testing.T) {
	tmp := helperChdirTemp(t)

	content := "abs path content"
	if err := os.WriteFile(filepath.Join(tmp, "abs.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	val, err := ResolveFileInput("file://" + filepath.Join(tmp, "abs.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != content {
		t.Fatalf("expected %q, got %q", content, val)
	}
}

func TestResolveFileInput_ReadSuccessSubdir(t *testing.T) {
	tmp := helperChdirTemp(t)

	subdir := filepath.Join(tmp, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "subdir content"
	if err := os.WriteFile(filepath.Join(subdir, "file.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	val, err := ResolveFileInput("file://subdir/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != content {
		t.Fatalf("expected %q, got %q", content, val)
	}
}

func TestResolveFileInput_PathTraversalBlocked(t *testing.T) {
	helperChdirTemp(t)

	_, err := ResolveFileInput("file://../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}

	if !strings.Contains(err.Error(), "escapes working directory") {
		t.Fatalf("expected 'escapes working directory' error, got: %v", err)
	}
}

func TestResolveFileInput_SymlinkWithinCWDAllowed(t *testing.T) {
	tmp := helperChdirTemp(t)

	// Create a real file in CWD subtree
	content := "symlinked content"
	if err := os.WriteFile(filepath.Join(tmp, "real.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to it within CWD
	if err := os.Symlink(filepath.Join(tmp, "real.txt"), filepath.Join(tmp, "link.txt")); err != nil {
		t.Fatal(err)
	}

	val, err := ResolveFileInput("file://link.txt")
	if err != nil {
		t.Fatalf("unexpected error for symlink within CWD: %v", err)
	}

	if val != content {
		t.Fatalf("expected %q, got %q", content, val)
	}
}

func TestResolveFileInput_SymlinkOutsideCWDRejected(t *testing.T) {
	// Create a file outside CWD
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmp := helperChdirTemp(t)

	// Create a symlink from CWD pointing outside
	if err := os.Symlink(filepath.Join(outsideDir, "secret.txt"), filepath.Join(tmp, "escape.txt")); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveFileInput("file://escape.txt")
	if err == nil {
		t.Fatal("expected error for symlink outside CWD, got nil")
	}

	if !strings.Contains(err.Error(), "symlink target escapes working directory") {
		t.Fatalf("expected 'symlink target escapes working directory' error, got: %v", err)
	}
}

func TestResolveFileInput_SensitiveFileRejected(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"dotenv", ".env"},
		{"dotenv local", ".env.local"},
		{"dotenv production", ".env.production"},
		{"ssh dir", ".ssh/id_rsa"},
		{"aws dir", ".aws/credentials"},
		{"gcloud dir", ".gcloud/config"},
		{"credentials in name", "my_credentials.json"},
		{"secret in name", "api_secret.txt"},
		{"token in name", "access_token.json"},
		{"pem file", "server.pem"},
		{"key file", "private.key"},
		{"p12 file", "cert.p12"},
		{"pfx file", "cert.pfx"},
		{"id_rsa", "id_rsa"},
		{"id_ed25519", "id_ed25519"},
		{"id_dsa", "id_dsa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := helperChdirTemp(t)

			// Create any needed subdirectories
			dir := filepath.Dir(tt.filename)
			if dir != "." {
				if err := os.MkdirAll(filepath.Join(tmp, dir), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			// Create the file
			if err := os.WriteFile(filepath.Join(tmp, tt.filename), []byte("sensitive"), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := ResolveFileInput("file://" + tt.filename)
			if err == nil {
				t.Fatalf("expected error for sensitive file %q, got nil", tt.filename)
			}

			if !strings.Contains(err.Error(), "access to sensitive file blocked") {
				t.Fatalf("expected 'access to sensitive file blocked' error for %q, got: %v", tt.filename, err)
			}
		})
	}
}

func TestResolveFileInput_CaseInsensitiveSensitiveMatch(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"uppercase ENV", ".ENV"},
		{"mixed case Env", ".Env"},
		{"uppercase PEM", "cert.PEM"},
		{"mixed case Key", "private.Key"},
		{"uppercase CREDENTIALS", "MY_CREDENTIALS.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := helperChdirTemp(t)

			if err := os.WriteFile(filepath.Join(tmp, tt.filename), []byte("sensitive"), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := ResolveFileInput("file://" + tt.filename)
			if err == nil {
				t.Fatalf("expected error for sensitive file %q, got nil", tt.filename)
			}

			if !strings.Contains(err.Error(), "access to sensitive file blocked") {
				t.Fatalf("expected 'access to sensitive file blocked' error for %q, got: %v", tt.filename, err)
			}
		})
	}
}

func TestResolveFileInput_SizeLimitExceeded(t *testing.T) {
	tmp := helperChdirTemp(t)

	// Create a file larger than 10MB
	bigData := make([]byte, 10*1024*1024+1)
	if err := os.WriteFile(filepath.Join(tmp, "big.txt"), bigData, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveFileInput("file://big.txt")
	if err == nil {
		t.Fatal("expected error for oversized file, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("expected 'exceeds maximum size' error, got: %v", err)
	}
}

func TestResolveFileInput_FilebBase64Encoding(t *testing.T) {
	tmp := helperChdirTemp(t)

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	if err := os.WriteFile(filepath.Join(tmp, "data.bin"), binaryData, 0o644); err != nil {
		t.Fatal(err)
	}

	val, err := ResolveFileInput("fileb://data.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := base64.StdEncoding.EncodeToString(binaryData)
	if val != expected {
		t.Fatalf("expected base64 %q, got %q", expected, val)
	}
}

func TestResolveFileInput_FileNotFound(t *testing.T) {
	helperChdirTemp(t)

	_, err := ResolveFileInput("file://nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestResolveFileInput_IntermediateSymlinkEscape(t *testing.T) {
	// A parent directory is a symlink pointing outside CWD.
	// filepath.Clean/Abs don't resolve intermediate symlinks,
	// so the raw path passes CWD check but the real target is outside.
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "data.txt"), []byte("outside-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmp := helperChdirTemp(t)

	// Create a symlink directory inside CWD that points outside
	if err := os.Symlink(outsideDir, filepath.Join(tmp, "safe-linkdir")); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveFileInput("file://safe-linkdir/data.txt")
	if err == nil {
		t.Fatal("expected error for intermediate symlink escape, got nil")
	}

	// Should be blocked by symlink escape or CWD containment
	errMsg := err.Error()
	if !strings.Contains(errMsg, "symlink target escapes working directory") &&
		!strings.Contains(errMsg, "escapes working directory") {
		t.Fatalf("expected symlink/CWD escape error, got: %v", err)
	}
}

func TestResolveFileInput_SymlinkedCWDAllowsValidFiles(t *testing.T) {
	// Simulate running from a symlinked working directory (common in worktree
	// setups). The real directory is "realdir" and we chdir into a symlink
	// pointing to it. Files inside should still be accessible.
	parentDir := t.TempDir()

	realDir := filepath.Join(parentDir, "realdir")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "hello from symlinked cwd"
	if err := os.WriteFile(filepath.Join(realDir, "test.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the real directory
	linkDir := filepath.Join(parentDir, "linkdir")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	// Save original CWD and chdir into the symlinked directory
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(linkDir); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chdir(orig) })

	// This should succeed: the file is within the real CWD even though we
	// entered through a symlink.
	val, err := ResolveFileInput("file://test.txt")
	if err != nil {
		t.Fatalf("unexpected error for file in symlinked CWD: %v", err)
	}

	if val != content {
		t.Fatalf("expected %q, got %q", content, val)
	}
}

func TestResolveFileInput_SensitivePatternInPathComponent(t *testing.T) {
	// Paths like "safe/secret/config.txt" should be blocked because "secret"
	// is a path component matching the sensitive pattern, even though the
	// basename "config.txt" is not sensitive.
	tests := []struct {
		name     string
		filename string
	}{
		{"secret dir component", "safe/secret/config.txt"},
		{"credentials dir component", "data/credentials/app.json"},
		{"token dir component", "cache/token/refresh.dat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := helperChdirTemp(t)

			dir := filepath.Dir(tt.filename)
			if err := os.MkdirAll(filepath.Join(tmp, dir), 0o755); err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(filepath.Join(tmp, tt.filename), []byte("data"), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := ResolveFileInput("file://" + tt.filename)
			if err == nil {
				t.Fatalf("expected error for sensitive path %q, got nil", tt.filename)
			}

			if !strings.Contains(err.Error(), "access to sensitive file blocked") {
				t.Fatalf("expected 'access to sensitive file blocked' error for %q, got: %v", tt.filename, err)
			}
		})
	}
}

func TestResolveFileInput_NonSensitiveFileAllowed(t *testing.T) {
	// Ensure files that look somewhat like sensitive names but are not actually
	// in the sensitive pattern list are allowed.
	tmp := helperChdirTemp(t)

	if err := os.WriteFile(filepath.Join(tmp, "readme.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	val, err := ResolveFileInput("file://readme.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != "ok" {
		t.Fatalf("expected 'ok', got %q", val)
	}
}
