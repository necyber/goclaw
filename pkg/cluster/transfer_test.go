package cluster

import "testing"

func TestOwnershipTransferManager_DedupeScopedByShard(t *testing.T) {
	manager := NewOwnershipTransferManager()
	manager.SetActiveToken("shard-a", 1)
	manager.SetActiveToken("shard-b", 1)

	if err := manager.StartInFlight("shard-a", "workload-1", 1); err != nil {
		t.Fatalf("StartInFlight(shard-a) error = %v", err)
	}
	if err := manager.StartInFlight("shard-b", "workload-1", 1); err != nil {
		t.Fatalf("StartInFlight(shard-b) error = %v", err)
	}

	completed, err := manager.CompleteInFlight("shard-a", "workload-1", 1)
	if err != nil {
		t.Fatalf("CompleteInFlight(shard-a) error = %v", err)
	}
	if !completed {
		t.Fatal("expected shard-a completion to succeed")
	}

	completed, err = manager.CompleteInFlight("shard-b", "workload-1", 1)
	if err != nil {
		t.Fatalf("CompleteInFlight(shard-b) error = %v", err)
	}
	if !completed {
		t.Fatal("expected shard-b completion to succeed independently")
	}
}

func TestOwnershipTransferManager_DuplicateSuppressionRemainsPerShard(t *testing.T) {
	manager := NewOwnershipTransferManager()
	manager.SetActiveToken("shard-a", 7)

	if err := manager.StartInFlight("shard-a", "workload-dup", 7); err != nil {
		t.Fatalf("StartInFlight() error = %v", err)
	}
	completed, err := manager.CompleteInFlight("shard-a", "workload-dup", 7)
	if err != nil {
		t.Fatalf("CompleteInFlight(first) error = %v", err)
	}
	if !completed {
		t.Fatal("expected first completion to succeed")
	}

	completed, err = manager.CompleteInFlight("shard-a", "workload-dup", 7)
	if err != nil {
		t.Fatalf("CompleteInFlight(second) error = %v", err)
	}
	if completed {
		t.Fatal("expected duplicate completion on same shard to be suppressed")
	}
}

func TestOwnershipTransferManager_QueueWorkAllowsSameWorkloadAcrossShardsAfterCompletion(t *testing.T) {
	manager := NewOwnershipTransferManager()
	manager.SetActiveToken("shard-a", 3)
	manager.SetActiveToken("shard-b", 3)

	if err := manager.StartInFlight("shard-a", "workload-x", 3); err != nil {
		t.Fatalf("StartInFlight() error = %v", err)
	}
	completed, err := manager.CompleteInFlight("shard-a", "workload-x", 3)
	if err != nil || !completed {
		t.Fatalf("CompleteInFlight() = (%v, %v), want (true, nil)", completed, err)
	}

	if err := manager.QueueWork("shard-b", "workload-x", "payload"); err != nil {
		t.Fatalf("QueueWork() error = %v", err)
	}

	snapshot := manager.TransferShard("shard-b", 4)
	if len(snapshot.Queued) != 1 || snapshot.Queued[0].ID != "workload-x" {
		t.Fatalf("queued snapshot = %+v, want workload-x in shard-b queue", snapshot.Queued)
	}
}
