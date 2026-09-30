package eventtool

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kslamph/tronlib/v2/event"
)

// v1Event is the corpus's old schema: a 4-byte selector plus the canonical
// signature. Migration recomputes the 32-byte sighash and asserts the old
// selector is that hash's first four bytes — the check that proves the move to
// full-width keys is lossless.
type v1Event struct {
	Selector  string       `json:"selector"`
	Signature string       `json:"signature"`
	Name      string       `json:"name"`
	Inputs    []SavedInput `json:"inputs"`
}

// Migrate converts a v1 {selector,...} corpus into the 32-byte schema. It also
// accepts a corpus already in the new schema and returns it after verification,
// so re-running migration is a no-op (idempotent).
//
// For every entry it recomputes sighash = keccak256(signature) via
// event.SignatureKey and fails loudly if the first four hash bytes do not equal
// the stored selector: a mismatch means the corpus and the registry's
// derivation disagree, and silently rewriting would hide that.
func Migrate(data []byte) ([]SavedEvent, error) {
	var probes []struct {
		Selector string `json:"selector"`
		Sighash  string `json:"sighash"`
	}
	if err := json.Unmarshal(data, &probes); err != nil {
		return nil, fmt.Errorf("corpus is not a JSON array: %w", err)
	}
	if len(probes) == 0 {
		return nil, nil
	}
	// Already the new schema: verify and return as-is (idempotent re-run).
	if probes[0].Selector == "" && probes[0].Sighash != "" {
		return decodeCorpus(data)
	}

	var v1 []v1Event
	if err := json.Unmarshal(data, &v1); err != nil {
		return nil, fmt.Errorf("decode v1 corpus: %w", err)
	}
	out := make([]SavedEvent, 0, len(v1))
	seen := make(map[string]bool, len(v1))
	for i, e := range v1 {
		types := inputTypes(e.Inputs)
		sig := event.CanonicalSignature(e.Name, types)
		if e.Signature != sig {
			return nil, fmt.Errorf("entry %d (%s): signature %q is not canonical form %q", i, e.Name, e.Signature, sig)
		}
		full := event.SignatureKey(e.Name, types)
		sum := hex.EncodeToString(full[:])
		wantSel := sum[:8]
		if !strings.EqualFold(e.Selector, wantSel) {
			return nil, fmt.Errorf("entry %d (%s): selector %q != first 4 bytes of keccak256(%q) = %q",
				i, e.Name, e.Selector, e.Signature, wantSel)
		}
		if seen[sum] {
			continue // first-wins
		}
		seen[sum] = true
		out = append(out, SavedEvent{
			Sighash:   sum,
			Signature: e.Signature,
			Name:      e.Name,
			Inputs:    e.Inputs,
		})
	}
	return out, nil
}
