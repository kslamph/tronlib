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
