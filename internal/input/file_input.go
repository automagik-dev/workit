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

// readSecureFile implements the 6-step security chain:
//  1. filepath.Abs — resolve to absolute path
//  2. filepath.Clean — normalize
//  3. strings.HasPrefix — must be within CWD
//  4. os.Lstat — check WITHOUT following symlinks
//  5. If symlink: EvalSymlinks -> re-check target in CWD
//  6. os.ReadFile — safe to read
func readSecureFile(path string) ([]byte, error) {
	// Step 1: resolve to absolute path
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	// Step 2: normalize
	cleaned := filepath.Clean(absPath)
	cwdClean := filepath.Clean(cwd)

	// Step 3: validate within CWD subtree
	if !isWithinDir(cleaned, cwdClean) {
		return nil, fmt.Errorf("%w: %s", errPathEscapesCWD, path)
	}

	// Check sensitive file patterns before any I/O
	if isSensitiveFile(cleaned, cwdClean) {
		relPath, relErr := filepath.Rel(cwdClean, cleaned)
		if relErr != nil {
			relPath = path
		}

		return nil, fmt.Errorf("%w: %s", errSensitiveFile, relPath)
	}

	// Step 4: Lstat (no symlink follow)
	info, err := os.Lstat(cleaned)
	if err != nil {
		return nil, fmt.Errorf("lstat: %w", err)
	}

	// Step 5: if symlink, evaluate and re-check
	if info.Mode()&os.ModeSymlink != 0 {
		target, evalErr := filepath.EvalSymlinks(cleaned)
		if evalErr != nil {
			return nil, fmt.Errorf("resolve symlink: %w", evalErr)
		}

		target = filepath.Clean(target)

		if !isWithinDir(target, cwdClean) {
			return nil, fmt.Errorf("%w: %s", errSymlinkEscape, target)
		}

		// Re-check sensitive pattern on the resolved target
		if isSensitiveFile(target, cwdClean) {
			return nil, fmt.Errorf("%w: %s", errSensitiveFile, path)
		}

		// Stat the target to get size
		info, err = os.Stat(target)
		if err != nil {
			return nil, fmt.Errorf("stat symlink target: %w", err)
		}
	}

	// Check file size
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("%w (%d bytes > %d bytes)", errFileTooLarge, info.Size(), maxFileSize)
	}

	// Step 6: read the file
	data, err := os.ReadFile(cleaned)
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
	dir := filepath.Dir(lower)

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
	// Also check direct dir
	if dir == ".ssh" || dir == ".aws" || dir == ".gcloud" {
		return true
	}

	// *credentials*, *secret*, *token* in the basename
	if strings.Contains(base, "credentials") ||
		strings.Contains(base, "secret") ||
		strings.Contains(base, "token") {
		return true
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
