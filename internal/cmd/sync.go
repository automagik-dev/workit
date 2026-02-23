package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"

	"github.com/steipete/gogcli/internal/googleapi"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/sync"
	"github.com/steipete/gogcli/internal/ui"
)

// SyncCmd is the top-level command for Drive sync operations.
type SyncCmd struct {
	Init   SyncInitCmd   `cmd:"" help:"Initialize sync between a local folder and Drive folder"`
	List   SyncListCmd   `cmd:"" help:"List all sync configurations"`
	Remove SyncRemoveCmd `cmd:"" help:"Remove a sync configuration"`
	Status SyncStatusCmd `cmd:"" help:"Show sync status for all configurations"`
	Start  SyncStartCmd  `cmd:"" help:"Start sync daemon (placeholder)"`
	Stop   SyncStopCmd   `cmd:"" help:"Stop sync daemon (placeholder)"`
}

// SyncInitCmd initializes a new sync configuration.
type SyncInitCmd struct {
	LocalPath   string `arg:"" name:"local-path" help:"Local directory path to sync"`
	DriveFolder string `name:"drive-folder" required:"" help:"Drive folder name or ID"`
	DriveID     string `name:"drive-id" help:"Shared drive ID (optional)"`
}

func (c *SyncInitCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	localPath := strings.TrimSpace(c.LocalPath)
	if localPath == "" {
		return usage("empty local-path")
	}

	driveFolder := strings.TrimSpace(c.DriveFolder)
	if driveFolder == "" {
		return usage("empty --drive-folder")
	}

	// Resolve folder name/URL to a Drive folder ID.
	driveFolder = normalizeGoogleID(driveFolder)
	driveID := strings.TrimSpace(c.DriveID)
	if strings.ContainsAny(driveFolder, " \t\r\n") || len(driveFolder) < 16 {
		// Looks like a human-readable name — resolve via Drive API.
		driveSvc, err := getDriveService(ctx, flags)
		if err != nil {
			return fmt.Errorf("resolve Drive folder name: %w", err)
		}
		resolved, err := resolveDriveFolderID(ctx, driveSvc, driveFolder, driveID)
		if err != nil {
			return err
		}
		driveFolder = resolved
	}

	db, err := sync.OpenDB()
	if err != nil {
		return fmt.Errorf("open sync database: %w", err)
	}
	defer db.Close()

	// Check if config already exists
	existing, err := db.GetConfig(localPath)
	if err != nil {
		return fmt.Errorf("check existing config: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("sync config already exists for path: %s", existing.LocalPath)
	}

	cfg, err := db.CreateConfig(localPath, driveFolder, driveID)
	if err != nil {
		return fmt.Errorf("create sync config: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"config":  cfg,
			"created": true,
		})
	}

	u.Out().Printf("created\ttrue")
	u.Out().Printf("id\t%d", cfg.ID)
	u.Out().Printf("local_path\t%s", cfg.LocalPath)
	u.Out().Printf("drive_folder\t%s", cfg.DriveFolderID)
	if cfg.DriveID != "" {
		u.Out().Printf("drive_id\t%s", cfg.DriveID)
	}
	return nil
}

// SyncListCmd lists all sync configurations.
type SyncListCmd struct{}

func (c *SyncListCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	db, err := sync.OpenDB()
	if err != nil {
		return fmt.Errorf("open sync database: %w", err)
	}
	defer db.Close()

	configs, err := db.ListConfigs()
	if err != nil {
		return fmt.Errorf("list configs: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"configs": configs,
			"count":   len(configs),
		})
	}

	if len(configs) == 0 {
		u.Err().Println("No sync configurations")
		return nil
	}

	w, flush := tableWriter(ctx)
	defer flush()
	fmt.Fprintln(w, "ID\tLOCAL PATH\tDRIVE FOLDER\tCREATED\tLAST SYNC")
	for _, cfg := range configs {
		lastSync := "-"
		if !cfg.LastSyncAt.IsZero() {
			lastSync = cfg.LastSyncAt.Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			cfg.ID,
			cfg.LocalPath,
			cfg.DriveFolderID,
			cfg.CreatedAt.Format(time.RFC3339),
			lastSync,
		)
	}
	return nil
}

// SyncRemoveCmd removes a sync configuration.
type SyncRemoveCmd struct {
	LocalPath string `arg:"" name:"local-path" help:"Local directory path of sync to remove"`
}

func (c *SyncRemoveCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	localPath := strings.TrimSpace(c.LocalPath)
	if localPath == "" {
		return usage("empty local-path")
	}

	db, err := sync.OpenDB()
	if err != nil {
		return fmt.Errorf("open sync database: %w", err)
	}
	defer db.Close()

	// Get the config first for confirmation
	cfg, err := db.GetConfig(localPath)
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("sync config not found: %s", localPath)
	}

	if confirmErr := confirmDestructive(ctx, flags, fmt.Sprintf("remove sync config for %s", cfg.LocalPath)); confirmErr != nil {
		return confirmErr
	}

	if err := db.RemoveConfig(localPath); err != nil {
		return fmt.Errorf("remove config: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"removed":    true,
			"local_path": cfg.LocalPath,
		})
	}

	u.Out().Printf("removed\ttrue")
	u.Out().Printf("local_path\t%s", cfg.LocalPath)
	return nil
}

// SyncStatusCmd shows the sync status for all configurations.
type SyncStatusCmd struct{}

