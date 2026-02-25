---
name: docx-tables
description: Table operations in DOCX files — list, add rows, update cells, delete rows
---

# DOCX Table Operations Workflow

Read and modify tables in local `.docx` files. All table mutation commands overwrite the input file by default; use `-o` to write to a separate file.

## Before you start

1. Run `wk setup docx` to verify environment.
2. List tables first to understand the document structure.

## List all tables

```bash
wk docx table document.docx --list
```

Shows each table's index, dimensions (rows x cols), and cell content. Table indices are 0-based.

JSON output for programmatic use:
```bash
wk docx table document.docx --list --json
```

## Add a row

```bash
wk docx table document.docx --id 0 --add-row "Cell 1,Cell 2,Cell 3"
```

Appends a new row to table 0 with comma-separated cell values. The number of values should match the table's column count.

## Update a cell

```bash
wk docx table document.docx --id 0 --update-cell "2,1,Updated value"
```

Format: `row,col,value` (1-based row and column indices). This updates the cell at row 2, column 1 in table 0.

## Delete a row

```bash
wk docx table document.docx --id 0 --delete-row 3
```

Deletes row 3 (1-based index) from table 0.

## Target a specific table

When a document has multiple tables, use `--id` to select which one:

```bash
# List all tables to find the right index
wk docx table document.docx --list

# Operate on the second table (0-based index)
wk docx table document.docx --id 1 --add-row "A,B,C"
```

If `--id` is omitted, it defaults to the first table (index 0).

## Output to a separate file

```bash
wk docx table document.docx --id 0 --add-row "New,Row,Data" -o updated.docx
```

## Common patterns

### Inspect table structure
```bash
# Get table data as JSON for analysis
wk docx table report.docx --list --json | jq '.[0]'
```

### Build a table row by row
```bash
wk docx table data.docx --add-row "January,1200,15%"
wk docx table data.docx --add-row "February,1350,12%"
wk docx table data.docx --add-row "March,1500,11%"
```

### Update specific cells
```bash
# Fix a typo in row 1, column 2
wk docx table report.docx --update-cell "1,2,Corrected Value"

# Update the total in the last row
wk docx table report.docx --update-cell "5,3,$4,050"
```

### Delete and rebuild
```bash
# Remove outdated rows
wk docx table report.docx --delete-row 4
wk docx table report.docx --delete-row 3

# Add new rows
wk docx table report.docx --add-row "New Item,100,Active"
```

## Error handling

- **"no tables found"**: The document has no tables. Use `wk docx cat --structure` to confirm.
- **"table index out of range"**: Check available table indices with `--list`.
- **"row index out of range"**: Row indices are 1-based. Use `--list` to see how many rows exist.
- **Column count mismatch**: When using `--add-row`, the number of comma-separated values should match the table's column count.
