package googleauth

import (
	"sort"
	"testing"
)

func TestScopesForCommands_Empty(t *testing.T) {
	scopes, err := ScopesForCommands(nil)
	if err != nil {
		t.Fatalf("ScopesForCommands(nil) err: %v", err)
	}

	if len(scopes) == 0 {
		t.Fatal("expected non-empty scopes for nil input")
	}

	allScopes := AllScopes()
	if len(scopes) != len(allScopes) {
		t.Fatalf("ScopesForCommands(nil) returned %d scopes, AllScopes() returned %d", len(scopes), len(allScopes))
	}
}

func TestScopesForCommands_Gmail(t *testing.T) {
	scopes, err := ScopesForCommands([]string{"gmail"})
	if err != nil {
		t.Fatalf("ScopesForCommands([gmail]) err: %v", err)
	}

	gmailScopes, err := Scopes(ServiceGmail)
	if err != nil {
		t.Fatalf("Scopes(ServiceGmail) err: %v", err)
	}

	if len(scopes) != len(gmailScopes) {
		t.Fatalf("expected %d scopes, got %d: %v", len(gmailScopes), len(scopes), scopes)
	}

	for _, want := range gmailScopes {
		if !containsScope(scopes, want) {
			t.Fatalf("missing scope %q in %v", want, scopes)
		}
	}
}

func TestScopesForCommands_Multiple(t *testing.T) {
	scopes, err := ScopesForCommands([]string{"gmail", "drive"})
	if err != nil {
		t.Fatalf("ScopesForCommands([gmail,drive]) err: %v", err)
	}
	gmailScopes, _ := Scopes(ServiceGmail)
	driveScopes, _ := Scopes(ServiceDrive)

	for _, want := range gmailScopes {
		if !containsScope(scopes, want) {
			t.Fatalf("missing gmail scope %q in %v", want, scopes)
		}
	}

	for _, want := range driveScopes {
		if !containsScope(scopes, want) {
			t.Fatalf("missing drive scope %q in %v", want, scopes)
		}
	}
}

func TestScopesForCommands_Unknown(t *testing.T) {
	_, err := ScopesForCommands([]string{"nosuchcommand"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestScopesForCommands_Dedup(t *testing.T) {
	scopes, err := ScopesForCommands([]string{"gmail", "gmail"})
	if err != nil {
		t.Fatalf("ScopesForCommands([gmail,gmail]) err: %v", err)
	}

	gmailScopes, _ := Scopes(ServiceGmail)
	if len(scopes) != len(gmailScopes) {
		t.Fatalf("expected %d scopes (no duplicates), got %d: %v", len(gmailScopes), len(scopes), scopes)
	}
}

func TestScopesForCommands_CaseInsensitive(t *testing.T) {
	scopes, err := ScopesForCommands([]string{"GMAIL"})
	if err != nil {
		t.Fatalf("ScopesForCommands([GMAIL]) err: %v", err)
	}

	gmailScopes, _ := Scopes(ServiceGmail)
	if len(scopes) != len(gmailScopes) {
		t.Fatalf("expected %d scopes, got %d: %v", len(gmailScopes), len(scopes), scopes)
	}
}

func TestScopesForCommands_TrimWhitespace(t *testing.T) {
	scopes, err := ScopesForCommands([]string{" drive "})
	if err != nil {
		t.Fatalf("ScopesForCommands([ drive ]) err: %v", err)
	}

	driveScopes, _ := Scopes(ServiceDrive)
	if len(scopes) != len(driveScopes) {
		t.Fatalf("expected %d scopes, got %d: %v", len(driveScopes), len(scopes), scopes)
	}
}

func TestAllScopes(t *testing.T) {
	scopes := AllScopes()

	if len(scopes) == 0 {
		t.Fatal("AllScopes returned empty")
	}

	if !sort.StringsAreSorted(scopes) {
		t.Fatal("AllScopes result is not sorted")
	}
	// Should contain scopes from multiple services.
	for _, want := range []string{
		"https://www.googleapis.com/auth/gmail.modify",
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/calendar",
	} {
		if !containsScope(scopes, want) {
			t.Fatalf("AllScopes missing expected scope %q", want)
		}
	}
}

func TestAllScopes_NoDuplicates(t *testing.T) {
	scopes := AllScopes()

	seen := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		if seen[s] {
			t.Fatalf("duplicate scope %q in AllScopes()", s)
		}
		seen[s] = true
	}
}

func TestCommandServiceMap_CoversAllUserServices(t *testing.T) {
	userSvcs := UserServices()

	mappedServices := make(map[Service]bool)
	for _, svc := range CommandServiceMap {
		mappedServices[svc] = true
	}

	for _, svc := range userSvcs {
		if !mappedServices[svc] {
			t.Fatalf("user service %q not covered by CommandServiceMap", svc)
		}
	}
}
