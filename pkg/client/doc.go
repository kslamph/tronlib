// Package client provides connection management and RPC helpers for interacting
// with TRON full nodes over gRPC.
//
// The core type is Client, which maintains a small pool of gRPC connections
// to a single node endpoint. Endpoints are expressed with an explicit scheme:
//   - grpc://host:port   for plaintext
//   - grpcs://host:port  for TLS
//
// Construction uses functional options:
//   - WithTimeout sets a default timeout used when a context has no deadline
//   - WithPool configures initial and maximum connections for the pool
//
// Typical usage:
//
//	cli, err := client.NewClient("grpc://127.0.0.1:50051", client.WithTimeout(30*time.Second))
//	if err != nil { /* handle */ }
//	defer cli.Close()
//
// Packages in this module accept a *client.Client to perform network calls
// (e.g., smartcontract, trc20, voting). This keeps transport concerns
// centralized and makes it straightforward to swap node endpoints.
package client

