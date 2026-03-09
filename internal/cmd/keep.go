package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	keepapi "google.golang.org/api/keep/v1"

	"github.com/automagik-dev/workit/internal/config"
	"github.com/automagik-dev/workit/internal/googleapi"
	"github.com/automagik-dev/workit/internal/outfmt"
	"github.com/automagik-dev/workit/internal/ui"
)

var newKeepServiceWithSA = googleapi.NewKeepWithServiceAccount

type KeepCmd struct {
	ServiceAccount string `name:"service-account" help:"Path to service account JSON file"`
	Impersonate    string `name:"impersonate" help:"Email to impersonate (required with service-account)"`

	List        KeepListCmd        `cmd:"" default:"withargs" help:"List notes"`
	Get         KeepGetCmd         `cmd:"" name:"get" help:"Get a note"`
	Search      KeepSearchCmd      `cmd:"" name:"search" help:"Search notes by text (client-side)"`
	Attachment  KeepAttachmentCmd  `cmd:"" name:"attachment" help:"Download an attachment"`
	Create      KeepCreateCmd      `cmd:"" name:"create" help:"Create a note"`
	Delete      KeepDeleteCmd      `cmd:"" name:"delete" help:"Delete a note"`
	Permissions KeepPermissionsCmd `cmd:"" name:"permissions" help:"Manage note permissions"`
}

type KeepListCmd struct {
	Max       int64  `name:"max" aliases:"limit" help:"Max results" default:"100"`
	Page      string `name:"page" aliases:"cursor" help:"Page token"`
	All       bool   `name:"all" aliases:"all-pages,allpages" help:"Fetch all pages"`
	FailEmpty bool   `name:"fail-empty" aliases:"non-empty,require-results" help:"Exit with code 3 if no results"`
	Filter    string `name:"filter" help:"Filter expression (e.g. 'create_time > \"2024-01-01T00:00:00Z\"')"`
}

func (c *KeepListCmd) Run(ctx context.Context, flags *RootFlags, keep *KeepCmd) error {
	u := ui.FromContext(ctx)

	svc, err := getKeepService(ctx, flags, keep)
	if err != nil {
		return err
	}

	effectiveMax, effectivePage := applyPagination(flags, c.Max, c.Page)

	fetch := func(pageToken string) ([]*keepapi.Note, string, error) {
		call := svc.Notes.List().PageSize(effectiveMax).Context(ctx)
		if strings.TrimSpace(pageToken) != "" {
			call = call.PageToken(pageToken)
		}
		if strings.TrimSpace(c.Filter) != "" {
			call = call.Filter(strings.TrimSpace(c.Filter))
		}
		resp, callErr := call.Do()
		if callErr != nil {
			return nil, "", callErr
		}
		return resp.Notes, resp.NextPageToken, nil
	}

	var notes []*keepapi.Note
	nextPageToken := ""
	if c.All {
		all, err := collectAllPages(effectivePage, fetch)
		if err != nil {
			return err
		}
		notes = all
	} else {
		var err error
		notes, nextPageToken, err = fetch(effectivePage)
		if err != nil {
			return err
		}
	}

	if outfmt.IsJSON(ctx) {
		if err := outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"notes":         notes,
			"nextPageToken": nextPageToken,
		}); err != nil {
			return err
		}
		if len(notes) == 0 {
			return failEmptyExit(c.FailEmpty)
		}
		return nil
	}

	if len(notes) == 0 {
		u.Err().Println("No notes")
		return failEmptyExit(c.FailEmpty)
	}

	w, flush := tableWriter(ctx)
	defer flush()
	fmt.Fprintln(w, "NAME\tTITLE\tUPDATED")
	for _, n := range notes {
		title := n.Title
		if title == "" {
			title = noteSnippet(n)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", n.Name, title, n.UpdateTime)
	}
	printNextPageHint(u, nextPageToken)
	return nil
}

func noteSnippet(n *keepapi.Note) string {
	if n.Body == nil || n.Body.Text == nil {
		return "(no content)"
	}
	text := n.Body.Text.Text
	if len(text) > 50 {
		text = text[:50] + "..."
	}
	text = strings.ReplaceAll(text, "\n", " ")
	return text
}

func noteContains(n *keepapi.Note, query string) bool {
	query = strings.ToLower(query)
	if strings.Contains(strings.ToLower(n.Title), query) {
		return true
	}
	if n.Body != nil && n.Body.Text != nil {
		if strings.Contains(strings.ToLower(n.Body.Text.Text), query) {
			return true
		}
	}
	return false
}

type KeepSearchCmd struct {
	Query string `arg:"" name:"query" help:"Text to search for in title and body"`
	Max   int64  `name:"max" aliases:"limit" help:"Max results to fetch before filtering" default:"500"`
}

