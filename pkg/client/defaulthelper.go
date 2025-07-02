package client

import "time"

// NewMainnetClient creates a client connected to a mainnet node
func NewMainnetClient() (*Client, error) {
	config := ClientConfig{
		NodeAddress: "grpc.trongrid.io:50051",
		Timeout:     30 * time.Second,
	}
	return NewClient(config)
}

// NewShastaClient creates a client connected to Shasta testnet
func NewShastaClient() (*Client, error) {
	config := ClientConfig{
		NodeAddress: "grpc.shasta.trongrid.io:50051",
		Timeout:     30 * time.Second,
	}
	return NewClient(config)
}

// NewNileClient creates a client connected to Nile testnet
func NewNileClient() (*Client, error) {
	config := ClientConfig{
		NodeAddress: "grpc.nile.trongrid.io:50051",
		Timeout:     30 * time.Second,
	}
	return NewClient(config)
}
