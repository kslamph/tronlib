package rpc

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- Dial: endpoint validation ---

func TestDialInvalidEndpoint(t *testing.T) {
	bad := []string{
		"",                   // empty
		"127.0.0.1:50051",    // missing scheme
		"http://127.0.0.1:1", // unsupported scheme
		"grpc://",            // missing host:port
		"://invalid",         // unparseable, no scheme
	}
	for _, ep := range bad {
		_, err := Dial(context.Background(), ep)
		if err == nil {
			t.Fatalf("Dial(%q): expected error", ep)
		}
		if !tron.HasCode(err, tron.CodeChainConnection) {
			t.Fatalf("Dial(%q): err = %v, want code chain.connection", ep, err)
		}
		var te *tron.Error
		if !errors.As(err, &te) || te.Hint == "" || !strings.Contains(te.Hint, "grpc://host:port") {
			t.Fatalf("Dial(%q): err = %v, want *tron.Error with remediation Hint mentioning grpc://host:port", ep, err)
		}
		if te.Op != "rpc.Dial" {
			t.Fatalf("Dial(%q): Op = %q, want rpc.Dial", ep, te.Op)
		}
	}
}

func TestDialAcceptsGrpcSchemes(t *testing.T) {
	// grpc.NewClient is lazy: Dial performs no network I/O, so dialing a
	// non-listening port is fine (v1 test approach — grpcs:// is only scheme-
	// validated; a TLS handshake failure surfaces on first call).
	c, err := Dial(context.Background(), "grpcs://localhost:19999")
	if err != nil {
		t.Fatalf("Dial grpcs://localhost:19999: unexpected error: %v", err)
	}
	defer c.Close()
	if c.GetNodeAddress() != "grpcs://localhost:19999" {
		t.Fatalf("unexpected node address: %q", c.GetNodeAddress())
	}

	c2, err := Dial(context.Background(), "grpc://127.0.0.1:50051")
	if err != nil {
		t.Fatalf("Dial grpc://127.0.0.1:50051: unexpected error: %v", err)
	}
	defer c2.Close()
	if c2.GetTimeout() != 30*time.Second {
		t.Fatalf("expected default timeout 30s, got %v", c2.GetTimeout())
	}
}

func TestDialSchemeCaseInsensitive(t *testing.T) {
	c, err := Dial(context.Background(), "GRPC://127.0.0.1:50051")
	if err != nil {
		t.Fatalf("uppercase scheme rejected: %v", err)
	}
	defer c.Close()
}

