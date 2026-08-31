// Package event decodes TRON transaction log entries into typed event data.
//
// # Registry
//
// Decoding works off a signature registry: the first log topic is the
// keccak-256 of the canonical event signature ("Transfer(address,address,uint256)"),
// and the first 4 bytes of that topic key into the registry. Register ABI
// sources with RegisterABIJSON (Solidity JSON ABI) or RegisterABIObject
// (a *core.SmartContract_ABI), or use the built-in TRC-20 definitions via
// BuiltinTRC20 (Transfer and Approval).
//
// The registry is global mutable state, preserving v1's eventdecoder
// semantics: registrations are process-wide, last write wins per signature,
// and built-in definitions never overwrite an explicitly registered one.
// A package-level registry was chosen over receiver-based decoders because
// log decoding is done opportunistically (receipts, streams) where threading
// a decoder instance through every caller costs more than the shared state.
//
// Unlike v1 — which returned a placeholder for unknown signatures — Decode
// returns an error with code tron.CodeEventUnknown when no definition
// matches. Decode errors are *tron.Error; malformed log shapes and data
// that doesn't match the registered ABI types carry
// tron.CodeContractArgMismatch, so callers can distinguish "unknown event"
// from "known event, corrupt log".
//
// # Quick Start
//
//	event.BuiltinTRC20()
//	log, err := event.Decode(topics, data)
//	// log.EventName == "Transfer", log.Parameters[i].Value is the decoded value
//
// # Parameters
//
// Parameters are returned in the event's declared input order. Indexed
// parameters are decoded from topics[1:], non-indexed from the data blob
// via standard ABI decoding. Values are typed: address → tron.Address,
// integers → *big.Int, bool → bool, string → string, bytesN/bytes → []byte.
package event
