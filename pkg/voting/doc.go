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
