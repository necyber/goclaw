# api-documentation Specification

## Purpose
Define and preserve the API documentation baseline, including OpenAPI artifacts, interactive documentation endpoints, and generation workflow.

## Requirements

### Requirement: Legacy specification baseline
The system SHALL preserve and implement the legacy behavior documented for api-documentation.

#### Scenario: Baseline conformance
- **WHEN** implementations reference this capability
- **THEN** they MUST conform to the legacy details captured in the notes section.

## Notes

### Scope
The HTTP API documentation includes:
- OpenAPI/Swagger specification artifacts
- Interactive documentation endpoints
- Example requests and responses

### OpenAPI baseline
- OpenAPI version SHOULD be 3.0.3 or higher.
- The specification SHOULD include API metadata, servers, tags, schemas, and endpoint examples.

### Interactive documentation
- Swagger UI endpoint: `GET /docs`
- Optional ReDoc endpoint: `GET /redoc`
- Documentation pages SHOULD allow interactive request testing when enabled.

### Generation workflow
A typical generation workflow uses `swag`:

```bash
# Install generator
go install github.com/swaggo/swag/cmd/swag@latest

# Generate OpenAPI docs from code annotations
swag init -g cmd/goclaw/main.go -o docs/swagger
```

Generated artifacts commonly include:
- `docs/swagger/swagger.json`
- `docs/swagger/swagger.yaml`
- `docs/swagger/docs.go`

### Documentation quality expectations
- Endpoint docs SHOULD describe parameters, request body, and response status codes.
- Example payloads SHOULD be consistent with runtime API behavior.
- Data models in docs SHOULD match server-side request and response structures.

### Typical dependencies
- `github.com/swaggo/swag`
- `github.com/swaggo/http-swagger`
