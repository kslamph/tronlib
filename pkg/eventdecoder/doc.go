// Package eventdecoder maintains a compact registry of event signatures and
// helpers to decode logs into typed values using minimal ABI fragments.
//
// Two usage modes are supported:
//  1. Register ABI sources at runtime (JSON or *core.SmartContract_ABI)
//  2. Use builtin signatures generated from common ecosystems
//
// Given topics and data from a TransactionInfo_Log, DecodeLog will look up
// the first topic's 4-byte prefix, build a minimal ABI for the matched
// signature, and return a DecodedEvent. Unknown signatures are surfaced with
// a placeholder name and no parameters rather than failing.
package eventdecoder
