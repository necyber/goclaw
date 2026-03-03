## MODIFIED Requirements

### Requirement: Cluster membership lifecycle
The runtime MUST maintain cluster membership with join, heartbeat, and leave semantics via an explicitly supported coordination backend.

#### Scenario: Node joins cluster
- **WHEN** a node starts in distributed mode with valid coordination backend connectivity
- **THEN** the node MUST register membership and begin periodic heartbeat updates

#### Scenario: Node heartbeat timeout
- **WHEN** a node heartbeat is not observed within configured lease TTL
- **THEN** the node MUST be marked unhealthy and excluded from new ownership assignments

#### Scenario: Unsupported distributed backend configuration
- **WHEN** runtime is configured with `etcd` or `consul` backend without implemented backend support
- **THEN** startup MUST fail fast or enter explicit unsupported degraded mode
- **AND** runtime MUST NOT silently emulate distributed semantics using in-memory coordinator behavior
