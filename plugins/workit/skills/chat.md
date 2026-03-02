# chat.md

Google Chat for spaces, threads, messages, and DMs.

## Spaces
- `wk chat spaces list`
- `wk chat spaces find <display-name>`
- `wk chat spaces create <displayName> [--member email,...]`

## Messages and threads
- `wk chat messages list <space>` (space = `spaces/AAAA...`)
- `wk chat messages send <space> --text "..."  [--thread spaces/.../threads/...]`
- `wk chat threads list <space>`

## Direct messages
- `wk chat dm space <email-or-userId>` (find/create DM space)
- `wk chat dm send <email-or-userId> --text "..."`

## Examples
```bash
wk chat spaces find 'Engineering' --read-only
wk chat messages send spaces/AAAA... --text 'Deploy complete ✅' --dry-run
wk chat dm send user@acme.com --text 'Can you review the doc?' --dry-run
```
