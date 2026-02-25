# tasks.md

Google Tasks command guide.

## Task lists
- `wk tasks lists list`
- `wk tasks lists create <title>`

## Tasks CRUD
- `wk tasks list <tasklistId>`
- `wk tasks get <tasklistId> <taskId>`
- `wk tasks add <tasklistId> --title "..." [--notes ...] [--due YYYY-MM-DD]`
- `wk tasks update <tasklistId> <taskId> [--title ...] [--notes ...] [--due ...] [--status needsAction|completed]`
- `wk tasks done <tasklistId> <taskId>`
- `wk tasks undo <tasklistId> <taskId>`
- `wk tasks delete <tasklistId> <taskId>`
- `wk tasks clear <tasklistId>` (clears completed)

## Example
```bash
wk tasks add <tasklistId> --title 'Prepare launch checklist' --due 2026-02-20 --dry-run
wk tasks done <tasklistId> <taskId> --dry-run
```
