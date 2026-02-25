package cmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/automagik-dev/workit/internal/config"
	"github.com/automagik-dev/workit/internal/docx"
	"github.com/automagik-dev/workit/internal/ui"
)

// DocxCreateCmd creates a DOCX from a template + values or from markdown.
type DocxCreateCmd struct {
	From     string `help:"Input file (markdown or JSON key-value)" required:""`
	Template string `help:"Template DOCX file path (required for JSON input)"`
	Out      string `help:"Output DOCX file path" required:""`
}

// Run executes the docx create command.
func (c *DocxCreateCmd) Run(ctx context.Context) error {
	// Determine if the input is JSON or markdown.
	inputData, err := os.ReadFile(c.From)
	if err != nil {
		return fmt.Errorf("read input %s: %w", c.From, err)
	}

	ext := strings.ToLower(filepath.Ext(c.From))

	if ext == ".json" || (c.Template != "" && ext != ".md" && ext != ".markdown") {
		return c.runTemplateFill(ctx, inputData)
	}

	return c.runFromMarkdown(ctx, inputData)
}

// resolveTemplatePath resolves a --template value.
// If the value looks like a file path (contains / or \ or ends with .docx), use it directly.
// Otherwise, look it up in the template management directory.
func resolveTemplatePath(tmpl string) string {
	// If it already looks like a path, use as-is.
	if strings.ContainsAny(tmpl, `/\`) || strings.HasSuffix(strings.ToLower(tmpl), ".docx") {
		return tmpl
	}

	// Try to resolve via template management.
	configDir, err := config.Dir()
	if err != nil {
		return tmpl // fallback to raw value
	}

	templatesDir := docx.TemplatesDir(configDir)
	resolved := docx.GetTemplatePath(templatesDir, tmpl)

	// Only return the resolved path if the file exists.
	if _, err := os.Stat(resolved); err == nil {
		return resolved
	}

	// File doesn't exist in templates dir; return original value
	// so the error message refers to what the user typed.
	return tmpl
}

// runTemplateFill fills a template with JSON key-value pairs.
func (c *DocxCreateCmd) runTemplateFill(ctx context.Context, inputData []byte) error {
	if c.Template == "" {
		return fmt.Errorf("--template is required when using JSON input")
	}

	// Resolve template name through the template management system.
	templatePath := resolveTemplatePath(c.Template)

	// Parse JSON values.
	var values map[string]string
	if err := json.Unmarshal(inputData, &values); err != nil {
		return fmt.Errorf("parse JSON values from %s: %w", c.From, err)
	}

	// Copy template to output path.
	templateData, err := os.ReadFile(templatePath) //nolint:gosec // user-provided template path
	if err != nil {
		return fmt.Errorf("read template %s: %w", templatePath, err)
	}

	outDir := filepath.Dir(c.Out)
	if mkdirErr := os.MkdirAll(outDir, 0o750); mkdirErr != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, mkdirErr)
	}

	if writeErr := os.WriteFile(c.Out, templateData, 0o600); writeErr != nil {
		return fmt.Errorf("copy template to %s: %w", c.Out, writeErr)
	}

	// Open the output copy and fill the template.
	session, err := docx.Open(c.Out)
	if err != nil {
		return fmt.Errorf("open output docx: %w", err)
	}
	defer session.Close()

	n, err := docx.FillTemplate(session, values)
	if err != nil {
		return fmt.Errorf("fill template: %w", err)
	}

	if err := session.SaveInPlace(); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("created %s (filled %d placeholder(s))", c.Out, n)
	} else {
		fmt.Fprintf(os.Stderr, "created %s (filled %d placeholder(s))\n", c.Out, n)
	}
	return nil
}

// runFromMarkdown creates a minimal DOCX from markdown content.
func (c *DocxCreateCmd) runFromMarkdown(ctx context.Context, inputData []byte) error {
	// Create a minimal blank DOCX, then rewrite its content.
	if err := writeBlankDOCX(c.Out); err != nil {
		return fmt.Errorf("create blank docx: %w", err)
	}

	session, err := docx.Open(c.Out)
	if err != nil {
		return fmt.Errorf("open blank docx: %w", err)
	}
	defer session.Close()

	if err := docx.Rewrite(session, string(inputData)); err != nil {
		return fmt.Errorf("rewrite: %w", err)
	}

	if err := session.SaveInPlace(); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("created %s from markdown", c.Out)
	} else {
		fmt.Fprintf(os.Stderr, "created %s from markdown\n", c.Out)
	}
	return nil
}

// writeBlankDOCX creates a minimal valid DOCX file at the given path.
func writeBlankDOCX(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	f, err := os.Create(path) //nolint:gosec // user output path
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	entries := map[string]string{
		"[Content_Types].xml":          blankContentTypes,
		"_rels/.rels":                  blankRels,
		"word/document.xml":            blankDocumentXML,
		"word/_rels/document.xml.rels": blankDocumentRels,
	}

	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("create zip entry %s: %w", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return fmt.Errorf("write zip entry %s: %w", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("close zip: %w", err)
	}
	return nil
}

const blankContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const blankRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const blankDocumentXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t></w:t></w:r></w:p>
  </w:body>
</w:document>`

const blankDocumentRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
