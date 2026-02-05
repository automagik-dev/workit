package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
)

// Uploader handles uploading files to Google Drive.
type Uploader struct {
	service    *drive.Service
	rootFolder string
	driveID    string // For shared drives
	folderIDs  map[string]string // Maps relative paths to Drive folder IDs
}

// UploadResult contains the result of an upload operation.
type UploadResult struct {
	DriveID string
	MD5     string
	ModTime time.Time
}

// NewUploader creates a new uploader.
func NewUploader(service *drive.Service, rootFolder, driveID string) *Uploader {
	return &Uploader{
		service:    service,
		rootFolder: rootFolder,
		driveID:    driveID,
		folderIDs:  make(map[string]string),
	}
}

// UploadFile uploads a file to Drive.
func (u *Uploader) UploadFile(ctx context.Context, relPath, absPath string) (*UploadResult, error) {
	// Get parent folder ID
	parentID, err := u.ensureParentFolders(ctx, relPath)
	if err != nil {
		return nil, fmt.Errorf("ensure parent folders: %w", err)
	}

	// Open the file
	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Get file info
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Compute MD5 before upload
	md5Hash, err := computeMD5(absPath)
	if err != nil {
		return nil, fmt.Errorf("compute md5: %w", err)
	}

	// Check if file already exists in Drive
	existingID, err := u.findFileByName(ctx, parentID, filepath.Base(relPath))
	if err != nil {
		return nil, fmt.Errorf("check existing file: %w", err)
	}

	var driveFile *drive.File

	if existingID != "" {
		// Update existing file
		driveFile, err = u.updateFile(ctx, existingID, f)
	} else {
		// Create new file
		driveFile, err = u.createFile(ctx, parentID, filepath.Base(relPath), f)
	}

	if err != nil {
		return nil, err
	}

	return &UploadResult{
		DriveID: driveFile.Id,
		MD5:     md5Hash,
		ModTime: info.ModTime(),
	}, nil
}

// createFile creates a new file in Drive.
func (u *Uploader) createFile(ctx context.Context, parentID, name string, reader io.Reader) (*drive.File, error) {
	file := &drive.File{
		Name:    name,
		Parents: []string{parentID},
	}

	call := u.service.Files.Create(file).
		Context(ctx).
		Media(reader).
		Fields("id,md5Checksum")

	if u.driveID != "" {
		call = call.SupportsAllDrives(true)
	}

	result, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("create file in Drive: %w", err)
	}

	return result, nil
}

// updateFile updates an existing file in Drive.
func (u *Uploader) updateFile(ctx context.Context, fileID string, reader io.Reader) (*drive.File, error) {
	file := &drive.File{}

	call := u.service.Files.Update(fileID, file).
		Context(ctx).
		Media(reader).
		Fields("id,md5Checksum")

	if u.driveID != "" {
		call = call.SupportsAllDrives(true)
	}

	result, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("update file in Drive: %w", err)
	}

	return result, nil
}

// CreateFolder creates a folder in Drive.
func (u *Uploader) CreateFolder(ctx context.Context, relPath string) error {
	_, err := u.ensureParentFolders(ctx, relPath+"/dummy")
	if err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	// Now create the actual folder
	parentID, err := u.ensureParentFolders(ctx, relPath)
	if err != nil {
		return fmt.Errorf("get parent: %w", err)
	}

	folderName := filepath.Base(relPath)

	// Check if folder already exists
	existingID, err := u.findFolderByName(ctx, parentID, folderName)
	if err != nil {
		return fmt.Errorf("check existing folder: %w", err)
	}

	if existingID != "" {
		u.folderIDs[relPath] = existingID

		return nil
	}

	// Create the folder
	folder := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentID},
	}

	call := u.service.Files.Create(folder).
		Context(ctx).
		Fields("id")

	if u.driveID != "" {
		call = call.SupportsAllDrives(true)
	}

	result, err := call.Do()
	if err != nil {
		return fmt.Errorf("create folder in Drive: %w", err)
	}

	u.folderIDs[relPath] = result.Id

	return nil
}

