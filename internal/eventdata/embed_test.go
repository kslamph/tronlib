package eventdata

import (
	"encoding/json"
	"testing"
)

// TestEmbeddedRegistryIsPresent guards the corpus relocation: the file must
// ship inside the module (go:embed), parse as JSON, and hold at least the v1
// corpus's 747 curated entries. capture grows the corpus over time, so the
// count is a floor, not an equality.
func TestEmbeddedRegistryIsPresent(t *testing.T) {
	if len(RegistryJSON) == 0 {
		t.Fatal("RegistryJSON is empty: events_registry.json did not embed")
	}
	var entries []map[string]any
	if err := json.Unmarshal(RegistryJSON, &entries); err != nil {
		t.Fatalf("registry is not a JSON array: %v", err)
	}
	if len(entries) < 747 {
		t.Fatalf("registry has %d entries, want at least the 747 curated entries", len(entries))
	}
}
