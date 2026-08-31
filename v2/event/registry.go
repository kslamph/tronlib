package event

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kslamph/tronlib/pb/core"
	"golang.org/x/crypto/sha3"
)

// ParamDef is a compact representation of an event parameter definition.
type ParamDef struct {
	Type    string
	Indexed bool
	Name    string
}

// EventDef is a compact representation of an event definition, keyed in the
// registry by the first 4 bytes of keccak256("Name(type,...)").
type EventDef struct {
	Name   string
	Inputs []ParamDef
}

var (
	mu   sync.RWMutex
	sig4 = make(map[[4]byte]*EventDef)
)

// RegisterABIJSON registers all event entries from a Solidity JSON ABI
// string. Non-event entries are ignored; registering an empty or
// event-free ABI is a no-op, not an error. A signature registered here
// overwrites any previous definition (including built-ins).
func RegisterABIJSON(abiJSON string) error {
	parsed, err := newSimpleABIParser().parseABI(abiJSON)
	if err != nil {
		return err
	}
	return RegisterABIObject(parsed)
}

// RegisterABIObject registers all event entries from a SmartContract_ABI
// protobuf object. Same semantics as RegisterABIJSON.
func RegisterABIObject(abi *core.SmartContract_ABI) error {
	if abi == nil {
		return fmt.Errorf("nil ABI")
	}
	return registerABIEntries(abi.Entrys)
}

// registerABIEntries registers all event entries from the provided list
// (non-event entries are ignored). Shared by both register entry points;
// ported from v1 RegisterABIEntries.
func registerABIEntries(entries []*core.SmartContract_ABI_Entry) error {
	if len(entries) == 0 {
		return nil
	}

	local := make(map[[4]byte]*EventDef)

	for _, entry := range entries {
		if entry == nil || entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}
		// Build canonical signature string: Name(types...)
		inputs := make([]string, len(entry.Inputs))
		compactInputs := make([]ParamDef, len(entry.Inputs))
		for i, in := range entry.Inputs {
			if in == nil {
				continue
			}
			inputs[i] = in.Type
			compactInputs[i] = ParamDef{Type: in.Type, Indexed: in.Indexed, Name: in.Name}
		}
		sigStr := fmt.Sprintf("%s(%s)", entry.Name, strings.Join(inputs, ","))

		// Compute 4-byte signature key.
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write([]byte(sigStr))
		sum := hasher.Sum(nil)

		var key [4]byte
		copy(key[:], sum[:4])

		local[key] = &EventDef{
			Name:   entry.Name,
			Inputs: compactInputs,
		}
	}

	if len(local) == 0 {
		return nil
	}

	mu.Lock()
	for k, v := range local {
		sig4[k] = v // overwrite by design
	}
	mu.Unlock()
	return nil
}

// registerBuiltin inserts defs into the registry without overwriting
// explicitly registered definitions. Called once from BuiltinTRC20.
func registerBuiltin(defs map[[4]byte]*EventDef) {
	mu.Lock()
	for k, v := range defs {
		if _, exists := sig4[k]; !exists {
			sig4[k] = v
		}
	}
	mu.Unlock()
}
