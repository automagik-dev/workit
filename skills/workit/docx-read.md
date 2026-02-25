---
name: docx-read
description: Read and extract content from DOCX files
---

# DOCX Reading Workflow

Read content and metadata from local `.docx` files. No Google account needed.

## Before you start

Run `wk setup docx` to verify environment dependencies.

## Read content as markdown

```bash
wk docx cat document.docx
```

Returns the full text content of the document converted to markdown format.

## Read structured content (paragraphs, styles, IDs)

```bash
wk docx cat document.docx --structure
```

Returns JSON with paragraph-level detail: index, style name, and text for each paragraph. Useful for identifying target paragraphs before editing.

## Get document metadata

```bash
wk docx info document.docx
```

Shows title, author, description, created/modified dates, page count, and internal part list.

## JSON output (for programmatic use)

```bash
wk docx info document.docx --json
wk docx cat document.docx --json
```

Both commands support `--json` for machine-readable output that can be piped to `jq`.

## Common patterns

### Inspect before editing
Always read structure first to identify paragraph indices and headings:
```bash
wk docx cat report.docx --structure | jq '.[].style'
```

### Quick content check
```bash
wk docx cat report.docx | head -50
```

### Extract metadata for logging
```bash
wk docx info report.docx --json | jq '{title, author, pages}'
```

## Error handling

- **File not found**: Verify the path exists. Relative and absolute paths are both supported.
- **Not a valid DOCX**: The file must be a valid ZIP-based `.docx` (Office Open XML). Older `.doc` binary format is not supported.
