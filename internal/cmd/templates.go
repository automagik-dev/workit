package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/automagik-dev/workit/internal/config"
	"github.com/automagik-dev/workit/internal/docx"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

// TemplatesCmd manages document templates.
type TemplatesCmd struct {
	List    TemplatesListCmd    `cmd:"" help:"List available templates"`
	Add     TemplatesAddCmd     `cmd:"" help:"Add a template"`
	Inspect TemplatesInspectCmd `cmd:"" help:"Inspect a template for placeholders"`
}

// TemplatesListCmd lists all installed templates.
type TemplatesListCmd struct{}

// Run executes the templates list command.
func (c *TemplatesListCmd) Run(ctx context.Context) error {
	configDir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}

	templatesDir := docx.TemplatesDir(configDir)

	if outfmt.IsJSON(ctx) {
		infos, jsonErr := docx.ListTemplateInfos(templatesDir, true)
		if jsonErr != nil {
			return fmt.Errorf("list templates: %w", jsonErr)
		}
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"templates": infos,
			"dir":       templatesDir,
		})
	}

	names, err := docx.ListTemplates(templatesDir)
	if err != nil {
		return fmt.Errorf("list templates: %w", err)
	}

	u := ui.FromContext(ctx)

	if len(names) == 0 {
		if u != nil {
			u.Out().Printf("no templates installed (dir: %s)", templatesDir)
		} else {
			fmt.Printf("no templates installed (dir: %s)\n", templatesDir)
		}
		return nil
	}

	if u != nil {
		u.Out().Printf("%d template(s):", len(names))
		for _, name := range names {
			u.Out().Printf("  %s", name)
		}
	} else {
		fmt.Printf("%d template(s):\n", len(names))
		for _, name := range names {
			fmt.Printf("  %s\n", name)
		}
	}
	return nil
}

// TemplatesAddCmd adds a template to the templates directory.
type TemplatesAddCmd struct {
	Name   string `arg:"" help:"Template name"`
	Source string `arg:"" help:"Path to DOCX template file"`
}

// Run executes the templates add command.
func (c *TemplatesAddCmd) Run(ctx context.Context) error {
	configDir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}

	templatesDir, err := docx.EnsureTemplatesDir(configDir)
	if err != nil {
		return fmt.Errorf("ensure templates dir: %w", err)
	}

	if err := docx.AddTemplate(templatesDir, c.Name, c.Source); err != nil {
		return fmt.Errorf("add template: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]string{
			"name": c.Name,
			"path": docx.GetTemplatePath(templatesDir, c.Name),
		})
	}

	if u := ui.FromContext(ctx); u != nil {
		u.Err().Printf("template %q added from %s", c.Name, c.Source)
	} else {
		fmt.Fprintf(os.Stderr, "template %q added from %s\n", c.Name, c.Source)
	}
	return nil
}

// TemplatesInspectCmd inspects a template for placeholders.
type TemplatesInspectCmd struct {
	Name string `arg:"" help:"Template name or file path"`
}

// Run executes the templates inspect command.
func (c *TemplatesInspectCmd) Run(ctx context.Context) error {
	configDir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}

	templatesDir := docx.TemplatesDir(configDir)

	names, err := docx.InspectTemplateByName(templatesDir, c.Name)
	if err != nil {
		return fmt.Errorf("inspect template: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, docx.TemplateInfo{
			Name:         c.Name,
			Path:         docx.GetTemplatePath(templatesDir, c.Name),
			Placeholders: names,
		})
	}

	u := ui.FromContext(ctx)

	if len(names) == 0 {
		if u != nil {
			u.Out().Print("no placeholders found")
		} else {
			fmt.Println("no placeholders found")
		}
		return nil
	}

	if u != nil {
		u.Out().Printf("found %d placeholder(s):", len(names))
		for _, name := range names {
			u.Out().Printf("  {{%s}}", name)
		}
	} else {
		fmt.Printf("found %d placeholder(s):\n", len(names))
		for _, name := range names {
			fmt.Printf("  {{%s}}\n", name)
		}
	}
	return nil
}
