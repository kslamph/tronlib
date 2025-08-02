package smartcontract

import (
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// Helper function to create a mock client for testing
func createMockClient(t *testing.T) *client.Client {
	// For tests where ABI is provided, we don't need a fully functional client.
	// We just need a non-nil client.Client pointer.
	return &client.Client{}
}

// Helper function to create a mock address for testing
func createMockAddress() *types.Address {
	addr, _ := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	return addr
}

func mustAddr(t *testing.T, s string) *types.Address {
	a, err := types.NewAddress(s)
	if err != nil {
		t.Fatalf("failed to create address from string %q: %v", s, err)
	}
	return a
}
