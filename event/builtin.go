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
// discoverability and as the documented opt-in path — e.g. to make the
// dependency on the built-ins visible at the call site, or to re-assert
// after code that registers an explicit ABI for a built-in signature
// (explicit registrations win; re-assertion does not overwrite them).
func BuiltinTRC20() {
	builtinOnce.Do(func() {
		registerBuiltin(builtinTRC20)
	})
}

// builtinTRC20 maps the first 4 bytes of
// keccak256("Transfer(address,address,uint256)") and
// keccak256("Approval(address,address,uint256)") to their definitions. It
// overlaps the generated table on purpose: registering it insert-if-absent
// is idempotent against init()'s registration of the same signatures.
var builtinTRC20 = map[[4]byte]*EventDef{
	{0xdd, 0xf2, 0x52, 0xad}: {
		Name: "Transfer",
		Inputs: []ParamDef{
			{Type: "address", Indexed: true, Name: "from"},
			{Type: "address", Indexed: true, Name: "to"},
			{Type: "uint256", Indexed: false, Name: "value"},
		},
	},
	{0x8c, 0x5b, 0xe1, 0xe5}: {
		Name: "Approval",
		Inputs: []ParamDef{
			{Type: "address", Indexed: true, Name: "owner"},
			{Type: "address", Indexed: true, Name: "spender"},
			{Type: "uint256", Indexed: false, Name: "value"},
		},
	},
}
