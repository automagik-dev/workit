package sync

import (
	"context"
	"os"
	"time"

	"google.golang.org/api/drive/v3"
)

// DefaultPollIntervalDuration is the default poll interval for Drive changes.
const DefaultPollIntervalDuration = 5 * time.Second

// DriveChange represents a change detected on Google Drive.
type DriveChange struct {
	FileID    string
	FileName  string
	MimeType  string
	Op        DriveChangeOp
	Removed   bool // File was deleted/trashed
	Timestamp time.Time
}

// DriveChangeOp represents the type of Drive change.
type DriveChangeOp int

const (
	// DriveOpCreate indicates a file was created.
	DriveOpCreate DriveChangeOp = iota
	// DriveOpModify indicates a file was modified.
	DriveOpModify
	// DriveOpDelete indicates a file was deleted.
	DriveOpDelete
)

// String returns the string representation of the change operation.
func (op DriveChangeOp) String() string {
	switch op {
	case DriveOpCreate:
		return "create"
	case DriveOpModify:
		return "modify"
	case DriveOpDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// DrivePoller polls Google Drive for changes.
type DrivePoller struct {
	service      *drive.Service
	folderID     string        // Root folder to watch
	pollInterval time.Duration // Default 5s
	events       chan DriveChange
	errors       chan error

	// Page token management
	db       *DB
	configID int64
}

// DefaultPollInterval returns the poll interval from environment or default.
func DefaultPollInterval() time.Duration {
	if envVal := os.Getenv("WK_SYNC_POLL_INTERVAL"); envVal != "" {
		if d, err := time.ParseDuration(envVal); err == nil {
			return d
		}
	}
	return DefaultPollIntervalDuration
}

// NewDrivePoller creates a new Drive changes poller.
func NewDrivePoller(service *drive.Service, db *DB, configID int64, folderID string, pollInterval time.Duration) *DrivePoller {
	return &DrivePoller{
		service:      service,
		db:           db,
		configID:     configID,
		folderID:     folderID,
		pollInterval: pollInterval,
		events:       make(chan DriveChange, 100),
		errors:       make(chan error, 10),
	}
}

// Events returns the channel of drive changes.
func (p *DrivePoller) Events() <-chan DriveChange {
	return p.events
}

// Errors returns the channel of errors.
func (p *DrivePoller) Errors() <-chan error {
	return p.errors
}

// Start begins polling. Blocks until context is cancelled.
func (p *DrivePoller) Start(ctx context.Context) error {
	// Check for immediate cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Return early if service is nil (useful for testing)
	if p.service == nil {
		<-ctx.Done()
		return ctx.Err()
	}

	// Get or initialize start page token
	pageToken, err := p.getOrInitPageToken(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			changes, newToken, err := p.pollChanges(ctx, pageToken)
			if err != nil {
				select {
				case p.errors <- err:
				default:
					// Error channel full, skip
				}
				continue
			}

			// Send changes to events channel
			for _, change := range changes {
				select {
				case p.events <- change:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			// Save new page token
			if newToken != pageToken {
				if p.db != nil {
					if err := p.db.UpdateChangeToken(p.configID, newToken); err != nil {
						select {
						case p.errors <- err:
						default:
						}
					}
				}
				pageToken = newToken
			}
		}
	}
}

// getOrInitPageToken gets the page token from DB or initializes a new one.
func (p *DrivePoller) getOrInitPageToken(ctx context.Context) (string, error) {
	// Try to get from DB first
	if p.db != nil {
		cfg, err := p.db.GetConfigByID(p.configID)
		if err != nil {
			return "", err
		}
		if cfg != nil && cfg.ChangeToken != "" {
			return cfg.ChangeToken, nil
		}
	}

	// Initialize a new start page token
	return p.initStartPageToken(ctx)
}

// initStartPageToken gets the initial page token from Drive API.
func (p *DrivePoller) initStartPageToken(ctx context.Context) (string, error) {
	resp, err := p.service.Changes.GetStartPageToken().Context(ctx).Do()
	if err != nil {
		return "", err
	}

	token := resp.StartPageToken

	// Save to DB
	if p.db != nil {
		if err := p.db.UpdateChangeToken(p.configID, token); err != nil {
			return "", err
		}
	}

	return token, nil
}

// pollChanges fetches changes since last pageToken.
func (p *DrivePoller) pollChanges(ctx context.Context, pageToken string) ([]DriveChange, string, error) {
	var allChanges []DriveChange
	currentToken := pageToken

	for {
		req := p.service.Changes.List(currentToken).
			Context(ctx).
			PageSize(1000).
			IncludeRemoved(true).
			Fields("nextPageToken,newStartPageToken,changes(fileId,file(id,name,mimeType,parents,trashed),removed,time)")

		resp, err := req.Do()
		if err != nil {
			return nil, pageToken, err
		}

		// Process changes
		for _, change := range resp.Changes {
			// Filter to only files in our folder
			if change.File != nil && !p.isInFolderByParents(change.File.Parents) {
				continue
			}

			driveChange := p.convertChange(change)
			if driveChange != nil {
				allChanges = append(allChanges, *driveChange)
			}
		}

		// Check for more pages
		if resp.NextPageToken != "" {
			currentToken = resp.NextPageToken
			continue
		}

		// Use new start page token for next poll
		if resp.NewStartPageToken != "" {
			return allChanges, resp.NewStartPageToken, nil
		}

		return allChanges, currentToken, nil
	}
}

// convertChange converts a Drive change to our DriveChange type.
func (p *DrivePoller) convertChange(change *drive.Change) *DriveChange {
	if change == nil {
		return nil
	}

	driveChange := &DriveChange{
		FileID:    change.FileId,
		Removed:   change.Removed,
		Timestamp: time.Now(),
	}

	// Parse time if available
	if change.Time != "" {
		if t, err := time.Parse(time.RFC3339, change.Time); err == nil {
			driveChange.Timestamp = t
		}
	}

	if change.Removed {
		driveChange.Op = DriveOpDelete
		return driveChange
	}

	if change.File != nil {
		driveChange.FileName = change.File.Name
		driveChange.MimeType = change.File.MimeType

		if change.File.Trashed {
			driveChange.Op = DriveOpDelete
			driveChange.Removed = true
		} else {
			// We cannot easily distinguish create vs modify from changes.list
			// Default to modify for existing files
			driveChange.Op = DriveOpModify
		}
	}

	return driveChange
}

// isInFolderByParents checks if a file is within our synced folder by checking parents.
func (p *DrivePoller) isInFolderByParents(parents []string) bool {
	if parents == nil {
		return false
	}
	for _, parent := range parents {
		if parent == p.folderID {
			return true
		}
	}
	return false
}
