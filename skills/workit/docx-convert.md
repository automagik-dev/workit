---
name: docx-convert
description: Convert DOCX files to PDF via LibreOffice
---

# DOCX Conversion Workflow

Convert `.docx` files to PDF using LibreOffice as the rendering engine.

## Before you start

1. Run `wk setup docx` to verify LibreOffice is installed.
2. LibreOffice is **required** for PDF conversion. The setup command will show install instructions if it is missing.

## Convert DOCX to PDF

```bash
wk docx to-pdf document.docx
```

Creates a PDF in the same directory as the input file, with `.pdf` extension.

## Specify output path

```bash
wk docx to-pdf document.docx -o /path/to/output.pdf
```

## JSON output (for automation)

```bash
wk docx to-pdf document.docx --json
```

Returns `{"path": "/absolute/path/to/document.pdf"}`.

## Common patterns

### Create and convert pipeline
```bash
# Generate DOCX from template, then export to PDF
wk docx create --from values.json --template invoice --out invoice.docx
wk docx to-pdf invoice.docx
```

### Batch conversion
```bash
for f in *.docx; do
  wk docx to-pdf "$f"
done
```

### Edit then export
```bash
# Make edits
wk docx replace report.docx "DRAFT" "FINAL"
wk docx accept-changes report.docx

# Export
wk docx to-pdf report.docx -o report-final.pdf
```

## Installing LibreOffice

If `wk setup docx` reports LibreOffice as missing:

- **macOS**: `brew install --cask libreoffice`
- **Ubuntu/Debian**: `apt install libreoffice-common`
- **Snap**: `snap install libreoffice`

Only the headless mode is used (`soffice --headless`), so no GUI is needed.

## Error handling

- **"convert to pdf: soffice not found"**: LibreOffice is not installed or not in PATH. Run `wk setup docx` for install instructions.
- **"convert to pdf: exit status 1"**: LibreOffice failed. Check the input file is a valid DOCX and that LibreOffice is not already running (headless mode may conflict with a running instance).
