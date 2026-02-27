package engine

import "testing"

func TestValidateWorkflowTransition(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus string
		newStatus string
		wantErr   bool
	}{
		{name: "create pending", oldStatus: "", newStatus: workflowStatusPending, wantErr: false},
		{name: "pending to scheduled", oldStatus: workflowStatusPending, newStatus: workflowStatusScheduled, wantErr: false},
		{name: "running to completed", oldStatus: workflowStatusRunning, newStatus: workflowStatusCompleted, wantErr: false},
		{name: "pending to running invalid", oldStatus: workflowStatusPending, newStatus: workflowStatusRunning, wantErr: true},
		{name: "terminal immutable", oldStatus: workflowStatusCompleted, newStatus: workflowStatusFailed, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkflowTransition(tt.oldStatus, tt.newStatus)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateWorkflowTransition(%q -> %q) error = %v, wantErr %v", tt.oldStatus, tt.newStatus, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTaskTransition(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus string
		newStatus string
		wantErr   bool
	}{
		{name: "create pending", oldStatus: "", newStatus: taskStatusPending, wantErr: false},
		{name: "pending to scheduled", oldStatus: taskStatusPending, newStatus: taskStatusScheduled, wantErr: false},
		{name: "running to scheduled retry", oldStatus: taskStatusRunning, newStatus: taskStatusScheduled, wantErr: false},
		{name: "scheduled to completed invalid", oldStatus: taskStatusScheduled, newStatus: taskStatusCompleted, wantErr: true},
		{name: "terminal immutable", oldStatus: taskStatusFailed, newStatus: taskStatusCompleted, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskTransition(tt.oldStatus, tt.newStatus)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateTaskTransition(%q -> %q) error = %v, wantErr %v", tt.oldStatus, tt.newStatus, err, tt.wantErr)
			}
		})
	}
}
