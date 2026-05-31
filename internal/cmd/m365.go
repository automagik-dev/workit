package cmd

import (
	"context"
	"strings"

	"github.com/automagik-dev/workit/internal/ui"
)

type M365Cmd struct {
	Outlook  M365OutlookCmd  `cmd:"" help:"Microsoft Outlook read-only pilot commands"`
	Calendar M365CalendarCmd `cmd:"" help:"Microsoft Calendar read-only pilot commands"`
}

type M365OutlookCmd struct {
	Search  M365OutlookSearchCmd  `cmd:"" help:"Search Outlook messages (read-only pilot)"`
	Message M365OutlookMessageCmd `cmd:"" help:"Read Outlook messages (read-only pilot)"`
}

type M365OutlookSearchCmd struct {
	Query string `name:"query" help:"Microsoft Graph message search query"`
	Top   int    `name:"top" help:"Maximum messages to return" default:"10"`
}

type M365OutlookMessageCmd struct {
	Get M365OutlookMessageGetCmd `cmd:"" help:"Get an Outlook message by id (read-only pilot)"`
}

type M365OutlookMessageGetCmd struct {
	ID string `arg:"" name:"id" help:"Microsoft Graph message id"`
}

type M365CalendarCmd struct {
	Events   M365CalendarEventsCmd   `cmd:"" help:"List calendar events (read-only pilot)"`
	FreeBusy M365CalendarFreeBusyCmd `cmd:"" name:"freebusy" help:"Check free/busy availability (read-only pilot)"`
}

type M365CalendarEventsCmd struct {
	From string `name:"from" help:"Start time (RFC3339)"`
	To   string `name:"to" help:"End time (RFC3339)"`
	Top  int    `name:"top" help:"Maximum events to return" default:"10"`
}

type M365CalendarFreeBusyCmd struct {
	Users string `name:"users" help:"Comma-separated email addresses/resources"`
	From  string `name:"from" help:"Start time (RFC3339)"`
	To    string `name:"to" help:"End time (RFC3339)"`
}

func (c *M365OutlookSearchCmd) Run(ctx context.Context, flags *RootFlags) error {
	return writeM365PilotResult(ctx, flags, "m365.outlook.search", map[string]any{
		"query": strings.TrimSpace(c.Query),
		"top":   c.Top,
	})
}

func (c *M365OutlookMessageGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	return writeM365PilotResult(ctx, flags, "m365.outlook.message.get", map[string]any{
		"id": strings.TrimSpace(c.ID),
	})
}

func (c *M365CalendarEventsCmd) Run(ctx context.Context, flags *RootFlags) error {
	return writeM365PilotResult(ctx, flags, "m365.calendar.events", map[string]any{
		"from": strings.TrimSpace(c.From),
		"to":   strings.TrimSpace(c.To),
		"top":  c.Top,
	})
}

func (c *M365CalendarFreeBusyCmd) Run(ctx context.Context, flags *RootFlags) error {
	users := splitCommaList(c.Users)
	if users == nil {
		users = []string{}
	}

	return writeM365PilotResult(ctx, flags, "m365.calendar.freebusy", map[string]any{
		"users": users,
		"from":  strings.TrimSpace(c.From),
		"to":    strings.TrimSpace(c.To),
	})
}

func writeM365PilotResult(ctx context.Context, flags *RootFlags, operation string, request map[string]any) error {
	if flags == nil || !flags.ReadOnly {
		return usage("m365 pilot commands require explicit --read-only")
	}

	u := ui.FromContext(ctx)
	return writeResult(ctx, u,
		kv("operation", operation),
		kv("provider", "microsoft_graph"),
		kv("mode", "read_only_pilot"),
		kv("status", "ready_for_m365_auth"),
		kv("request", request),
	)
}
