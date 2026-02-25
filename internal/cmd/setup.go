package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/namastexlabs/workit/internal/outfmt"
	"github.com/namastexlabs/workit/internal/ui"
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

	sofficePath, err := exec.LookPath("soffice")
	if err != nil {
		d.Status = statusMissing
		d.Install = installHintLibreOffice()
		return d
	}

	out, err := exec.CommandContext(ctx, sofficePath, "--version").CombinedOutput() //nolint:gosec // sofficePath from LookPath
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

	python3, err := exec.LookPath("python3")
	if err != nil {
		d.Status = statusMissing
		d.Install = "apt install python3 python3-lxml  OR  pip3 install lxml"
		return d
	}

	out, err := exec.CommandContext(ctx, python3, "-c", "import lxml; print(lxml.__version__)").CombinedOutput() //nolint:gosec // python3 from LookPath
	if err != nil {
		d.Status = statusMissing
		d.Install = "pip3 install lxml  OR  apt install python3-lxml"
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
	default:
		return "apt install libreoffice-common  OR  snap install libreoffice"
	}
}
