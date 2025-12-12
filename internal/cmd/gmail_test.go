package cmd

import (
	"testing"

	"google.golang.org/api/gmail/v1"
)

func TestHeaderValue(t *testing.T) {
	p := &gmail.MessagePart{
		Headers: []*gmail.MessagePartHeader{
			{Name: "From", Value: "a@example.com"},
			{Name: "Subject", Value: "Hello"},
		},
	}
	if got := headerValue(p, "from"); got != "a@example.com" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := headerValue(p, "subject"); got != "Hello" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := headerValue(p, "date"); got != "" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestSanitizeTab(t *testing.T) {
	if got := sanitizeTab("a\tb"); got != "a b" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestFormatGmailDate(t *testing.T) {
	got := formatGmailDate("Mon, 02 Jan 2006 15:04:05 -0700")
	if got != "2006-01-02 15:04" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := formatGmailDate("not a date"); got != "not a date" {
		t.Fatalf("unexpected: %q", got)
	}
}
