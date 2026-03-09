package sync

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// ServiceManager represents a process management system.
type ServiceManager string

const (
	ServiceManagerSystemd       ServiceManager = "systemd"
	ServiceManagerPM2           ServiceManager = "pm2"
	ServiceManagerLaunchd       ServiceManager = "launchd"
	ServiceManagerTaskScheduler ServiceManager = "schtasks"
)

// DetectServiceManager auto-detects the available service manager.
func DetectServiceManager() (ServiceManager, error) {
	switch runtime.GOOS {
	case "darwin":
		return ServiceManagerLaunchd, nil
	case "linux":
		// Check for systemd first (most common)
		if _, err := exec.LookPath("systemctl"); err == nil {
			return ServiceManagerSystemd, nil
		}
		// Fall back to pm2
		if _, err := exec.LookPath("pm2"); err == nil {
			return ServiceManagerPM2, nil
		}
		return "", fmt.Errorf("no supported service manager found (need systemctl or pm2)")
	case "windows":
		return ServiceManagerTaskScheduler, nil
	default:
		// Check pm2 as universal fallback
		if _, err := exec.LookPath("pm2"); err == nil {
			return ServiceManagerPM2, nil
		}
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// ServiceConfig holds the parameters for service installation.
type ServiceConfig struct {
	LocalPath  string
	Account    string
	Conflict   string
	Executable string // Path to wk binary
}

// validateServiceConfig checks that service config values are safe for template interpolation.
func validateServiceConfig(cfg ServiceConfig) error {
	for _, check := range []struct {
		name, val string
	}{
		{"executable", cfg.Executable},
		{"local-path", cfg.LocalPath},
		{"account", cfg.Account},
		{"conflict", cfg.Conflict},
	} {
		if strings.ContainsAny(check.val, "\n\r\x00`${}") {
			return fmt.Errorf("%s contains invalid characters", check.name)
		}
	}
	if cfg.Executable == "" || cfg.LocalPath == "" || cfg.Account == "" {
		return fmt.Errorf("executable, local-path, and account are required")
	}
	return nil
}

// InstallService installs the sync daemon as a managed service.
func InstallService(cfg ServiceConfig, manager ServiceManager) error {
	if err := validateServiceConfig(cfg); err != nil {
		return fmt.Errorf("invalid service config: %w", err)
	}
	switch manager {
	case ServiceManagerSystemd:
		return installSystemd(cfg)
	case ServiceManagerPM2:
		return installPM2(cfg)
	case ServiceManagerLaunchd:
		return installLaunchd(cfg)
	case ServiceManagerTaskScheduler:
		return installTaskScheduler(cfg)
	default:
		return fmt.Errorf("unsupported service manager: %s", manager)
	}
}

// UninstallService removes the sync service.
func UninstallService(manager ServiceManager) error {
	switch manager {
	case ServiceManagerSystemd:
		return uninstallSystemd()
	case ServiceManagerPM2:
		return uninstallPM2()
	case ServiceManagerLaunchd:
		return uninstallLaunchd()
	case ServiceManagerTaskScheduler:
		return uninstallTaskScheduler()
	default:
		return fmt.Errorf("unsupported service manager: %s", manager)
	}
}

// ServiceStatus checks the status of the installed service.
// Returns (running, statusText, error).
func ServiceStatus(manager ServiceManager) (bool, string, error) {
	switch manager {
	case ServiceManagerSystemd:
		return statusSystemd()
	case ServiceManagerPM2:
		return statusPM2()
	case ServiceManagerLaunchd:
		return statusLaunchd()
	case ServiceManagerTaskScheduler:
		return statusTaskScheduler()
	default:
		return false, "", fmt.Errorf("unsupported service manager: %s", manager)
	}
}

// --- systemd backend ---

var systemdUnitTemplate = template.Must(template.New("systemd").Parse(`[Unit]
Description=Workit Sync Daemon
After=network-online.target

[Service]
Type=simple
ExecStart="{{.Executable}}" sync start "{{.LocalPath}}" --account "{{.Account}}" --conflict "{{.Conflict}}"
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`))

func systemdUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "systemd", "user", "wk-sync.service"), nil
}

func installSystemd(cfg ServiceConfig) error {
	unitPath, err := systemdUnitPath()
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}

	f, err := os.Create(unitPath)
	if err != nil {
		return fmt.Errorf("create unit file: %w", err)
	}
	defer f.Close()

	if err := systemdUnitTemplate.Execute(f, cfg); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	// Reload unit files and enable+start the service.
	reloadCmd := exec.Command("systemctl", "--user", "daemon-reload")
	if out, err := reloadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %s: %w", strings.TrimSpace(string(out)), err)
	}

	cmd := exec.Command("systemctl", "--user", "enable", "--now", "wk-sync")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

