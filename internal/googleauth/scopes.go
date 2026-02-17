package googleauth

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrUnknownCommand = errors.New("unknown command")

// CommandServiceMap maps CLI command names to their required Google API service.
// Used for dynamic scope filtering when --enable-commands is specified.
var CommandServiceMap = map[string]Service{
	"gmail":     ServiceGmail,
	"drive":     ServiceDrive,
	"calendar":  ServiceCalendar,
	"docs":      ServiceDocs,
	"sheets":    ServiceSheets,
	"slides":    ServiceSlides,
	"forms":     ServiceForms,
	"tasks":     ServiceTasks,
	"chat":      ServiceChat,
	"classroom": ServiceClassroom,
	"contacts":  ServiceContacts,
	"people":    ServicePeople,
	"keep":      ServiceKeep,
	"appscript": ServiceAppScript,
	"groups":    ServiceGroups,
}

// ScopesForCommands returns the union of OAuth scopes needed for the given CLI
// command names. If commands is empty, returns all scopes (backward compatible).
func ScopesForCommands(commands []string) ([]string, error) {
	if len(commands) == 0 {
		return AllScopes(), nil
	}

	seen := make(map[string]bool)
	var result []string

	for _, cmd := range commands {
		cmd = strings.ToLower(strings.TrimSpace(cmd))
		if cmd == "" {
			continue
		}

		svc, ok := CommandServiceMap[cmd]
		if !ok {
			return nil, fmt.Errorf("%w %q (known: %s)", ErrUnknownCommand, cmd, knownCommandNames())
		}

		scopes, err := Scopes(svc)
		if err != nil {
			return nil, fmt.Errorf("scopes for %q: %w", cmd, err)
		}

		for _, s := range scopes {
			if !seen[s] {
				seen[s] = true
				result = append(result, s)
			}
		}
	}

	sort.Strings(result)

	return result, nil
}

// AllScopes returns all known OAuth scopes across all services, sorted and deduplicated.
func AllScopes() []string {
	seen := make(map[string]bool)
	var result []string

	for _, svc := range serviceOrder {
		scopes, err := Scopes(svc)
		if err != nil {
			continue
		}

		for _, s := range scopes {
			if !seen[s] {
				seen[s] = true
				result = append(result, s)
			}
		}
	}

	sort.Strings(result)

	return result
}

// knownCommandNames returns a sorted, comma-separated list of known command names.
func knownCommandNames() string {
	names := make([]string, 0, len(CommandServiceMap))
	for k := range CommandServiceMap {
		names = append(names, k)
	}

	sort.Strings(names)

	return strings.Join(names, ", ")
}
