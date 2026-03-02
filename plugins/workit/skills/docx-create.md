---
name: docx-create
description: Create DOCX documents from templates with JSON values or from markdown
---

# DOCX Creation Workflow

Create new `.docx` files from templates with placeholder substitution, or generate a DOCX from markdown content.

## Before you start

1. Run `wk setup docx` to verify environment.
2. For template workflows, register templates first with `wk templates add`.

## Create from template + JSON values

### Step 1: Register or inspect a template

```bash
# Add a template to the template library
wk templates add invoice ./templates/invoice-template.docx

# List available templates
wk templates list

# Inspect placeholders in a template
wk templates inspect invoice
```

### Step 2: Prepare a JSON values file

Create `values.json`:
```json
{
  "CLIENT_NAME": "ACME Corp",
  "INVOICE_NUMBER": "INV-2026-042",
  "DATE": "2026-02-24",
  "AMOUNT": "$12,500.00"
}
```

Placeholders in the template are `{{KEY}}` patterns (e.g. `{{CLIENT_NAME}}`).

### Step 3: Generate the document

```bash
wk docx create --from values.json --template invoice --out invoice-acme.docx
```

The `--template` flag accepts either:
- A registered template name (e.g. `invoice`)
- A direct file path (e.g. `./templates/invoice-template.docx`)

## Create from markdown

```bash
wk docx create --from content.md --out report.docx
```

Converts markdown content into a minimal DOCX. Headings, paragraphs, bold, italic, and lists are converted to DOCX elements.

## Template management commands

```bash
# List all registered templates
wk templates list
wk templates list --json

# Add a template
wk templates add <name> <path-to-docx>

# Inspect placeholders
wk templates inspect <name>
wk templates inspect <name> --json
```

## Common patterns

### Batch document generation
```bash
for client in acme globex initech; do
  wk docx create --from "${client}-values.json" --template contract --out "contracts/${client}-contract.docx"
done
```

### Inspect before filling
```bash
# See what placeholders are needed
wk docx inspect template.docx

# Or via the templates command
wk templates inspect proposal
```

### Markdown to DOCX pipeline
```bash
# Generate report content as markdown, then convert
wk docx create --from report.md --out report.docx

# Optionally convert to PDF
wk docx to-pdf report.docx
```

## Error handling

- **"--template is required when using JSON input"**: JSON values need a template. Provide `--template`.
- **"no placeholders found"**: The template has no `{{...}}` patterns. Verify the template file.
- **"read template: file not found"**: Check the template name with `wk templates list` or use a direct file path.
