package config

// Build-time injected defaults via -ldflags.
// Example:
//
//	go build -ldflags "\
//	  -X 'github.com/namastexlabs/gog-cli/internal/config.DefaultClientID=...' \
//	  -X 'github.com/namastexlabs/gog-cli/internal/config.DefaultClientSecret=...' \
//	  -X 'github.com/namastexlabs/gog-cli/internal/config.DefaultCallbackServer=...'"
var (
	DefaultClientID       string
	DefaultClientSecret   string
	DefaultCallbackServer string
)
