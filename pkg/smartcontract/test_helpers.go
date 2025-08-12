package smartcontract

import (
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// Helper function to create a mock client for testing
func createMockClient() *client.Client {
	// For tests where ABI is provided, we don't need a fully functional client.
	// We just need a non-nil client.Client pointer.
	return &client.Client{}
}

// Helper function to create a mock address for testing
func createMockAddress() *types.Address {
	addr, _ := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	return addr
}
