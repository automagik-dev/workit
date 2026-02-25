package cmd

import (
	"fmt"

	"github.com/automagik-dev/workit/internal/input"
)

// resolveFileFlag resolves a file:// or fileb:// prefix on a flag value in-place.
// If the value is empty or has no file prefix, it is left unchanged.
// fieldName is used in the error message (e.g. "body", "text", "content").
func resolveFileFlag(value *string, fieldName string) error {
	if *value == "" {
		return nil
	}

	resolved, err := input.ResolveFileInput(*value)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", fieldName, err)
	}

	*value = resolved

	return nil
}
