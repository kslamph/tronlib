// Package event decodes TRON transaction log entries into typed event data.
//
// # Registry
//
// Decoding works off a signature registry. The first log topic is the
// keccak-256 of the canonical event signature
// ("Transfer(address,address,uint256)"), and the whole 32-byte topic is the
// registry key — not a prefix of it. A 4-byte prefix collides often enough to
// matter (two unrelated signatures can share one), and a colliding prefix made
// a log decode against someone else's definition. Decode rejects a first topic
// that is not exactly 32 bytes as malformed.
//
// The registry comes pre-loaded with 747 built-in ecosystem event definitions
// (including TRC-20's Transfer and Approval), vendored from v1's generated
// table and auto-registered in init() — decoding is zero-config, as in v1.
// Register additional ABIs with RegisterABIJSON (Solidity JSON ABI) or
// RegisterABIObject (a *core.SmartContract_ABI); BuiltinTRC20 explicitly
// re-asserts the built-in definitions (idempotent, a near no-op after init).
//
// Indexed parameters are not part of the hashed signature, so two contracts
// can share a topic while disagreeing about which parameters are indexed — and
// only the emitting contract can say which layout a log uses. That motivates
// the two rules the registry enforces:
//
//   - Registrations never overwrite silently. Re-registering an identical
//     definition is a no-op; registering a DIFFERENT one for a signature
//     already on file makes that signature ambiguous, and it then decodes
//     nowhere (event.unknown) rather than guessing a layout.
//   - Definitions can be registered per emitting address with
//     RegisterABIJSONForAddress / RegisterABIObjectForAddress and decoded with
//     DecodeFor / DecodeLenientFor. Address-scoped definitions win for that
//     contract; signatures it does not define still resolve through the global
//     registry, so zero-config decoding keeps working.
//
// The registry is global mutable state, preserving v1's eventdecoder
// semantics: registrations are process-wide and built-in definitions never
// overwrite an explicitly registered one. A package-level registry was chosen
// over receiver-based decoders because log decoding is done opportunistically
// (receipts, streams) where threading a decoder instance through every caller
// costs more than the shared state. Per-address scopes are the escape hatch
// from the sharing that decision implies.
//
// Unlike v1 — which returned a placeholder for unknown signatures — Decode
// returns an error with code tron.CodeEventUnknown when no definition matches
// or when the match is ambiguous. Decode errors are *tron.Error; malformed log
// shapes and data that doesn't match the registered ABI types carry
// tron.CodeContractArgMismatch, so callers can distinguish "unknown event" from
// "known event, corrupt log". DecodeLenient is the tolerant variant: it
// materializes unknown and ambiguous signatures as a Log with an empty
// EventName and the raw Topics/Data preserved, so receipt consumers never drop
// logs they cannot name. Receipt.Logs and Events() use DecodeLenientFor for
// exactly that reason, with the emitting address as the scope.
//
// # Quick Start
//
//	log, err := event.Decode(topics, data) // zero-config: built-ins pre-registered
//	// log.EventName == "Transfer", log.Parameters[i].Value is the decoded value
//
//	// A contract whose ABI is known, or whose layout the global registry
//	// cannot settle, decodes through its address instead:
//	event.RegisterABIJSONForAddress(contract, abiJSON)
//	log, err := event.DecodeFor(contract, topics, data)
//
// # Parameters
//
// Parameters are returned in the event's declared input order. Indexed
// parameters are decoded from topics[1:], non-indexed from the data blob
// via standard ABI decoding. Values are typed: address → tron.Address,
// integers → *big.Int, bool → bool, string → string, bytesN/bytes → []byte.
// Tuple and array parameters pass through as geth-decoded Go shapes
// (go-ethereum's representation for tuples and slices for arrays) without
// further mapping.
package event
