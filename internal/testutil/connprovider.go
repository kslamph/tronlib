package testutil

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
)

// MockConnProvider is a fake connection provider handing out a single fixed
// connection — typically one obtained from DialBufconn. Its zero value returns
// a nil connection and is still useful for construction/validation tests.
//
// It satisfies every package-local ConnProvider interface (and
// pkg/client/lowlevel.ConnProvider) structurally:
//
//	GetConnection(ctx) (*grpc.ClientConn, error)
//	ReturnConnection(*grpc.ClientConn)
//	GetTimeout() time.Duration
type MockConnProvider struct {
	// Conn is returned from GetConnection. May be nil.
	Conn *grpc.ClientConn
	// Err, when non-nil, is returned from GetConnection instead of Conn —
	// use it to exercise connection-failure paths.
	Err error
	// Timeout is returned from GetTimeout. Defaults to 30s when zero.
	Timeout time.Duration
}

// NewMockConnProvider returns a MockConnProvider serving conn with a 30s
// timeout.
func NewMockConnProvider(conn *grpc.ClientConn) *MockConnProvider {
	return &MockConnProvider{Conn: conn, Timeout: 30 * time.Second}
}

// GetConnection returns the configured connection or error. A zero-value
// provider with neither Conn nor Err set fails loudly rather than handing
// out a nil connection.
func (m *MockConnProvider) GetConnection(context.Context) (*grpc.ClientConn, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if m.Conn == nil {
		return nil, fmt.Errorf("mock connection provider: no connection configured")
	}
	return m.Conn, nil
}

// ReturnConnection is a no-op.
func (m *MockConnProvider) ReturnConnection(*grpc.ClientConn) {}

// GetTimeout returns the configured timeout, defaulting to 30 seconds.
func (m *MockConnProvider) GetTimeout() time.Duration {
	if m.Timeout == 0 {
		return 30 * time.Second
	}
	return m.Timeout
}
