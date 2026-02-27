## 1. Scope and Baseline

- [x] 1.1 Enumerate invalid main specs with `openspec validate --specs --json`
- [x] 1.2 Classify legacy specs into structured (has requirements/scenarios) and narrative-only buckets

## 2. Bulk Migration

- [x] 2.1 Migrate structured legacy specs to canonical top-level sections while preserving requirement/scenario blocks
- [x] 2.2 Migrate narrative-only specs by adding baseline requirement/scenario and retaining historical text in `## Notes`
- [x] 2.3 Ensure all migrated files are saved under `openspec/specs/<capability>/spec.md`

## 3. Verification

- [x] 3.1 Run `openspec validate --specs --json --no-interactive`
- [x] 3.2 Confirm validation summary reports zero failures
- [x] 3.3 Record migration as a dedicated OpenSpec change for future reference
