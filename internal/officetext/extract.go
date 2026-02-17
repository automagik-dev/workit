// Package officetext extracts plain text from Office Open XML files
// (DOCX, XLSX, PPTX) using only Go stdlib (archive/zip + encoding/xml).
package officetext

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ErrUnsupportedFormat is returned when the file format is not supported
// for text extraction.
var ErrUnsupportedFormat = errors.New("unsupported format for text extraction")

// ErrNotFound is returned when an expected file is not found in the ZIP archive.
var ErrNotFound = errors.New("file not found in archive")

// ExtractText reads Office content from r, dispatching to a format-specific
// extractor based on the file extension of filename.
// Supported extensions: .docx, .xlsx, .pptx.
// Returns an error for unsupported or corrupted files.
func ExtractText(r io.Reader, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".docx":
		return extractFromReader(r, extractDocx)
	case ".xlsx":
		return extractFromReader(r, extractXlsx)
	case ".pptx":
		return extractFromReader(r, extractPptx)
	default:
		return "", fmt.Errorf("%w: %q (supported: .docx, .xlsx, .pptx)", ErrUnsupportedFormat, ext)
	}
}

// ExtractTextByMIME reads Office content from r, dispatching to a format-specific
// extractor based on the MIME type.
func ExtractTextByMIME(r io.Reader, mimeType string) (string, error) {
	switch mimeType {
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return extractFromReader(r, extractDocx)
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return extractFromReader(r, extractXlsx)
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return extractFromReader(r, extractPptx)
	default:
		return "", fmt.Errorf("%w: MIME type %q", ErrUnsupportedFormat, mimeType)
	}
}

// IsSupportedExtension returns true if the given filename has a supported
// Office extension for text extraction.
func IsSupportedExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".docx", ".xlsx", ".pptx":
		return true
	default:
		return false
	}
}

// IsSupportedMIME returns true if the given MIME type is supported
// for text extraction.
func IsSupportedMIME(mimeType string) bool {
	switch mimeType {
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return true
	default:
		return false
	}
}

// extractFromReader buffers the io.Reader into memory (required for zip.NewReader)
// and calls the given extraction function.
func extractFromReader(r io.Reader, fn func([]byte) (string, error)) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read file data: %w", err)
	}

	return fn(data)
}

// openZip creates a *zip.Reader from raw bytes.
func openZip(data []byte) (*zip.Reader, error) {
	r := bytes.NewReader(data)

	zr, err := zip.NewReader(r, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	return zr, nil
}

// maxZipEntrySize is the maximum decompressed size allowed for a single ZIP
// entry. This guards against zip bombs where a small compressed entry inflates
// to gigabytes of XML content.
const maxZipEntrySize = 100 * 1024 * 1024 // 100 MB decompressed

// readZipEntry finds and reads a named file from a zip.Reader.
// It caps the decompressed content at maxZipEntrySize to prevent
// unbounded memory allocation from crafted archives.
func readZipEntry(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open %s: %w", name, err)
			}
			defer rc.Close()

			lr := io.LimitReader(rc, maxZipEntrySize+1)
			content, err := io.ReadAll(lr)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", name, err)
			}
			if int64(len(content)) > maxZipEntrySize {
				return nil, fmt.Errorf("entry %s exceeds %d byte decompressed limit", name, maxZipEntrySize)
			}

			return content, nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
}
