# chat.md

Google Chat for spaces, threads, messages, and DMs.

## Spaces
- `gog chat spaces list`
- `gog chat spaces find <display-name>`
- `gog chat spaces create <displayName> [--member email,...]`

## Messages and threads
- `gog chat messages list <space>` (space = `spaces/AAAA...`)
- `gog chat messages send <space> --text "..."  [--thread spaces/.../threads/...]`
- `gog chat threads list <space>`

## Direct messages
- `gog chat dm space <email-or-userId>` (find/create DM space)
- `gog chat dm send <email-or-userId> --text "..."`

## Examples
```bash
gog chat spaces find 'Engineering' --read-only
gog chat messages send spaces/AAAA... --text 'Deploy complete ✅' --dry-run
gog chat dm send user@acme.com --text 'Can you review the doc?' --dry-run
```
