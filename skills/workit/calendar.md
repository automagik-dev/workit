# calendar.md

Google Calendar (`wk calendar` / alias `cal`).

Use `--read-only` for inspection, `--dry-run` for writes, and `--json`/`--plain` for automation.

## Top-level commands (from `wk calendar --help`)
- `calendars`
- `acl <calendarId>`
- `events [calendarId]`
- `event <calendarId> <eventId>`
- `create <calendarId>`
- `update <calendarId> <eventId>`
- `delete <calendarId> <eventId>`
- `freebusy <calendarIds>`
- `respond <calendarId> <eventId>`
- `propose-time <calendarId> <eventId>`
- `colors`
- `conflicts`
- `search <query>`
- `time`
- `users`
- `team <group-email>`
- `focus-time --from ... --to ... [calendarId]`
- `out-of-office --from ... --to ... [calendarId]`
- `working-location --from ... --to ... --type ... [calendarId]`

## Examples
```bash
wk calendar events primary --from 2026-02-18T09:00:00-03:00 --to 2026-02-18T18:00:00-03:00 --read-only --plain

wk calendar focus-time --from 2026-02-19T13:00:00-03:00 --to 2026-02-19T15:00:00-03:00 primary --dry-run
```