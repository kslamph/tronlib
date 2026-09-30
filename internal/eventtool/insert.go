package eventtool

import (
	"encoding/json"
	"fmt"
)

// abiEntry is the minimal slice of a Solidity ABI entry that event ingestion
// needs: enough to build a canonical signature. Outputs, mutability and
// constant are irrelevant here.
type abiEntry struct {
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Anonymous bool       `json:"anonymous"`
	Inputs    []abiParam `json:"inputs"`
}

type abiParam struct {
	Type    string `json:"type"`
	Indexed bool   `json:"indexed"`
	Name    string `json:"name"`
}

// InsertABI ingests a Solidity ABI file into the store and returns how many
// new events were added. It accepts a raw ABI array or the {"abi":[...]} form
// a TronScan/compiler artifact commonly uses.
//
// Only named, non-anonymous event entries are kept: anonymous events have no
// topic0 and cannot be signature-decoded, function/constructor entries are not
// events, and an unnamed event has no canonical signature. Upsert is
// insert-if-absent, so re-inserting an ABI adds nothing.
func InsertABI(data []byte, s *Store) (int, error) {
	entries, err := parseABIEntries(data)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, e := range entries {
		if e.Type != "event" || e.Anonymous || e.Name == "" {
			continue
		}
		inputs := make([]SavedInput, len(e.Inputs))
		for i, p := range e.Inputs {
			inputs[i] = SavedInput(p)
		}
		if s.Upsert(makeSavedEvent(e.Name, inputs)) {
			added++
		}
	}
	return added, nil
}

// parseABIEntries accepts either a top-level array of entries or an object
// wrapping that array under "abi".
func parseABIEntries(data []byte) ([]abiEntry, error) {
	var entries []abiEntry
	if err := json.Unmarshal(data, &entries); err == nil {
		return entries, nil
	}
	var wrapped struct {
		ABI []abiEntry `json:"abi"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("eventtool: ABI is neither an array nor {\"abi\":[...]}: %w", err)
	}
	return wrapped.ABI, nil
}
