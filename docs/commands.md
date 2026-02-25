# Full Command Reference

> Back to [README](../README.md)

## Top-Level Aliases

These are convenient shortcuts for the most common operations:

| Alias | Expands To |
|---|---|
| `wk send` | `wk gmail send` |
| `wk ls` (or `wk list`) | `wk drive ls` |
| `wk search` (or `wk find`) | `wk drive search` |
| `wk open` (or `wk browse`) | Print a best-effort web URL for a Google URL/ID |
| `wk download` (or `wk dl`) | `wk drive download` |
| `wk upload` (or `wk up`, `wk put`) | `wk drive upload` |
| `wk login` | `wk auth add` |
| `wk logout` | `wk auth remove` |
| `wk status` (or `wk st`) | `wk auth status` |
| `wk me` | `wk people me` |
| `wk whoami` (or `wk who-am-i`) | `wk people me` |

## Flag Aliases

- `--out` also accepts `--output`.
- `--out-dir` also accepts `--output-dir` (Gmail thread attachment downloads).

---

## Authentication

```bash
wk auth credentials <path>           # Store OAuth client credentials
wk auth credentials list             # List stored OAuth client credentials
wk --client work auth credentials <path>  # Store named OAuth client credentials
wk auth add <email>                  # Authorize and store refresh token
wk auth service-account set <email> --key <path>  # Configure service account impersonation (Workspace only)
wk auth service-account status <email>            # Show service account status
wk auth service-account unset <email>             # Remove service account
wk auth keep <email> --key <path>                 # Legacy alias (Keep)
wk auth keyring [backend]            # Show/set keyring backend (auto|keychain|file)
wk auth status                       # Show current auth state/services
wk auth services                     # List available services and OAuth scopes
wk auth list                         # List stored accounts
wk auth list --check                 # Validate stored refresh tokens
wk auth remove <email>               # Remove a stored refresh token
wk auth manage                       # Open accounts manager in browser
wk auth tokens                       # Manage stored refresh tokens
```

See [docs/auth.md](auth.md) for full authentication documentation.

---

## Gmail

```bash
# Search and read
wk gmail search 'newer_than:7d' --max 10
wk gmail thread get <threadId>
wk gmail thread get <threadId> --download              # Download attachments to current dir
wk gmail thread get <threadId> --download --out-dir ./attachments
wk gmail get <messageId>
wk gmail get <messageId> --format metadata
wk gmail attachment <messageId> <attachmentId>
wk gmail attachment <messageId> <attachmentId> --out ./attachment.bin
wk gmail url <threadId>              # Print Gmail web URL
wk gmail thread modify <threadId> --add STARRED --remove INBOX

# Send and compose
wk gmail send --to a@b.com --subject "Hi" --body "Plain fallback"
wk gmail send --to a@b.com --subject "Hi" --body-file ./message.txt
wk gmail send --to a@b.com --subject "Hi" --body-file -   # Read body from stdin
wk gmail send --to a@b.com --subject "Hi" --body "Plain fallback" --body-html "<p>Hello</p>"
# Reply + include quoted original message (auto-generates HTML quote unless you pass --body-html)
wk gmail send --reply-to-message-id <messageId> --quote --to a@b.com --subject "Re: Hi" --body "My reply"

# Message-level search (one row per email; add --include-body to fetch/decode bodies)
wk gmail messages search 'newer_than:7d' --max 3
wk gmail messages search 'newer_than:7d' --max 1 --include-body --json

# Drafts
wk gmail drafts list
wk gmail drafts create --subject "Draft" --body "Body"
wk gmail drafts create --to a@b.com --subject "Draft" --body "Body"
wk gmail drafts update <draftId> --subject "Draft" --body "Body"
wk gmail drafts update <draftId> --to a@b.com --subject "Draft" --body "Body"
wk gmail drafts send <draftId>

# Labels
wk gmail labels list
wk gmail labels get INBOX --json  # Includes message counts
wk gmail labels create "My Label"
wk gmail labels modify <threadId> --add STARRED --remove INBOX
wk gmail labels delete <labelIdOrName>  # Deletes user label (guards system labels; confirm)

# Batch operations
wk gmail batch delete <messageId> <messageId>
wk gmail batch modify <messageId> <messageId> --add STARRED --remove INBOX

# Filters
wk gmail filters list
wk gmail filters create --from 'noreply@example.com' --add-label 'Notifications'
wk gmail filters delete <filterId>

# Settings
wk gmail autoforward get
wk gmail autoforward enable --email forward@example.com
wk gmail autoforward disable
wk gmail forwarding list
wk gmail forwarding add --email forward@example.com
wk gmail sendas list
wk gmail sendas create --email alias@example.com
wk gmail vacation get
wk gmail vacation enable --subject "Out of office" --message "..."
wk gmail vacation disable

# Delegation (Google Workspace)
wk gmail delegates list
wk gmail delegates add --email delegate@example.com
wk gmail delegates remove --email delegate@example.com

# Watch (Pub/Sub push)
wk gmail watch start --topic projects/<p>/topics/<t> --label INBOX
wk gmail watch serve --bind 127.0.0.1 --token <shared> --hook-url http://127.0.0.1:18789/hooks/agent
wk gmail watch serve --bind 0.0.0.0 --verify-oidc --oidc-email <svc@...> --hook-url <url>
wk gmail watch serve --bind 127.0.0.1 --token <shared> --exclude-labels SPAM,TRASH --hook-url http://127.0.0.1:18789/hooks/agent
wk gmail history --since <historyId>
```

