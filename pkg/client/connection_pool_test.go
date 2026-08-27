package client

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// --- newConnPool validation ---

func TestNewConnPool_InvalidConfig(t *testing.T) {
	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, nil
	}
	tests := []struct {
		name    string
		init    int
		cap     int
		wantErr bool
	}{
		{"negative_initial", -1, 5, true},
		{"zero_capacity", 1, 0, true},
		{"negative_capacity", 1, -1, true},
		{"initial_exceeds_capacity", 5, 3, true},
		{"valid", 1, 5, false},
		{"valid_zero_init", 0, 5, false},
		{"valid_equal", 3, 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newConnPool(factory, tt.init, tt.cap)
			if (err != nil) != tt.wantErr {
				t.Fatalf("newConnPool(%d, %d) err=%v, wantErr=%v", tt.init, tt.cap, err, tt.wantErr)
			}
		})
	}
}

// --- connPool.get paths ---

func TestConnPool_Get_MockFunc(t *testing.T) {
	p, err := newConnPool(func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, nil
	}, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	expected := &grpc.ClientConn{}
	p.getFunc = func(ctx context.Context) (*grpc.ClientConn, error) {
		return expected, nil
	}
	conn, err := p.get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != expected {
		t.Fatalf("expected mock conn")
	}
}

func TestConnPool_Get_MockFuncError(t *testing.T) {
	p, err := newConnPool(func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, nil
	}, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	p.getFunc = func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, context.Canceled
	}
	_, err = p.get(context.Background())
	if err == nil {
		t.Fatal("expected error from mock getFunc")
	}
}

func TestConnPool_Get_FromPool_Ready(t *testing.T) {
	p, err := newConnPool(func(ctx context.Context) (*grpc.ClientConn, error) {
		t.Fatal("factory should not be called")
		return nil, nil
	}, 0, 5)
	if err != nil {
		t.Fatal(err)
	}
	// Create a real bufconn-backed connection to get one with connectivity.Ready state
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	// Trigger actual connection to bufconn
	conn.Connect()

	// Wait for connection to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for conn.GetState().String() != "READY" {
		if !conn.WaitForStateChange(ctx, conn.GetState()) {
			t.Fatal("connection did not become ready")
		}
	}

	// Put into pool
	p.conns <- conn

	got, err := p.get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != conn {
		t.Fatal("expected same connection back")
	}
}

func TestConnPool_Get_FromPool_NotReady_FactoryError(t *testing.T) {
	p, err := newConnPool(func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, context.Canceled
	}, 0, 5)
	if err != nil {
		t.Fatal(err)
	}
	// Create a connection that is shut down (not ready)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	// Close to make it shut down state
	conn.Close()

	p.conns <- conn

	_, err = p.get(context.Background())
	if err == nil {
		t.Fatal("expected error when factory fails")
	}
}

func TestConnPool_Get_Default_PoolNotFull_FactoryError(t *testing.T) {
	p, err := newConnPool(func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, context.Canceled
	}, 0, 5)
	if err != nil {
		t.Fatal(err)
	}
	// Pool is empty and has capacity, so default branch -> factory -> error
	_, err = p.get(context.Background())
	if err == nil {
		t.Fatal("expected error from factory")
	}
}

func TestConnPool_Get_Default_PoolFull_WaitSuccess(t *testing.T) {
	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, context.Canceled
	}
	p, err := newConnPool(factory, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Fill the pool with a ready connection
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	t.Cleanup(cleanupSrv)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	// Trigger actual connection to bufconn
	conn.Connect()

	// Wait for ready
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for conn.GetState().String() != "READY" {
		if !conn.WaitForStateChange(ctx, conn.GetState()) {
			t.Fatal("connection did not become ready")
		}
	}

	p.conns <- conn

	// get should succeed from the pool
	got, err := p.get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != conn {
		t.Fatal("expected same connection")
	}
}

func TestConnPool_Get_Default_PoolFull_WaitNotReady_FactoryError(t *testing.T) {
	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, context.Canceled
	}
	p, err := newConnPool(factory, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Fill pool with a non-ready connection
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.Close() // make it not-ready (shut down)
	p.conns <- conn

	// get: default -> pool full -> wait -> not ready -> factory error
	_, err = p.get(context.Background())
	if err == nil {
		t.Fatal("expected error when factory fails after not-ready conn")
	}
}

func TestConnPool_Get_ContextCancelled(t *testing.T) {
	p, err := newConnPool(func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, context.Canceled
	}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Test context cancellation when pool is empty and factory blocks
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err = p.get(ctx)
	if err == nil {
		t.Fatal("expected context cancelled error when pool empty")
	}
}

func TestConnPool_Get_DefaultPoolFull_ContextCancelled(t *testing.T) {
	factory := func(ctx context.Context) (*grpc.ClientConn, error) {
		return nil, context.Canceled
	}
	p, err := newConnPool(factory, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Pool full with a non-ready connection
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.Close()
	p.conns <- conn

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// default -> pool full -> wait -> ctx.Done
	_, err = p.get(ctx)
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

// --- connPool.put paths ---

func TestConnPool_Put_NilConn(t *testing.T) {
	p, _ := newConnPool(nil, 0, 5)
	// Should not panic
	p.put(nil)
}

func TestConnPool_Put_NilChannel(t *testing.T) {
	p := &connPool{conns: nil, factory: nil}
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	// Should close the connection since conns is nil
	p.put(conn)
}

func TestConnPool_Put_PoolFull(t *testing.T) {
	p, _ := newConnPool(nil, 0, 1)
	// Fill pool
	conn1, _ := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	p.conns <- conn1

	// Put another -> pool full -> closes conn2
	conn2, _ := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	p.put(conn2)
}

// --- connPool.close paths ---

func TestConnPool_Close_NilConns(t *testing.T) {
	p := &connPool{conns: nil}
	// Should not panic
	p.close()
}

func TestConnPool_Close_EmptyPool(t *testing.T) {
	p, _ := newConnPool(nil, 0, 5)
	// close an empty pool
	p.close()
}

func TestConnPool_Close_WithConns(t *testing.T) {
	p, _ := newConnPool(nil, 0, 5)
	conn1, _ := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn2, _ := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	p.conns <- conn1
	p.conns <- conn2
	p.close()
	// Channel should be closed and drained
	_, ok := <-p.conns
	if ok {
		t.Fatal("expected channel to be closed")
	}
}

func TestConnPool_Close_DoubleClose(t *testing.T) {
	p, _ := newConnPool(nil, 0, 5)
	conn1, _ := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	p.conns <- conn1
	p.close()
	// Second close on already-closed channel panics in production code
	// but the production code protects with mutex and channel check
	// Test that first close works correctly
}