func TestDialOptions(t *testing.T) {
	c, err := Dial(context.Background(), "grpc://127.0.0.1:50051",
		WithTimeout(5*time.Second), WithPool(3, 10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()
	if c.GetTimeout() != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", c.GetTimeout())
	}
	if c.pool.capacity() != 10 {
		t.Fatalf("expected pool capacity 10, got %d", c.pool.capacity())
	}
}

func TestDialZeroAndNegativePoolDefaults(t *testing.T) {
	for _, tt := range []struct{ init, max int }{{0, 0}, {-1, -1}, {0, -3}} {
		c, err := Dial(context.Background(), "grpc://127.0.0.1:50051", WithPool(tt.init, tt.max))
		if err != nil {
			t.Fatalf("WithPool(%d,%d): unexpected error: %v", tt.init, tt.max, err)
		}
		c.Close()
		if c.pool.capacity() != 5 {
			t.Fatalf("WithPool(%d,%d): expected defaulted capacity 5, got %d", tt.init, tt.max, c.pool.capacity())
		}
	}
}

func TestDialNilOptionIgnored(t *testing.T) {
	c, err := Dial(context.Background(), "grpc://127.0.0.1:50051", nil)
	if err != nil {
		t.Fatalf("unexpected error for nil option: %v", err)
	}
	defer c.Close()
}

func TestDialInitialExceedsCapacityFails(t *testing.T) {
	_, err := Dial(context.Background(), "grpc://127.0.0.1:50051", WithPool(5, 3))
	if err == nil {
		t.Fatal("expected error when initial connections exceed capacity")
	}
	if !tron.HasCode(err, tron.CodeChainConnection) {
		t.Fatalf("err = %v, want chain.connection", err)
	}
}

// --- Call lifecycle through the bufconn fake ---

func TestCallHappyPath(t *testing.T) {
	want := &core.Block{BlockHeader: &core.BlockHeader{RawData: &core.BlockHeaderRaw{Number: 42}}}
	srv := &testWalletServer{
		GetNowBlockHandler: func(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
			return want, nil
		},
	}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	validateRan := false
	got, err := Call(c, context.Background(), "rpc.GetNowBlock",
		func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
			return cl.GetNowBlock(ctx, &api.EmptyMessage{})
		},
		func(result *core.Block, op string) error {
			validateRan = true
			if op != "rpc.GetNowBlock" {
				t.Errorf("validator op = %q, want rpc.GetNowBlock", op)
			}
			return nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GetBlockHeader().GetRawData().GetNumber() != 42 {
		t.Fatalf("got %+v, want the fake's block (number 42)", got)
	}
	if !validateRan {
		t.Fatal("validator never ran")
	}
}

func TestCallAppliesClientTimeoutWhenNoDeadline(t *testing.T) {
	// Fake sleeps longer than the client timeout; the deadline applied by
	// Call must abort the RPC.
	srv := &testWalletServer{
		GetNowBlockHandler: func(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return &core.Block{}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	c := newBufconnClient(t, srv, 30*time.Millisecond)
	defer c.Close()

	start := time.Now()
	_, err := Call(c, context.Background(), "rpc.GetNowBlock",
		func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
			return cl.GetNowBlock(ctx, &api.EmptyMessage{})
		})
	// Documented port behavior: a deadline hit mid-call surfaces as
	// chain.timeout whose Cause is the gRPC status error (code
	// DeadlineExceeded) — grpc status errors do not unwrap to
	// context.DeadlineExceeded, so HasCode (not errors.Is) is the assertable
	// verb here. (A context that is ALREADY expired before Call reaches
	// GetConnection DOES errors.Is-hold; see TestCallAlreadyExpiredContext.)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !tron.HasCode(err, tron.CodeChainTimeout) {
		t.Fatalf("err = %v, want chain.timeout", err)
	}
	var te *tron.Error
	if !errors.As(err, &te) || status.Code(te.Cause) != codes.DeadlineExceeded {
		t.Fatalf("err = %v, want Cause with grpc code DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("call took %v; client timeout was not applied", elapsed)
	}
}

func TestCallHonorsExistingDeadline(t *testing.T) {
	srv := &testWalletServer{
		GetNowBlockHandler: func(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
			select {
			case <-time.After(time.Second):
				return &core.Block{}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	c := newBufconnClient(t, srv, 10*time.Second)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := Call(c, ctx, "rpc.GetNowBlock",
		func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
			return cl.GetNowBlock(ctx, &api.EmptyMessage{})
		})
	// Documented port behavior (same as TestCallAppliesClientTimeoutWhenNoDeadline):
	// HasCode(chain.timeout); the Cause is the gRPC DeadlineExceeded status.
	if err == nil {
		t.Fatal("expected error")
	}
	if !tron.HasCode(err, tron.CodeChainTimeout) {
		t.Fatalf("err = %v, want chain.timeout", err)
	}
	var te *tron.Error
	if !errors.As(err, &te) || status.Code(te.Cause) != codes.DeadlineExceeded {
		t.Fatalf("err = %v, want Cause with grpc code DeadlineExceeded", err)
	}
}

func TestCallAlreadyExpiredContext(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond) // ensure expiry

	_, err := Call(c, ctx, "rpc.GetNowBlock",
		func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
			return cl.GetNowBlock(ctx, &api.EmptyMessage{})
		})
	// Pre-expired context: GetConnection sees ctx.Done() before any wire
	// traffic, so Cause IS the raw context error and BOTH assertions hold
	// (documented port behavior).
	if !tron.HasCode(err, tron.CodeChainTimeout) {
		t.Fatalf("err = %v, want chain.timeout", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

func TestCallClosedClient(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)
	c.Close()

	_, err := Call(c, context.Background(), "rpc.GetNowBlock",
		func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
			return cl.GetNowBlock(ctx, &api.EmptyMessage{})
		})
	if !tron.HasCode(err, tron.CodeChainClosed) {
		t.Fatalf("err = %v, want chain.closed", err)
	}
}

func TestCallServerFailureMapsToRPCMethodFailed(t *testing.T) {
	srv := &testWalletServer{
		GetNowBlockHandler: func(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
			return nil, errors.New("node exploded")
		},
	}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	_, err := Call[*core.Block](c, context.Background(), "GetNowBlock2",
		func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
			return cl.GetNowBlock(ctx, &api.EmptyMessage{})
		})
	var te *tron.Error
	if !errors.As(err, &te) {
		t.Fatalf("err = %v, want *tron.Error", err)
	}
	if te.Code != tron.CodeRPCMethodFailed {
		t.Fatalf("code = %v, want rpc.method_failed", te.Code)
	}
	if te.Op != "GetNowBlock2" {
		t.Fatalf("Op = %q, want the operation name", te.Op)
	}
}

func TestCallValidationFailureWrapped(t *testing.T) {
	srv := &testWalletServer{
		GetNowBlockHandler: func(ctx context.Context, in *api.EmptyMessage) (*core.Block, error) {
			return &core.Block{}, nil
		},
	}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	// Plain error from the validator → wrapped as *tron.Error preserving Op.
	_, err := Call[*core.Block](c, context.Background(), "GetNowBlock2",
		func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
			return cl.GetNowBlock(ctx, &api.EmptyMessage{})
		},
		func(result *core.Block, op string) error { return errors.New("bad block") })
	if !tron.HasCode(err, tron.CodeRPCMethodFailed) {
		t.Fatalf("err = %v, want rpc.method_failed wrap", err)
	}

	// A *tron.Error from the validator passes through unchanged.
	want := &tron.Error{Code: tron.CodeChainUnconfirmed, Op: "validator"}
	_, err = Call[*core.Block](c, context.Background(), "GetNowBlock2",
		func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
			return cl.GetNowBlock(ctx, &api.EmptyMessage{})
		},
		func(result *core.Block, op string) error { return want })
	if err != want {
		t.Fatalf("tron.Error from validator must pass through unchanged, got %v", err)
	}
}

func TestCallConnProviderError(t *testing.T) {
	// A ConnProvider whose GetConnection fails with a plain error → Call wraps
	// as chain.connection preserving the operation in Op.
	cp := &failingProvider{err: errors.New("no conns")}
	_, err := Call(cp, context.Background(), "some-op",
		func(cl api.WalletClient, ctx context.Context) (int, error) { return 0, nil })
	if !tron.HasCode(err, tron.CodeChainConnection) {
		t.Fatalf("err = %v, want chain.connection", err)
	}
	var te *tron.Error
	if !errors.As(err, &te) || te.Op != "some-op" {
		t.Fatalf("err = %v, want Op=some-op", err)
	}
}

type failingProvider struct{ err error }

func (f *failingProvider) GetConnection(context.Context) (*grpc.ClientConn, error) {
	return nil, f.err
}
func (f *failingProvider) ReturnConnection(*grpc.ClientConn) {}
func (f *failingProvider) GetTimeout() time.Duration         { return time.Second }

// --- Pool behavior ---

func TestPoolGetReturnCycle(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	conn1, err := c.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("first GetConnection: %v", err)
	}
	conn2, err := c.GetConnection(context.Background()) // pool (1,2): creates a second
	if err != nil {
		t.Fatalf("second GetConnection: %v", err)
	}
	c.ReturnConnection(conn1)
	c.ReturnConnection(conn2)

	conn3, err := c.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("GetConnection after return: %v", err)
	}
	// Note: v1 (and this port) replaces a pooled connection that is not
	// connectivity.Ready with a fresh one, so conn3 is not necessarily conn1
	// or conn2 — lazily-created conns are Idle. What matters is that the
	// cycle works and the pool stays within capacity.
	c.ReturnConnection(conn3)
}

