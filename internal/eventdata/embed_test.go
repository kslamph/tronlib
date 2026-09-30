package eventdata

import (
	"encoding/json"
	"testing"
)

// TestEmbeddedRegistryIsPresent guards the corpus relocation: the file must
// ship inside the module (go:embed), parse as JSON, and still hold the v1
// corpus's 747 entries. The count is schema-independent, so this test stays
// valid across the migration to the 32-byte schema.
func TestEmbeddedRegistryIsPresent(t *testing.T) {
	if len(RegistryJSON) == 0 {
		t.Fatal("RegistryJSON is empty: events_registry.json did not embed")
	}
	var entries []map[string]any
	if err := json.Unmarshal(RegistryJSON, &entries); err != nil {
		t.Fatalf("registry is not a JSON array: %v", err)
	}
	if len(entries) != 747 {
		t.Fatalf("registry has %d entries, want 747", len(entries))
	}
}
