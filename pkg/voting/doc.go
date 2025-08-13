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

// Package voting provides a high-level manager for witness voting and related
// governance operations.
//
// # Manager Features
//
// The voting manager provides methods for voting operations:
//
//	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
//	defer cli.Close()
//
//	vm := voting.NewManager(cli)
//	voter, _ := types.NewAddress("Tvoterxxxxxxxxxxxxxxxxxxxxxxxxxx")
//	witness, _ := types.NewAddress("Twitnessxxxxxxxxxxxxxxxxxxxxxxxx")
//
//	// Vote for witnesses
//	txExt, err := vm.VoteWitness(context.Background(), voter, map[types.Address]int64{
//	    witness: 1000000, // 1,000,000 TRX equivalent votes
//	})
//	if err != nil { /* handle */ }
//
//	// Get votes
//	votes, err := vm.ListVotes(context.Background(), voter)
//	if err != nil { /* handle */ }
//
// # Error Handling
//
// Common error types:
//   - ErrInvalidAddress - Invalid TRON address
//   - ErrInvalidVoteCount - Invalid vote count
//   - ErrInsufficientVotes - Insufficient votes available
//
// Always check for errors in production code.
package voting
