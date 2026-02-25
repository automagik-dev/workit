package docx

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ConvertToPDF converts a DOCX file to PDF using LibreOffice headless.
// outputPath can be empty (defaults to same directory as input, .pdf extension).
// Returns the path to the generated PDF.
func ConvertToPDF(ctx context.Context, inputPath, outputPath string) (string, error) {
	// Check LibreOffice is available.
	sofficePath, err := exec.LookPath("soffice")
	if err != nil {
		return "", fmt.Errorf("LibreOffice not found: install with 'apt install libreoffice-common' or 'brew install libreoffice': %w", err)
	}

	outDir := filepath.Dir(inputPath)
	if outputPath != "" {
		outDir = filepath.Dir(outputPath)
	}

	// Ensure the output directory exists.
	if mkdirErr := os.MkdirAll(outDir, 0o750); mkdirErr != nil {
		return "", fmt.Errorf("create output dir %s: %w", outDir, mkdirErr)
	}

	// soffice --headless --convert-to pdf --outdir DIR INPUT
	cmd := exec.CommandContext(ctx, sofficePath, "--headless", "--convert-to", "pdf", "--outdir", outDir, inputPath) //nolint:gosec // sofficePath from LookPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("libreoffice conversion failed: %s: %w", output, err)
	}

	// LibreOffice names the output same as input but with .pdf extension.
	baseName := filepath.Base(inputPath)
	pdfName := baseName[:len(baseName)-len(filepath.Ext(baseName))] + ".pdf"
	generatedPath := filepath.Join(outDir, pdfName)

	// If user specified a different output path, rename.
	if outputPath != "" && outputPath != generatedPath {
		if renameErr := os.Rename(generatedPath, outputPath); renameErr != nil {
			return "", fmt.Errorf("rename pdf: %w", renameErr)
		}

		return outputPath, nil
	}

	return generatedPath, nil
}

// LibreOfficeVersion returns the LibreOffice version string, or an error
// if soffice is not installed.
func LibreOfficeVersion(ctx context.Context) (string, error) {
	sofficePath, err := exec.LookPath("soffice")
	if err != nil {
		return "", fmt.Errorf("soffice not found: %w", err)
	}

	out, err := exec.CommandContext(ctx, sofficePath, "--version").CombinedOutput() //nolint:gosec // sofficePath from LookPath
	if err != nil {
		return "", fmt.Errorf("soffice --version failed: %w", err)
	}

	return string(out), nil
}
