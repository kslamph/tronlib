package event

import (
	"fmt"
	"sync"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// ParamDef is a compact representation of an event parameter definition.
type ParamDef struct {
	Type    string
	Indexed bool
	Name    string
}

// Definition is a compact representation of an event definition. It is keyed in
// the registry by the FULL 32 bytes of keccak256("Name(type,...)") — a log's
// entire first topic, not a prefix of it. Keying on a 4-byte prefix let two
// unrelated signatures share one slot, and one of them silently decoded
// against the other's definition.
type Definition struct {
	Name   string
	Inputs []ParamDef
}

// signature returns the canonical event signature whose keccak256 is the
// registry key ("Transfer(address,address,uint256)"). It delegates to
// CanonicalSignature, the exported single source of the join rule, so the
// registry and out-of-process tooling derive identical hashes.
func (d *Definition) signature() string {
	types := make([]string, len(d.Inputs))
	for i, in := range d.Inputs {
		types[i] = in.Type
	}
	return CanonicalSignature(d.Name, types)
}

// sigKey is the registry key: the whole signature hash, matching the length of
// a real topics[0].
type sigKey [32]byte

// sigKeyOf hashes a full signature string through the package's one keccak
// primitive (hashSignature), the same one SignatureKey uses.
func sigKeyOf(signature string) sigKey {
	return sigKey(hashSignature(signature))
}

// scope is one namespace of event definitions: the process-wide global
// registry, or the per-address registry of one emitting contract.
//
// ambiguous marks keys for which two different definitions were registered.
// Those keys refuse to decode instead of keeping the last writer: the
// disagreement is about which parameters are indexed, so either choice would
// mis-assign values for at least one of the two contracts.
type scope struct {
	defs      map[sigKey]*Definition
	ambiguous map[sigKey]bool
}

func newScope() *scope {
	return &scope{defs: make(map[sigKey]*Definition), ambiguous: make(map[sigKey]bool)}
}

var (
	mu     sync.RWMutex
	global = newScope()
	// scoped holds per-emitting-contract registries, consulted by DecodeFor
	// before the global registry. tron.Address is a value struct (a base58
	// string plus a [21]byte array), so it is usable as a map key.
	scoped = make(map[tron.Address]*scope)
)

// RegisterABIJSON registers all event entries from a Solidity JSON ABI
// string in the global registry. Non-event entries are ignored; registering an
// empty or event-free ABI is a no-op, not an error.
//
// A definition already registered under the same signature is replaced only
// when it is identical (which makes re-registering the same ABI a no-op). A
// DIFFERENT definition — same hashed signature, different indexed flags or
// parameter names — makes the signature ambiguous: it no longer decodes at
// all, globally or through DecodeFor, until the emitting contract registers
// its own ABI with RegisterABIJSONForAddress. Overwriting was v1's behavior
// and the defect this replaces: last writer won, and the other contract's logs
// decoded against the wrong layout.
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
	registerDefs(tron.Address{}, eventDefs(abi.Entrys))
	return nil
}

// RegisterABIJSONForAddress registers all event entries from a Solidity JSON
// ABI string for one emitting contract. Logs from that address decode with
// these definitions even when the global registry disagrees about the layout;
// logs from any other address never see them. addr must be a real address: the
// unset one is the global registry, not a scope.
func RegisterABIJSONForAddress(addr tron.Address, abiJSON string) error {
	if err := requireScopeAddress("event.RegisterABIJSONForAddress", addr); err != nil {
		return err
	}
	parsed, err := newSimpleABIParser().parseABI(abiJSON)
	if err != nil {
		return err
	}
	return RegisterABIObjectForAddress(addr, parsed)
}

// RegisterABIObjectForAddress registers all event entries from a
// SmartContract_ABI protobuf object for one emitting contract. Same semantics
// as RegisterABIJSONForAddress.
func RegisterABIObjectForAddress(addr tron.Address, abi *core.SmartContract_ABI) error {
	if err := requireScopeAddress("event.RegisterABIObjectForAddress", addr); err != nil {
		return err
	}
	if abi == nil {
		return fmt.Errorf("nil ABI")
	}
	registerDefs(addr, eventDefs(abi.Entrys))
	return nil
}

