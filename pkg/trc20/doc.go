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