Gmail watch (Pub/Sub push):
- Create Pub/Sub topic + push subscription (OIDC preferred; shared token ok for dev).
- Full flow + payload details: [docs/watch.md](watch.md).
- `watch serve --exclude-labels` defaults to `SPAM,TRASH`; IDs are case-sensitive.

---

## Email Tracking

Track when recipients open your emails:

```bash
# Set up local tracking config (per-account; generates keys; follow printed deploy steps)
wk gmail track setup --worker-url https://wk-email-tracker.<acct>.workers.dev

# Send with tracking
wk gmail send --to recipient@example.com --subject "Hello" --body-html "<p>Hi!</p>" --track

# Check opens
wk gmail track opens <tracking_id>
wk gmail track opens --to recipient@example.com

# View status
wk gmail track status
```

Docs: [docs/email-tracking.md](email-tracking.md) (setup/deploy) + [docs/email-tracking-worker.md](email-tracking-worker.md) (internals).

**Notes:** `--track` requires exactly 1 recipient (no cc/bcc) and an HTML body (`--body-html` or `--quote`). Use `--track-split` to send per-recipient messages with individual tracking ids. The tracking worker stores IP/user-agent + coarse geo by default.

---

## Calendar

```bash
# Calendars
wk calendar calendars
wk calendar acl <calendarId>         # List access control rules
wk calendar colors                   # List available event/calendar colors
wk calendar time --timezone America/New_York
wk calendar users                    # List workspace users (use email as calendar ID)

# Events (with timezone-aware time flags)
wk calendar events <calendarId> --today                    # Today's events
wk calendar events <calendarId> --tomorrow                 # Tomorrow's events
wk calendar events <calendarId> --week                     # This week (Mon-Sun by default; use --week-start)
wk calendar events <calendarId> --days 3                   # Next 3 days
wk calendar events <calendarId> --from today --to friday   # Relative dates
wk calendar events <calendarId> --from today --to friday --weekday   # Include weekday columns
wk calendar events <calendarId> --from 2025-01-01T00:00:00Z --to 2025-01-08T00:00:00Z
wk calendar events --all             # Fetch events from all calendars
wk calendar event <calendarId> <eventId>
wk calendar get <calendarId> <eventId>                     # Alias for event
wk calendar search "meeting" --today
wk calendar search "meeting" --tomorrow
wk calendar search "meeting" --days 365
wk calendar search "meeting" --from 2025-01-01T00:00:00Z --to 2025-01-31T00:00:00Z --max 50

# Search defaults to 30 days ago through 90 days ahead unless you set --from/--to/--today/--week/--days.
# Tip: set WK_CALENDAR_WEEKDAY=1 to default --weekday for calendar events output.

# JSON event output includes timezone and localized times (useful for agents).
wk calendar get <calendarId> <eventId> --json
# {
#   "event": {
#     "id": "...",
#     "summary": "...",
#     "startDayOfWeek": "Friday",
#     "endDayOfWeek": "Friday",
#     "timezone": "America/Los_Angeles",
#     "eventTimezone": "America/New_York",
#     "startLocal": "2026-01-23T20:45:00-08:00",
#     "endLocal": "2026-01-23T22:45:00-08:00",
#     "start": { "dateTime": "2026-01-23T23:45:00-05:00" },
#     "end": { "dateTime": "2026-01-24T01:45:00-05:00" }
#   }
# }

# Team calendars (requires Cloud Identity API for Google Workspace)
wk calendar team <group-email> --today           # Show team's events for today
wk calendar team <group-email> --week            # Show team's events for the week (use --week-start)
wk calendar team <group-email> --freebusy        # Show only busy/free blocks (faster)
wk calendar team <group-email> --query "standup" # Filter by event title

# Create and update
wk calendar create <calendarId> \
  --summary "Meeting" \
  --from 2025-01-15T10:00:00Z \
  --to 2025-01-15T11:00:00Z

wk calendar create <calendarId> \
  --summary "Team Sync" \
  --from 2025-01-15T14:00:00Z \
  --to 2025-01-15T15:00:00Z \
  --attendees "alice@example.com,bob@example.com" \
  --location "Zoom"

wk calendar update <calendarId> <eventId> \
  --summary "Updated Meeting" \
  --from 2025-01-15T11:00:00Z \
  --to 2025-01-15T12:00:00Z

# Send notifications when creating/updating
wk calendar create <calendarId> \
  --summary "Team Sync" \
  --from 2025-01-15T14:00:00Z \
  --to 2025-01-15T15:00:00Z \
  --send-updates all

wk calendar update <calendarId> <eventId> \
  --send-updates externalOnly

# Default: no attendee notifications unless you pass --send-updates.
wk calendar delete <calendarId> <eventId> \
  --send-updates all --force

# Recurrence + reminders
wk calendar create <calendarId> \
  --summary "Payment" \
  --from 2025-02-11T09:00:00-03:00 \
  --to 2025-02-11T09:15:00-03:00 \
  --rrule "RRULE:FREQ=MONTHLY;BYMONTHDAY=11" \
  --reminder "email:3d" \
  --reminder "popup:30m"

# Special event types via --event-type (focus-time/out-of-office/working-location)
wk calendar create primary \
  --event-type focus-time \
  --from 2025-01-15T13:00:00Z \
  --to 2025-01-15T14:00:00Z

wk calendar create primary \
  --event-type out-of-office \
  --from 2025-01-20 \
  --to 2025-01-21 \
  --all-day

wk calendar create primary \
  --event-type working-location \
  --working-location-type office \
  --working-office-label "HQ" \
  --from 2025-01-22 \
  --to 2025-01-23

# Dedicated shortcuts (same event types, more opinionated defaults)
wk calendar focus-time --from 2025-01-15T13:00:00Z --to 2025-01-15T14:00:00Z
wk calendar out-of-office --from 2025-01-20 --to 2025-01-21 --all-day
wk calendar working-location --type office --office-label "HQ" --from 2025-01-22 --to 2025-01-23

# Add attendees without replacing existing attendees/RSVP state
wk calendar update <calendarId> <eventId> \
  --add-attendee "alice@example.com,bob@example.com"

wk calendar delete <calendarId> <eventId>

# Invitations
wk calendar respond <calendarId> <eventId> --status accepted
wk calendar respond <calendarId> <eventId> --status declined
wk calendar respond <calendarId> <eventId> --status tentative
wk calendar respond <calendarId> <eventId> --status declined --send-updates externalOnly

# Propose a new time (browser-only flow; API limitation)
wk calendar propose-time <calendarId> <eventId>
wk calendar propose-time <calendarId> <eventId> --open
wk calendar propose-time <calendarId> <eventId> --decline --comment "Can we do 5pm?"

# Availability
wk calendar freebusy --calendars "primary,work@example.com" \
  --from 2025-01-15T00:00:00Z \
  --to 2025-01-16T00:00:00Z

wk calendar conflicts --calendars "primary,work@example.com" \
  --today                             # Today's conflicts
```

