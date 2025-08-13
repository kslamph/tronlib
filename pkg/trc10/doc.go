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

// Package trc10 offers helpers to query TRC10 token metadata and balances.
//
// # Manager Features
//
// The TRC10 manager provides methods for TRC10 token operations:
//
//	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
//	defer cli.Close()
//
//	tm := trc10.NewManager(cli)
//	tokenID := int64(1000001)
//
//	// Get token metadata
//	asset, err := tm.GetAssetIssueByID(context.Background(), tokenID)
//	if err != nil { /* handle */ }
//
//	// Get account balance
//	balance, err := tm.GetAccountAssetBalance(context.Background(), account, tokenID)
//	if err != nil { /* handle */ }
//
// # Error Handling
//
// Common error types:
//   - ErrInvalidTokenID - Invalid token ID
//   - ErrTokenNotFound - Token not found on chain
//
// Always check for errors in production code.
package trc10