// requireScopeAddress rejects the unset address: registering against it would
// silently mean "global" while looking like an address-scoped call.
func requireScopeAddress(op string, addr tron.Address) error {
	if addr.IsZero() {
		return &tron.Error{
			Code: tron.CodeAddressInvalid,
			Op:   op,
			Hint: "the unset address is not a scope; pass the emitting contract's address, or use RegisterABIJSON to register globally",
		}
	}
	return nil
}

// eventDefs converts ABI entries into definitions, ignoring entries that are
// not events. Shared by every registration entry point; the event half of v1's
// RegisterABIEntries.
func eventDefs(entries []*core.SmartContract_ABI_Entry) []*Definition {
	var defs []*Definition
	for _, entry := range entries {
		if entry == nil || entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}
		inputs := make([]ParamDef, len(entry.Inputs))
		for i, in := range entry.Inputs {
			if in == nil {
				continue
			}
			inputs[i] = ParamDef{Type: in.Type, Indexed: in.Indexed, Name: in.Name}
		}
		defs = append(defs, &Definition{Name: entry.Name, Inputs: inputs})
	}
	return defs
}

// registerDefs registers defs in the scope for addr (global when addr is
// unset). Registrations are applied in order under one lock so that a
// self-conflicting ABI is deterministic to decode.
func registerDefs(addr tron.Address, defs []*Definition) {
	if len(defs) == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	s := global
	if !addr.IsZero() {
		s = scoped[addr]
		if s == nil {
			s = newScope()
			scoped[addr] = s
		}
	}
	for _, def := range defs {
		s.register(def)
	}
}

// register applies the identity/ambiguity rule to one definition.
func (s *scope) register(def *Definition) {
	key := sigKeyOf(def.signature())
	existing, ok := s.defs[key]
	if !ok {
		s.defs[key] = def
		return
	}
	if !sameDef(existing, def) {
		s.ambiguous[key] = true
	}
	// Identical definition: no-op, and an existing ambiguity is not repaired —
	// one of the two conflicting definitions is still on file.
}

// sameDef reports whether two definitions decode identically: same event name
// and same inputs in the same order, including indexed flags and names.
func sameDef(a, b *Definition) bool {
	if a.Name != b.Name || len(a.Inputs) != len(b.Inputs) {
		return false
	}
	for i := range a.Inputs {
		if a.Inputs[i] != b.Inputs[i] {
			return false
		}
	}
	return true
}

// registerBuiltin inserts defs into the global registry without overwriting or
// conflicting with what is already there — built-ins never displace an
// explicit registration. Called by BuiltinTRC20's explicit re-assertion and by
// the generated table's init().
//
// Keys come from the definitions themselves, not from the selector each
// generated entry is stored under, so built-ins land on the same full-hash keys
// as ABI registrations. TestBuiltinTableCountAndKeys pins that the derived key
// of every generated definition is the v1 selector it ships with, extended to
// 32 bytes.
func registerBuiltin(defs []*Definition) {
	mu.Lock()
	defer mu.Unlock()
	for _, def := range defs {
		key := sigKeyOf(def.signature())
		if _, exists := global.defs[key]; !exists {
			global.defs[key] = def
		}
	}
}

// lookup resolves the definition for a signature topic emitted by addr,
// consulting that address's registry first and the global registry when it has
// no entry for the topic. The second result reports the signature is ambiguous:
// two different layouts are on file for it, so no decoding is safe.
func lookup(addr tron.Address, key sigKey) (def *Definition, ambiguous bool) {
	mu.RLock()
	defer mu.RUnlock()
	if !addr.IsZero() {
		if s, ok := scoped[addr]; ok {
			if s.ambiguous[key] {
				return nil, true
			}
			if def, ok := s.defs[key]; ok {
				return def, false
			}
			// This address knows nothing about the topic; the global registry
			// still answers for it.
		}
	}
	if global.ambiguous[key] {
		return nil, true
	}
	return global.defs[key], false
}

// globalDef returns the definition the global registry holds for key, ignoring
// ambiguity. DecodeEventSignature can afford that: two definitions sharing a
// key have the same hashed signature by construction, so the signature string
// is the same either way even when the layouts disagree.
func globalDef(key sigKey) *Definition {
	mu.RLock()
	defer mu.RUnlock()
	return global.defs[key]
}
