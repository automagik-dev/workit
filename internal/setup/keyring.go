// Package setup provides automatic configuration for workit's runtime
// environment, including keyring password generation for Linux headless systems.
package setup

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/automagik-dev/workit/internal/config"
	"github.com/automagik-dev/workit/internal/secrets"
)

const (
	keyringKeyFile     = "keyring.key"
	credentialsEnvFile = "credentials.env"
	passwordBytes      = 32 // 32 bytes = 64 hex chars

	envKeyringPassword = "WK_KEYRING_PASSWORD" //nolint:gosec // env var name, not a credential
	envKeyringBackend  = "WK_KEYRING_BACKEND"  //nolint:gosec // env var name, not a credential
	envAccount         = "WK_ACCOUNT"

	sourceLine = "source" // prefix used in shell profiles
)

// Stubs for testability.
var (
	resolveBackendInfo = secrets.ResolveKeyringBackendInfo
	configDirFunc      = config.Dir
	ensureConfigDir    = config.EnsureDir
	runtimeGOOS        = runtime.GOOS
	getenv             = os.Getenv
	setenv             = os.Setenv
	userHomeDir        = os.UserHomeDir
)

// NeedsFileBackendSetup reports whether the current environment will use
// the keyring file backend and therefore needs automatic password setup.
//
// Returns true on Linux when the resolved backend is "auto" and no
// DBUS_SESSION_BUS_ADDRESS is set (headless environment).
func NeedsFileBackendSetup(goos string, dbusAddr string) (bool, error) {
	if goos != "linux" {
		return false, nil
	}

	info, err := resolveBackendInfo()
	if err != nil {
		return false, fmt.Errorf("resolve keyring backend: %w", err)
	}

	// Only auto-setup when the backend would auto-detect to "file".
	// If user explicitly set "file" or "keychain", respect their choice.
	if info.Value != "auto" {
		return false, nil
	}

	// D-Bus present means SecretService/gnome-keyring is likely available.
	if dbusAddr != "" {
		return false, nil
	}

	return true, nil
}

// SetupKeyringIfNeeded detects whether the file keyring backend will be used
// and, if so, generates a random password, saves it to keyring.key, writes
// credentials.env, and configures the user's shell profile to source it.
//
// On macOS or Linux with D-Bus present, this is a no-op.
// If keyring.key already exists, the existing password is reused.
//
// Status messages are written to w (typically os.Stderr).
func SetupKeyringIfNeeded(w io.Writer) error {
	needed, err := NeedsFileBackendSetup(runtimeGOOS, getenv("DBUS_SESSION_BUS_ADDRESS"))
	if err != nil {
		return err
	}

	if !needed {
		return nil
	}

	// Already configured via environment? Skip.
	if getenv(envKeyringPassword) != "" {
		return nil
	}

	configDir, err := ensureConfigDir()
	if err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	password, err := ensureKeyringKey(configDir)
	if err != nil {
		return fmt.Errorf("ensure keyring key: %w", err)
	}

	credPath := filepath.Join(configDir, credentialsEnvFile)

	if err := writeCredentialsEnv(configDir, password); err != nil {
		return fmt.Errorf("write credentials.env: %w", err)
	}

	// Set env vars in current process so the keyring opens correctly
	// in this same session without requiring the user to restart.
	if err := setenv(envKeyringPassword, password); err != nil {
		return fmt.Errorf("setenv %s: %w", envKeyringPassword, err)
	}

	if err := setenv(envKeyringBackend, "file"); err != nil {
		return fmt.Errorf("setenv %s: %w", envKeyringBackend, err)
	}

	if err := configureShellProfile(credPath); err != nil {
		// Non-fatal: user can source manually.
		fmt.Fprintf(w, "Warning: could not update shell profile: %v\n", err)
	}

	fmt.Fprintf(w, "Keyring auto-setup complete.\n")
	fmt.Fprintf(w, "  Password saved to: %s\n", filepath.Join(configDir, keyringKeyFile))
	fmt.Fprintf(w, "  Environment file:  %s\n", credPath)
	fmt.Fprintf(w, "\n  Run: source %s\n\n", credPath)

	return nil
}

// generatePassword creates a random hex password of passwordBytes length.
func generatePassword() (string, error) {
	b := make([]byte, passwordBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random password: %w", err)
	}

	return hex.EncodeToString(b), nil
}

