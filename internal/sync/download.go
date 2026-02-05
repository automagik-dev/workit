package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/api/drive/v3"
)

// Downloader handles downloading files from Google Drive.
type Downloader struct {
	service   *drive.Service
	localRoot string
}

// DownloadResult contains the result of a download operation.
type DownloadResult struct {
	LocalPath string
	MD5       string
}

// NewDownloader creates a new downloader.
func NewDownloader(service *drive.Service, localRoot string) *Downloader {
	return &Downloader{
		service:   service,
		localRoot: localRoot,
	}
}

// DownloadFile downloads a file from Drive to the local filesystem.
func (d *Downloader) DownloadFile(ctx context.Context, fileID, fileName string) (*DownloadResult, error) {
	// Get file metadata to determine the path
	file, err := d.service.Files.Get(fileID).
		Context(ctx).
		Fields("id,name,mimeType,md5Checksum,parents").
		SupportsAllDrives(true).
		Do()
	if err != nil {
		return nil, fmt.Errorf("get file metadata: %w", err)
	}

	// Skip Google Docs, Sheets, etc. (they don't have binary content)
	if isGoogleDocsType(file.MimeType) {
		return nil, fmt.Errorf("cannot download Google Docs type: %s", file.MimeType)
	}

	// Build local path (for now, just use the filename in root)
	// TODO: Reconstruct full path from Drive folder hierarchy
	localPath := file.Name

	absPath := filepath.Join(d.localRoot, localPath)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, fmt.Errorf("create parent directory: %w", err)
	}

	// Download the file
	resp, err := d.service.Files.Get(fileID).
		Context(ctx).
		SupportsAllDrives(true).
		Download()
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	// Create local file
	f, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("create local file: %w", err)
	}
	defer f.Close()

	// Copy content
	if _, err := io.Copy(f, resp.Body); err != nil {
		return nil, fmt.Errorf("write file content: %w", err)
	}

	// Compute local MD5 for verification
	md5Hash, err := computeMD5(absPath)
	if err != nil {
		return nil, fmt.Errorf("compute md5: %w", err)
	}

	// Verify MD5 if available
	if file.Md5Checksum != "" && md5Hash != file.Md5Checksum {
		return nil, fmt.Errorf("md5 mismatch: local=%s, remote=%s", md5Hash, file.Md5Checksum)
	}

	return &DownloadResult{
		LocalPath: localPath,
		MD5:       md5Hash,
	}, nil
}

// DownloadFolder creates a local folder.
func (d *Downloader) DownloadFolder(ctx context.Context, folderID, folderName string) error {
	absPath := filepath.Join(d.localRoot, folderName)

	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	return nil
}

// isGoogleDocsType returns true if the MIME type is a Google Docs type.
func isGoogleDocsType(mimeType string) bool {
	googleDocTypes := []string{
		"application/vnd.google-apps.document",
		"application/vnd.google-apps.spreadsheet",
		"application/vnd.google-apps.presentation",
		"application/vnd.google-apps.drawing",
		"application/vnd.google-apps.form",
		"application/vnd.google-apps.script",
		"application/vnd.google-apps.site",
		"application/vnd.google-apps.map",
		"application/vnd.google-apps.jam",
	}

	for _, docType := range googleDocTypes {
		if mimeType == docType {
			return true
		}
	}

	return false
}
