package sync

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// computeMD5 computes the MD5 hash of a file.
func computeMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := md5.New()

	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
