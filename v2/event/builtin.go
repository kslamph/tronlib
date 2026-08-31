package event

import "sync"

// Builtin TRC-20 event definitions. v1 generated a large ecosystem-wide
// table; v2 scopes the built-ins to the standard TRC-20 interface (Transfer
// and Approval) — the two events every TRC-20 contract emits and the ones
// tx.Receipt decoding and the facade consume. Contracts with custom events
// register their ABI via RegisterABIJSON/RegisterABIObject.

// builtinTRC20 maps the first 4 bytes of
// keccak256("Transfer(address,address,uint256)") and
// keccak256("Approval(address,address,uint256)") to their definitions.
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

// builtinOnce guards the one-time insertion of the built-in definitions.
var builtinOnce sync.Once

// BuiltinTRC20 registers the standard TRC-20 event definitions
// (Transfer, Approval) in the global registry. It is idempotent and safe
// for concurrent use; built-in definitions never overwrite a previously
// registered signature.
//
// v1 registered these implicitly via package init(). v2 requires the
// explicit call (keeping package import free of global side effects);
// call it once at startup or immediately before decoding.
func BuiltinTRC20() {
	builtinOnce.Do(func() {
		registerBuiltin(builtinTRC20)
	})
}
