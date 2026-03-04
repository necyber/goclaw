package cluster

import (
	"fmt"
	"os"
	"strings"
)

const (
	// CoordinationEmulationEnv enables explicit etcd/consul in-memory emulation for dev/test only.
	CoordinationEmulationEnv = "GOCLAW_CLUSTER_ALLOW_COORDINATION_EMULATION"
)

// EtcdCoordinator is the etcd adapter exposed through the unified Coordinator interface.
// It currently reuses in-memory semantics and should only be enabled through explicit emulation gate.
type EtcdCoordinator struct {
	*MemoryCoordinator
}

// ConsulCoordinator is the Consul adapter exposed through the unified Coordinator interface.
// It currently reuses in-memory semantics and should only be enabled through explicit emulation gate.
type ConsulCoordinator struct {
	*MemoryCoordinator
}

// NewEtcdCoordinator creates an etcd adapter.
func NewEtcdCoordinator() *EtcdCoordinator {
	return &EtcdCoordinator{MemoryCoordinator: NewMemoryCoordinator("etcd")}
}

// NewConsulCoordinator creates a Consul adapter.
func NewConsulCoordinator() *ConsulCoordinator {
	return &ConsulCoordinator{MemoryCoordinator: NewMemoryCoordinator("consul")}
}

// CoordinationEmulationEnabled reports whether explicit dev/test emulation override is enabled.
func CoordinationEmulationEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(CoordinationEmulationEnv)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// NewCoordinator creates a coordinator by backend name.
func NewCoordinator(backend string) (Coordinator, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "memory":
		return NewMemoryCoordinator("memory"), nil
	case "etcd":
		if CoordinationEmulationEnabled() {
			return NewEtcdCoordinator(), nil
		}
		return nil, fmt.Errorf(
			"cluster: etcd coordinator is not implemented; set %s=true for dev/test in-memory emulation",
			CoordinationEmulationEnv,
		)
	case "consul":
		if CoordinationEmulationEnabled() {
			return NewConsulCoordinator(), nil
		}
		return nil, fmt.Errorf(
			"cluster: consul coordinator is not implemented; set %s=true for dev/test in-memory emulation",
			CoordinationEmulationEnv,
		)
	default:
		return nil, fmt.Errorf("cluster: unsupported coordination backend %q", backend)
	}
}
