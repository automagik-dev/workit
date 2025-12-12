package cmd

import (
	"strings"
	"testing"
)

func TestBuildRFC822Plain(t *testing.T) {
	raw, err := buildRFC822(mailOptions{
		From:    "a@b.com",
		To:      []string{"c@d.com"},
		Subject: "Hi",
		Body:    "Hello",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "Content-Type: text/plain") {
		t.Fatalf("missing content-type: %q", s)
	}
	if !strings.Contains(s, "\r\n\r\nHello\r\n") {
		t.Fatalf("missing body: %q", s)
	}
}

func TestBuildRFC822WithAttachment(t *testing.T) {
	raw, err := buildRFC822(mailOptions{
		From:    "a@b.com",
		To:      []string{"c@d.com"},
		Subject: "Hi",
		Body:    "Hello",
		Attachments: []mailAttachment{
			{Filename: "x.txt", MIMEType: "text/plain", Data: []byte("abc")},
		},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "multipart/mixed") {
		t.Fatalf("expected multipart: %q", s)
	}
	if !strings.Contains(s, "Content-Disposition: attachment; filename=\"x.txt\"") {
		t.Fatalf("missing attachment header: %q", s)
	}
}

func TestEncodeHeaderIfNeeded(t *testing.T) {
	if got := encodeHeaderIfNeeded("Hello"); got != "Hello" {
		t.Fatalf("unexpected: %q", got)
	}
	got := encodeHeaderIfNeeded("Grüße")
	if got == "Grüße" || !strings.Contains(got, "=?utf-8?") {
		t.Fatalf("expected encoded-word, got: %q", got)
	}
}

func TestContentDispositionFilename(t *testing.T) {
	if got := contentDispositionFilename("a.txt"); got != "filename=\"a.txt\"" {
		t.Fatalf("unexpected: %q", got)
	}
	got := contentDispositionFilename("Grüße.txt")
	if !strings.HasPrefix(got, "filename*=UTF-8''") {
		t.Fatalf("unexpected: %q", got)
	}
}
