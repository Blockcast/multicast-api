package api

import "testing"

func TestRepairSessionIDWireContract(t *testing.T) {
	if RepairSessionIDHeader != "X-Repair-Session-ID" {
		t.Fatalf("RepairSessionIDHeader = %q", RepairSessionIDHeader)
	}

	id, err := NewRepairSessionID(42)
	if err != nil {
		t.Fatalf("NewRepairSessionID: %v", err)
	}
	if id != "42" {
		t.Fatalf("NewRepairSessionID(42) = %q", id)
	}
	if parsed, err := ParseRepairSessionID(string(id)); err != nil || parsed != id {
		t.Fatalf("ParseRepairSessionID(%q) = %q, %v", id, parsed, err)
	}
}

func TestRepairSessionIDRejectsNonCanonicalValues(t *testing.T) {
	for _, value := range []string{"", "0", "01", "-1", "session-1"} {
		t.Run(value, func(t *testing.T) {
			if _, err := ParseRepairSessionID(value); err == nil {
				t.Fatalf("ParseRepairSessionID(%q) succeeded", value)
			}
		})
	}
	if _, err := NewRepairSessionID(0); err == nil {
		t.Fatal("NewRepairSessionID(0) succeeded")
	}
}