---

## Time

```bash
wk time now
wk time now --timezone UTC
```

---

## Drive

```bash
# List and search
wk drive ls --max 20
wk drive ls --parent <folderId> --max 20
wk drive ls --no-all-drives            # Only list from "My Drive"
wk drive search "invoice" --max 20
wk drive search "invoice" --no-all-drives
wk drive search "mimeType = 'application/pdf'" --raw-query
wk drive get <fileId>                # Get file metadata
wk drive url <fileId>                # Print Drive web URL
wk drive copy <fileId> "Copy Name"

# Upload and download
wk drive upload ./path/to/file --parent <folderId>
wk drive upload ./path/to/file --replace <fileId>  # Replace file content in-place (preserves shared link)
wk drive upload ./report.docx --convert
wk drive upload ./chart.png --convert-to sheet
wk drive upload ./report.docx --convert --name report.docx
wk drive download <fileId> --out ./downloaded.bin
wk drive download <fileId> --format pdf --out ./exported.pdf     # Google Workspace files only
wk drive download <fileId> --format docx --out ./doc.docx
wk drive download <fileId> --format pptx --out ./slides.pptx

# Organize
wk drive mkdir "New Folder"
wk drive mkdir "New Folder" --parent <parentFolderId>
wk drive rename <fileId> "New Name"
wk drive move <fileId> --parent <destinationFolderId>
wk drive delete <fileId>             # Move to trash
wk drive delete <fileId> --permanent # Permanently delete

# Permissions
wk drive permissions <fileId>
wk drive share <fileId> --to user --email user@example.com --role reader
wk drive share <fileId> --to user --email user@example.com --role writer
wk drive share <fileId> --to domain --domain example.com --role reader
wk drive unshare <fileId> --permission-id <permissionId>

# Shared drives (Team Drives)
wk drive drives --max 100
```

