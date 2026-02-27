package eventbus

import "testing"

func TestCheckCompatibility(t *testing.T) {
	prev := VersionedSchema{
		SchemaVersion: "v1",
		Fields: []FieldSchema{
			{Name: "workflow_id", Type: "string", Required: true},
			{Name: "status", Type: "string", Required: true},
		},
	}
	nextAdditive := VersionedSchema{
		SchemaVersion: "v2",
		Fields: []FieldSchema{
			{Name: "workflow_id", Type: "string", Required: true},
			{Name: "status", Type: "string", Required: true},
			{Name: "trace_id", Type: "string", Required: false},
		},
	}
	nextBreaking := VersionedSchema{
		SchemaVersion: "v3",
		Fields: []FieldSchema{
			{Name: "workflow_id", Type: "string", Required: true},
			{Name: "status", Type: "int", Required: true},
		},
	}

	additive := CheckCompatibility(prev, nextAdditive)
	if !additive.Compatible || !additive.Additive {
		t.Fatalf("expected additive compatibility, got %+v", additive)
	}
	if len(additive.AddedOptional) != 1 || additive.AddedOptional[0] != "trace_id" {
		t.Fatalf("unexpected additive report: %+v", additive)
	}

	breaking := CheckCompatibility(prev, nextBreaking)
	if breaking.Compatible || breaking.Additive {
		t.Fatalf("expected breaking schema report, got %+v", breaking)
	}
	if len(breaking.TypeChanged) == 0 {
		t.Fatalf("expected type change details, got %+v", breaking)
	}
}
