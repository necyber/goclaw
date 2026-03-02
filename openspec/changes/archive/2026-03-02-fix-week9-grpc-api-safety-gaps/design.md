## Context

Week9 gRPC APIs are already implemented and archived, but current runtime behavior still has production-critical gaps: the interceptor chain is not effectively mounted in normal startup paths, batch pagination can panic on invalid offsets, streaming internals have concurrency hazards (data race/deadlock class), client health checks can report false-negative against service-scoped checks, and shutdown/connection semantics are partially inconsistent with intended behavior. This change is a cross-module hardening pass across server bootstrap, interceptors, handlers, streaming registry, and client SDK.

## Goals / Non-Goals

**Goals:**
- Make the default gRPC interceptor chain effective by default, with tracing optionality preserved.
- Guarantee batch pagination paths are panic-free and return deterministic `InvalidArgument` for malformed/out-of-range tokens.
- Remove known streaming concurrency hazards (shared filter mutation race, lock re-entry deadlock in cleanup paths).
- Align client and server health semantics for service-scoped checks.
- Fix server shutdown and connection-limit semantics so runtime state and config behavior are consistent.
- Add regression tests (including race-focused coverage) for each hardened path.

**Non-Goals:**
- No proto schema changes and no new RPC methods.
- No redesign of the authentication model beyond replacing placeholder behavior with enforceable semantics.
- No new external infrastructure dependency (no auth provider, no distributed cache).
- No behavioral expansion for unrelated Week10+ features.

## Decisions

### Decision 1: Always build from default interceptor chain, then conditionally include tracing

**Choice:** Server option construction will use the default chain builder for unary/stream interceptors in all normal startup paths, and tracing remains controlled by `EnableTracing`.

**Rationale:**
- Week9 requirements define an ordered chain as baseline behavior, not a tracing-only special case.
- Centralized chain construction avoids silent omission of security/metrics/recovery middleware.
- Preserves existing tracing toggle contract while making non-tracing interceptors always effective.

**Alternatives considered:**
- Keep tracing-only attachment and rely on handler-level checks: rejected due to fragmented enforcement and requirement drift.
- Split per-service manual chains: rejected due to inconsistent behavior and maintenance risk.

### Decision 2: Treat pagination token as validated bounded offset before slice operations

**Choice:** Batch handlers will validate parsed page offsets against request list length (`0 <= offset <= len`) and reject invalid values before any slicing.

**Rationale:**
- Prevents runtime panics and guarantees stable API behavior under malformed input.
- Keeps current offset-token model while making error behavior explicit.

**Alternatives considered:**
- Recover from panic with interceptor only: rejected because panic-driven control flow hides input bugs.
- Switch to opaque cursor tokens: rejected as larger behavior/API change.

### Decision 3: Eliminate shared mutable stream filter state between goroutines

**Choice:** `StreamLogs` will use synchronized filter state updates (single owner goroutine or lock-protected immutable snapshot handoff) so recv and send paths do not race on map/scalar state.

**Rationale:**
- Removes `concurrent map read/write` risk in long-lived streams.
- Keeps dynamic filter update capability without protocol changes.

**Alternatives considered:**
- Disable dynamic updates: rejected because it regresses an explicit Week9 capability.
- Deep copy per event without synchronization: rejected because races remain on pointer swaps without happens-before guarantees.

### Decision 4: Make subscriber cleanup non-reentrant with explicit remove path

**Choice:** Registry cleanup will avoid calling lock-taking public methods (`Unsubscribe`) while holding internal locks; cleanup performs direct in-lock removal or two-phase collect-and-remove.

**Rationale:**
- Prevents self-deadlock from lock re-entry.
- Keeps cleanup logic deterministic and testable.

**Alternatives considered:**
- Replace lock type with reentrant lock equivalent: not idiomatic in Go and adds complexity.

### Decision 5: Align health semantics by registering and setting service-specific health status

**Choice:** Server health initialization sets serving status for both global (`""`) and known service names; client health check keeps service-specific query and can optionally fallback to global if configured.

**Rationale:**
- Service-specific checks become meaningful and reliable.
- Maintains compatibility with existing health APIs.

**Alternatives considered:**
- Client checks only global status: rejected because service-scoped checks are required by spec and useful operationally.

### Decision 6: Normalize server stop state transitions and clarify connection-limit semantics

**Choice:** Force-stop timeout path must still transition server runtime state to stopped and release relevant references safely. Config key/field behavior will explicitly document and enforce whether limit applies to streams or connections.

**Rationale:**
- Prevents stale running-state flags after forced shutdown.
- Avoids operator confusion from mismatched naming vs behavior.

**Alternatives considered:**
- Keep current semantics and only update docs: rejected because state mismatch is operationally unsafe.

## Risks / Trade-offs

- [Risk] Enabling full interceptor chain by default may break previously unauthenticated clients.
  - Mitigation: keep auth behavior configurable where applicable and provide explicit migration notes; add integration tests for expected failures.
- [Risk] Hardening changes in streaming paths may alter timing/order in edge conditions.
  - Mitigation: add deterministic ordering and concurrency tests; preserve wire format and public RPC contracts.
- [Risk] Health behavior changes may expose existing misconfiguration in callers.
  - Mitigation: set both global and service statuses; keep fallback behavior where needed.
- [Risk] Tighter pagination validation may reject previously tolerated malformed tokens.
  - Mitigation: map all such cases to consistent `InvalidArgument` with clear message.

## Migration Plan

1. Implement server/interceptor wiring hardening and add focused tests first.
2. Implement batch pagination bounds validation and panic-regression tests.
3. Implement streaming concurrency fixes (filter synchronization and cleanup deadlock fix) with race-oriented tests.
4. Implement health status alignment across server/client and update related tests.
5. Validate with `go test ./pkg/grpc/...` and targeted `-race` suites.
6. Rollout in staged environments with interceptor/auth behavior monitoring.

Rollback strategy:
- Changes are code-path hardening without storage schema migration; rollback is safe by reverting this change set.
- If auth chain activation causes unexpected traffic impact, temporary config-based mitigation can be used while keeping panic/deadlock fixes.

## Open Questions

- Should client health check always fallback to global service status when service-specific is `NotFound`, or should this be opt-in?
- Should connection limit semantics be renamed at config layer (`max_connections` vs `max_concurrent_streams`) in a future compatibility window?
- For authentication placeholder replacement, which minimal production-safe token validation policy is accepted without introducing new dependencies?
