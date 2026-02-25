package config

// Build-time injected defaults via -ldflags.
// Example:
//
//	go build -ldflags "\
//	  -X 'github.com/automagik-dev/workit/internal/config.DefaultClientID=...' \
//	  -X 'github.com/automagik-dev/workit/internal/config.DefaultClientSecret=...' \
//	  -X 'github.com/automagik-dev/workit/internal/config.DefaultCallbackServer=...'"
var (
	DefaultClientID       string
	DefaultClientSecret   string
	DefaultCallbackServer string
)
