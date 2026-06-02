package main

import (
	"fmt"
	"os"
	"strings"
)

type serverConfigInput struct {
	Port           int
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	PublicBaseURL  string
	M365ClientID   string
	M365TenantID   string
	M365AdminToken string
}

type serverConfig struct {
	clientID       string
	clientSecret   string
	redirectURL    string
	publicBaseURL  string
	m365ClientID   string
	m365TenantID   string
	m365AdminToken string
}

func resolveServerConfig(input serverConfigInput) serverConfig {
	publicBaseURL := firstNonEmpty(input.PublicBaseURL, os.Getenv("WK_PUBLIC_BASE_URL"), os.Getenv("WK_CALLBACK_SERVER"))
	publicBaseURL = strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")

	redirectURL := firstNonEmpty(input.RedirectURL, os.Getenv("WK_REDIRECT_URL"))
	if redirectURL == "" && publicBaseURL != "" {
		redirectURL = publicBaseURL + "/callback"
	}
	if redirectURL == "" {
		redirectURL = fmt.Sprintf("http://localhost:%d/callback", input.Port)
	}

	return serverConfig{
		clientID:       firstNonEmpty(input.ClientID, os.Getenv("WK_CLIENT_ID")),
		clientSecret:   firstNonEmpty(input.ClientSecret, os.Getenv("WK_CLIENT_SECRET")),
		redirectURL:    redirectURL,
		publicBaseURL:  publicBaseURL,
		m365ClientID:   firstNonEmpty(input.M365ClientID, os.Getenv("WK_M365_CLIENT_ID")),
		m365TenantID:   firstNonEmpty(input.M365TenantID, os.Getenv("WK_M365_TENANT_ID")),
		m365AdminToken: firstNonEmpty(input.M365AdminToken, os.Getenv("WK_M365_BROKER_TOKEN"), os.Getenv("WK_BROKER_ADMIN_TOKEN")),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
