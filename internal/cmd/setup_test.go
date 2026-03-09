package cmd

import (
	"context"
	"runtime"
	"testing"
)

func TestInstallHintLibreOffice(t *testing.T) {
	t.Parallel()

	hint := installHintLibreOffice()
	if hint == "" {
		t.Fatal("expected non-empty hint")
	}

	switch runtime.GOOS {
	case "darwin":
		if hint != "brew install --cask libreoffice" {
			t.Errorf("unexpected darwin hint: %s", hint)
		}
	case "windows":
		if hint != "winget install TheDocumentFoundation.LibreOffice  OR  choco install libreoffice-fresh" {
			t.Errorf("unexpected windows hint: %s", hint)
		}
	default:
		if hint != "apt install libreoffice-common  OR  snap install libreoffice" {
			t.Errorf("unexpected linux hint: %s", hint)
		}
	}
}

func TestInstallHintPython(t *testing.T) {
	t.Parallel()

	hint := installHintPython()
	if hint == "" {
		t.Fatal("expected non-empty hint")
	}

	switch runtime.GOOS {
	case "darwin":
		if hint != "brew install python3" {
			t.Errorf("unexpected darwin hint: %s", hint)
		}
	case "windows":
		if hint != "winget install Python.Python.3  OR  choco install python3" {
			t.Errorf("unexpected windows hint: %s", hint)
		}
	default:
		if hint != "apt install python3  OR  snap install python3" {
			t.Errorf("unexpected linux hint: %s", hint)
		}
	}
}

func TestInstallHintLxml(t *testing.T) {
	t.Parallel()

	hint := installHintLxml()
	if hint == "" {
		t.Fatal("expected non-empty hint")
	}

	if runtime.GOOS == "windows" {
		if hint != "pip install lxml" {
			t.Errorf("unexpected windows hint: %s", hint)
		}
	} else {
		if hint != "pip3 install lxml  OR  apt install python3-lxml" {
			t.Errorf("unexpected hint: %s", hint)
		}
	}
}

func TestCheckGo(t *testing.T) {
	t.Parallel()

	d := checkGo()
	if d.Status != "ok" {
		t.Errorf("checkGo status = %q, want ok", d.Status)
	}
	if d.Version == "" {
		t.Error("checkGo version is empty")
	}
	if !d.Required {
		t.Error("checkGo should be required")
	}
}

func TestCheckLibreOffice(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	d := checkLibreOffice(ctx)

	// LibreOffice is typically not installed on CI.
	if d.Status == "ok" {
		if d.Version == "" {
			t.Error("status ok but version empty")
		}
	} else {
		if d.Status != statusMissing {
			t.Errorf("status = %q, want %q", d.Status, statusMissing)
		}
		if d.Install == "" {
			t.Error("missing install hint")
		}
	}

	if d.Required {
		t.Error("LibreOffice should not be required")
	}
}

func TestCheckPython3Lxml(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	d := checkPython3Lxml(ctx)

	// Python may or may not be installed.
	if d.Status == "ok" {
		if d.Version == "" {
			t.Error("status ok but version empty")
		}
	} else {
		if d.Status != statusMissing {
			t.Errorf("status = %q, want %q", d.Status, statusMissing)
		}
		if d.Install == "" {
			t.Error("missing install hint")
		}
	}

	if d.Required {
		t.Error("python3+lxml should not be required")
	}
}

func TestResolveServiceManager_Valid(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"systemd", "pm2", "launchd", "schtasks"} {
		m, err := resolveServiceManager(name)
		if err != nil {
			t.Errorf("resolveServiceManager(%q) error: %v", name, err)
		}
		if string(m) != name {
			t.Errorf("resolveServiceManager(%q) = %q", name, m)
		}
	}
}

func TestResolveServiceManager_Invalid(t *testing.T) {
	t.Parallel()

	_, err := resolveServiceManager("bogus")
	if err == nil {
		t.Error("expected error for bogus manager")
	}
}

func TestSetupDocxCmd_Run(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cmd := &SetupDocxCmd{}

	// Should not error even if deps are missing.
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("SetupDocxCmd.Run: %v", err)
	}
}
