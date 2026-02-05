package config

// Build-time injected defaults via -ldflags.
// Example:
//
//	go build -ldflags "\
//	  -X 'github.com/steipete/gogcli/internal/config.DefaultClientID=...' \
//	  -X 'github.com/steipete/gogcli/internal/config.DefaultClientSecret=...' \
//	  -X 'github.com/steipete/gogcli/internal/config.DefaultCallbackServer=...'"
var (
	DefaultClientID       string
	DefaultClientSecret   string
	DefaultCallbackServer string
)
