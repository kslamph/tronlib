package smartcontract

import (
	"context"
	"time"

	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
)

// Helper function to create a mock address for testing
func createMockAddress() *types.Address {
	addr, _ := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	return addr
}

// mockClient implements contractClient interface for testing
type mockClient struct{}

// GetConnection returns a nil connection for testing (should not be called in unit tests)
func (m *mockClient) GetConnection(ctx context.Context) (*grpc.ClientConn, error) {
	// For unit tests, this should not be called since we're only testing encoding/decoding
	return nil, nil
}

// ReturnConnection is a no-op for testing
func (m *mockClient) ReturnConnection(conn *grpc.ClientConn) {
	// No-op for testing
}

// GetTimeout returns a default timeout for testing
func (m *mockClient) GetTimeout() time.Duration {
	return 30 * time.Second
}

// createMockClient creates a mock client for testing
func createMockClient() contractClient {
	return &mockClient{}
}
