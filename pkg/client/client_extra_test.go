package client

import (
	"context"
	"strings"
	"testing"
	"time"
)

// --- NewClient: additional validation paths ---

func TestNewClient_GrpcsScheme(t *testing.T) {
	// grpcs:// is a valid scheme but connection will fail (no TLS server)
	// We just test that it doesn't return a validation error
	c, err := NewClient("grpcs://localhost:19999", WithTimeout(100*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()
	if c.GetNodeAddress() != "grpcs://localhost:19999" {
		t.Fatalf("unexpected node address: %v", c.GetNodeAddress())
	}
}

func TestNewClient_NilOption(t *testing.T) {
	c, err := NewClient("grpc://localhost:50051", nil)
	if err != nil {
		t.Fatalf("unexpected error for nil option: %v", err)
	}
	defer c.Close()
}

func TestNewClient_InvalidURL(t *testing.T) {
	// URL with no scheme but with valid-looking content
	_, err := NewClient("://invalid")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNewClient_GRPCSchemeWithPort(t *testing.T) {
	c, err := NewClient("grpc://127.0.0.1:50051")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()
	if c.GetTimeout() != 30*time.Second {
		t.Fatalf("expected default timeout 30s, got %v", c.GetTimeout())
	}
}

func TestNewClient_CustomOptions(t *testing.T) {
	c, err := NewClient("grpc://127.0.0.1:50051",
		WithTimeout(5*time.Second),
		WithPool(3, 10),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()
	if c.GetTimeout() != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", c.GetTimeout())
	}
}

func TestNewClient_ZeroPoolOptions(t *testing.T) {
	// Zero pool sizes should be defaulted
	c, err := NewClient("grpc://127.0.0.1:50051",
		WithPool(0, 0),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()
}

func TestNewClient_NegativePoolOptions(t *testing.T) {
	// Negative pool sizes should be defaulted
	c, err := NewClient("grpc://127.0.0.1:50051",
		WithPool(-1, -1),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()
}

// --- GetConnection: additional paths ---

func TestGetConnection_ClosedClient(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	c.Close()
	conn, err := c.GetConnection(context.Background())
	if err == nil {
		t.Fatal("expected error for closed client")
	}
	_ = conn
}

func TestGetConnection_CancelledContext(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	conn, err := c.GetConnection(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	_ = conn
}

func TestGetConnection_NilPool(t *testing.T) {
	c := &Client{
		closed:      0,
		timeout:     1 * time.Second,
		nodeAddress: "test",
		pool:        nil,
	}
	conn, err := c.GetConnection(context.Background())
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
	if !strings.Contains(err.Error(), "connection") {
		t.Fatalf("expected connection error, got: %v", err)
	}
	_ = conn
}

// --- ReturnConnection: additional paths ---

func TestReturnConnection_ClosedClient(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	c.Close()
	// Should not panic when returning to closed client
	c.ReturnConnection(nil)
}

func TestReturnConnection_NilConn(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	// Should not panic
	c.ReturnConnection(nil)
}

// --- Close: double close ---

func TestClose_DoubleClose(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	c.Close()
	c.Close() // should not panic
}

// --- TRC20: nil address path ---

func TestClient_TRC20_NilAddr(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	t.Cleanup(cleanupClient)

	// nil address should cause TRC20 to return nil
	mgr := c.TRC20(nil)
	if mgr != nil {
		t.Fatal("expected nil TRC20 manager for nil address")
	}
}
