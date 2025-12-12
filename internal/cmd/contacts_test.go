package cmd

import (
	"testing"

	"google.golang.org/api/people/v1"
)

func TestPrimaryName(t *testing.T) {
	p := &people.Person{
		Names: []*people.Name{
			{DisplayName: "Ada Lovelace", GivenName: "Ada", FamilyName: "Lovelace"},
		},
	}
	if got := primaryName(p); got != "Ada Lovelace" {
		t.Fatalf("unexpected: %q", got)
	}
	p.Names[0].DisplayName = ""
	if got := primaryName(p); got != "Ada Lovelace" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestPrimaryEmailPhone(t *testing.T) {
	p := &people.Person{
		EmailAddresses: []*people.EmailAddress{{Value: "a@b.com"}},
		PhoneNumbers:   []*people.PhoneNumber{{Value: "+1 555 0100"}},
	}
	if got := primaryEmail(p); got != "a@b.com" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := primaryPhone(p); got != "+1 555 0100" {
		t.Fatalf("unexpected: %q", got)
	}
}
