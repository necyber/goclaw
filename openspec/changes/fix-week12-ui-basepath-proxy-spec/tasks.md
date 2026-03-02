## 1. Backend UI Routing And Proxy

- [x] 1.1 Update `pkg/api/router.go` UI dev-proxy branch to rewrite upstream paths by stripping configured `ui.base_path`.
- [x] 1.2 Keep default `/ui` behavior unchanged when `ui.base_path` is empty and ensure normalization still works for custom paths.
- [x] 1.3 Add/adjust router tests for custom base path registration and proxy rewrite behavior (including nested/deep routes).

## 2. Frontend Base Path Handling

- [x] 2.1 Remove hardcoded `/ui` router basename in `web/src/App.tsx` and bind basename to server-provided/runtime UI base path.
- [x] 2.2 Update frontend build/base path configuration (for example in `web/vite.config.ts`) to avoid hardcoded `/ui/` asset URLs.
- [x] 2.3 Ensure WebSocket connection behavior remains `/ws/events` regardless of UI base path and keep existing behavior intact.

## 3. Documentation Consistency

- [x] 3.1 Fix conflicting WebSocket endpoint reference in `openspec/changes/archive/2026-02-27-week12-web-ui/proposal.md` from `/ws/workflows/{id}` to `/ws/events`.
- [x] 3.2 Verify week12 archived docs and relevant current specs consistently reference UI base-path behavior and canonical WebSocket endpoint.

## 4. Regression Tests And Validation

- [ ] 4.1 Add frontend tests covering non-default UI base path routing/bootstrap behavior.
- [ ] 4.2 Add backend tests covering dev proxy rewrite with custom `ui.base_path` and existing default-path compatibility.
- [ ] 4.3 Run targeted checks (`go test ./pkg/api ./config` and frontend tests) to confirm no regressions.
