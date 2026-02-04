# HEARTBEAT.md

## Periodic Checks

When receiving a heartbeat, check:

1. **Upstream changes?**
   ```bash
   git fetch upstream
   git log HEAD..upstream/main --oneline | head -5
   ```
   If new commits, consider merging.

2. **Current milestone progress?**
   Check `@./MILESTONES.md` - what's the next unchecked item?

3. **Any failing tests?**
   ```bash
   make test 2>&1 | tail -20
   ```

4. **Unpushed commits?**
   ```bash
   git log origin/main..HEAD --oneline
   ```

## If Nothing Needs Attention
Reply: `HEARTBEAT_OK`

## Active Tasks
*(Update this section when working on something)*

- [ ] Currently idle - pick up next milestone item
