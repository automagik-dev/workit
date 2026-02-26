package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/secrets"
)

func TestNeedsFileBackendSetup(t *testing.T) {
	// Save and restore the original resolveBackendInfo.
	origResolve := resolveBackendInfo
	t.Cleanup(func() { resolveBackendInfo = origResolve })

	// Mock: return "auto" backend with "default" source (typical fresh install).
	resolveBackendInfo = func() (secrets.KeyringBackendInfo, error) {
		return secrets.KeyringBackendInfo{Value: "auto", Source: "default"}, nil
	}

	tests := []struct {
		name     string
		goos     string
		dbusAddr string
		want     bool
	}{
		{"linux headless", "linux", "", true},
		{"linux with dbus", "linux", "/run/user/1000/bus", false},
		{"darwin", "darwin", "", false},
		{"darwin with dbus", "darwin", "/tmp/bus", false},
		{"windows", "windows", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NeedsFileBackendSetup(tt.goos, tt.dbusAddr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("NeedsFileBackendSetup(%q, %q) = %v, want %v", tt.goos, tt.dbusAddr, got, tt.want)
			}
		})
	}
}

func TestNeedsFileBackendSetup_ExplicitBackend(t *testing.T) {
	origResolve := resolveBackendInfo
	t.Cleanup(func() { resolveBackendInfo = origResolve })

	tests := []struct {
		name    string
		backend string
		want    bool
	}{
		{"explicit file", "file", false},     // User already configured — don't interfere.
		{"explicit keychain", "keychain", false},
		{"auto", "auto", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolveBackendInfo = func() (secrets.KeyringBackendInfo, error) {
				return secrets.KeyringBackendInfo{Value: tt.backend, Source: "config"}, nil
			}
			got, err := NeedsFileBackendSetup("linux", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	pw1, err := generatePassword()
	if err != nil {
		t.Fatalf("generatePassword: %v", err)
	}
	if len(pw1) != passwordBytes*2 {
		t.Errorf("password length = %d, want %d", len(pw1), passwordBytes*2)
	}

	pw2, err := generatePassword()
	if err != nil {
		t.Fatalf("generatePassword: %v", err)
	}
	if pw1 == pw2 {
		t.Error("two generated passwords are identical")
	}
}

func TestEnsureKeyringKey(t *testing.T) {
	dir := t.TempDir()

	// First call: creates file.
	pw1, err := ensureKeyringKey(dir)
	if err != nil {
		t.Fatalf("ensureKeyringKey: %v", err)
	}
	if pw1 == "" {
		t.Fatal("password is empty")
	}

	// Check permissions.
	path := filepath.Join(dir, keyringKeyFile)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat keyring.key: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}

	// Second call: returns same password (idempotent).
	pw2, err := ensureKeyringKey(dir)
	if err != nil {
		t.Fatalf("ensureKeyringKey (idempotent): %v", err)
	}
	if pw1 != pw2 {
		t.Errorf("password changed on second call: %q vs %q", pw1, pw2)
	}
}

func TestWriteCredentialsEnv(t *testing.T) {
	dir := t.TempDir()
	password := "abc123def456"

	if err := writeCredentialsEnv(dir, password); err != nil {
		t.Fatalf("writeCredentialsEnv: %v", err)
	}

	path := filepath.Join(dir, credentialsEnvFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read credentials.env: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "export WK_KEYRING_PASSWORD=abc123def456") {
		t.Error("missing WK_KEYRING_PASSWORD line")
	}
	if !strings.Contains(content, "export WK_KEYRING_BACKEND=file") {
		t.Error("missing WK_KEYRING_BACKEND line")
	}

	// Check permissions.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials.env: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}

func TestWriteCredentialsEnv_PreservesAccount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, credentialsEnvFile)

	// Pre-seed with account line.
	initial := "export WK_KEYRING_PASSWORD=oldpw\nexport WK_KEYRING_BACKEND=file\nexport WK_ACCOUNT=test@example.com\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	// Rewrite with new password.
	if err := writeCredentialsEnv(dir, "newpw"); err != nil {
		t.Fatalf("writeCredentialsEnv: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "export WK_KEYRING_PASSWORD=newpw") {
		t.Error("password not updated")
	}
	if !strings.Contains(content, "export WK_ACCOUNT=test@example.com") {
		t.Error("WK_ACCOUNT line lost")
	}
}

func TestConfigureShellProfile(t *testing.T) {
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".bashrc")
	credPath := filepath.Join(dir, "credentials.env")

	// Create the rc file.
	if err := os.WriteFile(rcFile, []byte("# existing content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override userHomeDir to use temp dir.
	origHome := userHomeDir
	origGetenv := getenv
	t.Cleanup(func() {
		userHomeDir = origHome
		getenv = origGetenv
	})
	userHomeDir = func() (string, error) { return dir, nil }
	getenv = func(key string) string {
		if key == "SHELL" {
			return "/bin/bash"
		}
		return ""
	}

	if err := configureShellProfile(credPath); err != nil {
		t.Fatalf("configureShellProfile: %v", err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	expectedLine := "source " + credPath
	if !strings.Contains(content, expectedLine) {
		t.Errorf("shell profile missing source line, content:\n%s", content)
	}

	// Idempotent: second call should not duplicate.
	if err := configureShellProfile(credPath); err != nil {
		t.Fatalf("configureShellProfile (idempotent): %v", err)
	}

	data2, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	count := strings.Count(string(data2), expectedLine)
	if count != 1 {
		t.Errorf("source line appears %d times, want 1", count)
	}
}

func TestAppendAccountToCredentialsEnv(t *testing.T) {
	dir := t.TempDir()

	origConfigDir := configDirFunc
	t.Cleanup(func() { configDirFunc = origConfigDir })
	configDirFunc = func() (string, error) { return dir, nil }

	// Create initial credentials.env.
	path := filepath.Join(dir, credentialsEnvFile)
	initial := "export WK_KEYRING_PASSWORD=testpw\nexport WK_KEYRING_BACKEND=file\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	// Append account.
	if err := AppendAccountToCredentialsEnv("user@example.com"); err != nil {
		t.Fatalf("AppendAccountToCredentialsEnv: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "export WK_ACCOUNT=user@example.com") {
		t.Errorf("missing WK_ACCOUNT line, content:\n%s", content)
	}

	// Update account.
	if err := AppendAccountToCredentialsEnv("other@example.com"); err != nil {
		t.Fatalf("AppendAccountToCredentialsEnv (update): %v", err)
	}

	data2, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content2 := string(data2)
	if !strings.Contains(content2, "export WK_ACCOUNT=other@example.com") {
		t.Error("account not updated")
	}
	if strings.Contains(content2, "user@example.com") {
		t.Error("old account still present")
	}

	// Count: only one WK_ACCOUNT line.
	count := strings.Count(content2, "export WK_ACCOUNT=")
	if count != 1 {
		t.Errorf("WK_ACCOUNT appears %d times, want 1", count)
	}
}

func TestAppendAccountToCredentialsEnv_NoFile(t *testing.T) {
	dir := t.TempDir()

	origConfigDir := configDirFunc
	t.Cleanup(func() { configDirFunc = origConfigDir })
	configDirFunc = func() (string, error) { return dir, nil }

	// No credentials.env exists — should be a no-op.
	if err := AppendAccountToCredentialsEnv("user@example.com"); err != nil {
		t.Fatalf("expected no-op, got error: %v", err)
	}

	// File should still not exist.
	path := filepath.Join(dir, credentialsEnvFile)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("credentials.env was created when it shouldn't have been")
	}
}

func TestSetupKeyringIfNeeded_NoOp_Darwin(t *testing.T) {
	origGOOS := runtimeGOOS
	origResolve := resolveBackendInfo
	origGetenv := getenv
	t.Cleanup(func() {
		runtimeGOOS = origGOOS
		resolveBackendInfo = origResolve
		getenv = origGetenv
	})

	runtimeGOOS = "darwin"
	resolveBackendInfo = func() (secrets.KeyringBackendInfo, error) {
		return secrets.KeyringBackendInfo{Value: "keychain", Source: "default"}, nil
	}
	getenv = func(key string) string { return "" }

	var buf bytes.Buffer
	if err := SetupKeyringIfNeeded(&buf); err != nil {
		t.Fatalf("expected no-op: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for no-op, got: %s", buf.String())
	}
}

func TestSetupKeyringIfNeeded_NoOp_LinuxWithDBus(t *testing.T) {
	origGOOS := runtimeGOOS
	origResolve := resolveBackendInfo
	origGetenv := getenv
	t.Cleanup(func() {
		runtimeGOOS = origGOOS
		resolveBackendInfo = origResolve
		getenv = origGetenv
	})

	runtimeGOOS = "linux"
	resolveBackendInfo = func() (secrets.KeyringBackendInfo, error) {
		return secrets.KeyringBackendInfo{Value: "auto", Source: "default"}, nil
	}
	getenv = func(key string) string {
		if key == "DBUS_SESSION_BUS_ADDRESS" {
			return "unix:path=/run/user/1000/bus"
		}
		return ""
	}

	var buf bytes.Buffer
	if err := SetupKeyringIfNeeded(&buf); err != nil {
		t.Fatalf("expected no-op: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for no-op, got: %s", buf.String())
	}
}

func TestSetupKeyringIfNeeded_SkipsWhenPasswordSet(t *testing.T) {
	origGOOS := runtimeGOOS
	origResolve := resolveBackendInfo
	origGetenv := getenv
	t.Cleanup(func() {
		runtimeGOOS = origGOOS
		resolveBackendInfo = origResolve
		getenv = origGetenv
	})

	runtimeGOOS = "linux"
	resolveBackendInfo = func() (secrets.KeyringBackendInfo, error) {
		return secrets.KeyringBackendInfo{Value: "auto", Source: "default"}, nil
	}
	getenv = func(key string) string {
		if key == envKeyringPassword {
			return "already-set"
		}
		return ""
	}

	var buf bytes.Buffer
	if err := SetupKeyringIfNeeded(&buf); err != nil {
		t.Fatalf("expected no-op: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output when password already set, got: %s", buf.String())
	}
}

func TestSetupKeyringIfNeeded_FullFlow(t *testing.T) {
	dir := t.TempDir()
	homeDir := t.TempDir()

	// Create a .bashrc in the fake home.
	bashrcPath := filepath.Join(homeDir, ".bashrc")
	if err := os.WriteFile(bashrcPath, []byte("# test bashrc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origGOOS := runtimeGOOS
	origResolve := resolveBackendInfo
	origGetenv := getenv
	origSetenv := setenv
	origEnsureDir := ensureConfigDir
	origHome := userHomeDir
	t.Cleanup(func() {
		runtimeGOOS = origGOOS
		resolveBackendInfo = origResolve
		getenv = origGetenv
		setenv = origSetenv
		ensureConfigDir = origEnsureDir
		userHomeDir = origHome
	})

	runtimeGOOS = "linux"
	resolveBackendInfo = func() (secrets.KeyringBackendInfo, error) {
		return secrets.KeyringBackendInfo{Value: "auto", Source: "default"}, nil
	}

	envMap := map[string]string{}
	getenv = func(key string) string {
		if key == "SHELL" {
			return "/bin/bash"
		}
		return envMap[key]
	}
	setenv = func(key, value string) error {
		envMap[key] = value
		return nil
	}
	ensureConfigDir = func() (string, error) { return dir, nil }
	userHomeDir = func() (string, error) { return homeDir, nil }

	var buf bytes.Buffer
	if err := SetupKeyringIfNeeded(&buf); err != nil {
		t.Fatalf("SetupKeyringIfNeeded: %v", err)
	}

	// Check keyring.key exists with correct permissions.
	keyPath := filepath.Join(dir, keyringKeyFile)
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("keyring.key not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("keyring.key permissions = %o, want 600", perm)
	}

	// Check credentials.env exists with correct content.
	credPath := filepath.Join(dir, credentialsEnvFile)
	credData, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatalf("credentials.env not created: %v", err)
	}
	credContent := string(credData)
	if !strings.Contains(credContent, "export WK_KEYRING_PASSWORD=") {
		t.Error("credentials.env missing WK_KEYRING_PASSWORD")
	}
	if !strings.Contains(credContent, "export WK_KEYRING_BACKEND=file") {
		t.Error("credentials.env missing WK_KEYRING_BACKEND")
	}

	// Check env vars were set in process.
	if envMap[envKeyringPassword] == "" {
		t.Error("WK_KEYRING_PASSWORD not set in process")
	}
	if envMap[envKeyringBackend] != "file" {
		t.Errorf("WK_KEYRING_BACKEND = %q, want 'file'", envMap[envKeyringBackend])
	}

	// Check shell profile was updated.
	bashrcData, err := os.ReadFile(bashrcPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bashrcData), "source "+credPath) {
		t.Error("shell profile not updated with source line")
	}

	// Check output contains instructions.
	output := buf.String()
	if !strings.Contains(output, "Keyring auto-setup complete") {
		t.Errorf("expected setup message, got: %s", output)
	}
	if !strings.Contains(output, "source") {
		t.Errorf("expected source instruction, got: %s", output)
	}
}

func TestDetectShellProfile_Zsh(t *testing.T) {
	dir := t.TempDir()

	origHome := userHomeDir
	origGetenv := getenv
	t.Cleanup(func() {
		userHomeDir = origHome
		getenv = origGetenv
	})

	userHomeDir = func() (string, error) { return dir, nil }
	getenv = func(key string) string {
		if key == "SHELL" {
			return "/bin/zsh"
		}
		return ""
	}

	path, err := detectShellProfile()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, ".zshrc")
	if path != want {
		t.Errorf("detectShellProfile() = %q, want %q", path, want)
	}
}

func TestDetectShellProfile_Bash(t *testing.T) {
	dir := t.TempDir()

	origHome := userHomeDir
	origGetenv := getenv
	t.Cleanup(func() {
		userHomeDir = origHome
		getenv = origGetenv
	})

	userHomeDir = func() (string, error) { return dir, nil }
	getenv = func(key string) string {
		if key == "SHELL" {
			return "/bin/bash"
		}
		return ""
	}

	path, err := detectShellProfile()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, ".bashrc")
	if path != want {
		t.Errorf("detectShellProfile() = %q, want %q", path, want)
	}
}
