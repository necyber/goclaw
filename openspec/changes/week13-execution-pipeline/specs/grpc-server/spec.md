## ADDED Requirements

### Requirement: Runtime service registration completeness
The gRPC runtime MUST register all enabled business services required for execution and control operations.

#### Scenario: Startup registers workflow and related services
- **WHEN** gRPC server starts with workflow execution features enabled
- **THEN** runtime MUST register Workflow, Streaming, Batch, and admin-control services according to configuration

### Requirement: Engine adapter wiring is mandatory for enabled services
Enabled gRPC services MUST be wired to concrete engine adapters before server enters serving state.

#### Scenario: Missing adapter blocks startup
- **WHEN** a required service adapter is missing or invalid
- **THEN** gRPC runtime MUST fail startup with explicit wiring error instead of serving partial capabilities

#### Scenario: Valid adapter enables runtime operations
- **WHEN** service adapters are fully wired
- **THEN** gRPC methods MUST execute against runtime engine operations instead of placeholder behavior

