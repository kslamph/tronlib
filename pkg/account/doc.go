// Package account provides high-level helpers to query and mutate TRON
// accounts, such as retrieving balances, resources, and building TRX transfers.
//
// # Manager Features
//
// The account manager provides methods for common account operations:
//
//	cli, _ := client.NewClient("grpc://127.0.0.1:50051")
//	defer cli.Close()
//
//	am := cli.Account()
//	from, _ := types.NewAddress("Txxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1")
//	to, _ := types.NewAddress("Tyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy2")
//
//	// Get account balance
//	balance, err := am.GetBalance(context.Background(), from)
//	if err != nil { /* handle */ }
//
//	// Transfer TRX
//	txExt, err := am.TransferTRX(context.Background(), from, to, 1_000_000, nil)
//	if err != nil { /* handle */ }
//
// # Error Handling
//
// Common error types:
//   - ErrInvalidAddress - Invalid TRON address
//   - ErrInsufficientBalance - Insufficient balance for transfer
//   - ErrAccountNotFound - Account not found on chain
//
// Always check for errors in production code.
package account
