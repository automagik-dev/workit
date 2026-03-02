---
name: workit
description: Router for Google Workspace and DOCX automation via workit CLI. Load only the relevant service file.
allowed-tools: Bash(wk *), Bash(jq *)
---

# wk skill router

Use `wk` for Gmail, Calendar, Drive, Docs, Sheets, Slides, Chat, Classroom, Tasks, Contacts, People, Keep, Groups, Forms, App Script, auth/sync setup, and local DOCX editing.

> **No GCP setup needed.** Run `wk auth manage` — auth.automagik.dev handles OAuth. Load `setup.md` for details.

## Safety defaults (always)
- Read/list/query flows: add `--read-only`.
- Any write/send/create/update/delete flow: start with `--dry-run`, then rerun without it after user confirms.
- Prefer `--no-input` in automation/CI.
- For dangerous operations, require explicit confirmation unless user provided it; `--force` only after confirmation.
- Use `--command-tier core|extended|complete` and `--enable-commands` to constrain capability.

Load `safety.md` before risky actions. Load `setup.md` for auth/account setup.

## Keyword → file routing

### Google Workspace services

| User intent / keywords | Load file |
|---|---|
| safety, dry-run, read-only, force, tiers, allowlist | `safety.md` |
| login, oauth, token, account, alias, service account | `setup.md` |
| email, gmail, send, labels, filters, drafts, tracking | `gmail.md` |
| calendar, events, freebusy, focus time, ooo, working location | `calendar.md` |
| drive, files, folders, upload, download, share, permissions | `drive.md` |
| sync, mirror, local folder sync daemon | `sync.md` |
| sheets, spreadsheet, range, tab, batch-update | `sheets.md` |
| docs, document, find-replace, header, footer, template | `docs.md` |
| slides, presentation, markdown deck, speaker notes | `slides.md` |
| chat, spaces, threads, dm, webhook-style message sends | `chat.md` |
| classroom, courses, students, teachers, coursework, submissions | `classroom.md` |
| tasks, task list, due date, complete/undo | `tasks.md` |
| contacts, address book, batch contacts, directory | `contacts.md` |
| people, profile, me, directory relations | `people.md` |
| keep, notes, keep attachments (workspace only) | `keep.md` |
| groups, group members, workspace groups | `groups.md` |
| forms, form responses, publish | `forms.md` |
| appscript, apps script, script run/deploy | `appscript.md` |

### DOCX local editing (no Google account needed)

| User intent / keywords | Load file |
|---|---|
| docx read, docx cat, docx info, extract text, document metadata | `docx-read.md` |
| docx replace, insert, delete, style, rewrite, edit docx | `docx-edit.md` |
| docx create, template fill, markdown to docx, generate document | `docx-create.md` |
| tracked changes, revisions, accept, reject, comments, review | `docx-track.md` |
| table, rows, cells, add row, update cell, delete row | `docx-tables.md` |
| pdf, convert, export, to-pdf, libreoffice | `docx-convert.md` |

### Agent helpers & config

| User intent / keywords | Load file |
|---|---|
| agent, help topics, exit codes, schema, generate-input, flags | `agent.md` |
| config, configuration, config path, config set, config get | `config.md` |
| templates, template list, template add, template inspect, placeholder | `templates.md` |

## Top-level shortcuts

Common shorthand patterns — no routing needed:

| Shortcut | Expands to |
|---|---|
| `wk send` | `wk gmail send` |
| `wk ls` | `wk drive ls` |
| `wk cat` | `wk docx cat` |
| `wk search` | `wk gmail search` |
| `wk events` | `wk calendar list` |
| `wk upload` | `wk drive upload` |
| `wk download` | `wk drive download` |
| `wk share` | `wk drive share` |

## DOCX quick reference

All DOCX commands work on local files. No Google account or network access required.

```bash
# Environment check
wk setup docx

# Read
wk docx cat file.docx                     # content as markdown
wk docx cat file.docx --structure          # structured JSON
wk docx info file.docx                     # metadata

# Edit
wk docx replace file.docx "old" "new"     # find-replace
wk docx insert file.docx --after "heading:Intro" --text "New para"
wk docx delete file.docx --section "Appendix"
wk docx style file.docx --paragraph 3 --style "Heading2"
wk docx rewrite file.docx --from content.md

# Create
wk docx create --from values.json --template tmpl.docx --out out.docx
wk docx create --from content.md --out out.docx

# Tracked changes + comments
wk docx track file.docx --replace "old" --new "new" --author "Agent"
wk docx accept-changes file.docx
wk docx reject-changes file.docx
wk docx comment file.docx --at "paragraph:1" --text "Review this" --author "Agent"
wk docx list-comments file.docx

# Tables
wk docx table file.docx --list
wk docx table file.docx --add-row "A,B,C"
wk docx table file.docx --update-cell "1,2,New Value"
wk docx table file.docx --delete-row 3

# Convert
wk docx to-pdf file.docx

# Templates
wk templates list
wk templates add <name> <path>
wk templates inspect <name>
```

## Fast command patterns
- Search/list: `wk <service> ... --read-only --json`
- Preview write: `wk <service> ... --dry-run`
- Execute write after confirmation: `wk <service> ...`
- DOCX read: `wk docx cat file.docx --json`
- DOCX edit: `wk docx replace file.docx "old" "new" -o output.docx`
