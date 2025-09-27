// Package network exposes simple accessors for network-level information
// such as chain parameters and node metadata.
//
// # Manager Features
//
// The network manager provides methods for network-level queries:
//
//	cli, _ := client.NewClient("grpc://grpc.trongrid.io:50051")
//	defer cli.Close()
//
//	nm := network.NewManager(cli)
//
//	// Get chain parameters
//	params, err := nm.GetChainParameters(context.Background())
//	if err != nil { /* handle */ }
//
//	// Get node info
//	info, err := nm.GetNodeInfo(context.Background())
//	if err != nil { /* handle */ }
//
// # Error Handling
//
// Common error types:
//   - ErrNetworkUnavailable - Network or node unavailable
//   - ErrInvalidResponse - Invalid response from node
//
// Always check for errors in production code.
package network
