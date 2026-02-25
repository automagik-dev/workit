# templates.md

Document template management. Register, inspect, and reuse DOCX templates for repeatable document generation with `wk docx create`.

## Read
- `wk templates list` — list available templates
- `wk templates inspect <name>` — inspect a template for `{{PLACEHOLDER}}` patterns (accepts template name or file path)

## Write
- `wk templates add <name> <source>` — register a template from a local DOCX file path

## Safety
- Read commands (`list`, `inspect`) are safe with `--read-only`.
- `add` registers a template; use `--dry-run` to preview without saving.

## Examples
```bash
# List all registered templates
wk templates list --read-only
wk templates list --json --read-only

# Register a new template
wk templates add invoice ./templates/invoice.docx
wk templates add nda ./legal/nda-template.docx

# Inspect a template for placeholders
wk templates inspect invoice
wk templates inspect ./templates/invoice.docx

# Full workflow: register, inspect, then generate
wk templates add contract ./templates/contract.docx
wk templates inspect contract
wk docx create --from data.json --template contract --out contract-acme.docx --dry-run
wk docx create --from data.json --template contract --out contract-acme.docx
```

## Command index
`list` `add` `inspect`
