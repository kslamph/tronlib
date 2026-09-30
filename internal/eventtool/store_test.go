package eventtool

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/v2/event"
)

// hexKey is the corpus's sighash encoding: lowercase hex of the full 32-byte
// key, no "0x".
func hexKey(name string, types []string) string {
	k := event.SignatureKey(name, types)
	return hex.EncodeToString(k[:])
}

func TestStoreRoundTripAndFirstWins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	s := New(path)
	first := SavedEvent{
		Sighash:   hexKey("Ping", []string{"uint256"}),
		Signature: "Ping(uint256)",
		Name:      "Ping",
		Inputs:    []SavedInput{{Type: "uint256", Name: "n"}},
	}
	if !s.Upsert(first) {
		t.Fatal("first upsert should insert")
	}
	// Same sighash, different body: first-wins must keep it and report no insert.
	if s.Upsert(SavedEvent{Sighash: first.Sighash, Signature: first.Signature, Name: "Other"}) {
		t.Fatal("second upsert with the same sighash must be a no-op")
	}
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Len() != 1 {
		t.Fatalf("Len = %d, want 1", got.Len())
	}
	e := got.Events()[0]
	if e.Name != "Ping" || e.Signature != "Ping(uint256)" {
		t.Fatalf("first-wins lost: %+v", e)
	}
}

func TestStoreRejectsOldSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.json")
	old := `[{"selector":"c0819c13","signature":"FeesWithdrawn(address,uint256)",` +
		`"name":"FeesWithdrawn","inputs":[{"type":"address","indexed":false,"name":"to"}]}]`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "migrate") {
		t.Fatalf("want a migrate-first error for the old schema, got %v", err)
	}
}

func TestStoreRejectsSighashMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	bad := `[{"sighash":"` + strings.Repeat("00", 32) + `","signature":"Ping(uint256)",` +
		`"name":"Ping","inputs":[{"type":"uint256","indexed":false,"name":"n"}]}]`
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "sighash") {
		t.Fatalf("want a sighash mismatch error, got %v", err)
	}
}

func TestStoreSaveIsSortedAndDeterministic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	s := New(path)
	// Insert in descending name order; names and sighashes share no order, so
	// alphabetical name order would be a false signal.
	s.Upsert(SavedEvent{Sighash: hexKey("Zed", nil), Signature: "Zed()", Name: "Zed"})
	s.Upsert(SavedEvent{Sighash: hexKey("Axe", nil), Signature: "Axe()", Name: "Axe"})
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Entries must be written in ascending sighash order.
	var written []SavedEvent
	if err := json.Unmarshal(first, &written); err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 || written[0].Sighash >= written[1].Sighash {
		t.Fatalf("corpus not sorted by sighash: %+v", written)
	}
	// Same set, different insertion order -> byte-identical file.
	other := New(path)
	other.Upsert(SavedEvent{Sighash: hexKey("Axe", nil), Signature: "Axe()", Name: "Axe"})
	other.Upsert(SavedEvent{Sighash: hexKey("Zed", nil), Signature: "Zed()", Name: "Zed"})
	if err := other.Save(); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("saved corpus depends on insertion order; diffs would be unstable")
	}
}
