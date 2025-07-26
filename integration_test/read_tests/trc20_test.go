package read_tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"

	// "github.com/kslamph/tronlib/pkg/utils" // Not needed in trc20_test.go
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTRC20TestClient creates a test TRC20 client instance
func setupTRC20TestClient(t *testing.T, contractAddress string) *trc20.TRC20Client {
	config := getTestConfig()

	clientConfig := client.DefaultClientConfig(config.Endpoint)
	clientConfig.Timeout = config.Timeout

	tronClient, err := client.NewClient(clientConfig)
	require.NoError(t, err, "Failed to create client")

	// Convert string address to *types.Address
	addr, err := types.NewAddress(contractAddress)
	require.NoError(t, err, "Failed to parse contract address")

	trc20Client, err := trc20.NewTRC20Client(tronClient, addr)
	require.NoError(t, err, "Failed to create TRC20 client")

	return trc20Client
}

// TestTRC20MainnetUSDTContract tests USDT contract functionality using the TRC20Client
func TestTRC20MainnetUSDTContract(t *testing.T) {
	// USDT contract address
	usdtAddress := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	// Test address for balance check
	testAddress := "TUHwTn3JhQqdys4ckqQ86EsWk3KC2p2tZc"

	trc20Client := setupTRC20TestClient(t, usdtAddress)
	_ = context.Background() // Keep context for future use, remove unused warning

	t.Run("USDT_BasicInfo", func(t *testing.T) {
		name, err := trc20Client.Name()
		require.NoError(t, err, "Should get USDT name")
		assert.Equal(t, "Tether USD", name, "USDT name should be 'Tether USD'")
		t.Logf("USDT Name: %s", name)

		symbol, err := trc20Client.Symbol()
		require.NoError(t, err, "Should get USDT symbol")
		assert.Equal(t, "USDT", symbol, "USDT symbol should be 'USDT'")
		t.Logf("USDT Symbol: %s", symbol)

		decimals, err := trc20Client.Decimals()
		require.NoError(t, err, "Should get USDT decimals")
		assert.Equal(t, uint8(6), decimals, "USDT decimals should be 6")
		t.Logf("USDT Decimals: %d", decimals)

		t.Logf("✅ USDT basic info test completed successfully")
	})

	t.Run("USDT_BalanceOf", func(t *testing.T) {
		ownerAddr, err := types.NewAddress(testAddress)
		require.NoError(t, err, "Should create owner address")

		balance, err := trc20Client.BalanceOf(ownerAddr)
		require.NoError(t, err, "Should get USDT balance")
		expectedBalance, _ := decimal.NewFromString("45967.732353")
		assert.True(t, balance.Equal(expectedBalance), "USDT balance should be %s, got %s", expectedBalance.String(), balance.String())
		t.Logf("USDT Balance of %s: %s", testAddress, balance.String())

		t.Logf("✅ USDT balance test completed successfully")
	})
}

// TestTRC20MainnetTRC20Contract tests standard TRC20 contract functionality using the TRC20Client
func TestTRC20MainnetTRC20Contract(t *testing.T) {
	// TRC20 contract address (USDD)
	trc20Address := "TPYmHEhy5n8TCEfYGqW2rPxsghSfzghPDn"
	// Test address for balance check
	testAddress := "TRQ4u4Qog3dFWupQ5DnF8GP9pnjyrc8q2X"
	// Spender address for allowance check
	spenderAddress := "TQrq2p1aoAkNK94q3Q69ubJcv5nQ9y675R"

	trc20Client := setupTRC20TestClient(t, trc20Address)
	_ = context.Background() // Keep context for future use, remove unused warning

	t.Run("TRC20_BasicInfo", func(t *testing.T) {
		name, err := trc20Client.Name()
		require.NoError(t, err, "Should get TRC20 name")
		t.Logf("TRC20 Name: %s", name)

		symbol, err := trc20Client.Symbol()
		require.NoError(t, err, "Should get TRC20 symbol")
		t.Logf("TRC20 Symbol: %s", symbol)

		decimals, err := trc20Client.Decimals()
		require.NoError(t, err, "Should get TRC20 decimals")
		t.Logf("TRC20 Decimals: %d", decimals)

		t.Logf("✅ TRC20 basic info test completed successfully")
	})

	t.Run("TRC20_BalanceOf", func(t *testing.T) {
		ownerAddr, err := types.NewAddress(testAddress)
		require.NoError(t, err, "Should create owner address")

		balance, err := trc20Client.BalanceOf(ownerAddr)
		require.NoError(t, err, "Should get TRC20 balance")
		t.Logf("TRC20 Balance of %s: %s", testAddress, balance.String())

		t.Logf("✅ TRC20 balance test completed successfully")
	})

	t.Run("TRC20_Allowance", func(t *testing.T) {
		ownerAddr, err := types.NewAddress(testAddress)
		require.NoError(t, err, "Should create owner address")
		spenderAddr, err := types.NewAddress(spenderAddress)
		require.NoError(t, err, "Should create spender address")

		allowance, err := trc20Client.Allowance(ownerAddr, spenderAddr)
		require.NoError(t, err, "Should get TRC20 allowance")
		t.Logf("TRC20 Allowance from %s to %s: %s", testAddress, spenderAddress, allowance.String())

		// For USDD, the test address typically has unlimited allowance (maxUint256)
		// We'll check if it's a very large number (effectively unlimited for practical purposes)
		maxUint256 := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
		expectedAllowance, _ := decimal.NewFromString(maxUint256.String())
		decimals, err := trc20Client.Decimals()
		require.NoError(t, err, "Should get TRC20 decimals for allowance check")
		// Adjust expected allowance based on decimals
		expectedAllowance = expectedAllowance.Shift(-int32(decimals))
		assert.True(t, allowance.GreaterThanOrEqual(expectedAllowance.Sub(decimal.NewFromInt(1))), "Allowance should be effectively unlimited")

		t.Logf("✅ TRC20 allowance test completed successfully")
	})
}
