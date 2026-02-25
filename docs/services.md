# Service Scope Matrix

> Back to [README](../README.md)

## Overview

By default, `wk auth add` requests access to the **user** services (see `wk auth services` for the current list and scopes).

## Scope Selection

### Select specific services

```bash
wk auth add you@gmail.com --services drive,calendar
```

### Request read-only scopes

Write operations will fail with 403 insufficient scopes:

```bash
wk auth add you@gmail.com --services drive,calendar --readonly
```

### Control Drive's scope

```bash
wk auth add you@gmail.com --services drive --drive-scope full      # Default: full access
wk auth add you@gmail.com --services drive --drive-scope readonly  # Read-only
wk auth add you@gmail.com --services drive --drive-scope file      # Only files created/opened by this app
```

Notes:
- `--drive-scope readonly` is enough for listing/downloading/exporting via Drive (write operations will 403).
- `--drive-scope file` is write-capable (limited to files created/opened by this app) and can't be combined with `--readonly`.

### Re-authorize with additional services

If you need to add services later and Google doesn't return a refresh token, re-run with `--force-consent`:

```bash
wk auth add you@gmail.com --services user --force-consent
# Or add just Sheets
wk auth add you@gmail.com --services sheets --force-consent
```

`--services all` is accepted as an alias for `user` for backwards compatibility.

Docs commands are implemented via the Drive API, and `docs` requests both Drive and Docs API scopes.

## Service Scope Table

Auto-generated; run `go run scripts/gen-auth-services-md.go` to regenerate.

| Service | User | APIs | Scopes | Notes |
| --- | --- | --- | --- | --- |
| gmail | yes | Gmail API | `https://www.googleapis.com/auth/gmail.modify`<br>`https://www.googleapis.com/auth/gmail.settings.basic`<br>`https://www.googleapis.com/auth/gmail.settings.sharing` |  |
| calendar | yes | Calendar API | `https://www.googleapis.com/auth/calendar` |  |
| chat | yes | Chat API | `https://www.googleapis.com/auth/chat.spaces`<br>`https://www.googleapis.com/auth/chat.messages`<br>`https://www.googleapis.com/auth/chat.memberships`<br>`https://www.googleapis.com/auth/chat.users.readstate.readonly` |  |
| classroom | yes | Classroom API | `https://www.googleapis.com/auth/classroom.courses`<br>`https://www.googleapis.com/auth/classroom.rosters`<br>`https://www.googleapis.com/auth/classroom.coursework.students`<br>`https://www.googleapis.com/auth/classroom.coursework.me`<br>`https://www.googleapis.com/auth/classroom.courseworkmaterials`<br>`https://www.googleapis.com/auth/classroom.announcements`<br>`https://www.googleapis.com/auth/classroom.topics`<br>`https://www.googleapis.com/auth/classroom.guardianlinks.students`<br>`https://www.googleapis.com/auth/classroom.profile.emails`<br>`https://www.googleapis.com/auth/classroom.profile.photos` |  |
| drive | yes | Drive API | `https://www.googleapis.com/auth/drive` |  |
| docs | yes | Docs API, Drive API | `https://www.googleapis.com/auth/drive`<br>`https://www.googleapis.com/auth/documents` | Export/copy/create via Drive |
| slides | yes | Slides API, Drive API | `https://www.googleapis.com/auth/drive`<br>`https://www.googleapis.com/auth/presentations` | Create/edit presentations |
| contacts | yes | People API | `https://www.googleapis.com/auth/contacts`<br>`https://www.googleapis.com/auth/contacts.other.readonly`<br>`https://www.googleapis.com/auth/directory.readonly` | Contacts + other contacts + directory |
| tasks | yes | Tasks API | `https://www.googleapis.com/auth/tasks` |  |
| sheets | yes | Sheets API, Drive API | `https://www.googleapis.com/auth/drive`<br>`https://www.googleapis.com/auth/spreadsheets` | Export via Drive |
| people | yes | People API | `profile` | OIDC profile scope |
| forms | yes | Forms API | `https://www.googleapis.com/auth/forms.body`<br>`https://www.googleapis.com/auth/forms.responses.readonly` |  |
| appscript | yes | Apps Script API | `https://www.googleapis.com/auth/script.projects`<br>`https://www.googleapis.com/auth/script.deployments`<br>`https://www.googleapis.com/auth/script.processes` |  |
| groups | no | Cloud Identity API | `https://www.googleapis.com/auth/cloud-identity.groups.readonly` | Workspace only |
| keep | no | Keep API | `https://www.googleapis.com/auth/keep.readonly` | Workspace only; service account (domain-wide delegation) |

**User column**: `yes` means the service is included in the default `user` service set (what `wk auth add` requests by default). `no` means it must be explicitly requested via `--services`.

## Service Accounts Setup

A service account is a non-human Google identity that belongs to a Google Cloud project. In Google Workspace, a service account can impersonate a user via **domain-wide delegation** (admin-controlled) and access APIs like Gmail/Calendar/Drive as that user.

In `wk`, service accounts are an **optional auth method** that can be configured per account email. If a service account key is configured for an account, it takes precedence over OAuth refresh tokens.

### 1) Create a Service Account (Google Cloud)

1. Create (or pick) a Google Cloud project.
2. Enable the APIs you'll use (e.g. Gmail, Calendar, Drive, Sheets, Docs, People, Tasks, Cloud Identity).
3. Go to **IAM & Admin -> Service Accounts** and create a service account.
4. In the service account details, enable **Domain-wide delegation**.
5. Create a key (**Keys -> Add key -> Create new key -> JSON**) and download the JSON key file.

### 2) Allowlist Scopes (Google Workspace Admin Console)

Domain-wide delegation is enforced by Workspace admin settings.

1. Open **Admin console -> Security -> API controls -> Domain-wide delegation**.
2. Add a new API client:
   - Client ID: use the service account's "Client ID" from Google Cloud.
   - OAuth scopes: comma-separated list of scopes you want to allow (copy from `wk auth services` and/or your `wk auth add --services ...` usage).

If a scope is missing from the allowlist, service-account token minting can fail (or API calls will 403 with insufficient permissions).

### 3) Configure `wk` to Use the Service Account

```bash
wk auth service-account set you@yourdomain.com --key ~/Downloads/service-account.json
```

Verify:

```bash
wk --account you@yourdomain.com auth status
wk auth list
```

See [docs/auth.md](auth.md) for complete authentication documentation.