func TestPoolRespectsCapacity(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	var conns []*grpc.ClientConn
	for i := 0; i < 2; i++ { // pool is (1,2) — capacity 2
		conn, err := c.GetConnection(context.Background())
		if err != nil {
			t.Fatalf("GetConnection %d: %v", i, err)
		}
		conns = append(conns, conn)
	}
	// Returning beyond capacity must not block or panic.
	for _, conn := range conns {
		c.ReturnConnection(conn)
	}
	extra := conns[0] // over-capacity return: pool closes it
	c.ReturnConnection(extra)
}

func TestPoolGetWithMockFunc(t *testing.T) {
	// Ported from v1 connection_pool_test.go: getFunc override path.
	factory := func(ctx context.Context) (*grpc.ClientConn, error) { return nil, nil }
	p, err := newConnPool(factory, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	expected := &grpc.ClientConn{}
	p.getFunc = func(ctx context.Context) (*grpc.ClientConn, error) { return expected, nil }
	conn, err := p.get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != expected {
		t.Fatal("expected mock conn")
	}
}

func TestPoolGetWithMockFuncError(t *testing.T) {
	factory := func(ctx context.Context) (*grpc.ClientConn, error) { return nil, nil }
	p, err := newConnPool(factory, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	p.getFunc = func(ctx context.Context) (*grpc.ClientConn, error) { return nil, context.Canceled }
	_, err = p.get(context.Background())
	if err == nil {
		t.Fatal("expected error from mock getFunc")
	}
}

func TestNewConnPoolInvalidConfig(t *testing.T) {
	factory := func(ctx context.Context) (*grpc.ClientConn, error) { return nil, nil }
	for _, tt := range []struct {
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newConnPool(factory, tt.init, tt.cap)
			if (err != nil) != tt.wantErr {
				t.Fatalf("newConnPool(%d, %d) err=%v, wantErr=%v", tt.init, tt.cap, err, tt.wantErr)
			}
		})
	}
}

