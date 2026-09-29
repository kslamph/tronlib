// Package contract binds a deployed smart contract's ABI to typed Go calls:
// Instance encodes sealed Arg values into call data, executes reads
// (Call), state-changing calls (Invoke — producing a *tx.ContractTx), and
// decodes raw ABI output into typed Result values with enumerable
// accessors (spec §9).
//
// The layering is strict: contract depends on tx (Invoke returns
// *tx.ContractTx), rpc, event and tron — never the reverse.
//
// The ABI is loaded once and cached: either explicitly with UseABI(json),
// or lazily from the chain (GetContract) on first use — contract.no_abi if
// the on-chain contract publishes none, contract.bad_abi if it cannot be
// parsed. Loading also registers the ABI's event definitions with the
// event package's registry, so event.Decode works for the instance's
// events without a second registration step.
//
// Address rule (spec §9.1, normative): AddressArg strips TRON's 0x41
// prefix and encodes the 20-byte ABI form; Result.Address() re-prepends
// 0x41. AddressSliceArg/Result.Addresses() apply the same rule element-wise
// to address[]/address[N]. Sending the 21-byte value padded to 32 is the
// most common ABI-encoding mistake on TRON — it encodes cleanly, executes,
// and reads the wrong slot. The round-trip is pinned by test.
//
// Result wraps ONE decoded ABI return value. Methods that return no
// values decode to an IsNil Result; methods returning multiple values
// decode to a multi-value Result whose singular accessors fail with
// contract.result_type_mismatch — the spec's Result deliberately has no
// positional accessor, so multi-value returns are out of its model.
//
// Call's owner: spec §9's Call takes no owner parameter (view calls do
// not spend anything), but the node's triggerconstantcontract RPC
// requires an owner_address field. Call sends the 0x41-prefixed null
// address (20 zero bytes). If a node ever rejects that owner, the
// signature will need an owner parameter — a spec change, not a local
// fix.
//
// Methods()/ABI() are I/O-free accessors over the loaded ABI (empty/nil
// until UseABI or the first ABI-using call triggers the lazy fetch).
//
// CallAtBlock: the node's Wallet API has no block-anchored constant call
// (see its doc); the method returns a classified error rather than
// faking head-state reads.
//
// live-verified: pending (spec §7.5).
package contract
