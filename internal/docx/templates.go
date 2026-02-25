package docx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TemplateInfo describes an installed template.
type TemplateInfo struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Placeholders []string `json:"placeholders,omitempty"`
}

// TemplatesDir returns the path to the template storage directory
// inside the given config directory.
func TemplatesDir(configDir string) string {
	return filepath.Join(configDir, "templates")
}

// EnsureTemplatesDir creates the templates directory if it doesn't exist.
func EnsureTemplatesDir(configDir string) (string, error) {
	dir := TemplatesDir(configDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("ensure templates dir: %w", err)
	}

	return dir, nil
}

// ListTemplates returns names of all installed templates (sorted).
// Names are derived from .docx filenames with the extension stripped.
func ListTemplates(templatesDir string) ([]string, error) {
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("read templates dir: %w", err)
	}
	var names []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".docx") {
			names = append(names, strings.TrimSuffix(name, filepath.Ext(name)))
		}
	}

	sort.Strings(names)

	return names, nil
}

// AddTemplate copies a DOCX file into the templates directory with the given name.
// The template is stored as <name>.docx.
func AddTemplate(templatesDir, name, sourcePath string) error {
	if err := os.MkdirAll(templatesDir, 0o750); err != nil {
		return fmt.Errorf("ensure templates dir: %w", err)
	}

	destPath := filepath.Join(templatesDir, name+".docx")

	src, err := os.Open(sourcePath) //nolint:gosec // user-provided template path
	if err != nil {
		return fmt.Errorf("open source %s: %w", sourcePath, err)
	}
	defer src.Close()

	dst, err := os.Create(destPath) //nolint:gosec // user-provided template destination path
	if err != nil {
		return fmt.Errorf("create template %s: %w", destPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy template: %w", err)
	}

	return nil
}

// GetTemplatePath returns the full path for a template name.
// If the name is already a file path (contains / or \), it is returned directly.
// Otherwise, it is resolved as <templatesDir>/<name>.docx.
func GetTemplatePath(templatesDir, name string) string {
	if strings.ContainsAny(name, `/\`) || strings.HasSuffix(strings.ToLower(name), ".docx") {
		return name
	}

	return filepath.Join(templatesDir, name+".docx")
}

// InspectTemplateByName opens a template by name and returns its placeholders.
func InspectTemplateByName(templatesDir, name string) ([]string, error) {
	path := GetTemplatePath(templatesDir, name)

	session, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("open template %s: %w", path, err)
	}
	defer session.Close()

	return InspectTemplate(session)
}

// ListTemplateInfos returns detailed info for all installed templates.
// If inspectPlaceholders is true, each template is opened and inspected for placeholders.
func ListTemplateInfos(templatesDir string, inspectPlaceholders bool) ([]TemplateInfo, error) {
	names, err := ListTemplates(templatesDir)
	if err != nil {
		return nil, err
	}

	infos := make([]TemplateInfo, 0, len(names))
	for _, name := range names {
		info := TemplateInfo{
			Name: name,
			Path: GetTemplatePath(templatesDir, name),
		}

		if inspectPlaceholders {
			placeholders, err := InspectTemplateByName(templatesDir, name)
			if err == nil {
				info.Placeholders = placeholders
			}
			// Silently skip templates that fail to inspect.
		}

		infos = append(infos, info)
	}

	return infos, nil
}
