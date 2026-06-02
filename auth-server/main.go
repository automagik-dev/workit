// Package main implements the OAuth callback server for headless authentication.
// This server receives OAuth callbacks from Google, exchanges authorization codes
// for tokens, and holds them temporarily for CLI retrieval.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// DefaultPort is the default port the server listens on.
	DefaultPort = 8080
	// DefaultTTL is the default time-to-live for stored tokens.
	DefaultTTL = 15 * time.Minute
	// CleanupInterval is how often to run the token cleanup routine.
	CleanupInterval = 1 * time.Minute
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", DefaultPort, "Port to listen on")
	clientID := flag.String("client-id", "", "OAuth client ID")
	clientSecret := flag.String("client-secret", "", "OAuth client secret")
	redirectURL := flag.String("redirect-url", "", "OAuth redirect URL (defaults to http://localhost:{port}/callback)")
	publicBaseURL := flag.String("public-base-url", "", "Public HTTPS base URL for this auth server (env WK_PUBLIC_BASE_URL; defaults redirect URLs when set)")
	m365ClientID := flag.String("m365-client-id", "", "Microsoft 365 OAuth client ID (env WK_M365_CLIENT_ID)")
	m365TenantID := flag.String("m365-tenant-id", "", "Microsoft 365 tenant ID (env WK_M365_TENANT_ID; default organizations)")
	credentialsFile := flag.String("credentials-file", "", "Path to OAuth credentials JSON file (workit format)")
	ttl := flag.Duration("ttl", DefaultTTL, "Token time-to-live")
	flag.Parse()

	resolved := resolveServerConfig(serverConfigInput{
		Port:          *port,
		ClientID:      *clientID,
		ClientSecret:  *clientSecret,
		RedirectURL:   *redirectURL,
		PublicBaseURL: *publicBaseURL,
		M365ClientID:  *m365ClientID,
		M365TenantID:  *m365TenantID,
	})
	*clientID = resolved.clientID
	*clientSecret = resolved.clientSecret
	*redirectURL = resolved.redirectURL
	*publicBaseURL = resolved.publicBaseURL
	*m365ClientID = resolved.m365ClientID
	*m365TenantID = resolved.m365TenantID

	// Load credentials from file if specified (fills empty client ID/secret)
	if *credentialsFile != "" {
		creds, err := loadCredentialsFile(*credentialsFile)
		if err != nil {
			log.Fatalf("Failed to load credentials from %s: %v", *credentialsFile, err)
		}
		if *clientID == "" {
			*clientID = creds.clientID
		}
		if *clientSecret == "" {
			*clientSecret = creds.clientSecret
		}
		log.Printf("Loaded credentials from %s", *credentialsFile)
	}

	// Validate required configuration. Google OAuth is optional when the pod is deployed
	// as an M365-only enterprise broker.
	if *clientID == "" && *m365ClientID == "" {
		log.Fatal("OAuth client ID is required (--client-id, WK_CLIENT_ID, --m365-client-id, or WK_M365_CLIENT_ID)")
	}
	if *clientID != "" && *clientSecret == "" {
		log.Fatal("OAuth client secret is required for Google OAuth (--client-secret, WK_CLIENT_SECRET, or --credentials-file)")
	}
	if *m365ClientID != "" && *publicBaseURL == "" {
		log.Fatal("Public base URL is required for M365 broker (--public-base-url, WK_PUBLIC_BASE_URL, or WK_CALLBACK_SERVER)")
	}

	// Default redirect URL if not specified
	if *redirectURL == "" {
		*redirectURL = fmt.Sprintf("http://localhost:%d/callback", *port)
	}

	// Create token store with TTL and start cleanup
	store := NewTokenStore(*ttl)
	store.StartCleanup(CleanupInterval)
	defer store.StopCleanup()

	// Create server
	server := NewServerWithOptions(ServerOptions{
		Store:              store,
		GoogleClientID:     *clientID,
		GoogleClientSecret: *clientSecret,
		GoogleRedirectURL:  *redirectURL,
		M365Enabled:        *m365ClientID != "",
		M365ClientID:       *m365ClientID,
		M365TenantID:       *m365TenantID,
		PublicBaseURL:      *publicBaseURL,
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		if err := httpServer.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
		close(done)
	}()

	// Start server
	log.Printf("Auth callback server starting on port %d", *port)
	log.Printf("Redirect URL: %s", *redirectURL)
	if *publicBaseURL != "" {
		log.Printf("Public base URL: %s", *publicBaseURL)
	}
	if *m365ClientID != "" {
		log.Printf("M365 broker enabled for tenant: %s", firstNonEmpty(*m365TenantID, defaultM365TenantID))
	}
	log.Printf("Token TTL: %s", *ttl)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	<-done
	log.Println("Server stopped")
}

type oauthCredentials struct {
	clientID     string
	clientSecret string
}

type credentialsFileFormat struct {
	Installed *struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"installed"`
	Web *struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"web"`
}

func loadCredentialsFile(path string) (oauthCredentials, error) {
	data, err := os.ReadFile(path) //nolint:gosec // credentials file path from flag
	if err != nil {
		return oauthCredentials{}, fmt.Errorf("read file: %w", err)
	}

	var f credentialsFileFormat
	if err := json.Unmarshal(data, &f); err != nil {
		return oauthCredentials{}, fmt.Errorf("parse JSON: %w", err)
	}

	if f.Web != nil && f.Web.ClientID != "" {
		return oauthCredentials{clientID: f.Web.ClientID, clientSecret: f.Web.ClientSecret}, nil
	}

	if f.Installed != nil && f.Installed.ClientID != "" {
		return oauthCredentials{clientID: f.Installed.ClientID, clientSecret: f.Installed.ClientSecret}, nil
	}

	return oauthCredentials{}, fmt.Errorf("no client credentials found in file")
}
