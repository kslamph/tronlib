package client

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

type testServer struct {
	listener *bufconn.Listener
	server   *grpc.Server
	mock     *mockWalletServer
}

// mockWalletServer implements a mock of the WalletClient interface
type mockWalletServer struct {
	api.UnimplementedWalletServer
	failureCount int           // number of times to fail before succeeding
	currentFails int           // current failure count
	delay        time.Duration // delay before responding
	errorCode    codes.Code    // error code to return when failing
	t            *testing.T    // for test logging
}

func (m *mockWalletServer) GetAccount(ctx context.Context, req *core.Account) (*core.Account, error) {
	select {
	case <-ctx.Done():
		m.t.Logf("GetAccount context cancelled")
		return nil, ctx.Err()
	default:
		if m.currentFails < m.failureCount {
			m.currentFails++
			m.t.Logf("GetAccount failing attempt %d/%d with code %v", m.currentFails, m.failureCount, m.errorCode)
			return nil, status.Error(m.errorCode, fmt.Sprintf("server unavailable (attempt %d)", m.currentFails))
		}
		m.t.Logf("GetAccount succeeding after %d failures", m.currentFails)
		return &core.Account{Address: req.Address}, nil
	}
}

func (m *mockWalletServer) GetAccountResource(ctx context.Context, req *core.Account) (*api.AccountResourceMessage, error) {
	select {
	case <-ctx.Done():
		m.t.Logf("GetAccountResource context cancelled")
		return nil, ctx.Err()
	default:
		if m.currentFails < m.failureCount {
			m.currentFails++
			m.t.Logf("GetAccountResource failing attempt %d/%d with code %v", m.currentFails, m.failureCount, m.errorCode)
			return nil, status.Error(m.errorCode, fmt.Sprintf("server unavailable (attempt %d)", m.currentFails))
		}
		m.t.Logf("GetAccountResource succeeding after %d failures", m.currentFails)
		return &api.AccountResourceMessage{}, nil
	}
}

type testClient struct {
	*Client
	servers map[string]*testServer
	t       *testing.T
}

// connect overrides the real client's connect method for testing
func (tc *testClient) connect(ctx context.Context) error {
	server, ok := tc.servers[tc.endpoints[tc.current]]
	if !ok {
		return fmt.Errorf("unknown endpoint: %s", tc.endpoints[tc.current])
	}

	tc.t.Logf("Connecting to endpoint: %s", tc.endpoints[tc.current])

	conn, err := grpc.DialContext(ctx, tc.endpoints[tc.current],
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return server.listener.Dial()
		}),
		grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to dial test server: %v", err)
	}

	tc.conn = conn
	tc.wallet = api.NewWalletClient(conn)
	tc.t.Logf("Successfully connected to endpoint: %s", tc.endpoints[tc.current])
	return nil
}

func createTestServer(t *testing.T, failureCount int, delay time.Duration, errorCode codes.Code) *testServer {
	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	mock := &mockWalletServer{
		failureCount: failureCount,
		delay:        delay,
		errorCode:    errorCode,
		t:            t,
	}
	api.RegisterWalletServer(server, mock)

	ts := &testServer{
		listener: listener,
		server:   server,
		mock:     mock,
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			if err != grpc.ErrServerStopped {
				t.Logf("Test server stopped with error: %v", err)
			}
		}
	}()

	return ts
}

func setupTestClient(t *testing.T, endpoints []string, servers map[string]*testServer) (*testClient, error) {
	opts := &ClientOptions{
		Endpoints: endpoints,
		Timeout:   time.Second,
		RetryConfig: &RetryConfig{
			MaxAttempts:     2,
			InitialBackoff:  10 * time.Millisecond,
			MaxBackoff:      100 * time.Millisecond,
			BackoffFactor:   1.5,
			RetryableErrors: []ErrorCode{ErrNetwork},
		},
	}

	client := &testClient{
		Client: &Client{
			opts:      opts,
			endpoints: opts.Endpoints,
			current:   0,
		},
		servers: servers,
		t:       t,
	}

	// Make initial connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := client.connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to initial endpoint: %v", err)
	}

	return client, nil
}

func TestFailoverBehavior(t *testing.T) {
	// Create test servers
	failingServer := createTestServer(t, 1, 0, codes.Unavailable)
	defer failingServer.server.GracefulStop()

	workingServer := createTestServer(t, 0, 0, codes.OK)
	defer workingServer.server.GracefulStop()

	// Set up test client
	client, err := setupTestClient(t, []string{"failing", "working"}, map[string]*testServer{
		"failing": failingServer,
		"working": workingServer,
	})
	if err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}
	defer client.Close()

	// Test cases
	testCases := []struct {
		name     string
		testFunc func(context.Context) error
	}{
		{
			name: "GetAccount should failover and succeed",
			testFunc: func(ctx context.Context) error {
				testAddr := &types.Address{}
				_, err := client.GetAccount(testAddr)
				return err
			},
		},
		{
			name: "GetAccountResource should failover and succeed",
			testFunc: func(ctx context.Context) error {
				testAddr := &types.Address{}
				_, err := client.GetAccountResource(testAddr)
				return err
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Reset state
			failingServer.mock.currentFails = 0
			workingServer.mock.currentFails = 0
			client.current = 0 // Start with failing server

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			t.Logf("Starting test case: %s", tc.name)
			err := tc.testFunc(ctx)
			if err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}
			t.Logf("Completed test case: %s", tc.name)
		})
	}
}

func TestRetryBehavior(t *testing.T) {
	// Create test server that fails twice then succeeds
	server := createTestServer(t, 2, 10*time.Millisecond, codes.Unavailable)
	defer server.server.GracefulStop()

	// Set up test client
	client, err := setupTestClient(t, []string{"test"}, map[string]*testServer{
		"test": server,
	})
	if err != nil {
		t.Fatalf("Failed to setup test client: %v", err)
	}
	defer client.Close()

	// Test retry behavior
	testAddr := &types.Address{}
	start := time.Now()

	t.Log("Starting retry behavior test")
	_, err = client.GetAccount(testAddr)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected success after retries but got error: %v", err)
	}

	// Verify retry behavior
	if server.mock.currentFails != 2 {
		t.Errorf("Expected 2 failures before success, got %d", server.mock.currentFails)
	}

	// Verify timing
	minExpectedDuration := 20 * time.Millisecond // 2 failures * 10ms delay
	if duration < minExpectedDuration {
		t.Errorf("Expected operation to take at least %v, but took %v", minExpectedDuration, duration)
	}
	t.Logf("Test completed in %v with %d failures", duration, server.mock.currentFails)
}
