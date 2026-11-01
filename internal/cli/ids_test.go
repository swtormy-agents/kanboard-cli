package cli

import (
	"testing"
)

func TestParsePositiveID(t *testing.T) {
	ok, err := parsePositiveID("--project", "42")
	if err != nil || ok != 42 {
		t.Fatalf("42 => %d err=%v", ok, err)
	}
	bad := []string{"0", "-1", "1a", "1/2", "%2e", "18446744073709551616", "99999999999999999999999999999"}
	for _, raw := range bad {
		if _, err := parsePositiveID("--project", raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestParseTaskStatus(t *testing.T) {
	id, err := parseTaskStatus("active")
	if err != nil || id != 1 {
		t.Fatalf("active => %d err=%v", id, err)
	}
	id, err = parseTaskStatus("inactive")
	if err != nil || id != 0 {
		t.Fatalf("inactive => %d err=%v", id, err)
	}
	if _, err := parseTaskStatus("bogus"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRPCMethodValidation(t *testing.T) {
	if err := validateRPCMethod("getAllProjects"); err != nil {
		t.Fatal(err)
	}
	if err := validateRPCMethod("9bad"); err == nil {
		t.Fatal("expected error for numeric start")
	}
	if !rpcMethodReadOnly("getVersion") {
		t.Fatal("getVersion should be read-only")
	}
	if rpcMethodReadOnly("createTask") {
		t.Fatal("createTask should not be read-only")
	}
}
