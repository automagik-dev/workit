---
name: docx-track
description: Tracked changes and comments in DOCX files
---

# DOCX Tracked Changes and Comments Workflow

Manage tracked changes (revisions) and comments in `.docx` files. Useful for review workflows where changes need approval before finalizing.

## Before you start

1. Run `wk setup docx` to verify environment.
2. Read the document first: `wk docx cat document.docx`

## Tracked changes

### Create a tracked replacement

```bash
wk docx track document.docx --replace "old text" --new "new text" --author "Agent"
```

Inserts a revision mark that shows the old text as deleted and the new text as inserted, attributed to the specified author.

Save to a separate file:
```bash
wk docx track document.docx --replace "draft" --new "final" --author "Review Bot" -o reviewed.docx
```

### Accept all tracked changes

```bash
wk docx accept-changes document.docx
```

Accepts all pending revisions: inserted text becomes permanent, deleted text is removed. The document is flattened to its "accepted" state.

Output to a new file:
```bash
wk docx accept-changes document.docx -o accepted.docx
```

### Reject all tracked changes

```bash
wk docx reject-changes document.docx
```

Rejects all pending revisions: inserted text is removed, deleted text is restored. The document reverts to its pre-revision state.

Output to a new file:
```bash
wk docx reject-changes document.docx -o original.docx
```

## Comments

### Add a comment

```bash
wk docx comment document.docx --at "paragraph:3" --text "Please review this section." --author "Agent"
```

The `--at` reference uses `paragraph:N` format (0-based index). Use `wk docx cat document.docx --structure` to find the right paragraph index.

### List all comments

```bash
wk docx list-comments document.docx
```

Shows comment ID, author, date, and text for each comment.

JSON output:
```bash
wk docx list-comments document.docx --json
```

## Common patterns

### Review workflow
```bash
# 1. Read the document
wk docx cat report.docx

# 2. Make tracked changes
wk docx track report.docx --replace "preliminary" --new "final" --author "Reviewer"
wk docx track report.docx --replace "$10,000" --new "$12,500" --author "Finance"

# 3. Add review comments
wk docx comment report.docx --at "paragraph:0" --text "Title needs updating for Q4." --author "Reviewer"

# 4. Check all comments
wk docx list-comments report.docx

# 5. After approval, accept all changes
wk docx accept-changes report.docx
```

### Safe review with backup
```bash
cp report.docx report-original.docx
wk docx track report.docx --replace "2025" --new "2026" --author "Agent"
# If something went wrong:
wk docx reject-changes report.docx
```

### Accept and export
```bash
wk docx accept-changes report.docx -o report-final.docx
wk docx to-pdf report-final.docx
```

## Error handling

- **"no occurrences found"**: The `--replace` text was not found in the document body.
- **"paragraph index out of range"**: Use `--structure` to verify valid paragraph indices for `--at`.
- **Default author**: If `--author` is omitted, it defaults to "Workit".
