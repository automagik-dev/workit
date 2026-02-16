package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

// generateInputTemplate creates a JSON template showing all available flags
// for the selected command with their types, defaults, and required status.
func generateInputTemplate(kctx *kong.Context) (map[string]any, error) {
	selected := kctx.Selected()
	if selected == nil {
		return nil, fmt.Errorf("no command selected")
	}

	template := make(map[string]any)

	// Walk all flags (including inherited/global ones).
	for _, group := range selected.AllFlags(false) {
		for _, f := range group {
			if f == nil || f.Hidden {
				continue
			}
			// Skip Kong built-ins.
			name := f.Name
			if name == "help" || name == "version" {
				continue
			}

			// Determine the template value.
			var value any
			typeName := reflectTypeString(f.Target)

			switch {
			case f.Enum != "":
				value = fmt.Sprintf("<enum: %s>", strings.Join(f.EnumSlice(), "|"))
			case f.Required:
				value = fmt.Sprintf("(required) %s", typeName)
			case f.HasDefault && f.Default != "":
				value = f.Default
			case typeName == "bool":
				value = false
			case typeName == "int" || typeName == "int64":
				value = 0
			default:
				value = fmt.Sprintf("<%s>", typeName)
			}

			template[name] = value
		}
	}

	// Also include positional arguments.
	for _, p := range selected.Positional {
		if p == nil {
			continue
		}
		name := p.Name
		if p.Required {
			template[name] = fmt.Sprintf("(required) %s", reflectTypeString(p.Target))
		} else if p.HasDefault && p.Default != "" {
			template[name] = p.Default
		} else {
			template[name] = fmt.Sprintf("<%s>", reflectTypeString(p.Target))
		}
	}

	return template, nil
}

func printGenerateInput(kctx *kong.Context) error {
	template, err := generateInputTemplate(kctx)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(template)
}
