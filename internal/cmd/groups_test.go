package cmd

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/api/cloudidentity/v1"
)

func TestWrapCloudIdentityError(t *testing.T) {
	err := wrapCloudIdentityError(errors.New("accessNotConfigured: boom"), "user@company.com")
	if !strings.Contains(err.Error(), "cloudidentity API is not enabled") {
		t.Fatalf("unexpected error: %v", err)
	}

	err = wrapCloudIdentityError(errors.New("insufficientPermissions: nope"), "user@company.com")
	if !strings.Contains(err.Error(), "Insufficient permissions") {
		t.Fatalf("unexpected error: %v", err)
	}

	other := errors.New("other")
	if !errors.Is(wrapCloudIdentityError(other, "user@company.com"), other) {
		t.Fatalf("expected passthrough error")
	}
}

func TestWrapCloudIdentityError_ConsumerAccount(t *testing.T) {
	err := wrapCloudIdentityError(errors.New("badRequest: Request contains an invalid argument."), "person@gmail.com")
	if !strings.Contains(err.Error(), "consumer accounts") {
		t.Fatalf("unexpected consumer error: %v", err)
	}
}

func TestWrapCloudIdentityError_WorkspaceInvalidArg(t *testing.T) {
	err := wrapCloudIdentityError(errors.New("badRequest: Request contains an invalid argument."), "user@company.com")
	if !strings.Contains(err.Error(), "invalid argument error for user@company.com") {
		t.Fatalf("unexpected workspace error: %v", err)
	}
	if !strings.Contains(err.Error(), "Cloud Identity Premium") {
		t.Fatalf("expected licensing hint: %v", err)
	}
}

func TestGetRelationType(t *testing.T) {
	if got := getRelationType("DIRECT"); got != "direct" {
		t.Fatalf("unexpected relation: %q", got)
	}
	if got := getRelationType("INDIRECT"); got != "indirect" {
		t.Fatalf("unexpected relation: %q", got)
	}
	if got := getRelationType("CUSTOM"); got != "CUSTOM" {
		t.Fatalf("unexpected relation: %q", got)
	}
}

func TestGetMemberRole(t *testing.T) {
	if got := getMemberRole(nil); got != "MEMBER" {
		t.Fatalf("unexpected role: %q", got)
	}
	got := getMemberRole([]*cloudidentity.MembershipRole{
		{Name: "MEMBER"},
		{Name: "OWNER"},
	})
	if got != "OWNER" {
		t.Fatalf("unexpected role: %q", got)
	}
	got = getMemberRole([]*cloudidentity.MembershipRole{
		{Name: "MANAGER"},
	})
	if got != "MANAGER" {
		t.Fatalf("unexpected role: %q", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("unexpected truncate: %q", got)
	}
	if got := truncate("hello world", 5); got != "he..." {
		t.Fatalf("unexpected truncate: %q", got)
	}
	if got := truncate("hello", 3); got != "hel" {
		t.Fatalf("unexpected truncate: %q", got)
	}
}
