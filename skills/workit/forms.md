# forms.md

Google Forms command guide.

## Form CRUD-ish
- `wk forms create --title "..." [--description ...]`
- `wk forms get <formId>`
- `wk forms publish <formId> [--publish-as-template] [--require-authentication]`

## Responses
- `wk forms responses list <formId>`
- `wk forms responses get <formId> <responseId>`

## Template creation pattern
- Keep a baseline form; clone via Drive copy patterns and then adjust settings/questions externally if needed.

## Example
```bash
wk forms create --title 'Customer Feedback - Q1' --dry-run
wk forms responses list <formId> --read-only
```
