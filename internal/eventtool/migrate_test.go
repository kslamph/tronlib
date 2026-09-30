package eventtool

import (
	"encoding/json"
	"strings"
	"testing"
)

const v1FeesWithdrawn = `[{
	"selector":"c0819c13",
	"signature":"FeesWithdrawn(address,uint256)",
	"name":"FeesWithdrawn",
	"inputs":[
		{"type":"address","indexed":false,"name":"to"},
		{"type":"uint256","indexed":false,"name":"amount"}
	]
}]`

func TestMigrateDerivesSighashAndChecksSelector(t *testing.T) {
	got, err := Migrate([]byte(v1FeesWithdrawn))
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("entries = %d, want 1", len(got))
	}
	e := got[0]
	if e.Sighash != hexKey("FeesWithdrawn", []string{"address", "uint256"}) {
		t.Fatalf("sighash = %s, want %s", e.Sighash, hexKey("FeesWithdrawn", []string{"address", "uint256"}))
	}
	if e.Signature != "FeesWithdrawn(address,uint256)" || e.Name != "FeesWithdrawn" {
		t.Fatalf("unexpected entry %+v", e)
	}
	if len(e.Inputs) != 2 || e.Inputs[1].Name != "amount" {
		t.Fatalf("inputs lost in migration: %+v", e.Inputs)
	}
}

func TestMigrateRejectsSelectorMismatch(t *testing.T) {
	bad := strings.Replace(v1FeesWithdrawn, `"c0819c13"`, `"deadbeef"`, 1)
	_, err := Migrate([]byte(bad))
	if err == nil || !strings.Contains(err.Error(), "selector") {
		t.Fatalf("want selector mismatch error, got %v", err)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	first, err := Migrate([]byte(v1FeesWithdrawn))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Migrate(encoded)
	if err != nil {
		t.Fatalf("re-migrating the new schema should verify, not fail: %v", err)
	}
	if len(second) != 1 || second[0].Sighash != first[0].Sighash {
		t.Fatalf("not idempotent: %+v vs %+v", first, second)
	}
}
