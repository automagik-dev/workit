# appscript.md

Google Apps Script command guide.

## Commands
- `wk appscript create --title "..."`
- `wk appscript get <scriptId>`
- `wk appscript content <scriptId>`
- `wk appscript run <scriptId> <function> [--params '[...]'] [--dev-mode]`

## Typical workflow
1. Create project
2. Inspect metadata/content
3. Run deployed function for automation tasks

## Example
```bash
wk appscript create --title 'Daily Ops Automation' --dry-run
wk appscript run <scriptId> syncSheets --params '["2026-02-18"]' --dry-run
```
