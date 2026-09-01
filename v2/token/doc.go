// Package token is the TRC-20 layer of tronlib v2: a Handle pins one
// token contract with its decimals fetched eagerly at construction, and
// Amount is an exact immutable amount in the token's atomic unit.
//
// # Eager decimals
//
// New performs the decimals() view call once, at construction, through
// v2/contract's Instance. There is no Refresh: the scale is immutable for
// the life of the Handle, so Amount parsing (Handle.Amount) and whole-
// token scaling (Handle.Whole) are pure in-memory operations that never do
// I/O and never need the scale passed. A malformed metadata response — a
// decimals value outside the uint8 range 0..255, such as a non-standard
// contract packing decimals as a wide integer — fails construction with
// contract.bad_metadata rather than yielding a Handle that formats every
// amount wrong forever. (Values 0..255 are all accepted: the scale comes
// off the wire as a uint8, and the controller's finding stands — refusing
// scales above 18 would conflate unusual-but-valid tokens with malformed
// responses.)
//
// # The uint8 accessor gap
//
// go-ethereum decodes a declared-uint8 ABI output as a Go byte, and
// v2/contract's Result has no accessor for that shape (Uint64 reads
// uint64, BigInt reads *big.Int) — reading decimals as declared would be
// a dead end. Handle therefore supplies its own ABI in which decimals is
// declared uint256: Result.BigInt() then works for BOTH the standard
// uint8 wire form (a uint8 value fits a uint256 word exactly) and the
// uint256-packed form some non-standard contracts emit (the case v1's
// decimals_uint256_test.go pinned), and the bad_metadata check becomes an
// explicit range test instead of a decode failure. The same asymmetry
// means BalanceOf needs no workaround: balanceOf is declared uint256 and
// Result.BigInt() is its accessor.
//
// # Immutable amounts
//
// Amount is a value type over the raw *big.Int: Raw() returns a copy, so
// mutating a returned raw cannot corrupt an Amount, and String() is the
// canonical decimal form (no separators, no rounding, trailing zeros
// trimmed) that round-trips through Handle.Amount. Whole converts
// whole-token counts with the same compile-time safety tron.TRX has: the
// parameter is a plain int64, so float literals and foreign numeric types
// (including tron.SUN) do not compile. Transfer rejects an
// Amount whose decimals do not match the Handle's
// (amount.decimals_mismatch) — a mixed-scale transfer would silently move
// the wrong quantity — and encodes transfer(address,uint256) with the
// amount's raw value.
package token
