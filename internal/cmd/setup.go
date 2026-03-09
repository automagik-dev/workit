package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/automagik-dev/workit/internal/docx"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

// SetupCmd validates environment dependencies.
type SetupCmd struct {
	Docx SetupDocxCmd `cmd:"" help:"Validate DOCX dependencies"`
}

// SetupDocxCmd checks for required DOCX dependencies.
type SetupDocxCmd struct{}

const statusMissing = "missing"

type depStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // "ok" or "missing"
	Version  string `json:"version,omitempty"`
	Required bool   `json:"required"`
	Install  string `json:"install,omitempty"`
}

// Run executes the setup docx command.
func (c *SetupDocxCmd) Run(ctx context.Context) error {
	deps := []depStatus{
		checkGo(),
		checkLibreOffice(ctx),
		checkPython3Lxml(ctx),
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"dependencies": deps,
		})
	}

	u := ui.FromContext(ctx)
	printDep := func(d depStatus) {
		icon := "ok"
		if d.Status != "ok" {
			icon = "MISSING"
		}
		reqLabel := "optional"
		if d.Required {
			reqLabel = "required"
		}

		line := fmt.Sprintf("[%s] %s (%s)", icon, d.Name, reqLabel)
		if d.Version != "" {
			line += " - " + d.Version
		}

		if u != nil {
			u.Out().Print(line)
		} else {
			fmt.Println(line)
		}

		if d.Status != "ok" && d.Install != "" {
			hint := "  install: " + d.Install
			if u != nil {
				u.Out().Print(hint)
			} else {
				fmt.Println(hint)
			}
		}
	}

	if u != nil {
		u.Out().Print("DOCX dependency check:")
	} else {
		fmt.Println("DOCX dependency check:")
	}

	for _, d := range deps {
		printDep(d)
	}

	return nil
}

func checkGo() depStatus {
	return depStatus{
		Name:     "Go runtime",
		Status:   "ok",
		Version:  runtime.Version(),
		Required: true,
	}
}

func checkLibreOffice(ctx context.Context) depStatus {
	d := depStatus{
		Name:     "LibreOffice (PDF export)",
		Required: false,
	}

	sofficePath, err := docx.LookPathSoffice()
	if err != nil {
		d.Status = statusMissing
		d.Install = installHintLibreOffice()
		return d
	}

	out, err := exec.CommandContext(ctx, sofficePath, "--version").CombinedOutput() //nolint:gosec // sofficePath from LookPathSoffice
	if err != nil {
		d.Status = statusMissing
		d.Install = installHintLibreOffice()
		return d
	}

	d.Status = "ok"
	d.Version = strings.TrimSpace(string(out))
	return d
}

func checkPython3Lxml(ctx context.Context) depStatus {
	d := depStatus{
		Name:     "Python3 + lxml (XSD validation)",
		Required: false,
	}

	pythonBin, err := exec.LookPath("python3")
	if err != nil {
		pythonBin, err = exec.LookPath("python")
	}
	if err != nil {
		d.Status = statusMissing
		d.Install = installHintPython()
		return d
	}

	out, err := exec.CommandContext(ctx, pythonBin, "-c", "import lxml; print(lxml.__version__)").CombinedOutput() //nolint:gosec // pythonBin from LookPath
	if err != nil {
		d.Status = statusMissing
		d.Install = installHintLxml()
		return d
	}

	d.Status = "ok"
	d.Version = "lxml " + strings.TrimSpace(string(out))
	return d
}

func installHintLibreOffice() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install --cask libreoffice"
	case windowsOS:
		return "winget install TheDocumentFoundation.LibreOffice  OR  choco install libreoffice-fresh"
	default:
		return "apt install libreoffice-common  OR  snap install libreoffice"
	}
}

func installHintPython() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install python3"
	case windowsOS:
		return "winget install Python.Python.3  OR  choco install python3"
	default:
		return "apt install python3  OR  snap install python3"
	}
}

func installHintLxml() string {
	if runtime.GOOS == windowsOS {
		return "pip install lxml"
	}
	return "pip3 install lxml  OR  apt install python3-lxml"
}
