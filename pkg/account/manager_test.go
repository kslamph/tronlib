package account_test

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	go func() {
		_ = s.Serve(lis)
	}()
}

func createTestClient(t *testing.T) *client.Client {
	t.Helper()
	c, err := client.NewClient("bufnet")
	if err != nil {
		// bufnet requires a real connection, so we just create a client that will fail gracefully
		c, _ = client.NewClient("grpc://127.0.0.1:1")
	}
	return c
}

func TestNewManager(t *testing.T) {
	client, _ := client.NewClient("grpc://127.0.0.1:1")
	manager := account.NewManager(client)

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	t.Log("Account manager created successfully")
}

func TestTransferValidation(t *testing.T) {
	client, _ := client.NewClient("grpc://127.0.0.1:1")
	manager := account.NewManager(client)
	ctx := context.Background()

	testCases := []struct {
		name        string
		from        string
		to          string
		amount      int64
		expectError bool
	}{
		{
			name:        "Empty from address",
			from:        "",
			to:          "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
			amount:      1000000,
			expectError: true,
		},
		{
			name:        "Empty to address",
			from:        "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY",
			to:          "",
			amount:      1000000,
			expectError: true,
		},
		{
			name:        "Zero amount",
			from:        "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY",
			to:          "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
			amount:      0,
			expectError: true,
		},
		{
			name:        "Negative amount",
			from:        "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY",
			to:          "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
			amount:      -1000000,
			expectError: true,
		},
		{
			name:        "Same from and to address",
			from:        "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY",
			to:          "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY",
			amount:      1000000,
			expectError: true,
		},
		{
			name:        "Invalid from address",
			from:        "invalid_address",
			to:          "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
			amount:      1000000,
			expectError: true,
		},
		{
			name:        "Invalid to address",
			from:        "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY",
			to:          "invalid_address",
			amount:      1000000,
			expectError: true,
		},
		{
			name:        "Very large amount",
			from:        "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY",
			to:          "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
			amount:      int64(200_000_000_000 * types.SunPerTRX), // 200B TRX (exceeds max supply)
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fromAddr, _ := types.NewAddressFromBase58(tc.from)
			toAddr, _ := types.NewAddressFromBase58(tc.to)
			_, err := manager.TransferTRX(ctx, fromAddr, toAddr, tc.amount)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error for test case '%s', but got none", tc.name)
				}
				t.Logf("Test case '%s' correctly failed with error: %v", tc.name, err)
			}
		})
	}
}

func TestGetAccount(t *testing.T) {
	client, _ := client.NewClient("grpc://127.0.0.1:1")
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test invalid address (empty Address struct)
	_, err := manager.GetAccount(ctx, &types.Address{})
	if err == nil {
		t.Fatal("Expected error for invalid address")
	}
	t.Logf("GetAccount correctly failed for invalid address: %v", err)

	// Test nil address
	_, err = manager.GetAccount(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil address")
	}
	t.Logf("GetAccount correctly failed for nil address: %v", err)
}

func TestGetAccountNet(t *testing.T) {
	client, _ := client.NewClient("grpc://127.0.0.1:1")
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test nil address
	_, err := manager.GetAccountNet(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil address")
	}
	t.Logf("GetAccountNet correctly failed for nil address: %v", err)
}

func TestGetAccountResource(t *testing.T) {
	client, _ := client.NewClient("grpc://127.0.0.1:1")
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test nil address
	_, err := manager.GetAccountResource(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil address")
	}
	t.Logf("GetAccountResource correctly failed for nil address: %v", err)
}

func TestGetBalance(t *testing.T) {
	client, _ := client.NewClient("grpc://127.0.0.1:1")
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test nil address
	_, err := manager.GetBalance(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil address")
	}
	t.Logf("GetBalance correctly failed for nil address: %v", err)
}

func TestAmountConversions(t *testing.T) {
	testCases := []struct {
		name     string
		trx      float64
		expected int64
	}{
		{"1 TRX", 1.0, 1000000},
		{"0.1 TRX", 0.1, 100000},
		{"0.000001 TRX", 0.000001, 1},
		{"100 TRX", 100.0, 100000000},
		{"1000 TRX", 1000.0, 1000000000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert TRX to SUN (int64)
			sun := int64(tc.trx * float64(types.SunPerTRX))

			if sun != tc.expected {
				t.Errorf("Expected %d SUN, got %d SUN", tc.expected, sun)
			}

			t.Logf("%s = %d SUN", tc.name, sun)
		})
	}
}