---

## Docs

```bash
wk docs info <docId>
wk docs cat <docId> --max-bytes 10000
wk docs create "My Doc"
wk docs create "My Doc" --file ./doc.md            # Import markdown
wk docs copy <docId> "My Doc Copy"
wk docs export <docId> --format pdf --out ./doc.pdf
wk docs export <docId> --format docx --out ./doc.docx
wk docs export <docId> --format txt --out ./doc.txt
wk docs list-tabs <docId>
wk docs cat <docId> --tab "Notes"
wk docs cat <docId> --all-tabs
wk docs update <docId> --format markdown --content-file ./doc.md
wk docs write <docId> --replace --markdown --file ./doc.md
wk docs find-replace <docId> "old" "new"
```

---

## Slides

```bash
wk slides info <presentationId>
wk slides create "My Deck"
wk slides create-from-markdown "My Deck" --content-file ./slides.md
wk slides copy <presentationId> "My Deck Copy"
wk slides export <presentationId> --format pdf --out ./deck.pdf
wk slides export <presentationId> --format pptx --out ./deck.pptx
wk slides list-slides <presentationId>
wk slides add-slide <presentationId> ./slide.png --notes "Speaker notes"
wk slides update-notes <presentationId> <slideId> --notes "Updated notes"
wk slides replace-slide <presentationId> <slideId> ./new-slide.png --notes "New notes"
```

---

## Sheets

```bash
# Read
wk sheets metadata <spreadsheetId>
wk sheets get <spreadsheetId> 'Sheet1!A1:B10'

# Export (via Drive)
wk sheets export <spreadsheetId> --format pdf --out ./sheet.pdf
wk sheets export <spreadsheetId> --format xlsx --out ./sheet.xlsx
wk sheets copy <spreadsheetId> "My Sheet Copy"

# Write
wk sheets update <spreadsheetId> 'A1' 'val1|val2,val3|val4'
wk sheets update <spreadsheetId> 'A1' --values-json '[["a","b"],["c","d"]]'
wk sheets update <spreadsheetId> 'Sheet1!A1:C1' 'new|row|data' --copy-validation-from 'Sheet1!A2:C2'
wk sheets append <spreadsheetId> 'Sheet1!A:C' 'new|row|data'
wk sheets append <spreadsheetId> 'Sheet1!A:C' 'new|row|data' --copy-validation-from 'Sheet1!A2:C2'
wk sheets clear <spreadsheetId> 'Sheet1!A1:B10'

# Format
wk sheets format <spreadsheetId> 'Sheet1!A1:B2' --format-json '{"textFormat":{"bold":true}}' --format-fields 'userEnteredFormat.textFormat.bold'

# Create
wk sheets create "My New Spreadsheet" --sheets "Sheet1,Sheet2"
```

