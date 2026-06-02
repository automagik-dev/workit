package main

import "testing"

func TestResolveServerConfigUsesDynamicPublicBaseURL(t *testing.T) {
	t.Setenv("WK_PUBLIC_BASE_URL", "https://auth.hv.example/")
	t.Setenv("WK_M365_CLIENT_ID", "m365-client")
	t.Setenv("WK_M365_TENANT_ID", "tenant-id")
	t.Setenv("WK_M365_BROKER_TOKEN", "broker-token")

	cfg := resolveServerConfig(serverConfigInput{Port: 8080})

	if cfg.publicBaseURL != "https://auth.hv.example" {
		t.Fatalf("publicBaseURL = %q", cfg.publicBaseURL)
	}
	if cfg.redirectURL != "https://auth.hv.example/callback" {
		t.Fatalf("redirectURL = %q", cfg.redirectURL)
	}
	if cfg.m365ClientID != "m365-client" || cfg.m365TenantID != "tenant-id" || cfg.m365AdminToken != "broker-token" {
		t.Fatalf("m365 config = %#v", cfg)
	}
}

func TestResolveServerConfigKeepsExplicitRedirectURL(t *testing.T) {
	t.Setenv("WK_PUBLIC_BASE_URL", "https://auth.hv.example")

	cfg := resolveServerConfig(serverConfigInput{Port: 8080, RedirectURL: "https://custom.example/callback"})

	if cfg.redirectURL != "https://custom.example/callback" {
		t.Fatalf("redirectURL = %q", cfg.redirectURL)
	}
}
