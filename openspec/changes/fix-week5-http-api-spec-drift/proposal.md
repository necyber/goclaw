## Why

Week 5 HTTP API behavior has drifted from its archived baseline specs, and several endpoints now expose incompatible contracts. This blocks reliable client integration, creates ambiguous API documentation, and makes verification/maintenance difficult.

## What Changes

- Re-align workflow HTTP endpoint contracts with the Week 5 baseline (request/response field names, status code semantics, and validation behavior).
- Fix request ID propagation so error payloads always include the middleware-generated request ID instead of fallback values.
- Re-align health/readiness/status response payload schemas to baseline monitoring semantics.
- Re-align API documentation behavior (documentation route contract and generated spec/version contract), and ensure docs match runtime endpoints.
- Re-align server lifecycle/config semantics for shutdown timeout and related HTTP config validation.
- Harden API integration tests to remove flaky startup/shutdown timing behavior observed in current suite.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `workflow-api-endpoints`: Restore canonical workflow endpoint contract and non-terminal task-result semantics (including 409 behavior), with compatibility-safe migration rules.
- `http-server-core`: Enforce request-id propagation and HTTP shutdown/config semantics defined by baseline behavior.
- `health-monitoring-endpoints`: Restore response payload structure and probe semantics for `/health`, `/ready`, and `/status`.
- `api-documentation`: Restore documentation endpoint/versioning contract and ensure generated docs are aligned with implemented API paths.

## Impact

- Affected code: `pkg/api/handlers/*`, `pkg/api/models/workflow.go`, `pkg/api/middleware/*`, `pkg/api/router.go`, `pkg/api/server.go`, `pkg/engine/workflow_manager.go`, `cmd/goclaw/main.go`, `config/*`, `docs/swagger/*`, and related tests.
- Affected APIs: `/api/v1/workflows*`, `/health`, `/ready`, `/status`, documentation endpoints.
- Backward compatibility: Existing clients using current drifted payload shapes may require migration unless compatibility aliases are provided during transition.
- No new external dependencies expected.