---

## Contacts

```bash
# Personal contacts
wk contacts list --max 50
wk contacts search "Ada" --max 50
wk contacts get people/<resourceName>
wk contacts get user@example.com     # Get by email

# Other contacts (people you've interacted with)
wk contacts other list --max 50
wk contacts other search "John" --max 50

# Create and update
wk contacts create \
  --given "John" \
  --family "Doe" \
  --email "john@example.com" \
  --phone "+1234567890"

wk contacts update people/<resourceName> \
  --given "Jane" \
  --email "jane@example.com" \
  --birthday "1990-05-12" \
  --notes "Met at WWDC"

# Update via JSON (see docs/contacts-json-update.md)
wk contacts get people/<resourceName> --json | \
  jq '(.contact.urls //= []) | (.contact.urls += [{"value":"obsidian://open?vault=notes&file=People/John%20Doe","type":"profile"}])' | \
  wk contacts update people/<resourceName> --from-file -

wk contacts delete people/<resourceName>

# Workspace directory (requires Google Workspace)
wk contacts directory list --max 50
wk contacts directory search "Jane" --max 50
```

See [docs/contacts-json-update.md](contacts-json-update.md) for JSON update details.

---

## Tasks

```bash
# Task lists
wk tasks lists --max 50
wk tasks lists create <title>

# Tasks in a list
wk tasks list <tasklistId> --max 50
wk tasks get <tasklistId> <taskId>
wk tasks add <tasklistId> --title "Task title"
wk tasks add <tasklistId> --title "Weekly sync" --due 2025-02-01 --repeat weekly --repeat-count 4
wk tasks add <tasklistId> --title "Daily standup" --due 2025-02-01 --repeat daily --repeat-until 2025-02-05
wk tasks update <tasklistId> <taskId> --title "New title"
wk tasks done <tasklistId> <taskId>
wk tasks undo <tasklistId> <taskId>
wk tasks delete <tasklistId> <taskId>
wk tasks clear <tasklistId>
```

Note: Google Tasks treats due dates as date-only; time components may be ignored. See [docs/dates.md](dates.md) for all supported date/time input formats across commands.

---

## People

```bash
# Profile
wk people me
wk people get people/<userId>

# Search the Workspace directory
wk people search "Ada Lovelace" --max 5

# Relations (defaults to people/me)
wk people relations
wk people relations people/<userId> --type manager
```

---

## Chat

```bash
# Spaces
wk chat spaces list
wk chat spaces find "Engineering"
wk chat spaces create "Engineering" --member alice@company.com --member bob@company.com

# Messages
wk chat messages list spaces/<spaceId> --max 5
wk chat messages list spaces/<spaceId> --thread <threadId>
wk chat messages list spaces/<spaceId> --unread
wk chat messages send spaces/<spaceId> --text "Build complete!" --thread spaces/<spaceId>/threads/<threadId>

# Threads
wk chat threads list spaces/<spaceId>

# Direct messages
wk chat dm space user@company.com
wk chat dm send user@company.com --text "ping"
```

Note: Chat commands require a Google Workspace account (consumer @gmail.com accounts are not supported).

---

## Groups (Google Workspace)

```bash
# List groups you belong to
wk groups list

# List members of a group
wk groups members engineering@company.com
```

Note: Groups commands require the Cloud Identity API and the `cloud-identity.groups.readonly` scope. If you get a permissions error, re-authenticate:

```bash
wk auth add your@email.com --services groups --force-consent
```

---

## Classroom (Google Workspace for Education)

