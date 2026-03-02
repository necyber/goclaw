## Context

Week 12 introduced Web UI embedding, SPA routing, and dev proxy support, but the implementation still hardcodes `/ui` in the frontend router and Vite build base. The backend supports configurable `ui.base_path`, so current behavior diverges when deployments mount UI under non-default paths (for example `/dashboard`), and dev proxy forwarding does not normalize paths for Vite.

This change is cross-cutting across backend routing, static serving, frontend runtime bootstrap, and documentation consistency.

## Goals / Non-Goals

**Goals:**
- Make UI work correctly when mounted at any valid `ui.base_path`.
- Keep one frontend build artifact usable across different server `ui.base_path` values.
- Ensure dev proxy forwards UI requests to Vite with correct rewritten path.
- Remove endpoint naming drift and keep WebSocket endpoint references consistent with `/ws/events`.
- Add regression tests for the above behavior.

**Non-Goals:**
- No new WebSocket endpoint design.
- No auth, RBAC, or tenant-aware routing changes.
- No change to existing API payload formats.
- No redesign of dashboard feature set.

## Decisions

### Decision 1: Remove hardcoded `/ui` from frontend routing and bootstrap base path at runtime

**Choice:** Provide UI base path to frontend at runtime from server-rendered index metadata/config and consume it in router bootstrap.

**Why:**
- Server config controls mount path at runtime; a compile-time-only value is insufficient.
- A single binary/build should run under different mount paths without rebuilding frontend assets.

**Alternatives considered:**
- Keep hardcoded `/ui`: rejected, fails custom mount requirement.
- Build per-environment with different Vite `base`: rejected, operationally brittle and conflicts with single-artifact distribution goal.

### Decision 2: Use base-path-safe static asset references

**Choice:** Build static assets with path strategy that does not lock to `/ui` (base-path-safe), so embedded `index.html` resolves assets under whichever mount path serves it.

**Why:**
- Current absolute `/ui/...` asset paths break when mounted elsewhere.
- Keeps embedding workflow unchanged while removing mount-point coupling.

**Alternatives considered:**
- Rewrite all asset URLs in server at request time: rejected due to higher complexity and error risk.

### Decision 3: Rewrite dev proxy request path by stripping configured base path

**Choice:** In `newUIDevProxy`/route registration path, strip configured `ui.base_path` before forwarding to Vite upstream.

**Why:**
- UI route matcher receives `/base/...`; Vite expects app routes from root.
- Aligns dev mode behavior with production mount semantics.

**Alternatives considered:**
- Keep pass-through path and require Vite to also mount under same base: rejected because it adds fragile dual config and breaks default local workflow.

### Decision 4: Treat endpoint consistency as a contract check

**Choice:** Normalize references to WebSocket endpoint `/ws/events` in week12 change docs and keep specs/code aligned.

**Why:**
- Prevents operator/client confusion caused by stale `/ws/workflows/{id}` reference.

**Alternatives considered:**
- Ignore archived docs mismatch: rejected because it keeps conflicting guidance in-repo.

## Risks / Trade-offs

- **[Runtime base-path injection coupling]** Frontend bootstrap now depends on server-provided base path metadata/config. → Mitigation: provide deterministic fallback (`/ui`) and explicit tests for missing metadata.
- **[Proxy rewrite edge cases]** Trailing slash and nested base path normalization can produce malformed upstream paths. → Mitigation: table-driven router/proxy tests for `/ui`, `/ui/`, `/dashboard`, `/ops/ui`, and deep routes.
- **[Asset path strategy regressions]** Changing build path strategy can break existing embedded asset references. → Mitigation: run UI build and static-serving tests that validate `index.html` and asset loading under non-default base path.
- **[Documentation drift recurrence]** Future edits may reintroduce endpoint mismatch. → Mitigation: keep endpoint value defined in one canonical spec and reference it consistently.

## Migration Plan

1. Update backend proxy routing and static-serving metadata behavior behind existing config.
2. Update frontend bootstrap/base handling and build configuration.
3. Add/adjust tests (backend router tests + frontend router/bootstrap tests).
4. Verify with two configs: default `/ui` and custom `/dashboard`.
5. Update week12 archived proposal text to endpoint `/ws/events`.

Rollback:
- Revert change commit(s); system returns to previous `/ui`-only behavior.

## Open Questions

- Should runtime UI config be exposed via injected HTML metadata only, or also via a lightweight JSON endpoint for observability/debugging?
- Do we want explicit validation for multi-segment `ui.base_path` (e.g. `/ops/ui`) in config tests as a first-class supported case?
