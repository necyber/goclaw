# Event Bus Consumer Contract

This document defines the consumer-side contract for distributed lifecycle events published to canonical NATS subjects.

## Envelope Requirements

Consumers must parse the versioned envelope fields:

- `event_id`: globally unique idempotency key for duplicate suppression.
- `event_type`: lifecycle transition type.
- `timestamp`: producer-side event time in UTC.
- `schema_version`: payload contract version.
- `node_id`: source node identity.
- `shard_key`: ownership scope key.
- `workflow_id` / `task_id`: correlation identifiers.
- `ordering_key`: scoped ordering stream key (default: `workflow_id`).
- `sequence`: monotonic sequence within `ordering_key`.
- `payload`: schema-versioned event payload.

## Ordering Contract

- Ordering is guaranteed only within one `ordering_key`.
- Consumers must not assume global ordering across workflows or shards.
- For the same `ordering_key`, consumers should apply events by ascending `sequence`.
- Gaps or reordering should be handled by buffering/retry policy at consumer side.

## Duplicate Suppression Contract

- Delivery intent is at-least-once; duplicate events are expected.
- Consumers must deduplicate by `event_id`.
- Dedup stores should retain `event_id` for at least the maximum replay window.
- Idempotent handlers are required for all side effects.

## Schema Compatibility Contract

- Additive optional fields are backward compatible.
- Adding required fields, removing fields, or changing field types is breaking.
- Breaking changes must use a new `schema_version` and run with an explicit dual-read compatibility window.
- Consumers should route decoding by `schema_version` and reject unsupported versions explicitly.

