# contacts.md

Google Contacts: personal contacts, other contacts, directory search, and batch operations.

## Contact CRUD
- `gog contacts list`
- `gog contacts search <query...>`
- `gog contacts get <resourceName>`
- `gog contacts create --given <name> [--family ...] [--email ...] [--phone ...]`
- `gog contacts update <resourceName> [--given ...] [--family ...] [--email ...] [--phone ...] [--birthday YYYY-MM-DD] [--notes ...]`
- `gog contacts delete <resourceName>`

## Batch operations
- `gog contacts batch create --file contacts.json`
- `gog contacts batch delete <resourceName>...` (or `--file names.json`)

## Directory + other contacts
- `gog contacts directory list`
- `gog contacts directory search <query>`
- `gog contacts other list`
- `gog contacts other search <query>`
- `gog contacts other delete <resourceName>`

## Example
```bash
gog contacts search 'ana silva' --read-only
gog contacts create --given Ana --family Silva --email ana@acme.com --dry-run
```
