# Sync and Archive Plan

After implementation is complete for `week4-engine-spec-gap-fix`, finalize with:

1. Sync delta specs to main specs:
   - `openspec-sync-specs` for this change.
2. Re-check task/artifact completion:
   - `openspec instructions apply --change week4-engine-spec-gap-fix --json`
   - Expect `remaining: 0` and `state: all_done`.
3. Archive the change:
   - `openspec-archive-change` for this change.
4. Commit archive + synced spec updates.

This plan is tracked to satisfy task `3.3`.
