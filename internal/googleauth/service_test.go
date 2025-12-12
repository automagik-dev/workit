package googleauth

import "testing"

func TestParseService(t *testing.T) {
	tests := []struct {
		in   string
		want Service
	}{
		{"gmail", ServiceGmail},
		{"GMAIL", ServiceGmail},
		{"calendar", ServiceCalendar},
		{"drive", ServiceDrive},
		{"contacts", ServiceContacts},
	}
	for _, tt := range tests {
		got, err := ParseService(tt.in)
		if err != nil {
			t.Fatalf("ParseService(%q) err: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseService(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestExtractCodeAndState(t *testing.T) {
	code, state, err := extractCodeAndState("http://localhost:1/?code=abc&state=xyz")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if code != "abc" || state != "xyz" {
		t.Fatalf("unexpected: code=%q state=%q", code, state)
	}
}