func (c *KeepSearchCmd) Run(ctx context.Context, flags *RootFlags, keep *KeepCmd) error {
	u := ui.FromContext(ctx)

	if strings.TrimSpace(c.Query) == "" {
		return fmt.Errorf("search query cannot be empty")
	}

	svc, err := getKeepService(ctx, flags, keep)
	if err != nil {
		return err
	}

	effectiveMaxSearch, _ := applyPagination(flags, c.Max, "")

	fetch := func(pageToken string) ([]*keepapi.Note, string, error) {
		call := svc.Notes.List().PageSize(effectiveMaxSearch).Context(ctx)
		if strings.TrimSpace(pageToken) != "" {
			call = call.PageToken(pageToken)
		}
		resp, callErr := call.Do()
		if callErr != nil {
			return nil, "", callErr
		}

		matches := make([]*keepapi.Note, 0, len(resp.Notes))
		for _, n := range resp.Notes {
			if noteContains(n, c.Query) {
				matches = append(matches, n)
			}
		}
		return matches, resp.NextPageToken, nil
	}

	allNotes, err := collectAllPages("", fetch)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"notes": allNotes,
			"query": c.Query,
			"count": len(allNotes),
		})
	}

	if len(allNotes) == 0 {
		u.Err().Printf("No notes matching %q", c.Query)
		return nil
	}

	w, flush := tableWriter(ctx)
	defer flush()
	fmt.Fprintln(w, "NAME\tTITLE\tUPDATED")
	for _, n := range allNotes {
		title := n.Title
		if title == "" {
			title = noteSnippet(n)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", n.Name, title, n.UpdateTime)
	}
	u.Err().Printf("Found %d notes matching %q", len(allNotes), c.Query)
	return nil
}

type KeepGetCmd struct {
	NoteID string `arg:"" name:"noteId" help:"Note ID or name (e.g. notes/abc123)"`
}

func (c *KeepGetCmd) Run(ctx context.Context, flags *RootFlags, keep *KeepCmd) error {
	u := ui.FromContext(ctx)

	svc, err := getKeepService(ctx, flags, keep)
	if err != nil {
		return err
	}

	name := c.NoteID
	if !strings.HasPrefix(name, "notes/") {
		name = "notes/" + name
	}

	note, err := svc.Notes.Get(name).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"note": note})
	}

	u.Out().Printf("name\t%s", note.Name)
	u.Out().Printf("title\t%s", note.Title)
	u.Out().Printf("created\t%s", note.CreateTime)
	u.Out().Printf("updated\t%s", note.UpdateTime)
	u.Out().Printf("trashed\t%v", note.Trashed)
	if note.Body != nil && note.Body.Text != nil {
		u.Out().Println("")
		u.Out().Println(note.Body.Text.Text)
	}
	if len(note.Attachments) > 0 {
		u.Out().Println("")
		u.Out().Printf("attachments\t%d", len(note.Attachments))
		for _, a := range note.Attachments {
			u.Out().Printf("  %s\t%s", a.Name, a.MimeType)
		}
	}
	return nil
}

type KeepAttachmentCmd struct {
	AttachmentName string `arg:"" name:"attachmentName" help:"Attachment name (e.g. notes/abc123/attachments/xyz789)"`
	MimeType       string `name:"mime-type" help:"MIME type of attachment (e.g. image/jpeg)" default:"application/octet-stream"`
	Out            string `name:"out" help:"Output file path (default: attachment filename or ID)"`
}

func (c *KeepAttachmentCmd) Run(ctx context.Context, flags *RootFlags, keep *KeepCmd) error {
	u := ui.FromContext(ctx)

	name := strings.TrimSpace(c.AttachmentName)
	if !strings.Contains(name, "/attachments/") {
		return fmt.Errorf("invalid attachment name format, expected: notes/<noteId>/attachments/<attachmentId>")
	}

	outPath := strings.TrimSpace(c.Out)
	if outPath == "" {
		parts := strings.Split(name, "/")
		outPath = parts[len(parts)-1]
	}
	var err error
	outPath, err = config.ExpandPath(outPath)
	if err != nil {
		return err
	}

	// Avoid touching auth/keyring and avoid writing files in dry-run mode.
	if dryRunErr := dryRunExit(ctx, flags, "keep.attachment.download", map[string]any{
		"attachment_name": name,
		"mime_type":       strings.TrimSpace(c.MimeType),
		"out":             outPath,
	}); dryRunErr != nil {
		return dryRunErr
	}

	svc, err := getKeepService(ctx, flags, keep)
	if err != nil {
		return err
	}

	resp, err := svc.Media.Download(name).MimeType(c.MimeType).Download()
	if err != nil {
		return fmt.Errorf("download attachment: %w", err)
	}
	defer resp.Body.Close()

	if dir := filepath.Dir(outPath); dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0o700); mkdirErr != nil && !os.IsExist(mkdirErr) {
			return fmt.Errorf("create output directory: %w", mkdirErr)
		}
	}

	f, err := os.Create(outPath) //nolint:gosec // user-provided output path
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("write attachment: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"downloaded": true,
			"path":       outPath,
			"bytes":      written,
		})
	}

	u.Out().Printf("path\t%s", outPath)
	u.Out().Printf("bytes\t%d", written)
	return nil
}

