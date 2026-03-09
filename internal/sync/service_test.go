package sync

import (
	"bytes"
	"runtime"
	"testing"
)

func TestDetectServiceManager(t *testing.T) {
	manager, err := DetectServiceManager()
	// On any CI/dev machine, at least one manager should be available
	// or we get a clear error. We just verify the function doesn't panic.
	if err != nil {
		t.Logf("DetectServiceManager returned error (expected on some platforms): %v", err)
		return
	}

	switch manager {
	case ServiceManagerSystemd, ServiceManagerPM2, ServiceManagerLaunchd, ServiceManagerTaskScheduler:
		// valid
	default:
		t.Errorf("unexpected service manager: %q", manager)
	}

	// On darwin, should always return launchd
	if runtime.GOOS == "darwin" && manager != ServiceManagerLaunchd {
		t.Errorf("on darwin expected launchd, got %q", manager)
	}

	if runtime.GOOS == "windows" && manager != ServiceManagerTaskScheduler {
		t.Errorf("on windows expected schtasks, got %q", manager)
	}
}

func TestSystemdUnitTemplate(t *testing.T) {
	cfg := ServiceConfig{
		LocalPath:  "/home/user/sync",
		Account:    "user@example.com",
		Conflict:   "rename",
		Executable: "/usr/local/bin/wk",
	}

	var buf bytes.Buffer
	if err := systemdUnitTemplate.Execute(&buf, cfg); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}

	output := buf.String()

	// Verify key parts of the unit file
	expected := []string{
		"[Unit]",
		"Description=Workit Sync Daemon",
		"After=network-online.target",
		"[Service]",
		"Type=simple",
		`ExecStart="/usr/local/bin/wk" sync start "/home/user/sync" --account "user@example.com" --conflict "rename"`,
		"Restart=on-failure",
		"RestartSec=5",
		"[Install]",
		"WantedBy=default.target",
	}

	for _, s := range expected {
		if !bytes.Contains([]byte(output), []byte(s)) {
			t.Errorf("systemd unit missing %q\nGot:\n%s", s, output)
		}
	}
}

func TestLaunchdPlistTemplate(t *testing.T) {
	cfg := ServiceConfig{
		LocalPath:  "/Users/dev/sync",
		Account:    "dev@example.com",
		Conflict:   "local-wins",
		Executable: "/usr/local/bin/wk",
	}

	var buf bytes.Buffer
	if err := launchdPlistTemplate.Execute(&buf, cfg); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}

	output := buf.String()

	expected := []string{
		"<string>com.workit.sync</string>",
		"<string>/usr/local/bin/wk</string>",
		"<string>sync</string>",
		"<string>start</string>",
		"<string>/Users/dev/sync</string>",
		"<string>--account</string>",
		"<string>dev@example.com</string>",
		"<string>--conflict</string>",
		"<string>local-wins</string>",
		"<key>RunAtLoad</key>",
		"<true/>",
		"<key>KeepAlive</key>",
	}

	for _, s := range expected {
		if !bytes.Contains([]byte(output), []byte(s)) {
			t.Errorf("launchd plist missing %q\nGot:\n%s", s, output)
		}
	}
}

func TestInstallServiceUnsupportedManager(t *testing.T) {
	cfg := ServiceConfig{
		LocalPath:  "/tmp/test",
		Account:    "test@example.com",
		Conflict:   "rename",
		Executable: "/usr/local/bin/wk",
	}

	err := InstallService(cfg, ServiceManager("bogus"))
	if err == nil {
		t.Fatal("expected error for unsupported manager")
	}
	if got := err.Error(); got != "unsupported service manager: bogus" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestUninstallServiceUnsupportedManager(t *testing.T) {
	err := UninstallService(ServiceManager("bogus"))
	if err == nil {
		t.Fatal("expected error for unsupported manager")
	}
}

func TestServiceStatusUnsupportedManager(t *testing.T) {
	_, _, err := ServiceStatus(ServiceManager("bogus"))
	if err == nil {
		t.Fatal("expected error for unsupported manager")
	}
}

func TestResolveServiceManager(t *testing.T) {
	// Explicit manager should be returned as-is
	m, err := resolveServiceManager("systemd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != ServiceManagerSystemd {
		t.Errorf("expected systemd, got %q", m)
	}

	m, err = resolveServiceManager("pm2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != ServiceManagerPM2 {
		t.Errorf("expected pm2, got %q", m)
	}

	m, err = resolveServiceManager("launchd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != ServiceManagerLaunchd {
		t.Errorf("expected launchd, got %q", m)
	}

	m, err = resolveServiceManager("schtasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != ServiceManagerTaskScheduler {
		t.Errorf("expected schtasks, got %q", m)
	}
}

// resolveServiceManager is a helper that mirrors the cmd layer's logic.
// We test it here since it's a pure function.
func resolveServiceManager(name string) (ServiceManager, error) {
	if name != "" {
		return ServiceManager(name), nil
	}
	return DetectServiceManager()
}
