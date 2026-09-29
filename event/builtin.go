package event

import "sync"

// builtin.go keeps the explicit built-in entry point. The full generated
// ecosystem table (747 definitions) lives in builtin_gen.go and registers
// itself via init(); this file documents the opt-in trigger.

// builtinOnce guards the one-time explicit re-assertion of the built-ins.
var builtinOnce sync.Once

// BuiltinTRC20 asserts that the built-in event definitions are registered
// in the global registry: the full generated ecosystem table (747 defs,
// builtin_gen.go) is auto-registered by package init(), so after importing
// this package the built-ins are already present and this function is a
// near-no-op. It remains as the explicit, idempotent re-assertion for
// discoverability — to make the dependency on the built-ins visible at the
// call site. Re-assertion never overwrites an explicit registration, and it
// cannot resolve a signature that two explicit registrations made ambiguous:
// only RegisterABIJSONForAddress decides which layout an emitting contract
// means.
func BuiltinTRC20() {
	builtinOnce.Do(func() {
		registerBuiltin(builtinTRC20)
	})
}

// builtinTRC20 is the TRC-20 pair. The registry key is derived from each
// definition's canonical signature, exactly as ABI registrations derive theirs,
// so these land on the same keys as the overlapping entries in the generated
// table — inserting them is therefore idempotent against init()'s registration
// of the same signatures.
var builtinTRC20 = []*EventDef{
	{
		Name: "Transfer",
		Inputs: []ParamDef{
			{Type: "address", Indexed: true, Name: "from"},
			{Type: "address", Indexed: true, Name: "to"},
			{Type: "uint256", Indexed: false, Name: "value"},
		},
	},
	{
		Name: "Approval",
		Inputs: []ParamDef{
			{Type: "address", Indexed: true, Name: "owner"},
			{Type: "address", Indexed: true, Name: "spender"},
			{Type: "uint256", Indexed: false, Name: "value"},
		},
	},
}
