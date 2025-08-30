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

// Package resources manages TRON resource economics, including bandwidth and
// energy queries and operations.
//
// # Resource Types
//
// TRON uses two types of resources:
//   - Bandwidth (formerly Net): Used for most transactions
//   - Energy: Used for smart contract execution
//
// Both can be obtained by freezing TRX or consumed directly from the account balance.
//
// # Manager Features
//
// The resources manager provides methods for resource management:
//
//	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
//	defer cli.Close()
//
//	rm := resources.NewManager(cli)
//	account, _ := types.NewAddress("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
//
//	// Freeze TRX for bandwidth
//	txExt, err := rm.FreezeBalance(context.Background(), account, 1_000_000_000, "BANDWIDTH")
//	if err != nil { /* handle */ }
//
//	// Query resource usage
//	net, err := rm.GetAccountNet(context.Background(), account)
//	if err != nil { /* handle */ }
//
//	energy, err := rm.GetAccountResource(context.Background(), account)
//	if err != nil { /* handle */ }
//
// # Error Handling
//
// Common error types:
//   - ErrInsufficientBalance - Insufficient TRX for freezing
//   - ErrInvalidFreezeAmount - Invalid amount for freezing/unfreezing
//   - ErrResourceNotAvailable - Resource not available
//
// Always check for errors in production code.
package resources
