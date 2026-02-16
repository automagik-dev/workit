package outfmt

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

// DiscoverFields returns JSON field names for a struct type using reflection.
// It reads json struct tags to get the wire names. Fields tagged with json:"-"
// are excluded. Fields without a json tag use the Go field name (matching
// encoding/json default behavior). The returned order matches struct field order.
func DiscoverFields(v any) []string {
	if v == nil {
		return nil
	}

	t := reflect.TypeOf(v)
	// Dereference pointer.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Handle map types (e.g. map[string]any envelopes emitted by many commands).
	if t.Kind() == reflect.Map {
		val := reflect.ValueOf(v)
		for val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		keys := make([]string, 0, val.Len())
		for _, k := range val.MapKeys() {
			keys = append(keys, fmt.Sprintf("%v", k.Interface()))
		}
		sort.Strings(keys)
		return keys
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	fields := make([]string, 0, t.NumField())

	for i := range t.NumField() {
		f := t.Field(i)

		// Skip unexported fields (encoding/json skips them too).
		if !f.IsExported() {
			continue
		}

		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}

		name := jsonFieldName(tag, f.Name)
		fields = append(fields, name)
	}

	return fields
}

// jsonFieldName extracts the JSON wire name from a struct tag value.
// If the tag is empty, it falls back to the Go field name (matching encoding/json).
func jsonFieldName(tag, goName string) string {
	if tag == "" {
		return goName
	}

	// Tag format: "name,opts" -- extract the name part before the first comma.
	if idx := strings.IndexByte(tag, ','); idx != -1 {
		tag = tag[:idx]
	}

	if tag == "" {
		return goName
	}

	return tag
}

// IsFieldDiscovery returns true if --select was passed as an explicit empty string
// to trigger field discovery mode.
func IsFieldDiscovery(selectValue string, selectFlagSet bool) bool {
	return selectFlagSet && selectValue == ""
}

// SelectFlagExplicitlySet scans a raw argument slice to determine whether
// --select (or its aliases --pick, --project) was explicitly provided.
// This is needed because Kong cannot distinguish "flag absent" from
// "flag present with empty string" when the default value is also "".
func SelectFlagExplicitlySet(args []string) bool {
	selectFlags := map[string]bool{
		"--select":  true,
		"--pick":    true,
		"--project": true,
	}

	for i, a := range args {
		if a == "--" {
			break
		}

		// Handle --select=value and --select value forms.
		if strings.Contains(a, "=") {
			prefix := a[:strings.IndexByte(a, '=')]
			if selectFlags[prefix] {
				return true
			}

			continue
		}

		if selectFlags[a] {
			// The flag is present. It must be followed by a value (even if empty).
			// Kong requires a value for string flags, so if this is the last arg
			// or next arg looks like a flag, Kong will error. We still consider
			// the flag "set" if it appears.
			_ = i // value follows at args[i+1] if present

			return true
		}
	}

	return false
}

// PrintFieldDiscovery writes available JSON field names and a usage hint to w
// (intended to be stderr). The commandExample is used in the hint line, e.g.
// "gog drive ls".
func PrintFieldDiscovery(w io.Writer, fields []string, commandExample string) {
	fmt.Fprintln(w, "Available fields:")

	for _, f := range fields {
		fmt.Fprintf(w, "  %s\n", f)
	}

	fmt.Fprintln(w)

	if len(fields) > 0 && commandExample != "" {
		// Show a hint with up to 3 sample fields.
		sample := fields
		if len(sample) > 3 {
			sample = sample[:3]
		}

		fmt.Fprintf(w, "Usage: %s --json --select \"%s\"\n", commandExample, strings.Join(sample, ","))
	} else {
		fmt.Fprintln(w, "Usage: gog <command> --json --select \"field1,field2\"")
	}
}
