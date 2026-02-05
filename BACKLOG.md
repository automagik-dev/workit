# gog-cli Backlog

## Bugs

### `gog sync status` doesn't detect systemd-launched daemon
- **Date:** 2026-02-05
- **Severity:** Low (cosmetic)
- **Description:** When sync daemon is launched via systemd service, `gog sync status` reports "Daemon not running" even though the daemon is active and working. The PID file detection doesn't match the systemd-launched process.
- **Workaround:** Check `systemctl --user status gog-sync` instead
- **Fix:** Update PID file handling to work with systemd, or use a different detection method (e.g., check for running process by name)

### Progress tracking shows 0 synced items
- **Date:** 2026-02-05  
- **Severity:** Low (cosmetic)
- **Description:** `gog sync status` shows `SYNCED: 0` and `PENDING: 12807` even after sync is working. Progress counters don't update.
- **Workaround:** Test sync manually by creating a file and checking Drive
- **Fix:** Implement proper progress tracking in sync engine