```bash
# Courses
wk classroom courses list
wk classroom courses list --role teacher
wk classroom courses get <courseId>
wk classroom courses create --name "Math 101"
wk classroom courses update <courseId> --name "Math 102"
wk classroom courses archive <courseId>
wk classroom courses unarchive <courseId>
wk classroom courses url <courseId>

# Roster
wk classroom roster <courseId>
wk classroom roster <courseId> --students
wk classroom students add <courseId> <userId>
wk classroom teachers add <courseId> <userId>

# Coursework
wk classroom coursework list <courseId>
wk classroom coursework get <courseId> <courseworkId>
wk classroom coursework create <courseId> --title "Homework 1" --type ASSIGNMENT --state PUBLISHED
wk classroom coursework update <courseId> <courseworkId> --title "Updated"
wk classroom coursework assignees <courseId> <courseworkId> --mode INDIVIDUAL_STUDENTS --add-student <studentId>

# Materials
wk classroom materials list <courseId>
wk classroom materials create <courseId> --title "Syllabus" --state PUBLISHED

# Submissions
wk classroom submissions list <courseId> <courseworkId>
wk classroom submissions get <courseId> <courseworkId> <submissionId>
wk classroom submissions grade <courseId> <courseworkId> <submissionId> --grade 85
wk classroom submissions return <courseId> <courseworkId> <submissionId>
wk classroom submissions turn-in <courseId> <courseworkId> <submissionId>
wk classroom submissions reclaim <courseId> <courseworkId> <submissionId>

# Announcements
wk classroom announcements list <courseId>
wk classroom announcements create <courseId> --text "Welcome!"
wk classroom announcements update <courseId> <announcementId> --text "Updated"
wk classroom announcements assignees <courseId> <announcementId> --mode INDIVIDUAL_STUDENTS --add-student <studentId>

# Topics
wk classroom topics list <courseId>
wk classroom topics create <courseId> --name "Unit 1"
wk classroom topics update <courseId> <topicId> --name "Unit 2"

# Invitations
wk classroom invitations list
wk classroom invitations create <courseId> <userId> --role student
wk classroom invitations accept <invitationId>

# Guardians
wk classroom guardians list <studentId>
wk classroom guardians get <studentId> <guardianId>
wk classroom guardians delete <studentId> <guardianId>

# Guardian invitations
wk classroom guardian-invitations list <studentId>
wk classroom guardian-invitations create <studentId> --email parent@example.com

# Profiles
wk classroom profile get
wk classroom profile get <userId>
```

Note: Classroom commands require a Google Workspace for Education account. Personal Google accounts have limited Classroom functionality.

---

## Keep (Workspace only)

```bash
wk keep list --account you@yourdomain.com
wk keep get <noteId> --account you@yourdomain.com
wk keep search <query> --account you@yourdomain.com
wk keep attachment <attachmentName> --account you@yourdomain.com --out ./attachment.bin
```

---

## Forms

```bash
# Forms
wk forms get <formId>
wk forms create --title "Weekly Check-in" --description "Friday async update"

# Responses
wk forms responses list <formId> --max 20
wk forms responses get <formId> <responseId>
```

---

## Apps Script

```bash
# Projects
wk appscript get <scriptId>
wk appscript content <scriptId>
wk appscript create --title "Automation Helpers"
wk appscript create --title "Bound Script" --parent-id <driveFileId>

# Execute functions
wk appscript run <scriptId> myFunction --params '["arg1", 123, true]'
wk appscript run <scriptId> myFunction --dev-mode
```

---

## Sync (Drive Sync)

Bidirectional folder sync like Google Drive for Desktop, with daemon mode and conflict resolution.

```bash
wk sync init --drive-folder=<folderId> <local-path>   # Initialize sync
wk sync list                                            # List all sync configurations
wk sync remove <local-path>                             # Remove a sync configuration
wk sync status                                          # Show sync status
wk sync start <local-path>                              # Start sync daemon
wk sync stop                                            # Stop sync daemon
```

See [docs/sync.md](sync.md) for full sync documentation.

---

## DOCX (Local File Operations)

Local DOCX document operations -- no Google API calls required.

### Read and inspect

```bash
wk docx cat <file>                    # Extract content as markdown
wk docx cat <file> --json             # Structured JSON output
wk docx cat <file> --structure        # Structured JSON with paragraph IDs and styles
wk docx info <file>                   # Show document metadata and structure
wk docx inspect <file>                # Inspect template for {{PLACEHOLDER}} patterns
wk docx list-comments <file>          # List all comments
```

### Edit content