func getKeepService(ctx context.Context, flags *RootFlags, keepCmd *KeepCmd) (*keepapi.Service, error) {
	if keepCmd.ServiceAccount != "" {
		if keepCmd.Impersonate == "" {
			return nil, fmt.Errorf("--impersonate is required when using --service-account")
		}
		return newKeepServiceWithSA(ctx, keepCmd.ServiceAccount, keepCmd.Impersonate)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return nil, err
	}

	genericSAPath, err := config.ServiceAccountPath(account)
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(genericSAPath); statErr == nil {
		return newKeepServiceWithSA(ctx, genericSAPath, account)
	}

	saPath, err := config.KeepServiceAccountPath(account)
	if err != nil {
		return nil, err
	}

	if _, statErr := os.Stat(saPath); statErr == nil {
		return newKeepServiceWithSA(ctx, saPath, account)
	}

	legacyPath, legacyErr := config.KeepServiceAccountLegacyPath(account)
	if legacyErr == nil {
		if _, statErr := os.Stat(legacyPath); statErr == nil {
			return newKeepServiceWithSA(ctx, legacyPath, account)
		}
	}

	return nil, usage("Keep is Workspace-only and requires a service account. Configure it with: wk auth service-account set <email> --key <service-account.json> (or legacy: wk auth keep <email> --key <service-account.json>)")
}

// ---------------------------------------------------------------------------
// KeepCreateCmd
// ---------------------------------------------------------------------------

type KeepCreateCmd struct {
	Title     string `name:"title" required:"" help:"Note title"`
	Body      string `name:"body" help:"Note body text (for text notes)"`
	ListItems string `name:"list-items" help:"Comma-separated list items (for checklist notes)"`
}

func (c *KeepCreateCmd) Run(ctx context.Context, flags *RootFlags, keep *KeepCmd) error {
	u := ui.FromContext(ctx)

	title := strings.TrimSpace(c.Title)
	if title == "" {
		return usage("empty title")
	}

	note := &keepapi.Note{Title: title}

	body := strings.TrimSpace(c.Body)
	listItems := strings.TrimSpace(c.ListItems)

	if body != "" && listItems != "" {
		return usage("--body and --list-items are mutually exclusive")
	}

	if body != "" {
		note.Body = &keepapi.Section{
			Text: &keepapi.TextContent{Text: body},
		}
	} else if listItems != "" {
		items := strings.Split(listItems, ",")
		listEntries := make([]*keepapi.ListItem, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			listEntries = append(listEntries, &keepapi.ListItem{
				Text: &keepapi.TextContent{Text: item},
			})
		}
		note.Body = &keepapi.Section{
			List: &keepapi.ListContent{ListItems: listEntries},
		}
	}

	if dryRunErr := dryRunExit(ctx, flags, "keep.create", map[string]any{
		"title":      title,
		"body":       body,
		"list_items": listItems,
	}); dryRunErr != nil {
		return dryRunErr
	}

	svc, err := getKeepService(ctx, flags, keep)
	if err != nil {
		return err
	}

	created, err := svc.Notes.Create(note).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{"note": created})
	}

	u.Out().Printf("name\t%s", created.Name)
	u.Out().Printf("title\t%s", created.Title)
	return nil
}

// ---------------------------------------------------------------------------
// KeepDeleteCmd
// ---------------------------------------------------------------------------

type KeepDeleteCmd struct {
	NoteID string `arg:"" name:"noteId" help:"Note ID to delete"`
}

func (c *KeepDeleteCmd) Run(ctx context.Context, flags *RootFlags, keep *KeepCmd) error {
	u := ui.FromContext(ctx)

	noteID := strings.TrimSpace(c.NoteID)
	if noteID == "" {
		return usage("empty noteId")
	}

	name := noteID
	if !strings.HasPrefix(name, "notes/") {
		name = "notes/" + name
	}

	if confirmErr := confirmDestructive(ctx, flags, fmt.Sprintf("delete keep note %s", name)); confirmErr != nil {
		return confirmErr
	}

	svc, err := getKeepService(ctx, flags, keep)
	if err != nil {
		return err
	}

	if _, err := svc.Notes.Delete(name).Context(ctx).Do(); err != nil {
		return err
	}

	return writeResult(ctx, u,
		kv("deleted", true),
		kv("name", name),
	)
}