func (c *SyncStatusCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	db, err := sync.OpenDB()
	if err != nil {
		return fmt.Errorf("open sync database: %w", err)
	}
	defer db.Close()

	statuses, err := db.ListStatuses()
	if err != nil {
		return fmt.Errorf("list statuses: %w", err)
	}

	// Get daemon status
	daemonStatus, _ := sync.GetDaemonStatus()

	if outfmt.IsJSON(ctx) {
		result := map[string]any{
			"statuses": statuses,
			"count":    len(statuses),
			"running":  false,
		}
		if daemonStatus != nil {
			result["running"] = daemonStatus.Running
			if daemonStatus.PID > 0 {
				result["pid"] = daemonStatus.PID
			}
		}

		return outfmt.WriteJSON(ctx, os.Stdout, result)
	}

	// Print daemon status
	if daemonStatus != nil && daemonStatus.Running {
		u.Err().Printf("Daemon running (PID %d)", daemonStatus.PID)
	} else {
		u.Err().Println("Daemon not running")
	}

	u.Err().Println("")

	if len(statuses) == 0 {
		u.Err().Println("No sync configurations")

		return nil
	}

	w, flush := tableWriter(ctx)
	defer flush()
	fmt.Fprintln(w, "ID\tLOCAL PATH\tTOTAL\tSYNCED\tPENDING\tCONFLICT\tERROR\tLAST SYNC")

	for _, s := range statuses {
		lastSync := "-"
		if !s.Config.LastSyncAt.IsZero() {
			lastSync = s.Config.LastSyncAt.Format(time.RFC3339)
		}

		fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%d\t%d\t%d\t%s\n",
			s.Config.ID,
			s.Config.LocalPath,
			s.TotalItems,
			s.SyncedItems,
			s.PendingItems,
			s.ConflictItems,
			s.ErrorItems,
			lastSync,
		)
	}

	return nil
}

// SyncStartCmd starts the sync daemon (placeholder).
type SyncStartCmd struct {
	LocalPath      string `arg:"" name:"local-path" help:"Local directory path to sync"`
	Daemon         bool   `name:"daemon" short:"d" help:"Run as background daemon"`
	InternalDaemon bool   `name:"internal-daemon" hidden:""`
	Conflict       string `name:"conflict" help:"Conflict resolution strategy: rename (default), local-wins, remote-wins" default:"rename" enum:"rename,local-wins,remote-wins"`
}

func (c *SyncStartCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	localPath := strings.TrimSpace(c.LocalPath)
	if localPath == "" {
		return usage("empty local-path")
	}

	// Handle daemon mode
	if c.Daemon && !c.InternalDaemon {
		// Start daemon in background
		account := flags.Account
		if account == "" {
			return fmt.Errorf("--account flag is required for daemon mode")
		}

		pid, err := sync.StartDaemon(localPath, account, c.Conflict)
		if err != nil {
			return fmt.Errorf("start daemon: %w", err)
		}

		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"started": true,
				"pid":     pid,
			})
		}

		u.Out().Printf("started\ttrue")
		u.Out().Printf("pid\t%d", pid)

		return nil
	}

	// If running as internal daemon, write PID file
	if c.InternalDaemon {
		if err := sync.WritePIDFile(); err != nil {
			return fmt.Errorf("write PID file: %w", err)
		}
		defer func() { _ = sync.RemovePIDFile() }()
	}

	db, err := sync.OpenDB()
	if err != nil {
		return fmt.Errorf("open sync database: %w", err)
	}
	defer db.Close()

	cfg, err := db.GetConfig(localPath)
	if err != nil {
		return fmt.Errorf("get sync config: %w", err)
	}

	if cfg == nil {
		return fmt.Errorf("sync config not found: %s (use 'gog sync init' first)", localPath)
	}

	// Get authenticated Drive service
	// For now, require the user to specify account via --account flag
	driveService, err := getDriveService(ctx, flags)
	if err != nil {
		return fmt.Errorf("get Drive service: %w", err)
	}

	engine, err := sync.NewEngine(sync.EngineOptions{
		DB:           db,
		Config:       cfg,
		DriveService: driveService,
	})
	if err != nil {
		return fmt.Errorf("create sync engine: %w", err)
	}

	u.Err().Printf("Starting sync for %s -> %s", cfg.LocalPath, cfg.DriveFolderID)
	u.Err().Println("Press Ctrl+C to stop")

	if err := engine.Start(ctx); err != nil && ctx.Err() == nil {
		return fmt.Errorf("sync engine error: %w", err)
	}

	u.Err().Println("Sync stopped")

	return nil
}

// SyncStopCmd stops the sync daemon.
type SyncStopCmd struct{}

func (c *SyncStopCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)

	status, err := sync.GetDaemonStatus()
	if err != nil {
		return fmt.Errorf("get daemon status: %w", err)
	}

	if !status.Running {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"stopped": false,
				"error":   "daemon not running",
			})
		}

		u.Err().Println("daemon is not running")

		return nil
	}

	pid := status.PID

	if err := sync.StopDaemon(); err != nil {
		return fmt.Errorf("stop daemon: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"stopped": true,
			"pid":     pid,
		})
	}

	u.Out().Printf("stopped\ttrue")
	u.Out().Printf("pid\t%d", pid)

	return nil
}

// getDriveService creates an authenticated Drive service.
func getDriveService(ctx context.Context, flags *RootFlags) (*drive.Service, error) {
	account := flags.Account
	if account == "" {
		return nil, fmt.Errorf("--account flag is required for sync operations")
	}

	return googleapi.NewDrive(ctx, account)
}
