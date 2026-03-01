## 1. Workflow Contract Alignment

- [x] 1.1 Add canonical workflow DTO fields (`workflow_id`, `dependencies`) and compatibility mapping for legacy aliases (`id`, `depends_on`).
- [x] 1.2 Update workflow submit/get/list/cancel/task-result handlers to emit Week 5 canonical response shapes.
- [x] 1.3 Enforce workflow list pagination defaults/bounds (default 50, max 100, invalid values return 400).

## 2. Runtime Semantics Alignment

- [x] 2.1 Update task result retrieval flow to return HTTP 409 for non-terminal task states.
- [x] 2.2 Ensure terminal task-result responses include persisted terminal status/result/error fields.
- [x] 2.3 Add/adjust error mapping tests for 404/409/400 behavior across workflow endpoints.

## 3. Middleware and Server Lifecycle Fixes

- [x] 3.1 Unify request-id extraction between middleware and handlers so error payload `request_id` is always propagated.
- [x] 3.2 Wire HTTP shutdown context timeout to `server.http.shutdown_timeout` in main process shutdown flow.
- [x] 3.3 Add config validation for non-positive HTTP timeout values and update config tests accordingly.

## 4. Health and Documentation Contract Alignment

- [x] 4.1 Implement canonical `/health` and `/ready` payload schemas including `timestamp` and readiness `checks` fields.
- [x] 4.2 Implement canonical `/status` payload envelope with runtime metadata/component/system sections.
- [x] 4.3 Expose `/docs` route as primary documentation endpoint and keep `/swagger/*` as compatibility alias if required.
- [x] 4.4 Update generated API specification artifacts to satisfy declared docs contract and endpoint parity.

## 5. Test Stabilization and Coverage

- [x] 5.1 Refactor API integration tests to use dynamic ports/readiness probing instead of fixed-port sleeps.
- [x] 5.2 Add regression tests for request-id propagation in structured error responses.
- [x] 5.3 Add regression tests for workflow payload compatibility (`dependencies` and `depends_on`).
- [x] 5.4 Add regression tests for docs endpoint reachability and health payload schema fields.

## 6. Verification and Rollout Readiness

- [x] 6.1 Run focused test suites for `pkg/api`, `pkg/engine`, and config validation after changes.
- [x] 6.2 Update user-facing docs/examples to reflect canonical field names and docs endpoint usage.
- [x] 6.3 Document compatibility/deprecation guidance for legacy payload aliases.
