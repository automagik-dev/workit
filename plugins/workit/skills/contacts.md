# contacts.md

Google Contacts: personal contacts, other contacts, directory search, and batch operations.

## Contact CRUD
- `wk contacts list`
- `wk contacts search <query...>`
- `wk contacts get <resourceName>`
- `wk contacts create --given <name> [--family ...] [--email ...] [--phone ...]`
- `wk contacts update <resourceName> [--given ...] [--family ...] [--email ...] [--phone ...] [--birthday YYYY-MM-DD] [--notes ...]`
- `wk contacts delete <resourceName>`

## Batch operations
- `wk contacts batch create --file contacts.json`
- `wk contacts batch delete <resourceName>...` (or `--file names.json`)

## Directory + other contacts
- `wk contacts directory list`
- `wk contacts directory search <query>`
- `wk contacts other list`
- `wk contacts other search <query>`
- `wk contacts other delete <resourceName>`

## Example
```bash
wk contacts search 'ana silva' --read-only
wk contacts create --given Ana --family Silva --email ana@acme.com --dry-run
```
