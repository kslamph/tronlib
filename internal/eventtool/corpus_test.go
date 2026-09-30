package eventtool

import (
	"testing"

	"github.com/kslamph/tronlib/v2/internal/eventdata"
)

// TestTrackedCorpusVerifies is the corpus-vs-registry drift gate: decodeCorpus
// re-derives every entry's signature and sighash through event.SignatureKey, so
// a corpus entry that could not have come from the registry fails here. It also
// pins the total and two canonical fixtures that the event package depends on.
func TestTrackedCorpusVerifies(t *testing.T) {
	events, err := decodeCorpus(eventdata.RegistryJSON)
	if err != nil {
		t.Fatalf("tracked corpus does not verify against the registry: %v", err)
	}
	if len(events) != 747 {
		t.Fatalf("tracked corpus has %d entries, want 747", len(events))
	}
	bySighash := make(map[string]SavedEvent, len(events))
	for _, e := range events {
		bySighash[e.Sighash] = e
	}
	fixtures := []struct {
		name  string
		types []string
	}{
		{"Transfer", []string{"address", "address", "uint256"}},
		{"SubmitTransaction", []string{"uint256", "address", "uint256", "bytes"}},
	}
	for _, f := range fixtures {
		if _, ok := bySighash[hexKey(f.name, f.types)]; !ok {
			t.Errorf("tracked corpus is missing %s", f.name)
		}
	}
}
