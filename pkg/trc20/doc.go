// Package trc20 provides a typed, ergonomic interface for TRC20 tokens.
//
// It wraps a generic smart contract client with convenience methods that:
//   - Cache immutable properties (name, symbol, decimals)
//   - Convert between human decimals and on-chain integer amounts
//   - Expose common actions (balance, allowance, approve, transfer)
//
// The manager requires a configured *client.Client and the token contract
// address. It preloads common metadata using the client's timeout so that
// subsequent calls are efficient.
package trc20
