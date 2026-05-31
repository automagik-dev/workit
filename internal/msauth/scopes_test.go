package msauth

import (
	"errors"
	"strings"
	"testing"
)

func TestPilotAllowedScopesAreExactlyReadOnlyBaseline(t *testing.T) {
	got := PilotAllowedScopes()
	want := []string{"User.Read", "Mail.Read", "Calendars.Read"}

	if len(got) != len(want) {
		t.Fatalf("PilotAllowedScopes length = %d, want %d: %#v", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("PilotAllowedScopes()[%d] = %q, want %q; got %#v", i, got[i], want[i], got)
		}
	}
}

func TestPilotScopesDefaultToReadOnlyBaseline(t *testing.T) {
	got, err := GuardPilotScopes(nil)
	if err != nil {
		t.Fatalf("GuardPilotScopes(nil) err: %v", err)
	}

	want := []string{"User.Read", "Mail.Read", "Calendars.Read"}
	if len(got) != len(want) {
		t.Fatalf("GuardPilotScopes(nil) length = %d, want %d: %#v", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("GuardPilotScopes(nil)[%d] = %q, want %q; got %#v", i, got[i], want[i], got)
		}
	}
}

func TestGuardPilotScopesAcceptsOnlyPilotReadScopes(t *testing.T) {
	got, err := GuardPilotScopes([]string{" mail.read ", "user.read", "Mail.Read", "CALENDARS.READ"})
	if err != nil {
		t.Fatalf("GuardPilotScopes(read scopes) err: %v", err)
	}

	want := []string{"User.Read", "Mail.Read", "Calendars.Read"}
	if len(got) != len(want) {
		t.Fatalf("GuardPilotScopes returned %d scopes, want %d: %#v", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("GuardPilotScopes()[%d] = %q, want %q; got %#v", i, got[i], want[i], got)
		}
	}
}

func TestGuardPilotScopesRejectsKnownWriteScopes(t *testing.T) {
	denied := []string{
		"Mail.Send",
		"Calendars.ReadWrite",
		"Chat.ReadWrite",
		"ChannelMessage.Send",
		"Sites.ReadWrite.All",
		"Files.ReadWrite.All",
	}

	for _, scope := range denied {
		t.Run(scope, func(t *testing.T) {
			_, err := GuardPilotScopes([]string{"User.Read", scope})
			if err == nil {
				t.Fatalf("GuardPilotScopes accepted denied scope %q", scope)
			}

			if !errors.Is(err, ErrPilotScopeNotAllowed) {
				t.Fatalf("GuardPilotScopes error = %v, want ErrPilotScopeNotAllowed", err)
			}

			if !strings.Contains(err.Error(), scope) {
				t.Fatalf("GuardPilotScopes error %q does not name denied scope %q", err.Error(), scope)
			}
		})
	}
}

func TestGuardPilotScopesRejectsUnknownScopeFailClosed(t *testing.T) {
	_, err := GuardPilotScopes([]string{"User.Read", "Mail.ReadBasic"})
	if err == nil {
		t.Fatal("GuardPilotScopes accepted unknown scope")
	}

	if !errors.Is(err, ErrPilotScopeNotAllowed) {
		t.Fatalf("GuardPilotScopes error = %v, want ErrPilotScopeNotAllowed", err)
	}
}

func TestGuardPilotScopesDoesNotExposeMutableAllowlist(t *testing.T) {
	scopes := PilotAllowedScopes()
	scopes[0] = "Mail.Send"

	got, err := GuardPilotScopes(nil)
	if err != nil {
		t.Fatalf("GuardPilotScopes(nil) err: %v", err)
	}

	if got[0] != "User.Read" {
		t.Fatalf("pilot allowlist mutated through returned slice: %#v", got)
	}
}
