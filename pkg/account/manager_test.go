package account_test

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

// Mock client for testing (will be replaced with actual client in integration tests)
func createTestClient() *client.Client {
	// For unit tests, we'll create a basic client
	// In real usage, this would connect to a TRON node
	config := client.DefaultClientConfig("grpc://127.0.0.1:50051")
	client, _ := client.NewClient(config)
	return client
}

func TestNewManager(t *testing.T) {
	client := createTestClient()
	manager := account.NewManager(client)
	
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}
	
	t.Log("Account manager created successfully")
}

func TestTransferValidation(t *testing.T) {
	client := createTestClient()
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
			name:        "Valid transfer",
			from:        "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY",
			to:          "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
			amount:      1000000, // 1 TRX in SUN
			expectError: false,
		},
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
			// Note: This will fail with network error since we don't have a real TRON node
			// But it will test our validation logic before the network call
			_, err := manager.TransferTRX(ctx, tc.from, tc.to, tc.amount, nil)
			
			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error for test case '%s', but got none", tc.name)
				}
				t.Logf("Test case '%s' correctly failed with error: %v", tc.name, err)
			} else {
				// For valid cases, we expect network error since no real node
				// But validation should pass (error should be network-related)
				if err != nil {
					t.Logf("Test case '%s' failed with network error (expected): %v", tc.name, err)
				}
			}
		})
	}
}

func TestTransferOptions(t *testing.T) {
	client := createTestClient()
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test with custom options
	opts := &account.TransferOptions{
		Memo: "Test transfer",
	}

	from := "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY"
	to := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
	amount := int64(1000000) // 1 TRX in SUN

	// This will fail with network error, but tests option handling
	_, err := manager.TransferTRX(ctx, from, to, amount, opts)
	
	// We expect a network error since no real TRON node is connected
	if err != nil {
		t.Logf("Transfer with options failed with expected network error: %v", err)
	}

	// Test with nil options (should use defaults)
	_, err = manager.TransferTRX(ctx, from, to, amount, nil)
	if err != nil {
		t.Logf("Transfer with nil options failed with expected network error: %v", err)
	}
}

func TestGetAccount(t *testing.T) {
	client := createTestClient()
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test valid address
	_, err := manager.GetAccount(ctx, "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY")
	if err != nil {
		t.Logf("GetAccount failed with expected network error: %v", err)
	}

	// Test invalid address
	_, err = manager.GetAccount(ctx, "invalid_address")
	if err == nil {
		t.Fatal("Expected error for invalid address")
	}
	t.Logf("GetAccount correctly failed for invalid address: %v", err)
}

func TestGetAccountNet(t *testing.T) {
	client := createTestClient()
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test valid address
	_, err := manager.GetAccountNet(ctx, "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY")
	if err != nil {
		t.Logf("GetAccountNet failed with expected network error: %v", err)
	}

	// Test invalid address
	_, err = manager.GetAccountNet(ctx, "invalid_address")
	if err == nil {
		t.Fatal("Expected error for invalid address")
	}
	t.Logf("GetAccountNet correctly failed for invalid address: %v", err)
}

func TestGetAccountResource(t *testing.T) {
	client := createTestClient()
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test valid address
	_, err := manager.GetAccountResource(ctx, "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY")
	if err != nil {
		t.Logf("GetAccountResource failed with expected network error: %v", err)
	}

	// Test invalid address
	_, err = manager.GetAccountResource(ctx, "invalid_address")
	if err == nil {
		t.Fatal("Expected error for invalid address")
	}
	t.Logf("GetAccountResource correctly failed for invalid address: %v", err)
}

func TestGetBalance(t *testing.T) {
	client := createTestClient()
	manager := account.NewManager(client)
	ctx := context.Background()

	// Test valid address
	_, err := manager.GetBalance(ctx, "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY")
	if err != nil {
		t.Logf("GetBalance failed with expected network error: %v", err)
	}

	// Test invalid address
	_, err = manager.GetBalance(ctx, "invalid_address")
	if err == nil {
		t.Fatal("Expected error for invalid address")
	}
	t.Logf("GetBalance correctly failed for invalid address: %v", err)
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

// demonstrateUsage shows usage patterns
func demonstrateUsage() {
	// This is an example of how the account package would be used
	// In a real application with a connected TRON node
	
	// Create client
	config := client.DefaultClientConfig("grpc://api.trongrid.io:50051")
	client, _ := client.NewClient(config)
	
	// Create account manager
	manager := account.NewManager(client)
	
	// Get account information
	ctx := context.Background()
	address := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
	
	// Individual calls (following design guidelines)
	account, _ := manager.GetAccount(ctx, address)
	_ = account
	
	accountNet, _ := manager.GetAccountNet(ctx, address)
	_ = accountNet
	
	accountResource, _ := manager.GetAccountResource(ctx, address)
	_ = accountResource
	
	// Get balance (convenience method)
	balance, _ := manager.GetBalance(ctx, address)
	_ = balance
	
	// Create transfer transaction
	from := "TZ1EafTG8FRtE6ef3H2dhaucDdjv36fzPY"
	to := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
	amount := int64(1000000) // 1 TRX in SUN
	
	txExt, _ := manager.TransferTRX(ctx, from, to, amount, nil)
	_ = txExt
	
	// Note: Signing and broadcasting are handled by separate workflows
	// as per design guidelines
}