// Package main implements the OAuth callback server for headless authentication.
// This server receives OAuth callbacks from Google, exchanges authorization codes
// for tokens, and holds them temporarily for CLI retrieval.
package main

import (
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
	ttl := flag.Duration("ttl", DefaultTTL, "Token time-to-live")
	flag.Parse()

	// Allow environment variables to override flags
	if *clientID == "" {
		*clientID = os.Getenv("GOG_CLIENT_ID")
	}
	if *clientSecret == "" {
		*clientSecret = os.Getenv("GOG_CLIENT_SECRET")
	}
	if *redirectURL == "" {
		*redirectURL = os.Getenv("GOG_REDIRECT_URL")
	}

	// Validate required configuration
	if *clientID == "" {
		log.Fatal("OAuth client ID is required (--client-id or GOG_CLIENT_ID)")
	}
	if *clientSecret == "" {
		log.Fatal("OAuth client secret is required (--client-secret or GOG_CLIENT_SECRET)")
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
	server := NewServer(store, *clientID, *clientSecret, *redirectURL)

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
	log.Printf("Token TTL: %s", *ttl)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	<-done
	log.Println("Server stopped")
}
