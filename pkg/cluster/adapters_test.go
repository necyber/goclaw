package cluster

import "testing"

func TestNewCoordinator_MemoryBackend(t *testing.T) {
	t.Setenv(CoordinationEmulationEnv, "")
	coordinator, err := NewCoordinator("memory")
	if err != nil {
		t.Fatalf("NewCoordinator(memory) error = %v", err)
	}
	if _, ok := coordinator.(*MemoryCoordinator); !ok {
		t.Fatalf("coordinator type = %T, want *MemoryCoordinator", coordinator)
	}
}

func TestNewCoordinator_EtcdFailsWithoutExplicitEmulation(t *testing.T) {
	t.Setenv(CoordinationEmulationEnv, "")
	if _, err := NewCoordinator("etcd"); err == nil {
		t.Fatal("expected etcd backend to fail without explicit emulation override")
	}
}

func TestNewCoordinator_ConsulFailsWithoutExplicitEmulation(t *testing.T) {
	t.Setenv(CoordinationEmulationEnv, "")
	if _, err := NewCoordinator("consul"); err == nil {
		t.Fatal("expected consul backend to fail without explicit emulation override")
	}
}

func TestNewCoordinator_EtcdAllowsExplicitEmulation(t *testing.T) {
	t.Setenv(CoordinationEmulationEnv, "true")
	coordinator, err := NewCoordinator("etcd")
	if err != nil {
		t.Fatalf("NewCoordinator(etcd) error = %v", err)
	}
	if _, ok := coordinator.(*EtcdCoordinator); !ok {
		t.Fatalf("coordinator type = %T, want *EtcdCoordinator", coordinator)
	}
}

func TestNewCoordinator_ConsulAllowsExplicitEmulation(t *testing.T) {
	t.Setenv(CoordinationEmulationEnv, "1")
	coordinator, err := NewCoordinator("consul")
	if err != nil {
		t.Fatalf("NewCoordinator(consul) error = %v", err)
	}
	if _, ok := coordinator.(*ConsulCoordinator); !ok {
		t.Fatalf("coordinator type = %T, want *ConsulCoordinator", coordinator)
	}
}
