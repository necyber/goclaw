## Why

Week 12 Web UI currently breaks expected behavior when `ui.base_path` is customized, and dev-proxy routing does not align with mounted UI paths. The change is needed now to restore spec conformance and prevent deployment-time routing failures outside the default `/ui` path.

## What Changes

- Make frontend routing and asset base path respect configurable `ui.base_path` instead of hardcoding `/ui`.
- Fix UI dev proxy behavior so requests under configured base path are forwarded correctly to the Vite dev server.
- Align WebSocket endpoint documentation to the implemented endpoint (`/ws/events`) and remove conflicting references.
- Add/adjust tests for custom base path and proxy path behavior to prevent regressions.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `http-server`: clarify and enforce custom UI base path behavior and dev-proxy routing semantics.
- `static-embedding`: ensure static serving and SPA fallback requirements apply under configured UI base paths.
- `dashboard-layout`: require client-side router and build-time base URL to follow server-configured UI mount path.
- `realtime-updates`: enforce endpoint naming consistency for WebSocket updates at `/ws/events`.

## Impact

- Affected code: `pkg/api/router.go`, `pkg/api/router_test.go`, `web/src/App.tsx`, `web/vite.config.ts`, related frontend tests, and week12 archived docs.
- API/surface: no new endpoint; documentation corrected to existing `/ws/events`.
- Runtime behavior: custom `ui.base_path` deployments and dev mode proxy become reliable.