// ensureKeyringKey reads or creates the keyring.key file in configDir.
// If the file exists, the existing password is returned.
// If not, a new password is generated and saved with 0600 permissions.
func ensureKeyringKey(configDir string) (string, error) {
	path := filepath.Join(configDir, keyringKeyFile)

	data, err := os.ReadFile(path) //nolint:gosec // config path
	if err == nil {
		pw := strings.TrimSpace(string(data))
		if pw != "" {
			return pw, nil
		}
	}

	password, err := generatePassword()
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, []byte(password+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write keyring key: %w", err)
	}

	return password, nil
}

// writeCredentialsEnv writes (or rewrites) the credentials.env file with
// WK_KEYRING_PASSWORD and WK_KEYRING_BACKEND. Existing WK_ACCOUNT lines
// are preserved.
func writeCredentialsEnv(configDir, password string) error {
	path := filepath.Join(configDir, credentialsEnvFile)

	// Preserve existing WK_ACCOUNT line if present.
	var accountLine string

	if data, err := os.ReadFile(path); err == nil { //nolint:gosec // config path
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "export "+envAccount+"=") {
				accountLine = line
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("# workit keyring credentials (auto-generated)\n")
	sb.WriteString("# Source this file in your shell: source " + path + "\n")
	sb.WriteString(fmt.Sprintf("export %s=%s\n", envKeyringPassword, password))
	sb.WriteString(fmt.Sprintf("export %s=file\n", envKeyringBackend))

	if accountLine != "" {
		sb.WriteString(accountLine + "\n")
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("write credentials env: %w", err)
	}

	return nil
}

// configureShellProfile finds the user's shell profile and appends a
// source line for credentials.env if not already present.
func configureShellProfile(credentialsEnvPath string) error {
	profilePath, err := detectShellProfile()
	if err != nil {
		return err
	}

	if profilePath == "" {
		return nil // No profile found; user must source manually.
	}

	// Check if source line already present.
	sourceDirective := fmt.Sprintf("source %s", credentialsEnvPath)
	sourceDirectiveQuoted := fmt.Sprintf("source \"%s\"", credentialsEnvPath)

	data, err := os.ReadFile(profilePath) //nolint:gosec // shell profile path
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read shell profile %s: %w", profilePath, err)
	}

	content := string(data)
	if strings.Contains(content, sourceDirective) || strings.Contains(content, sourceDirectiveQuoted) {
		return nil // Already configured.
	}

	// Append source line.
	line := fmt.Sprintf("\n# workit keyring credentials\n%s\n", sourceDirective)

	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // shell profile
	if err != nil {
		return fmt.Errorf("open shell profile %s: %w", profilePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("write to shell profile %s: %w", profilePath, err)
	}

	return nil
}

// detectShellProfile returns the path to the user's shell profile file.
// Detection order: $SHELL → ~/.zshrc (zsh), ~/.bashrc (bash), ~/.profile (fallback).
func detectShellProfile() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect home dir: %w", err)
	}

	shell := getenv("SHELL")

	switch {
	case strings.HasSuffix(shell, "/zsh"):
		return filepath.Join(home, ".zshrc"), nil
	case strings.HasSuffix(shell, "/bash"):
		return filepath.Join(home, ".bashrc"), nil
	}

	// Fallback: check if common profiles exist.
	for _, name := range []string{".bashrc", ".profile"} {
		path := filepath.Join(home, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Last resort: use .profile (will be created).
	return filepath.Join(home, ".profile"), nil
}

// AppendAccountToCredentialsEnv appends or updates the WK_ACCOUNT line
// in credentials.env with the given email.
//
// If credentials.env doesn't exist (e.g., non-headless environment),
// this is a no-op.
func AppendAccountToCredentialsEnv(email string) error {
	configDir, err := configDirFunc()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}

	path := filepath.Join(configDir, credentialsEnvFile)

	data, err := os.ReadFile(path) //nolint:gosec // config path
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No credentials.env → not a headless setup, skip.
		}

		return fmt.Errorf("read credentials env: %w", err)
	}

	accountLine := fmt.Sprintf("export %s=%s", envAccount, email)

	// Check if WK_ACCOUNT already set to this email.
	content := string(data)
	if strings.Contains(content, accountLine) {
		return nil // Already set.
	}

	// Replace existing WK_ACCOUNT line or append new one.
	var sb strings.Builder
	replaced := false

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "export "+envAccount+"=") {
			sb.WriteString(accountLine + "\n")
			replaced = true
		} else {
			sb.WriteString(line + "\n")
		}
	}

	if !replaced {
		sb.WriteString(accountLine + "\n")
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("write credentials env: %w", err)
	}

	return nil
}
