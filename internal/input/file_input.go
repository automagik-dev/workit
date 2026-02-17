package input

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sentinel errors for security validation.
var (
	errPathEscapesCWD = errors.New("file path escapes working directory")
	errSensitiveFile  = errors.New("access to sensitive file blocked")
	errSymlinkEscape  = errors.New("symlink target escapes working directory")
	errFileTooLarge   = errors.New("file exceeds maximum size")
)

const (
	prefixFile  = "file://"
	prefixFileB = "fileb://"

	// maxFileSize is the maximum file size allowed for file:// input (10 MB).
	maxFileSize = 10 * 1024 * 1024
)

// ResolveFileInput checks if a string value starts with "file://" or "fileb://",
// and if so reads the referenced file with security validation.
// If the value has no file prefix, it is returned as-is (literal passthrough).
//
// Prefixes:
//   - file://  — read UTF-8 text, return as string
//   - fileb:// — read binary, return base64-encoded
//   - (none)   — literal passthrough
//
// Security: files must be within CWD, symlink targets are validated,
// sensitive file patterns are rejected, and a 10 MB size limit is enforced.
func ResolveFileInput(value string) (string, error) {
	var path string
	var binary bool

	switch {
	case strings.HasPrefix(value, prefixFileB):
		path = value[len(prefixFileB):]
		binary = true
	case strings.HasPrefix(value, prefixFile):
		path = value[len(prefixFile):]
	default:
		return value, nil
	}

	data, err := readSecureFile(path)
	if err != nil {
		return "", err
	}

	if binary {
		return base64.StdEncoding.EncodeToString(data), nil
	}

	return string(data), nil
}

// readSecureFile implements the security chain:
//  1. filepath.Abs + filepath.Clean — resolve to absolute, normalized path
//  2. CWD prefix check on raw path — catches obvious ../../../ traversals
//  3. filepath.EvalSymlinks — resolve ALL symlinks (including intermediate dirs)
//  4. CWD prefix re-check on resolved path — catches symlink escapes
//  5. Sensitive file pattern checks on both raw and resolved paths
//  6. Size check + os.ReadFile on resolved path
func readSecureFile(path string) ([]byte, error) {
	// Step 1: resolve to absolute path and normalize
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	cwdClean := filepath.Clean(cwd)
	// Resolve symlinks in CWD itself so that symlinked working directories
	// (common in worktree setups) don't cause false containment failures.
	if cwdResolved, evalErr := filepath.EvalSymlinks(cwdClean); evalErr == nil {
		cwdClean = cwdResolved
	}

	// Compute the absolute path. filepath.Abs uses os.Getwd() internally,
	// which may return the symlinked CWD. For relative paths, re-derive
	// the absolute path using the resolved CWD so containment checks work.
	var cleaned string
	if filepath.IsAbs(path) {
		cleaned = filepath.Clean(path)
		// Resolve symlinks so absolute paths match the resolved CWD.
		// On macOS /var -> /private/var; without this, containment fails.
		if absResolved, evalErr := filepath.EvalSymlinks(cleaned); evalErr == nil {
			cleaned = absResolved
		}
	} else {
		cleaned = filepath.Clean(filepath.Join(cwdClean, path))
	}

	// Step 2: CWD check on raw path (catches obvious ../../../ etc)
	if !isWithinDir(cleaned, cwdClean) {
		return nil, fmt.Errorf("%w: %s", errPathEscapesCWD, path)
	}

	// Step 3: Resolve ALL symlinks in the full path (including intermediate
	// directory components), then re-check CWD containment on the resolved path.
	// This prevents attacks where a parent directory is a symlink pointing
	// outside CWD (e.g., safe-linkdir -> /etc).
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("resolve path: %w", err)
		}
		// File doesn't exist; will fail at ReadFile with a clear error.
		resolved = cleaned
	} else {
		resolved = filepath.Clean(resolved)

		// Step 4: CWD check on the fully resolved path
		if !isWithinDir(resolved, cwdClean) {
			return nil, fmt.Errorf("%w: %s", errSymlinkEscape, path)
		}

		// Check sensitive patterns on the resolved path
		if isSensitiveFile(resolved, cwdClean) {
			return nil, fmt.Errorf("%w: %s", errSensitiveFile, path)
		}
	}

	// Check sensitive file patterns on the original (pre-symlink) path
	if isSensitiveFile(cleaned, cwdClean) {
		relPath, relErr := filepath.Rel(cwdClean, cleaned)
		if relErr != nil {
			relPath = path
		}

		return nil, fmt.Errorf("%w: %s", errSensitiveFile, relPath)
	}

	// Step 5: Stat the resolved path for size check
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("%w (%d bytes > %d bytes)", errFileTooLarge, info.Size(), maxFileSize)
	}

	// Step 6: read the resolved file
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return data, nil
}

// isWithinDir checks if path is within or equal to dir.
// Both paths must be absolute and cleaned.
func isWithinDir(path, dir string) bool {
	if path == dir {
		return true
	}
	// When dir is the filesystem root, every absolute path is within it.
	if dir == "/" {
		return true
	}
	// Ensure the dir ends with separator for prefix matching
	prefix := dir + string(filepath.Separator)

	return strings.HasPrefix(path, prefix)
}

// isSensitiveFile checks if the file path matches any sensitive file pattern.
// Matching is case-insensitive on the relative path components.
func isSensitiveFile(absPath, cwdPath string) bool {
	relPath, err := filepath.Rel(cwdPath, absPath)
	if err != nil {
		relPath = filepath.Base(absPath)
	}

	// Normalize for case-insensitive matching
	lower := strings.ToLower(relPath)
	base := filepath.Base(lower)

	// .env and .env.*
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}

	// .ssh/*, .aws/*, .gcloud/* — check if any path component is these dirs
	parts := strings.Split(lower, string(filepath.Separator))
	for _, part := range parts {
		if part == ".ssh" || part == ".aws" || part == ".gcloud" {
			return true
		}
	}

	// *credentials*, *secret*, *token* — check both the basename and each
	// path component so that paths like "safe/secret/config.txt" are caught.
	for _, part := range parts {
		if strings.Contains(part, "credentials") ||
			strings.Contains(part, "secret") ||
			strings.Contains(part, "token") {
			return true
		}
	}

	// Certificate/key file extensions
	ext := strings.ToLower(filepath.Ext(absPath))
	switch ext {
	case ".pem", ".key", ".p12", ".pfx":
		return true
	}

	// SSH private key basenames
	switch base {
	case "id_rsa", "id_ed25519", "id_dsa":
		return true
	}

	return false
}
