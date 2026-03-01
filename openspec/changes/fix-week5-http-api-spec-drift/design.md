## Context

Week 5 archived specs define the baseline HTTP API contract for workflow management, health probes, and API docs. Current runtime code has evolved beyond that baseline, but several externally visible behaviors now diverge (payload shape, status codes, request-id propagation, docs route/version contract, and shutdown/config semantics). The change is cross-cutting across API handlers, middleware, engine adapter logic, config validation, documentation generation, and integration tests.

## Goals / Non-Goals

**Goals:**
- Restore Week 5-compatible HTTP behavior for workflow, health, and docs contracts.
- Minimize production breakage during convergence by supporting explicit compatibility aliases where practical.
- Make config and shutdown semantics deterministic and testable.
- Remove flaky API integration test behavior tied to startup/shutdown timing and fixed port reuse.

**Non-Goals:**
- Re-architecting workflow execution internals or lane scheduling.
- Introducing new API capabilities beyond Week 5 drift fixes.
- Replacing the entire API documentation toolchain unless required to satisfy spec-version constraints.

## Decisions

1. Canonical workflow payloads will follow Week 5 names, with compatibility aliases.
- Decision: Treat Week 5 field names as canonical (`workflow_id`, `dependencies`), while accepting currently deployed aliases (`id`, `depends_on`) during transition.
- Rationale: Restores spec conformance without forcing immediate client cutover.
- Alternative considered: Hard switch to Week 5 names only. Rejected due to avoidable client break risk.

2. Non-terminal task-result queries will return explicit conflict semantics.
- Decision: `GET /api/v1/workflows/{workflow_id}/tasks/{task_id}/result` returns HTTP 409 for pending/scheduled/running tasks.
- Rationale: Matches Week 5 contract and avoids ambiguous empty-success payloads.
- Alternative considered: Keep 200 with status-only body. Rejected as spec-incompatible and harder for clients to reason about.

3. Request ID propagation will use one shared retrieval path.
- Decision: Handlers and response helpers will read request id via middleware accessor semantics, not ad-hoc context keys.
- Rationale: Prevents `unknown` request IDs and guarantees traceability in error payloads.
- Alternative considered: duplicate context key strings across packages. Rejected as fragile.

4. Health/readiness/status handlers will produce canonical envelopes.
- Decision: Introduce explicit response DTOs (or equivalent shaping layer) to guarantee required fields (`timestamp`, `checks`, `components`, etc.) and stable status vocabulary.
- Rationale: Enforces monitoring contract consistency and Kubernetes probe clarity.
- Alternative considered: continue returning minimal maps. Rejected due to schema drift and observability gaps.

5. Docs routing and spec generation will be contract-driven.
- Decision: Expose `/docs` as primary docs route and optionally keep `/swagger/*` as compatibility alias. Ensure generated API spec format/version meets declared requirement.
- Rationale: Restores advertised endpoint behavior while reducing operational surprises.
- Alternative considered: route rename only. Rejected because format/version mismatch would remain.

6. Shutdown timeout and HTTP timeout validation will be centralized in config/runtime wiring.
- Decision: Main shutdown flow derives HTTP shutdown context deadline from `server.http.shutdown_timeout`, and loader validation rejects non-positive timeout values.
- Rationale: Aligns operator configuration intent with runtime behavior.
- Alternative considered: fixed 30s process timeout. Rejected as config-ignoring behavior.

7. Integration test stabilization will remove fixed-port and race assumptions.
- Decision: Use dynamic ports where possible, readiness polling before requests, and deterministic cleanup ordering.
- Rationale: Eliminates intermittent EOF/failures caused by shared port reuse and shutdown overlap.
- Alternative considered: keep static port with longer sleeps. Rejected as non-deterministic.

## Risks / Trade-offs

- [Risk] Compatibility mode can prolong dual-contract complexity. -> Mitigation: document deprecation window and add explicit tests for canonical + alias inputs.
- [Risk] Docs toolchain upgrade (if needed for OpenAPI 3.x) may impact CI generation scripts. -> Mitigation: isolate generation command changes and add validation checks in CI.
- [Risk] Health payload expansion may reveal runtime details not intended for all environments. -> Mitigation: keep sensitive fields out of payload and scope to non-secret operational metadata.
- [Risk] Tightened validation can reject previously accepted configs. -> Mitigation: provide actionable validation error messages and update config examples.

## Migration Plan

1. Implement compatibility-first API shaping (accept aliases, emit canonical fields).
2. Update handler/engine integration for 409 non-terminal result behavior.
3. Unify request-id retrieval and error response path.
4. Implement canonical health payload DTO mapping.
5. Update router/docs generation and verify runtime/docs parity.
6. Wire shutdown timeout to config and enforce timeout validation.
7. Stabilize integration tests (dynamic port/readiness probing) and run focused API test suite.
8. After successful rollout window, decide whether to retire compatibility aliases.

Rollback strategy:
- Revert to previous handler payload mapping and docs route behavior via targeted commits if client impact is detected.
- Keep compatibility alias support behind small, isolated handler mapping logic to allow quick rollback.

## Open Questions

- Should compatibility alias output include both canonical and legacy fields temporarily, or canonical-only output with alias input acceptance?
- Is full OpenAPI 3.x generation mandatory in the short term, or is a transitional docs contract acceptable if `/docs` parity is restored first?
- What is the target deprecation window for legacy request/response field aliases?
