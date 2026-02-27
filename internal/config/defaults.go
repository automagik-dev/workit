package config

// Build-time injected defaults via -ldflags.
// Example:
//
//	go build -ldflags "\
//	  -X 'github.com/automagik-dev/workit/internal/config.DefaultClientID=...' \
//	  -X 'github.com/automagik-dev/workit/internal/config.DefaultClientSecret=...' \
//	  -X 'github.com/automagik-dev/workit/internal/config.DefaultCallbackServer=https://custom.example.com'"
var (
	DefaultClientID     string
	DefaultClientSecret string

	// DefaultCallbackServer is the default OAuth relay used when WK_CALLBACK_SERVER is not set.
	// Override at build time via: -ldflags "-X github.com/automagik-dev/workit/internal/config.DefaultCallbackServer=https://custom.example.com"
	DefaultCallbackServer = "https://auth.automagik.dev"
)
