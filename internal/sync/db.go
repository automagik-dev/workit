package sync

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/namastexlabs/gog-cli/internal/config"
)

// DB provides sync state persistence using SQLite.
type DB struct {
	db *sql.DB
}

// DBPath returns the path to the sync database file.
func DBPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sync.db"), nil
}

// OpenDB opens (or creates) the sync database.
func OpenDB() (*DB, error) {
	dbPath, err := DBPath()
	if err != nil {
		return nil, fmt.Errorf("get db path: %w", err)
	}

	// Ensure the directory exists
	if _, err := config.EnsureDir(); err != nil {
		return nil, fmt.Errorf("ensure config dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	syncDB := &DB{db: db}
	if err := syncDB.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return syncDB, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// migrate creates the database schema if it doesn't exist.
func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sync_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		local_path TEXT NOT NULL UNIQUE,
		drive_folder_id TEXT NOT NULL,
		drive_id TEXT DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_sync_at DATETIME,
		change_token TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_sync_configs_local_path ON sync_configs(local_path);

	CREATE TABLE IF NOT EXISTS sync_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		config_id INTEGER NOT NULL,
		local_path TEXT NOT NULL,
		drive_id TEXT NOT NULL DEFAULT '',
		local_md5 TEXT DEFAULT '',
		remote_md5 TEXT DEFAULT '',
		local_mtime DATETIME,
		remote_mtime DATETIME,
		sync_state TEXT NOT NULL DEFAULT 'pending_upload',
		FOREIGN KEY (config_id) REFERENCES sync_configs(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sync_items_config_id ON sync_items(config_id);
	CREATE INDEX IF NOT EXISTS idx_sync_items_local_path ON sync_items(local_path);
	CREATE INDEX IF NOT EXISTS idx_sync_items_sync_state ON sync_items(sync_state);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_sync_items_config_path ON sync_items(config_id, local_path);

	CREATE TABLE IF NOT EXISTS sync_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		config_id INTEGER NOT NULL,
		action TEXT NOT NULL,
		path TEXT NOT NULL,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		details TEXT DEFAULT '{}',
		FOREIGN KEY (config_id) REFERENCES sync_configs(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sync_log_config_id ON sync_log(config_id);
	CREATE INDEX IF NOT EXISTS idx_sync_log_timestamp ON sync_log(timestamp);
	`

	_, err := d.db.Exec(schema)
	return err
}

// CreateConfig creates a new sync configuration.
func (d *DB) CreateConfig(localPath, driveFolderID, driveID string) (*SyncConfig, error) {
	// Expand and clean the path
	expandedPath, err := config.ExpandPath(localPath)
	if err != nil {
		return nil, fmt.Errorf("expand path: %w", err)
	}
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("absolute path: %w", err)
	}

	// Check if the local path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the directory
			if mkErr := os.MkdirAll(absPath, 0o755); mkErr != nil {
				return nil, fmt.Errorf("create directory: %w", mkErr)
			}
		} else {
			return nil, fmt.Errorf("stat path: %w", err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absPath)
	}

	now := time.Now()
	result, err := d.db.Exec(
		`INSERT INTO sync_configs (local_path, drive_folder_id, drive_id, created_at)
		 VALUES (?, ?, ?, ?)`,
		absPath, driveFolderID, driveID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert config: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return &SyncConfig{
		ID:            id,
		LocalPath:     absPath,
		DriveFolderID: driveFolderID,
		DriveID:       driveID,
		CreatedAt:     now,
	}, nil
}

// GetConfig retrieves a sync configuration by local path.
func (d *DB) GetConfig(localPath string) (*SyncConfig, error) {
	expandedPath, err := config.ExpandPath(localPath)
	if err != nil {
		return nil, fmt.Errorf("expand path: %w", err)
	}
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("absolute path: %w", err)
	}

	var cfg SyncConfig
	var lastSyncAt sql.NullTime
	err = d.db.QueryRow(
		`SELECT id, local_path, drive_folder_id, drive_id, created_at, last_sync_at, change_token
		 FROM sync_configs WHERE local_path = ?`,
		absPath,
	).Scan(&cfg.ID, &cfg.LocalPath, &cfg.DriveFolderID, &cfg.DriveID,
		&cfg.CreatedAt, &lastSyncAt, &cfg.ChangeToken)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query config: %w", err)
	}
	if lastSyncAt.Valid {
		cfg.LastSyncAt = lastSyncAt.Time
	}
	return &cfg, nil
}

// GetConfigByID retrieves a sync configuration by ID.
func (d *DB) GetConfigByID(id int64) (*SyncConfig, error) {
	var cfg SyncConfig
	var lastSyncAt sql.NullTime
	err := d.db.QueryRow(
		`SELECT id, local_path, drive_folder_id, drive_id, created_at, last_sync_at, change_token
		 FROM sync_configs WHERE id = ?`,
		id,
	).Scan(&cfg.ID, &cfg.LocalPath, &cfg.DriveFolderID, &cfg.DriveID,
		&cfg.CreatedAt, &lastSyncAt, &cfg.ChangeToken)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query config: %w", err)
	}
	if lastSyncAt.Valid {
		cfg.LastSyncAt = lastSyncAt.Time
	}
	return &cfg, nil
}

// ListConfigs returns all sync configurations.
func (d *DB) ListConfigs() ([]SyncConfig, error) {
	rows, err := d.db.Query(
		`SELECT id, local_path, drive_folder_id, drive_id, created_at, last_sync_at, change_token
		 FROM sync_configs ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query configs: %w", err)
	}
	defer rows.Close()

	var configs []SyncConfig
	for rows.Next() {
		var cfg SyncConfig
		var lastSyncAt sql.NullTime
		if err := rows.Scan(&cfg.ID, &cfg.LocalPath, &cfg.DriveFolderID, &cfg.DriveID,
			&cfg.CreatedAt, &lastSyncAt, &cfg.ChangeToken); err != nil {
			return nil, fmt.Errorf("scan config: %w", err)
		}
		if lastSyncAt.Valid {
			cfg.LastSyncAt = lastSyncAt.Time
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

// RemoveConfig removes a sync configuration by local path.
func (d *DB) RemoveConfig(localPath string) error {
	expandedPath, err := config.ExpandPath(localPath)
	if err != nil {
		return fmt.Errorf("expand path: %w", err)
	}
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}

	result, err := d.db.Exec(`DELETE FROM sync_configs WHERE local_path = ?`, absPath)
	if err != nil {
		return fmt.Errorf("delete config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("sync config not found: %s", absPath)
	}
	return nil
}

// UpdateChangeToken updates the change token for a config.
func (d *DB) UpdateChangeToken(configID int64, token string) error {
	_, err := d.db.Exec(
		`UPDATE sync_configs SET change_token = ?, last_sync_at = ? WHERE id = ?`,
		token, time.Now(), configID,
	)
	return err
}

// GetStatus returns the sync status for a configuration.
func (d *DB) GetStatus(configID int64) (*SyncStatus, error) {
	cfg, err := d.GetConfigByID(configID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("config not found: %d", configID)
	}

	status := &SyncStatus{Config: *cfg}

	// Count items by state
	rows, err := d.db.Query(
		`SELECT sync_state, COUNT(*) FROM sync_items WHERE config_id = ? GROUP BY sync_state`,
		configID,
	)
	if err != nil {
		return nil, fmt.Errorf("query item counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		status.TotalItems += count
		switch SyncState(state) {
		case StateSynced:
			status.SyncedItems = count
		case StateConflict:
			status.ConflictItems = count
		case StateError:
			status.ErrorItems = count
		case StatePendingUpload, StatePendingDownload:
			status.PendingItems += count
		}
	}

	return status, rows.Err()
}

// ListStatuses returns the sync status for all configurations.
func (d *DB) ListStatuses() ([]SyncStatus, error) {
	configs, err := d.ListConfigs()
	if err != nil {
		return nil, err
	}

	statuses := make([]SyncStatus, 0, len(configs))
	for _, cfg := range configs {
		status, err := d.GetStatus(cfg.ID)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, *status)
	}
	return statuses, nil
}

// AddLogEntry adds an entry to the sync log.
func (d *DB) AddLogEntry(configID int64, action, path string, details map[string]any) error {
	detailsJSON := "{}"
	if details != nil {
		b, err := json.Marshal(details)
		if err == nil {
			detailsJSON = string(b)
		}
	}

	_, err := d.db.Exec(
		`INSERT INTO sync_log (config_id, action, path, timestamp, details)
		 VALUES (?, ?, ?, ?, ?)`,
		configID, action, path, time.Now(), detailsJSON,
	)
	return err
}

// GetRecentLogs returns recent log entries for a config.
func (d *DB) GetRecentLogs(configID int64, limit int) ([]SyncLogEntry, error) {
	rows, err := d.db.Query(
		`SELECT id, config_id, action, path, timestamp, details
		 FROM sync_log WHERE config_id = ? ORDER BY timestamp DESC LIMIT ?`,
		configID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()

	var entries []SyncLogEntry
	for rows.Next() {
		var entry SyncLogEntry
		if err := rows.Scan(&entry.ID, &entry.ConfigID, &entry.Action,
			&entry.Path, &entry.Timestamp, &entry.Details); err != nil {
			return nil, fmt.Errorf("scan log: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// CreateSyncItem creates a new sync item.
func (d *DB) CreateSyncItem(configID int64, localPath, driveID, localMD5, remoteMD5 string, localMtime, remoteMtime time.Time) error {
	_, err := d.db.Exec(
		`INSERT INTO sync_items (config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		configID, localPath, driveID, localMD5, remoteMD5, localMtime, remoteMtime, StatePendingUpload,
	)

	return err
}

// GetSyncItem retrieves a sync item by local path.
func (d *DB) GetSyncItem(configID int64, localPath string) (*SyncItem, error) {
	var item SyncItem
	var localMtime, remoteMtime sql.NullTime

	err := d.db.QueryRow(
		`SELECT id, config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state
		 FROM sync_items WHERE config_id = ? AND local_path = ?`,
		configID, localPath,
	).Scan(&item.ID, &item.ConfigID, &item.LocalPath, &item.DriveID,
		&item.LocalMD5, &item.RemoteMD5, &localMtime, &remoteMtime, &item.SyncState)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query sync item: %w", err)
	}

	if localMtime.Valid {
		item.LocalMtime = localMtime.Time
	}

	if remoteMtime.Valid {
		item.RemoteMtime = remoteMtime.Time
	}

	return &item, nil
}

// GetSyncItemByDriveID retrieves a sync item by Drive file ID.
func (d *DB) GetSyncItemByDriveID(configID int64, driveID string) (*SyncItem, error) {
	var item SyncItem
	var localMtime, remoteMtime sql.NullTime

	err := d.db.QueryRow(
		`SELECT id, config_id, local_path, drive_id, local_md5, remote_md5, local_mtime, remote_mtime, sync_state
		 FROM sync_items WHERE config_id = ? AND drive_id = ?`,
		configID, driveID,
	).Scan(&item.ID, &item.ConfigID, &item.LocalPath, &item.DriveID,
		&item.LocalMD5, &item.RemoteMD5, &localMtime, &remoteMtime, &item.SyncState)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query sync item by drive id: %w", err)
	}

	if localMtime.Valid {
		item.LocalMtime = localMtime.Time
	}

	if remoteMtime.Valid {
		item.RemoteMtime = remoteMtime.Time
	}

	return &item, nil
}

// UpdateSyncItem updates a sync item.
func (d *DB) UpdateSyncItem(itemID int64, driveID, localMD5, remoteMD5 string, state SyncState) error {
	_, err := d.db.Exec(
		`UPDATE sync_items SET drive_id = ?, local_md5 = ?, remote_md5 = ?, sync_state = ?
		 WHERE id = ?`,
		driveID, localMD5, remoteMD5, state, itemID,
	)

	return err
}

// RemoveSyncItem removes a sync item by local path.
func (d *DB) RemoveSyncItem(configID int64, localPath string) error {
	_, err := d.db.Exec(
		`DELETE FROM sync_items WHERE config_id = ? AND local_path = ?`,
		configID, localPath,
	)

	return err
}