// Delete removes a file or folder from Drive.
func (u *Uploader) Delete(ctx context.Context, relPath string) error {
	// Find the file/folder ID
	parentID, err := u.getParentFolderID(ctx, relPath)
	if err != nil {
		return fmt.Errorf("get parent folder: %w", err)
	}

	name := filepath.Base(relPath)

	// Try to find as file first
	fileID, err := u.findFileByName(ctx, parentID, name)
	if err != nil {
		return fmt.Errorf("find file: %w", err)
	}

	if fileID == "" {
		// Try as folder
		fileID, err = u.findFolderByName(ctx, parentID, name)
		if err != nil {
			return fmt.Errorf("find folder: %w", err)
		}
	}

	if fileID == "" {
		// Not found, nothing to delete
		return nil
	}

	// Trash the file (not permanent delete)
	call := u.service.Files.Update(fileID, &drive.File{Trashed: true}).
		Context(ctx)

	if u.driveID != "" {
		call = call.SupportsAllDrives(true)
	}

	_, err = call.Do()
	if err != nil {
		return fmt.Errorf("trash file in Drive: %w", err)
	}

	return nil
}

// ensureParentFolders ensures all parent folders exist and returns the parent ID.
func (u *Uploader) ensureParentFolders(ctx context.Context, relPath string) (string, error) {
	dir := filepath.Dir(relPath)
	if dir == "." || dir == "/" || dir == "" {
		return u.rootFolder, nil
	}

	// Check cache first
	if id, ok := u.folderIDs[dir]; ok {
		return id, nil
	}

	// Split path and ensure each component exists
	parts := strings.Split(dir, string(filepath.Separator))
	currentID := u.rootFolder
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		currentPath = filepath.Join(currentPath, part)

		// Check cache
		if id, ok := u.folderIDs[currentPath]; ok {
			currentID = id

			continue
		}

		// Check if folder exists in Drive
		existingID, err := u.findFolderByName(ctx, currentID, part)
		if err != nil {
			return "", fmt.Errorf("find folder %s: %w", part, err)
		}

		if existingID != "" {
			u.folderIDs[currentPath] = existingID
			currentID = existingID

			continue
		}

		// Create folder
		folder := &drive.File{
			Name:     part,
			MimeType: "application/vnd.google-apps.folder",
			Parents:  []string{currentID},
		}

		call := u.service.Files.Create(folder).
			Context(ctx).
			Fields("id")

		if u.driveID != "" {
			call = call.SupportsAllDrives(true)
		}

		result, err := call.Do()
		if err != nil {
			return "", fmt.Errorf("create folder %s: %w", part, err)
		}

		u.folderIDs[currentPath] = result.Id
		currentID = result.Id
	}

	return currentID, nil
}

// getParentFolderID gets the parent folder ID for a path.
func (u *Uploader) getParentFolderID(ctx context.Context, relPath string) (string, error) {
	dir := filepath.Dir(relPath)
	if dir == "." || dir == "/" || dir == "" {
		return u.rootFolder, nil
	}

	// Check cache
	if id, ok := u.folderIDs[dir]; ok {
		return id, nil
	}

	// Try to find it
	return u.ensureParentFolders(ctx, relPath)
}

// findFileByName finds a file by name in a parent folder.
func (u *Uploader) findFileByName(ctx context.Context, parentID, name string) (string, error) {
	query := fmt.Sprintf("'%s' in parents and name = '%s' and mimeType != 'application/vnd.google-apps.folder' and trashed = false",
		parentID, escapeDriveQuery(name))

	call := u.service.Files.List().
		Context(ctx).
		Q(query).
		Fields("files(id)").
		PageSize(1)

	if u.driveID != "" {
		call = call.SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			Corpora("drive").
			DriveId(u.driveID)
	}

	result, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("list files: %w", err)
	}

	if len(result.Files) > 0 {
		return result.Files[0].Id, nil
	}

	return "", nil
}

// findFolderByName finds a folder by name in a parent folder.
func (u *Uploader) findFolderByName(ctx context.Context, parentID, name string) (string, error) {
	query := fmt.Sprintf("'%s' in parents and name = '%s' and mimeType = 'application/vnd.google-apps.folder' and trashed = false",
		parentID, escapeDriveQuery(name))

	call := u.service.Files.List().
		Context(ctx).
		Q(query).
		Fields("files(id)").
		PageSize(1)

	if u.driveID != "" {
		call = call.SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			Corpora("drive").
			DriveId(u.driveID)
	}

	result, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("list folders: %w", err)
	}

	if len(result.Files) > 0 {
		return result.Files[0].Id, nil
	}

	return "", nil
}

// escapeDriveQuery escapes a string for use in Drive API queries.
func escapeDriveQuery(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")

	return s
}
