// Package client provides connection management and RPC helpers for interacting
// with TRON full nodes over gRPC.
//
// The core type is Client, which maintains a small pool of gRPC connections to
// a single node endpoint. Endpoints are expressed with an explicit scheme:
//   - grpc://host:port   plaintext
//   - grpcs://host:port  TLS
//
// Construction uses functional options:
//   - WithTimeout(d) applies a default timeout when a context has no deadline
//   - WithPool(init, max) configures the connection pool size
//
// # Connection Management
//
// The client maintains a pool of connections to improve performance. The pool
// size can be configured during client creation:
//
//	cli, err := client.NewClient("grpc://grpc.trongrid.io:50051",
//	    client.WithPool(5, 10),     // 5 initial, 10 max connections
//	    client.WithTimeout(30*time.Second))
//	if err != nil { /* handle */ }
//	defer cli.Close()
//
// # Quick Start
//
//	cli, err := client.NewClient("grpc://grpc.trongrid.io:50051", client.WithTimeout(30*time.Second))
//	if err != nil { /* handle */ }
//	defer cli.Close()
//
// Pass the client to higher-level managers (smartcontract, trc20, account,
// resources, network). Transport concerns remain centralized, making it easy to
// switch node endpoints.
//
// # Broadcasting
//
// Build a transaction via a manager, then sign and broadcast. Use
// DefaultBroadcastOptions to control receipt waiting and timing.
//
//	opts := client.DefaultBroadcastOptions()
//	opts.FeeLimit = 100_000_000
//	opts.WaitForReceipt = true
//	opts.WaitTimeout = 20 * time.Second
//	opts.PollInterval = 500 * time.Millisecond
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	res, err := cli.SignAndBroadcast(ctx, txExt, opts, signer)
//	_ = res; _ = err
//
// When WaitForReceipt is true and the receipt arrives in time, Success reflects
// the final on-chain execution status. res.TxID is always populated.
//
// # Simulation
//
// Predict execution result and estimate energy before sending any transaction:
//
//	sim, err := cli.Simulate(ctx, txExt /* or *core.Transaction */)
//	if err != nil { /* handle */ }
//	if !sim.Success { /* would fail */ }
//	_ = sim.EnergyUsage
//
// Simulation does not require signatures. Bandwidth (net usage) depends on
// signatures and payload; for accurate bandwidth, broadcast a signed
// transaction and inspect the receipt.
//
// # Error Handling
//
// The client returns specific error types for common issues:
//   - ErrNoConnection - No connection available in the pool
//   - ErrTimeout - Operation timed out
//   - ErrInvalidEndpoint - Invalid endpoint format
//
// Always check for errors in production code.
//
// # Best Practices
//
//  1. Always close the client when finished to free up resources:
//     defer cli.Close()
//
//  2. Use context with timeout for all operations to prevent hanging:
//     ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//     defer cancel()
//
// 3. Configure appropriate pool sizes based on your application's concurrency needs
//
// 4. Use simulation to estimate energy costs before broadcasting transactions
//
// 5. Handle errors appropriately, especially network and timeout errors
//
// # Common Usage Patterns
//
//  1. Creating a client with custom configuration:
//     cli, err := client.NewClient("grpc://grpc.trongrid.io:50051",
//     client.WithTimeout(30*time.Second),
//     client.WithPool(5, 10))
//
//  2. Broadcasting a transaction with custom options:
//     opts := client.DefaultBroadcastOptions()
//     opts.FeeLimit = 50_000_000
//     opts.WaitForReceipt = true
//     result, err := cli.SignAndBroadcast(ctx, tx, opts, signer)
//
//  3. Simulating a transaction before broadcasting:
//     sim, err := cli.Simulate(ctx, tx)
//     if err != nil { /* handle error */ }
//     if !sim.Success { /* would fail on chain */ }
package client
