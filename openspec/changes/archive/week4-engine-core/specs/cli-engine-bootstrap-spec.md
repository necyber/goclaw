# CLI Engine Bootstrap Spec (Week 4 Archive Backfill)

## Scope

Define command-entry integration responsibilities for constructing and running engine core.

## Requirements

### FR-1 Config and logger injection

CLI bootstrap SHALL construct engine with resolved config and logger instances.

### FR-2 Start/stop wiring

CLI path SHALL start engine at process bootstrap and invoke graceful stop on process shutdown signals.

### FR-3 Signal handling

CLI process SHALL listen for termination signals and trigger controlled engine shutdown.

### FR-4 Compatibility migration

Week4 bootstrap SHALL replace earlier placeholder engine construction paths with full runtime initialization.

## Archive Note

Historical backfill for archived change `week4-engine-core`.

