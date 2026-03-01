# HTTP API Compatibility and Deprecation Guidance

This document captures compatibility behavior for Week 5 HTTP API contract alignment.

## Canonical Fields

- Workflow identifier: `workflow_id`
- Task dependency list: `dependencies`

## Legacy Aliases (Compatibility Window)

- Workflow identifier alias: `id` (responses still include this for backward compatibility)
- Dependency alias in requests: `depends_on` (accepted and normalized to `dependencies`)

## Behavioral Alignment

- `GET /api/v1/workflows/{id}/tasks/{tid}/result` returns:
  - `409 Conflict` for non-terminal task states (`pending`, `scheduled`, `running`)
  - `200 OK` with persisted `status`/`result`/`error` for terminal states (`completed`, `failed`, `cancelled`)
- Workflow list pagination:
  - Default `limit=50`
  - Maximum effective `limit=100`
  - Invalid pagination values return `400 Bad Request`

## Docs Endpoint Contract

- Primary docs endpoint: `GET /docs`
- Compatibility alias: `GET /swagger/index.html`

## Deprecation Plan

1. Keep alias input support (`depends_on`) during migration.
2. Update clients to prefer canonical fields (`workflow_id`, `dependencies`).
3. Announce a removal timeline for alias output/input once ecosystem migration is complete.
