package docx_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// buildSampleDOCX creates a minimal valid DOCX file at the given path.
// The document contains:
//   - A title heading ("Sample Document")
//   - 3 body paragraphs with different styles
//   - A 3x3 table
//   - Core metadata (author, title, dates)
func buildSampleDOCX(t *testing.T, path string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	files := map[string]string{
		"[Content_Types].xml":          contentTypesXML,
		"_rels/.rels":                  relsXML,
		"word/document.xml":            documentXML,
		"word/_rels/document.xml.rels": documentRelsXML,
		"docProps/core.xml":            coreXML,
		"docProps/app.xml":             appXML,
		"word/styles.xml":              stylesXML,
	}

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}

		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
}

// ensureSampleDOCX builds the testdata/sample.docx fixture if it doesn't exist.
func ensureSampleDOCX(t *testing.T) string {
	t.Helper()
	// Use a path relative to the repo root — go test sets cwd to the package dir,
	// so we walk up to find the repo root.
	path := filepath.Join("..", "..", "testdata", "sample.docx")

	// Always rebuild to ensure consistency.
	buildSampleDOCX(t, path)

	return path
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`

const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`

const documentXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Title"/></w:pPr>
      <w:r><w:t>Sample Document</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Introduction</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>This is the first paragraph of the sample document. It contains plain text.</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading2"/></w:pPr>
      <w:r><w:t>Details</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Second paragraph with more details about the topic.</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Third paragraph wrapping up the content.</w:t></w:r>
    </w:p>
    <w:tbl>
      <w:tblPr>
        <w:tblStyle w:val="TableGrid"/>
        <w:tblW w:w="0" w:type="auto"/>
      </w:tblPr>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Name</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Role</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Status</w:t></w:r></w:p></w:tc>
      </w:tr>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Alice</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Engineer</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Active</w:t></w:r></w:p></w:tc>
      </w:tr>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Bob</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Designer</w:t></w:r></w:p></w:tc>
        <w:tc><w:p><w:r><w:t>Active</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>
    <w:p>
      <w:r><w:t>Final paragraph after the table.</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

const coreXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
                   xmlns:dc="http://purl.org/dc/elements/1.1/"
                   xmlns:dcterms="http://purl.org/dc/terms/"
                   xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>Sample Document</dc:title>
  <dc:creator>Test Author</dc:creator>
  <dc:description>A sample document for testing</dc:description>
  <dcterms:created xsi:type="dcterms:W3CDTF">2026-01-15T10:30:00Z</dcterms:created>
  <dcterms:modified xsi:type="dcterms:W3CDTF">2026-02-20T14:45:00Z</dcterms:modified>
</cp:coreProperties>`

const appXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties">
  <Pages>2</Pages>
  <Application>Test</Application>
</Properties>`

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Title">
    <w:name w:val="Title"/>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
  </w:style>
</w:styles>`

// letterTemplateDocXML is a document with {{RECIPIENT}} and {{DATE}} placeholders.
const letterTemplateDocXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Dear {{RECIPIENT}},</w:t></w:r></w:p>
    <w:p><w:r><w:t>Date: {{DATE}}</w:t></w:r></w:p>
  </w:body>
</w:document>`

// buildLetterTemplateDOCX creates a DOCX with {{RECIPIENT}} and {{DATE}} placeholders.
func buildLetterTemplateDOCX(t *testing.T, path string) {
	t.Helper()
	buildDOCXFromXML(t, path, letterTemplateDocXML)
}

// ensureTemplateDOCX builds the testdata/template.docx fixture with
// {{TITLE}}, {{AUTHOR}}, {{BODY}} placeholders.
func ensureTemplateDOCX(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "template.docx")
	buildDOCXFromXML(t, path, templateDocumentXML)

	return path
}

// TestGenerateFixtures regenerates testdata fixtures.
// Run with: go test ./internal/docx/ -run TestGenerateFixtures
func TestGenerateFixtures(t *testing.T) {
	ensureSampleDOCX(t)
	ensureTemplateDOCX(t)
}