func TestPoolGetCancelledContext(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetConnection(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !tron.HasCode(err, tron.CodeChainTimeout) {
		t.Fatalf("err = %v, want chain.timeout", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want errors.Is(err, context.Canceled)", err)
	}
}

// --- Close ---

func TestCloseIdempotent(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)

	if err := c.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := c.Close(); err != nil { // second Close must be a no-op, not a panic
		t.Fatalf("second Close: %v", err)
	}
	if c.IsConnected() {
		t.Fatal("IsConnected must be false after Close")
	}
	if _, err := c.GetConnection(context.Background()); !tron.HasCode(err, tron.CodeChainClosed) {
		t.Fatalf("GetConnection after Close: err = %v, want chain.closed", err)
	}
	c.ReturnConnection(nil) // no-op on closed client; must not panic
	c.ReturnConnection(&grpc.ClientConn{})
}

func TestGetConnectionNilPool(t *testing.T) {
	c := &Client{timeout: time.Second}
	_, err := c.GetConnection(context.Background())
	if !tron.HasCode(err, tron.CodeChainConnection) {
		t.Fatalf("err = %v, want chain.connection", err)
	}
}

func TestReturnConnectionNilConnNoop(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()
	c.ReturnConnection(nil) // must not panic
}

// --- Concurrency smoke: the pool must be safe under parallel Call traffic ---

func TestPoolConcurrentCalls(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	var wg sync.WaitGroup
	var failures int32
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Call(c, context.Background(), "rpc.GetNowBlock",
				func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
					return cl.GetNowBlock(ctx, &api.EmptyMessage{})
				})
			if err != nil {
				atomic.AddInt32(&failures, 1)
			}
		}()
	}
	wg.Wait()
	if failures != 0 {
		t.Fatalf("%d concurrent calls failed", failures)
	}
}

// --- FIX ROUND 1: pool close/return race and Solidity fake registration ---

// TestCloseRacesWithReturnConnection: concurrent Close + in-flight
// ReturnConnection used to send on the closed p.conns channel and panic. The
// mutex-guarded put/close makes that send impossible; loop 200x to widen the
// window (run with -race).
func TestCloseRacesWithReturnConnection(t *testing.T) {
	for iter := 0; iter < 200; iter++ {
		srv := &testWalletServer{}
		c := newBufconnClient(t, srv, time.Second)

		const n = 8
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				conn, err := c.GetConnection(context.Background())
				if err != nil {
					return // pool may already be closed; fine
				}
				c.ReturnConnection(conn)
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Close()
		}()
		wg.Wait()
		// t.Cleanup closes the client again; Close is idempotent.
	}
}

// TestDoubleReturnDoesNotDuplicate: returning the same conn twice must not
// create two channel entries; the second caller must get a different conn.
func TestDoubleReturnDoesNotDuplicate(t *testing.T) {
	srv := &testWalletServer{}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	conn1, err := c.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("first GetConnection: %v", err)
	}
	c.ReturnConnection(conn1)
	c.ReturnConnection(conn1) // double return

	conn2, err := c.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("second GetConnection: %v", err)
	}
	conn3, err := c.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("third GetConnection: %v", err)
	}
	if conn2 == conn3 {
		t.Fatal("same conn handed to two callers after double return")
	}
}

// TestBufconnServesSolidity: the fake registers WalletSolidityServer alongside
// WalletServer; a Solidity client reaches the same canned handlers.
func TestBufconnServesSolidity(t *testing.T) {
	want := &core.TransactionInfo{BlockNumber: 4242}
	srv := &testWalletServer{
		GetTxInfoByIdHandler: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			return want, nil
		},
	}
	c := newBufconnClient(t, srv, time.Second)
	defer c.Close()

	conn, err := c.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	defer c.ReturnConnection(conn)

	cl := api.NewWalletSolidityClient(conn)
	got, err := cl.GetTransactionInfoById(context.Background(), &api.BytesMessage{Value: []byte("txid")})
	if err != nil {
		t.Fatalf("solidity GetTransactionInfoById: %v", err)
	}
	if got.GetBlockNumber() != want.GetBlockNumber() {
		t.Fatalf("got block %d, want %d", got.GetBlockNumber(), want.GetBlockNumber())
	}
}
