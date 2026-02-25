package config

// Build-time injected defaults via -ldflags.
// Example:
//
//	go build -ldflags "\
//	  -X 'github.com/namastexlabs/workit/internal/config.DefaultClientID=...' \
//	  -X 'github.com/namastexlabs/workit/internal/config.DefaultClientSecret=...' \
//	  -X 'github.com/namastexlabs/workit/internal/config.DefaultCallbackServer=...'"
var (
	DefaultClientID       string
	DefaultClientSecret   string
	DefaultCallbackServer string
)
