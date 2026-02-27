# Cluster Event Bus Rollout Checklist

This checklist is for staged rollout of distributed coordination, ownership enforcement, and canonical lifecycle event bus.

## Stage 0: Preflight

- Verify coordination backend endpoint reachability (`etcd`/`consul`) from all nodes.
- Verify NATS endpoint reachability and auth configuration.
- Confirm node IDs are unique and stable.
- Confirm clock sync (NTP) across nodes for lease and ordering diagnostics.

## Stage 1: Feature Flags (Observe-Only)

- Enable coordination registration only:
  - `cluster.enabled=true`
  - ownership enforcement disabled.
- Enable canonical event publisher in shadow mode:
  - publish enabled, consumer bridge read-only.
- Verify metrics baseline:
  - `cluster_ownership_changes_total`
  - `event_bus_publish_total{status}`
  - `event_bus_degraded`
  - `event_bus_outages_total`

## Stage 2: Partial Enforcement

- Enable ownership guard for a canary lane subset.
- Enable distributed signal routing for canary workflows.
- Enable streaming event-bus bridge for canary subscribers.
- Watch for:
  - ownership denials (`redis_lane_ownership_decision_total{decision="deny"}`)
  - retry spikes (`event_bus_publish_retries_total`)
  - duplicate suppression mismatches in downstream consumers.

## Stage 3: Full Enforcement

- Expand ownership guard to all Redis lanes.
- Expand distributed signal routing to all tasks.
- Make canonical event stream the default source for cross-node streaming updates.
- Keep rollback toggle ready for each capability independently.

## Rollback Plan

- If coordination instability occurs:
  - disable ownership guard while keeping local processing active.
- If event bus instability occurs:
  - keep local execution active, monitor degraded mode, disable bridge consumers.
- If consumer compatibility issues occur:
  - pin consumers to previous `schema_version` route and keep dual-publish window.

## Exit Criteria

- No sustained degraded mode (`event_bus_degraded==0`) for at least 24 hours.
- No unresolved ownership conflicts or stale-fencing execution attempts.
- Streaming subscribers observe ordered per-workflow updates across node boundaries.
- On-call runbook updated with confirmed rollback execution steps.

