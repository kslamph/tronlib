package eventtool

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/kslamph/tronlib/v2/event"
)

// SavedInput is one event parameter as stored in the corpus. The JSON tags are
// the corpus schema's field names; the shape mirrors event.ParamDef, which has
// no tags of its own.
type SavedInput struct {
	Type    string `json:"type"`
	Indexed bool   `json:"indexed"`
	Name    string `json:"name"`
}

// SavedEvent is one corpus entry: a named event plus the canonical signature
// whose keccak256 is Sighash. The v1 corpus keyed events by a 4-byte selector;
// this schema keys them by the full 32-byte hash, matching the registry.
type SavedEvent struct {
	Sighash   string       `json:"sighash"`
	Signature string       `json:"signature"`
	Name      string       `json:"name"`
	Inputs    []SavedInput `json:"inputs"`
}

// Store is an in-memory corpus backed by a JSON file. Upserts are
// insert-if-absent (first-wins): a later capture that disagrees about an
// already-known signature cannot overwrite the first accepted layout.
type Store struct {
	path   string
	events map[string]SavedEvent
}

// New returns an empty store that saves to path.
func New(path string) *Store {
	return &Store{path: path, events: make(map[string]SavedEvent)}
}

// Path returns the file the store saves to.
func (s *Store) Path() string { return s.path }

// Len is the number of distinct signatures held.
func (s *Store) Len() int { return len(s.events) }

// Events returns the corpus sorted by Sighash, so saves and generators produce
// stable, reviewable diffs.
func (s *Store) Events() []SavedEvent {
	out := make([]SavedEvent, 0, len(s.events))
	for _, e := range s.events {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sighash < out[j].Sighash })
	return out
}

// Upsert inserts e if its sighash is not already present and reports whether it
// inserted. An existing entry is never overwritten (D6, first-wins).
func (s *Store) Upsert(e SavedEvent) bool {
	if _, ok := s.events[e.Sighash]; ok {
		return false
	}
	s.events[e.Sighash] = e
	return true
}

// Save writes the corpus atomically (temp file + rename) as a sorted JSON
// array, world-readable like the rest of the checked-in data.
func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.Events(), "", "  ")
	if err != nil {
		return fmt.Errorf("eventtool: encode corpus: %w", err)
	}
	data = append(data, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("eventtool: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("eventtool: replace %s: %w", s.path, err)
	}
	return nil
}

// Load reads a corpus file. It accepts only the 32-byte schema and verifies
// every entry's signature and sighash derivation; a v1 {selector,...} file is
// rejected with an explicit "migrate first" error rather than reinterpreted.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("eventtool: read corpus %s: %w", path, err)
	}
	events, err := decodeCorpus(data)
	if err != nil {
		return nil, fmt.Errorf("eventtool: %s: %w", path, err)
	}
	s := New(path)
	for _, e := range events {
		s.Upsert(e)
	}
	return s, nil
}

// decodeCorpus parses and verifies a corpus in the current (32-byte) schema.
// Exported callers get this through Load or the migrate/insert paths.
func decodeCorpus(data []byte) ([]SavedEvent, error) {
	var probe []struct {
		Selector string `json:"selector"`
		Sighash  string `json:"sighash"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("corpus is not a JSON array: %w", err)
	}
	for _, p := range probe {
		if p.Selector != "" {
			return nil, fmt.Errorf("corpus uses the v1 selector schema; run `eventtool migrate` first")
		}
		if p.Sighash == "" {
			return nil, fmt.Errorf("corpus entry has no sighash; run `eventtool migrate` first")
		}
	}
	var events []SavedEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("decode corpus: %w", err)
	}
	for i := range events {
		if err := verifyEvent(&events[i]); err != nil {
			return nil, err
		}
	}
	return events, nil
}

// verifyEvent checks an entry against the registry's derivation: the stored
// signature string must be the canonical form of name+inputs, and the stored
// sighash must be its keccak256. Together they make silent drift impossible.
func verifyEvent(e *SavedEvent) error {
	types := inputTypes(e.Inputs)
	if want := event.CanonicalSignature(e.Name, types); e.Signature != want {
		return fmt.Errorf("corpus entry %q: signature %q is not canonical form %q", e.Name, e.Signature, want)
	}
	if got := sighashOf(e.Name, types); e.Sighash != got {
		return fmt.Errorf("corpus entry %s: sighash %q does not match keccak256(%q) = %q", e.Name, e.Sighash, e.Signature, got)
	}
	return nil
}

// inputTypes lists the declared parameter types in order, the input to a
// canonical signature.
func inputTypes(inputs []SavedInput) []string {
	types := make([]string, len(inputs))
	for i, in := range inputs {
		types[i] = in.Type
	}
	return types
}

// sighashOf is the corpus's key encoding: lowercase hex of the 32-byte key.
func sighashOf(name string, types []string) string {
	k := event.SignatureKey(name, types)
	return hex.EncodeToString(k[:])
}

// makeSavedEvent builds a verified corpus entry from a name and its inputs.
func makeSavedEvent(name string, inputs []SavedInput) SavedEvent {
	types := inputTypes(inputs)
	return SavedEvent{
		Sighash:   sighashOf(name, types),
		Signature: event.CanonicalSignature(name, types),
		Name:      name,
		Inputs:    inputs,
	}
}
