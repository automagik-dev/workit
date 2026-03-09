package sync

import (
	"bytes"
	"encoding/xml"
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

func TestInstallService_ValidationFailure(t *testing.T) {
	t.Parallel()

	// Empty required fields should fail validation before reaching any backend.
	cfg := ServiceConfig{
		Executable: "",
		LocalPath:  "/tmp",
		Account:    "user@example.com",
	}

	err := InstallService(cfg, ServiceManagerSystemd)
	if err == nil {
		t.Error("expected validation error")
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

func TestValidateServiceConfig_Valid(t *testing.T) {
	t.Parallel()

	cfg := ServiceConfig{
		Executable: "/usr/local/bin/wk",
		LocalPath:  "/home/user/sync",
		Account:    "user@example.com",
		Conflict:   "rename",
	}

	if err := validateServiceConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateServiceConfig_RejectsEmpty(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		cfg  ServiceConfig
	}{
		{"empty executable", ServiceConfig{LocalPath: "/tmp", Account: "a@b.com"}},
		{"empty path", ServiceConfig{Executable: "/bin/wk", Account: "a@b.com"}},
		{"empty account", ServiceConfig{Executable: "/bin/wk", LocalPath: "/tmp"}},
	} {
		if err := validateServiceConfig(tc.cfg); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}
}

func TestValidateServiceConfig_RejectsBadChars(t *testing.T) {
	t.Parallel()

	base := ServiceConfig{
		Executable: "/usr/bin/wk",
		LocalPath:  "/tmp/path",
		Account:    "user@example.com",
		Conflict:   "rename",
	}

	for _, bad := range []string{"\n", "\r", "\x00", "`", "$", "{", "}"} {
		cfg := base
		cfg.LocalPath = "/tmp/path" + bad

		if err := validateServiceConfig(cfg); err == nil {
			t.Errorf("expected error for %q in path", bad)
		}
	}
}

func TestSchtasksXML_Running(t *testing.T) {
	t.Parallel()

	data := `<?xml version="1.0"?>
<ScheduledTasks><Task><State>Running</State></Task></ScheduledTasks>`

	var result schtasksXML
	if err := xml.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}

	if result.Tasks[0].State != schtasksStateRunning {
		t.Errorf("expected Running, got %q", result.Tasks[0].State)
	}
}

func TestSchtasksXML_NotRunning(t *testing.T) {
	t.Parallel()

	data := `<?xml version="1.0"?>
<ScheduledTasks><Task><State>Ready</State></Task></ScheduledTasks>`

	var result schtasksXML
	if err := xml.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	running := len(result.Tasks) > 0 && result.Tasks[0].State == schtasksStateRunning
	if running {
		t.Error("expected not running for Ready state")
	}
}

func TestSchtasksXML_Empty(t *testing.T) {
	t.Parallel()

	data := `<?xml version="1.0"?><ScheduledTasks></ScheduledTasks>`

	var result schtasksXML
	if err := xml.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	running := len(result.Tasks) > 0 && result.Tasks[0].State == schtasksStateRunning
	if running {
		t.Error("expected not running for empty tasks")
	}
}
