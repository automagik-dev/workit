package officetext

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- helpers to create minimal Office ZIP fixtures in memory ---

func createDocxFixture(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "test.docx")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// word/document.xml with two paragraphs
	f, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t>Hello</w:t></w:r>
      <w:r><w:t xml:space="preserve"> World</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Second paragraph</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`))

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func createXlsxFixture(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "test.xlsx")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// xl/sharedStrings.xml
	f, err := w.Create("xl/sharedStrings.xml")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="3" uniqueCount="3">
  <si><t>Name</t></si>
  <si><t>Age</t></si>
  <si><t>Alice</t></si>
</sst>`))

	// xl/worksheets/sheet1.xml -- row 1: Name, Age; row 2: Alice, 30
	f, err = w.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>0</v></c>
      <c r="B1" t="s"><v>1</v></c>
    </row>
    <row r="2">
      <c r="A2" t="s"><v>2</v></c>
      <c r="B2"><v>30</v></c>
    </row>
  </sheetData>
</worksheet>`))

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func createXlsxMultiSheetFixture(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "multi.xlsx")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// xl/sharedStrings.xml
	f, err := w.Create("xl/sharedStrings.xml")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="2" uniqueCount="2">
  <si><t>Sheet1Data</t></si>
  <si><t>Sheet2Data</t></si>
</sst>`))

	// Sheet 1
	f, err = w.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1" t="s"><v>0</v></c></row>
  </sheetData>
</worksheet>`))

	// Sheet 2
	f, err = w.Create("xl/worksheets/sheet2.xml")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1" t="s"><v>1</v></c></row>
  </sheetData>
</worksheet>`))

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func createPptxFixture(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "test.pptx")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// ppt/slides/slide1.xml
	f, err := w.Create("ppt/slides/slide1.xml")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
       xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p><a:r><a:t>Title Slide</a:t></a:r></a:p>
          <a:p><a:r><a:t>Subtitle</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`))

	// ppt/slides/slide2.xml
	f, err = w.Create("ppt/slides/slide2.xml")
	if err != nil {
		t.Fatal(err)
	}

	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
       xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p><a:r><a:t>Content slide</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`))

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

// --- actual tests ---

func TestExtractTextDocx(t *testing.T) {
	dir := t.TempDir()
	path := createDocxFixture(t, dir)

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	text, err := ExtractText(f, "report.docx")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "Hello World") {
		t.Errorf("expected text to contain 'Hello World', got: %q", text)
	}

	if !strings.Contains(text, "Second paragraph") {
		t.Errorf("expected text to contain 'Second paragraph', got: %q", text)
	}
}

func TestExtractTextXlsx(t *testing.T) {
	dir := t.TempDir()
	path := createXlsxFixture(t, dir)

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	text, err := ExtractText(f, "data.xlsx")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "Name") {
		t.Errorf("expected text to contain 'Name', got: %q", text)
	}

	if !strings.Contains(text, "Alice") {
		t.Errorf("expected text to contain 'Alice', got: %q", text)
	}

	if !strings.Contains(text, "30") {
		t.Errorf("expected text to contain '30', got: %q", text)
	}
}

func TestExtractTextXlsxMultiSheet(t *testing.T) {
	dir := t.TempDir()
	path := createXlsxMultiSheetFixture(t, dir)

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	text, err := ExtractText(f, "multi.xlsx")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "Sheet 1:") {
		t.Errorf("expected multi-sheet output to contain 'Sheet 1:', got: %q", text)
	}

	if !strings.Contains(text, "Sheet 2:") {
		t.Errorf("expected multi-sheet output to contain 'Sheet 2:', got: %q", text)
	}

	if !strings.Contains(text, "Sheet1Data") {
		t.Errorf("expected text to contain 'Sheet1Data', got: %q", text)
	}

	if !strings.Contains(text, "Sheet2Data") {
		t.Errorf("expected text to contain 'Sheet2Data', got: %q", text)
	}
}

func TestExtractTextPptx(t *testing.T) {
	dir := t.TempDir()
	path := createPptxFixture(t, dir)

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	text, err := ExtractText(f, "slides.pptx")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "--- Slide 1 ---") {
		t.Errorf("expected slide 1 header, got: %q", text)
	}

	if !strings.Contains(text, "Title Slide") {
		t.Errorf("expected 'Title Slide', got: %q", text)
	}

	if !strings.Contains(text, "--- Slide 2 ---") {
		t.Errorf("expected slide 2 header, got: %q", text)
	}

	if !strings.Contains(text, "Content slide") {
		t.Errorf("expected 'Content slide', got: %q", text)
	}
}

func TestExtractTextUnknownFormat(t *testing.T) {
	r := strings.NewReader("some data")

	_, err := ExtractText(r, "file.unknown")
	if err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got: %q", err.Error())
	}
}

func TestExtractTextCorruptedZip(t *testing.T) {
	r := strings.NewReader("this is not a zip file")

	_, err := ExtractText(r, "corrupt.docx")
	if err == nil {
		t.Fatal("expected error for corrupted ZIP, got nil")
	}
}

func TestExtractTextByMimeType(t *testing.T) {
	dir := t.TempDir()
	path := createDocxFixture(t, dir)

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	// Even with no extension, MIME type should work
	text, err := ExtractTextByMIME(f, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "Hello World") {
		t.Errorf("expected text to contain 'Hello World', got: %q", text)
	}
}

func TestExtractTextByMimeXlsx(t *testing.T) {
	dir := t.TempDir()
	path := createXlsxFixture(t, dir)

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	text, err := ExtractTextByMIME(f, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "Name") {
		t.Errorf("expected 'Name', got: %q", text)
	}
}

func TestExtractTextByMimePptx(t *testing.T) {
	dir := t.TempDir()
	path := createPptxFixture(t, dir)

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	text, err := ExtractTextByMIME(f, "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "Title Slide") {
		t.Errorf("expected 'Title Slide', got: %q", text)
	}
}

func TestExtractTextByMimeUnknown(t *testing.T) {
	r := strings.NewReader("data")

	_, err := ExtractTextByMIME(r, "application/pdf")
	if err == nil {
		t.Fatal("expected error for unsupported MIME type, got nil")
	}
}

func TestDocxEmptyDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.docx")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f, _ := w.Create("word/document.xml")
	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body></w:body>
</w:document>`))
	_ = w.Close()
	_ = os.WriteFile(path, buf.Bytes(), 0o644)

	file, _ := os.Open(path)
	defer file.Close()

	text, err := ExtractText(file, "empty.docx")
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(text) != "" {
		t.Errorf("expected empty text from empty doc, got: %q", text)
	}
}

func TestXlsxNoSharedStrings(t *testing.T) {
	// XLSX with only inline values, no shared strings
	dir := t.TempDir()
	path := filepath.Join(dir, "inline.xlsx")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f, _ := w.Create("xl/worksheets/sheet1.xml")
	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1"><v>42</v></c>
      <c r="B1"><v>99</v></c>
    </row>
  </sheetData>
</worksheet>`))

	_ = w.Close()
	_ = os.WriteFile(path, buf.Bytes(), 0o644)

	file, _ := os.Open(path)
	defer file.Close()

	text, err := ExtractText(file, "inline.xlsx")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(text, "42") {
		t.Errorf("expected '42', got: %q", text)
	}

	if !strings.Contains(text, "99") {
		t.Errorf("expected '99', got: %q", text)
	}
}