// ---------------------------------------------------------------------------
// KeepPermissionsCmd — container for add/remove subcommands
// ---------------------------------------------------------------------------

type KeepPermissionsCmd struct {
	Add    KeepPermissionsAddCmd    `cmd:"" name:"add" help:"Share a note"`
	Remove KeepPermissionsRemoveCmd `cmd:"" name:"remove" help:"Unshare a note"`
}

// ---------------------------------------------------------------------------
// KeepPermissionsAddCmd
// ---------------------------------------------------------------------------

type KeepPermissionsAddCmd struct {
	NoteID string `arg:"" name:"noteId" help:"Note ID"`
	Email  string `name:"email" required:"" help:"Email address to share with"`
	Role   string `name:"role" default:"WRITER" help:"Permission role (WRITER)"`
}

func (c *KeepPermissionsAddCmd) Run(ctx context.Context, flags *RootFlags, keep *KeepCmd) error {
	u := ui.FromContext(ctx)

	noteID := strings.TrimSpace(c.NoteID)
	if noteID == "" {
		return usage("empty noteId")
	}
	email := strings.TrimSpace(c.Email)
	if email == "" {
		return usage("empty email")
	}
	role := strings.TrimSpace(c.Role)
	if role == "" {
		role = "WRITER"
	}

	parent := noteID
	if !strings.HasPrefix(parent, "notes/") {
		parent = "notes/" + parent
	}

	if dryRunErr := dryRunExit(ctx, flags, "keep.permissions.add", map[string]any{
		"noteId": parent,
		"email":  email,
		"role":   role,
	}); dryRunErr != nil {
		return dryRunErr
	}

	svc, err := getKeepService(ctx, flags, keep)
	if err != nil {
		return err
	}

	req := &keepapi.BatchCreatePermissionsRequest{
		Requests: []*keepapi.CreatePermissionRequest{
			{
				Parent: parent,
				Permission: &keepapi.Permission{
					Email: email,
					Role:  role,
				},
			},
		},
	}

	resp, err := svc.Notes.Permissions.BatchCreate(parent, req).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"noteId":      parent,
			"permissions": resp.Permissions,
		})
	}

	u.Out().Printf("noteId\t%s", parent)
	u.Out().Printf("shared_with\t%s", email)
	u.Out().Printf("role\t%s", role)
	if len(resp.Permissions) > 0 {
		u.Out().Printf("permission\t%s", resp.Permissions[0].Name)
	}
	return nil
}

// ---------------------------------------------------------------------------
// KeepPermissionsRemoveCmd
// ---------------------------------------------------------------------------

type KeepPermissionsRemoveCmd struct {
	NoteID string `arg:"" name:"noteId" help:"Note ID"`
	Email  string `name:"email" required:"" help:"Email address to remove"`
}

func (c *KeepPermissionsRemoveCmd) Run(ctx context.Context, flags *RootFlags, keep *KeepCmd) error {
	u := ui.FromContext(ctx)

	noteID := strings.TrimSpace(c.NoteID)
	if noteID == "" {
		return usage("empty noteId")
	}
	email := strings.TrimSpace(c.Email)
	if email == "" {
		return usage("empty email")
	}

	parent := noteID
	if !strings.HasPrefix(parent, "notes/") {
		parent = "notes/" + parent
	}

	if confirmErr := confirmDestructive(ctx, flags, fmt.Sprintf("remove permission for %s from %s", email, parent)); confirmErr != nil {
		return confirmErr
	}

	svc, err := getKeepService(ctx, flags, keep)
	if err != nil {
		return err
	}

	// Retrieve the note to find the permission name for the given email.
	note, err := svc.Notes.Get(parent).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("get note permissions: %w", err)
	}

	var permNames []string
	for _, p := range note.Permissions {
		if strings.EqualFold(p.Email, email) {
			permNames = append(permNames, p.Name)
		}
	}
	if len(permNames) == 0 {
		return fmt.Errorf("no permission found for %s on %s", email, parent)
	}

	req := &keepapi.BatchDeletePermissionsRequest{
		Names: permNames,
	}

	if _, err := svc.Notes.Permissions.BatchDelete(parent, req).Context(ctx).Do(); err != nil {
		return err
	}

	return writeResult(ctx, u,
		kv("removed", true),
		kv("noteId", parent),
		kv("email", email),
	)
}