func uninstallSystemd() error {
	// Disable and stop the service
	cmd := exec.Command("systemctl", "--user", "disable", "--now", "wk-sync")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl disable: %s: %w", strings.TrimSpace(string(out)), err)
	}

	unitPath, err := systemdUnitPath()
	if err != nil {
		return err
	}

	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file: %w", err)
	}

	return nil
}

func statusSystemd() (bool, string, error) {
	cmd := exec.Command("systemctl", "--user", "is-active", "wk-sync")
	out, err := cmd.CombinedOutput()
	status := strings.TrimSpace(string(out))

	if err != nil {
		// is-active returns non-zero for inactive/failed
		return false, status, nil
	}

	return status == "active", status, nil
}

// --- pm2 backend ---

func installPM2(cfg ServiceConfig) error {
	// pm2 start <executable> --name wk-sync -- sync start <path> --account <email> --conflict <strategy>
	args := []string{
		"start", cfg.Executable,
		"--name", "wk-sync",
		"--",
		"sync", "start", cfg.LocalPath,
		"--account", cfg.Account,
		"--conflict", cfg.Conflict,
	}

	cmd := exec.Command("pm2", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pm2 start: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Save the pm2 process list
	saveCmd := exec.Command("pm2", "save")
	if out, err := saveCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pm2 save: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

func uninstallPM2() error {
	cmd := exec.Command("pm2", "delete", "wk-sync")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pm2 delete: %s: %w", strings.TrimSpace(string(out)), err)
	}

	saveCmd := exec.Command("pm2", "save")
	if out, err := saveCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pm2 save: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

func statusPM2() (bool, string, error) {
	cmd := exec.Command("pm2", "show", "wk-sync")
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		return false, output, nil
	}

	// pm2 show returns 0 even if stopped; check for "online" in output
	running := strings.Contains(output, "online")
	return running, output, nil
}

// --- launchd backend ---

var launchdPlistTemplate = template.Must(template.New("launchd").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.workit.sync</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.Executable}}</string>
		<string>sync</string>
		<string>start</string>
		<string>{{.LocalPath}}</string>
		<string>--account</string>
		<string>{{.Account}}</string>
		<string>--conflict</string>
		<string>{{.Conflict}}</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>
</dict>
</plist>
`))

func launchdPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.workit.sync.plist"), nil
}

func installLaunchd(cfg ServiceConfig) error {
	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("create plist file: %w", err)
	}
	defer f.Close()

	if err := launchdPlistTemplate.Execute(f, cfg); err != nil {
		return fmt.Errorf("write plist file: %w", err)
	}

	cmd := exec.Command("launchctl", "load", plistPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

func uninstallLaunchd() error {
	plistPath, err := launchdPlistPath()
	if err != nil {
		return err
	}

	cmd := exec.Command("launchctl", "unload", plistPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl unload: %s: %w", strings.TrimSpace(string(out)), err)
	}

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist file: %w", err)
	}

	return nil
}

func statusLaunchd() (bool, string, error) {
	cmd := exec.Command("launchctl", "list", "com.workit.sync")
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		return false, output, nil
	}

	// If launchctl list succeeds, the service is loaded (running)
	return true, output, nil
}

// --- task scheduler backend ---

const schtasksTaskName = "WorkitSync"

func installTaskScheduler(cfg ServiceConfig) error {
	// Build the command that Task Scheduler will run.
	tr := fmt.Sprintf(`"%s" sync start "%s" --account "%s" --conflict "%s"`,
		cfg.Executable, cfg.LocalPath, cfg.Account, cfg.Conflict)

	// Create the scheduled task to run at logon.
	createArgs := []string{
		"/Create",
		"/TN", schtasksTaskName,
		"/TR", tr,
		"/SC", "ONLOGON",
		"/F",
	}

	cmd := exec.Command("schtasks", createArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks create: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Start the task immediately.
	runCmd := exec.Command("schtasks", "/Run", "/TN", schtasksTaskName)
	if out, err := runCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks run: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

func uninstallTaskScheduler() error {
	cmd := exec.Command("schtasks", "/Delete", "/TN", schtasksTaskName, "/F")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks delete: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// schtasksXML mirrors the XML structure returned by schtasks /Query /FO XML.
type schtasksXML struct {
	XMLName xml.Name `xml:"ScheduledTasks"`
	Tasks   []struct {
		State string `xml:"State"`
	} `xml:"Task"`
}

func statusTaskScheduler() (bool, string, error) {
	cmd := exec.Command("schtasks", "/Query", "/TN", schtasksTaskName, "/FO", "XML")
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		return false, output, nil
	}

	// Parse XML (locale-independent) to determine task state.
	var result schtasksXML
	if xmlErr := xml.Unmarshal(out, &result); xmlErr != nil {
		// Fall back to raw output if XML parsing fails.
		return false, output, nil
	}

	running := len(result.Tasks) > 0 && result.Tasks[0].State == "Running"

	return running, output, nil
}
