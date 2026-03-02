# drive.md

Google Drive file/folder/document management.

## Discover and read
- `wk drive ls [--folder <id>]`
- `wk drive search <query...>`
- `wk drive get <fileId>`
- `wk drive cat <fileId>`
- `wk drive url <fileId...>`
- `wk drive drives` (shared drives)
- `wk drive check-public <fileId>`

## CRUD and movement
- `wk drive upload <localPath> [--parent <folderId>] [--convert-to doc|sheet|slides]`
- `wk drive download <fileId> [--out ...]`
- `wk drive mkdir <name> [--parent <folderId>]`
- `wk drive copy <fileId> <name>`
- `wk drive move <fileId> --parent <folderId>`
- `wk drive rename <fileId> <newName>`
- `wk drive delete <fileId> [--permanent]`

## Sharing and permissions
- `wk drive share <fileId> --to anyone|user|domain [--email user@acme.com] [--role reader|writer]`
- `wk drive permissions <fileId>`
- `wk drive unshare <fileId> <permissionId>`

## Comments
- `wk drive comments list <fileId>`
- `wk drive comments get <fileId> <commentId>`
- `wk drive comments create <fileId> <content>`
- `wk drive comments update <fileId> <commentId> <content>`
- `wk drive comments reply <fileId> <commentId> <content>`
- `wk drive comments delete <fileId> <commentId>`

## Examples
```bash
# Locate all quarterly reports
wk drive search 'name contains "Q4" and mimeType != "application/vnd.google-apps.folder"' --read-only

# Preview new share permission
wk drive share <fileId> --to user --email analyst@acme.com --role reader --dry-run

# Safe delete preview
wk drive delete <fileId> --dry-run
```
