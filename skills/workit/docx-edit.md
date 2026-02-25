---
name: docx-edit
description: Edit DOCX files — surgical find/replace, insert, delete, style, and full rewrite
---

# DOCX Editing Workflow

Surgical and full-content editing of local `.docx` files. All edit commands overwrite the input file by default; use `-o` to write to a separate file.

## Before you start

1. Run `wk setup docx` to verify environment.
2. Read the document structure first: `wk docx cat document.docx --structure`

## Find and replace text

```bash
wk docx replace document.docx "old text" "new text"
```

Replaces all occurrences. Reports the number of replacements made.

Save to a new file instead of overwriting:
```bash
wk docx replace document.docx "draft" "final" -o output.docx
```

## Insert a paragraph

```bash
wk docx insert document.docx --after "heading:Summary" --text "New paragraph content here."
```

The `--after` reference can be:
- `heading:Summary` -- insert after the heading with text "Summary"
- `paragraph:5` -- insert after paragraph index 5 (0-based)

## Delete a section

```bash
wk docx delete document.docx --section "Appendix A"
```

Removes the heading and all content under it until the next heading of equal or higher level.

## Change paragraph style

```bash
wk docx style document.docx --paragraph 3 --style "Heading2"
```

The `--paragraph` index is 0-based. Use `wk docx cat document.docx --structure` to find the right index.

## Full content rewrite from markdown

```bash
wk docx rewrite document.docx --from content.md
```

Replaces the entire body with content from a markdown file. The DOCX container (styles, fonts, headers/footers) is preserved.

Save to a new file:
```bash
wk docx rewrite document.docx --from content.md -o rewritten.docx
```

## Inspect template placeholders

```bash
wk docx inspect template.docx
```

Lists all `{{PLACEHOLDER}}` patterns found in the document. Useful before running `wk docx create`.

## Common patterns

### Safe edit workflow
```bash
# 1. Read structure to understand the document
wk docx cat report.docx --structure

# 2. Make a backup
cp report.docx report-backup.docx

# 3. Edit with output to new file
wk docx replace report.docx "Q3" "Q4" -o report-updated.docx

# 4. Verify the result
wk docx cat report-updated.docx
```

### Batch find-replace
Run multiple replacements sequentially on the same file:
```bash
wk docx replace contract.docx "{{CLIENT}}" "ACME Corp"
wk docx replace contract.docx "{{DATE}}" "2026-02-24"
wk docx replace contract.docx "{{AMOUNT}}" "$50,000"
```

### Restructure a document
```bash
wk docx delete report.docx --section "Draft Notes"
wk docx insert report.docx --after "heading:Conclusion" --text "Next steps will be shared in the follow-up."
wk docx style report.docx --paragraph 12 --style "Heading1"
```

## Error handling

- **"no occurrences found"**: The search text was not found. Check exact spelling and whitespace.
- **"paragraph index out of range"**: Use `--structure` to verify valid paragraph indices.
- **"section not found"**: The heading text must match exactly. Check with `wk docx cat` first.
