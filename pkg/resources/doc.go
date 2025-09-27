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
//	cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
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
