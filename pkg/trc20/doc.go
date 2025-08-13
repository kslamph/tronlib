// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
//
// # Manager Features
//
// The TRC20 manager provides methods for all standard TRC20 operations:
//
//	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
//	defer cli.Close()
//
//	token, _ := types.NewAddress("Ttokenxxxxxxxxxxxxxxxxxxxxxxxxxxx")
//	mgr, _ := trc20.NewManager(cli, token)
//
//	// Read operations
//	name, _ := mgr.Name(context.Background())
//	symbol, _ := mgr.Symbol(context.Background())
//	decimals, _ := mgr.Decimals(context.Background())
//	balance, _ := mgr.BalanceOf(context.Background(), holder)
//
//	// Write operations
//	txid, txExt, err := mgr.Transfer(context.Background(), from, to, amount)
//	if err != nil { /* handle */ }
//
// # Decimal Conversion
//
// The package uses shopspring/decimal for precise decimal arithmetic:
//
//	amount := decimal.NewFromFloat(12.34)
//	txid, txExt, err := mgr.Transfer(context.Background(), from, to, amount)
//
// # Caching
//
// Immutable properties (name, symbol, decimals) are cached after first retrieval,
// making subsequent calls more efficient.
//
// # Error Handling
//
// Common error types:
//   - ErrInvalidTokenAddress - Invalid token contract address
//   - ErrInvalidAmount - Invalid amount for transfer/approve
//   - ErrInsufficientBalance - Insufficient balance for transfer
//
// Always check for errors in production code.
package trc20
