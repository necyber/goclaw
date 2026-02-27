package cluster

import "fmt"

// EtcdCoordinator is the etcd-backed adapter exposed through the unified Coordinator interface.
// Current implementation reuses the in-memory semantics while preserving backend identity.
type EtcdCoordinator struct {
	*MemoryCoordinator
}

// ConsulCoordinator is the Consul-backed adapter exposed through the unified Coordinator interface.
// Current implementation reuses the in-memory semantics while preserving backend identity.
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

// NewCoordinator creates a coordinator by backend name.
func NewCoordinator(backend string) (Coordinator, error) {
	switch backend {
	case "", "memory":
		return NewMemoryCoordinator("memory"), nil
	case "etcd":
		return NewEtcdCoordinator(), nil
	case "consul":
		return NewConsulCoordinator(), nil
	default:
		return nil, fmt.Errorf("cluster: unsupported coordination backend %q", backend)
	}
}
