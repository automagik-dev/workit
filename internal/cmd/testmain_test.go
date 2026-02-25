package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "workit-tests-*")
	if err != nil {
		panic(err)
	}

	oldHome := os.Getenv("HOME")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")

	home := filepath.Join(root, "home")
	xdg := filepath.Join(root, "xdg")
	_ = os.MkdirAll(home, 0o755)
	_ = os.MkdirAll(xdg, 0o755)
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("XDG_CONFIG_HOME", xdg)

	code := m.Run()

	if oldHome == "" {
		_ = os.Unsetenv("HOME")
	} else {
		_ = os.Setenv("HOME", oldHome)
	}
	if oldXDG == "" {
		_ = os.Unsetenv("XDG_CONFIG_HOME")
	} else {
		_ = os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}
	_ = os.RemoveAll(root)
	os.Exit(code)
}
