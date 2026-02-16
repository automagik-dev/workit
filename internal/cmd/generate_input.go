package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
				value = parseDefaultTyped(f.Default, typeName)
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
		typeName := reflectTypeString(p.Target)
		switch {
		case p.Required:
			template[name] = fmt.Sprintf("(required) %s", typeName)
		case p.HasDefault && p.Default != "":
			template[name] = parseDefaultTyped(p.Default, typeName)
		default:
			template[name] = fmt.Sprintf("<%s>", typeName)
		}
	}

	return template, nil
}

// parseDefaultTyped converts a string default value to its native Go type based
// on the reflected type name. This ensures JSON output has typed values (e.g.
// integers as numbers, bools as booleans) instead of everything being a string.
func parseDefaultTyped(defaultVal, typeName string) any {
	switch typeName {
	case "bool":
		if b, err := strconv.ParseBool(defaultVal); err == nil {
			return b
		}
	case "int":
		if n, err := strconv.Atoi(defaultVal); err == nil {
			return n
		}
	case "int64":
		if n, err := strconv.ParseInt(defaultVal, 10, 64); err == nil {
			return n
		}
	case "float64":
		if f, err := strconv.ParseFloat(defaultVal, 64); err == nil {
			return f
		}
	}
	// Fall back to string for types we cannot parse or on parse error.
	return defaultVal
}

// generateInputTemplateFromNode builds the same JSON template as
// generateInputTemplate but works directly from a kong.Node, bypassing the
// need for a fully parsed kong.Context.  This is used when --generate-input is
// detected before parsing so that commands with required positional arguments
// do not fail.
func generateInputTemplateFromNode(node *kong.Node) (map[string]any, error) {
	if node == nil {
		return nil, fmt.Errorf("no command selected")
	}

	template := make(map[string]any)

	for _, group := range node.AllFlags(false) {
		for _, f := range group {
			if f == nil || f.Hidden {
				continue
			}
			name := f.Name
			if name == "help" || name == "version" {
				continue
			}

			var value any
			typeName := reflectTypeString(f.Target)

			switch {
			case f.Enum != "":
				value = fmt.Sprintf("<enum: %s>", strings.Join(f.EnumSlice(), "|"))
			case f.Required:
				value = fmt.Sprintf("(required) %s", typeName)
			case f.HasDefault && f.Default != "":
				value = parseDefaultTyped(f.Default, typeName)
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

	for _, p := range node.Positional {
		if p == nil {
			continue
		}
		name := p.Name
		typeName := reflectTypeString(p.Target)
		switch {
		case p.Required:
			template[name] = fmt.Sprintf("(required) %s", typeName)
		case p.HasDefault && p.Default != "":
			template[name] = parseDefaultTyped(p.Default, typeName)
		default:
			template[name] = fmt.Sprintf("<%s>", typeName)
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

// printGenerateInputFromNode prints the input template for a command node
// without requiring a fully parsed kong.Context.
func printGenerateInputFromNode(node *kong.Node) error {
	template, err := generateInputTemplateFromNode(node)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(template)
}
