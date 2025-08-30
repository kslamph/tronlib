// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
