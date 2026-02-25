package docx_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/namastexlabs/workit/internal/docx"
)

func TestConvertToPDF(t *testing.T) {
	// Skip if LibreOffice is not installed.
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("LibreOffice (soffice) not available; skipping PDF test")
	}

	samplePath := ensureSampleDOCX(t)

	// Convert to a temp directory so we don't litter testdata.
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "sample.pdf")

	pdfPath, err := docx.ConvertToPDF(context.Background(), samplePath, outPath)
	if err != nil {
		t.Fatalf("ConvertToPDF: %v", err)
	}

	if pdfPath != outPath {
		t.Errorf("expected output %q, got %q", outPath, pdfPath)
	}
}

func TestConvertToPDF_DefaultOutput(t *testing.T) {
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("LibreOffice (soffice) not available; skipping PDF test")
	}

	// Build sample in a temp dir.
	dir := t.TempDir()
	samplePath := filepath.Join(dir, "sample.docx")
	buildSampleDOCX(t, samplePath)

	pdfPath, err := docx.ConvertToPDF(context.Background(), samplePath, "")
	if err != nil {
		t.Fatalf("ConvertToPDF: %v", err)
	}

	wantPath := filepath.Join(dir, "sample.pdf")
	if pdfPath != wantPath {
		t.Errorf("expected output %q, got %q", wantPath, pdfPath)
	}
}

func TestLibreOfficeVersion(t *testing.T) {
	if _, err := exec.LookPath("soffice"); err != nil {
		t.Skip("LibreOffice (soffice) not available; skipping version test")
	}

	version, err := docx.LibreOfficeVersion(context.Background())
	if err != nil {
		t.Fatalf("LibreOfficeVersion: %v", err)
	}

	if version == "" {
		t.Error("expected non-empty version string")
	}
}