```bash
wk docx replace <file> "old" "new"                    # Find and replace text
wk docx replace <file> "old" "new" --output out.docx  # Write to separate file
wk docx insert <file> --after 'heading:Summary' --text "New paragraph"
wk docx insert <file> --after 'paragraph:5' --text "Inserted text"
wk docx delete <file> --section "Section Heading"      # Delete a section by heading
wk docx style <file> --paragraph 3 --style "Heading1"  # Change paragraph style
wk docx rewrite <file> --from content.md               # Replace all body with markdown
```

### Track changes

```bash
wk docx track <file> --replace "old text" --new "new text"               # Create tracked replacement
wk docx track <file> --replace "old text" --new "new text" --author "Me" # With author attribution
wk docx accept-changes <file>          # Accept all tracked changes
wk docx reject-changes <file>          # Reject all tracked changes
```

### Comments

```bash
wk docx comment <file> --at 'paragraph:1' --text "Review this"
wk docx comment <file> --at 'paragraph:1' --text "Needs update" --author "Reviewer"
wk docx list-comments <file>
```

### Tables

```bash
wk docx table <file> --list                                  # List all tables
wk docx table <file> --id 0 --add-row "col1,col2,col3"      # Add row to table 0
wk docx table <file> --id 0 --update-cell "2,1,new value"   # Update cell at row 2, col 1
wk docx table <file> --id 0 --delete-row 3                   # Delete row 3
```

### Create and convert

```bash
wk docx create --from content.md --out output.docx                          # From markdown
wk docx create --from values.json --template template.docx --out output.docx # From template + JSON
wk docx to-pdf <file>                                                        # Convert to PDF (requires LibreOffice)
wk docx to-pdf <file> --output report.pdf                                    # Specify output path
```

All edit commands default to overwriting the input file. Use `--output` (or `-o`) to write to a separate file instead.

---

## Templates

Manage reusable DOCX document templates.

```bash
wk templates list                      # List available templates
wk templates add <name> <source>       # Add a template from a DOCX file
wk templates inspect <name>            # Inspect a template for {{PLACEHOLDER}} patterns
```

Templates are used with `wk docx create --template <name>` to generate documents from JSON key-value pairs.

---

## Examples

### Search recent emails and download attachments

```bash
wk gmail search 'newer_than:7d has:attachment' --max 10
wk gmail thread get <threadId> --download
```

### Modify labels on a thread

```bash
wk gmail thread modify <threadId> --remove INBOX --add STARRED
```

### Create a calendar event with attendees

```bash
wk calendar freebusy --calendars "primary" \
  --from 2025-01-15T00:00:00Z \
  --to 2025-01-16T00:00:00Z

wk calendar create primary \
  --summary "Team Standup" \
  --from 2025-01-15T10:00:00Z \
  --to 2025-01-15T10:30:00Z \
  --attendees "alice@example.com,bob@example.com"
```

### Find and download files from Drive

```bash
wk drive search "invoice filetype:pdf" --max 20 --json | \
  jq -r '.files[] | .id' | \
  while read fileId; do
    wk drive download "$fileId"
  done
```

### Manage multiple accounts

```bash
wk gmail search 'is:unread' --account personal@gmail.com
wk gmail search 'is:unread' --account work@company.com

export WK_ACCOUNT=work@company.com
wk gmail search 'is:unread'
```

### Update a Google Sheet from a CSV

```bash
cat data.csv | tr ',' '|' | \
  wk sheets update <spreadsheetId> 'Sheet1!A1'
```

### Export Sheets / Docs / Slides

```bash
wk sheets export <spreadsheetId> --format pdf
wk docs export <docId> --format docx
wk slides export <presentationId> --format pptx
```

### Batch process Gmail threads

```bash
# Mark all emails from a sender as read
wk --json gmail search 'from:noreply@example.com' --max 200 | \
  jq -r '.threads[].id' | \
  xargs -n 50 wk gmail labels modify --remove UNREAD

# Archive old emails
wk --json gmail search 'older_than:1y' --max 200 | \
  jq -r '.threads[].id' | \
  xargs -n 50 wk gmail labels modify --remove INBOX

# Label important emails
wk --json gmail search 'from:boss@example.com' --max 200 | \
  jq -r '.threads[].id' | \
  xargs -n 50 wk gmail labels modify --add IMPORTANT
```
