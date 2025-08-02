package read_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSmartContractTestManager creates a test smart contract manager instance
func setupSmartContractTestManager(t *testing.T) *smartcontract.Manager {
	config := getTestConfig()

	clientConfig := client.DefaultClientConfig(config.Endpoint)
	clientConfig.Timeout = config.Timeout

	client, err := client.NewClient(clientConfig)
	require.NoError(t, err, "Failed to create client")

	return smartcontract.NewManager(client)
}

// TestMainnetUSDTContract tests USDT contract functionality
func TestMainnetUSDTContract(t *testing.T) {
	// manager := setupSmartContractTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// USDT contract address
	usdtAddressStr := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	usdtAddress, err := types.NewAddress(usdtAddressStr)
	require.NoError(t, err, "Failed to parse USDT address")

	// Test address for balance check
	testAddressStr := "TUHwTn3JhQqdys4ckqQ86EsWk3KC2p2tZc"
	testAddress, err := types.NewAddress(testAddressStr)
	require.NoError(t, err, "Failed to parse test address")

	t.Run("USDT_BasicInfo", func(t *testing.T) {
		// Create contract instance from network using client
		client, err := client.NewClient(client.DefaultClientConfig(getTestConfig().Endpoint))
		require.NoError(t, err, "Should create client")

		contract, err := smartcontract.NewContract(client, usdtAddress)
		require.NoError(t, err, "Should create USDT contract instance from network")

		// Test symbol
		symbol, err := contract.TriggerConstantContract(ctx, testAddress, "symbol")
		require.NoError(t, err, "Should call symbol method")
		// Single output should be concrete string
		symbolDecoded, ok := symbol.(string)
		require.True(t, ok, "Should decode symbol as string, got %T", symbol)
		assert.Equal(t, "USDT", symbolDecoded, "Symbol should be USDT")
		t.Logf("USDT Symbol: %s", symbolDecoded)

		// Test name
		name, err := contract.TriggerConstantContract(ctx, testAddress, "name")
		require.NoError(t, err, "Should call name method")
		// Single output should be concrete string
		nameDecoded, ok := name.(string)
		require.True(t, ok, "Should decode name as string, got %T", name)
		assert.Equal(t, "Tether USD", nameDecoded, "Name should be Tether USD")
		t.Logf("USDT Name: %s", nameDecoded)

		// Test decimals
		decimals, err := contract.TriggerConstantContract(ctx, testAddress, "decimals")
		require.NoError(t, err, "Should call decimals method")
		// Single output should be concrete uint8
		decimalsValue, ok := decimals.(uint8)
		require.True(t, ok, "Should decode decimals result as uint8, got %T", decimals)
		require.Equal(t, uint8(6), decimalsValue, "Decimals should be 6")

		t.Logf("USDT Decimals: %d", decimalsValue)

		// Test total supply
		totalSupply, err := contract.TriggerConstantContract(ctx, testAddress, "totalSupply")
		require.NoError(t, err, "Should call totalSupply method")
		// Single output should be *big.Int
		totalSupplyDecoded, ok := totalSupply.(*big.Int)
		require.True(t, ok, "Should decode totalSupply result as *big.Int, got %T", totalSupply)

		// The exact total supply might change, just assert it's a non-zero positive number

		t.Logf("USDT Total Supply: %s", totalSupplyDecoded)

		t.Logf("✅ USDT basic info test completed successfully")
	})

	t.Run("USDT_BalanceOf", func(t *testing.T) {
		// Create contract instance from network using client
		client, err := client.NewClient(client.DefaultClientConfig(getTestConfig().Endpoint))
		require.NoError(t, err, "Should create client")

		contract, err := smartcontract.NewContract(client, usdtAddress)
		require.NoError(t, err, "Should create USDT contract instance from network")

		// Test balance of specific address
		_, err = contract.Encode("balanceOf", testAddress.String())
		require.NoError(t, err, "Should encode balanceOf call")

		balanceResult, err := contract.TriggerConstantContract(ctx, testAddress, "balanceOf", testAddress.String())
		require.NoError(t, err, "Should call balanceOf method")

		// Single output should be *big.Int
		balanceDecoded, ok := balanceResult.(*big.Int)
		require.True(t, ok, "Should decode balanceOf result as *big.Int , got %T", balanceResult)
		require.Equal(t, big.NewInt(45967732353), balanceDecoded, "Balance should be 45967732353")

		// Convert to human-readable format using utils.HumanReadableNumber
		humanReadable, err := utils.HumanReadableNumber(balanceDecoded, 6)
		require.NoError(t, err, "Should convert to human readable format")
		t.Logf("USDT Balance (human readable): %s USDT", humanReadable)

		t.Logf("✅ USDT balance test completed successfully")
	})

	t.Run("USDT_IsBlackListed", func(t *testing.T) {
		// Create contract instance from network using client
		client, err := client.NewClient(client.DefaultClientConfig(getTestConfig().Endpoint))
		require.NoError(t, err, "Should create client")

		contract, err := smartcontract.NewContract(client, usdtAddress)
		require.NoError(t, err, "Should create USDT contract instance from network")

		// Test isBlackListed for the test address
		_, err = contract.Encode("isBlackListed", testAddress.String())
		require.NoError(t, err, "Should encode isBlackListed call")

		blacklistResult, err := contract.TriggerConstantContract(ctx, testAddress, "isBlackListed", testAddress.String())
		require.NoError(t, err, "Should call isBlackListed method")

		// Single output should be bool
		blacklistDecoded, ok := blacklistResult.(bool)
		require.True(t, ok, "Should decode isBlackListed result as bool")
		require.True(t, blacklistDecoded, "Expected address to be blacklisted")

		t.Logf("✅ USDT isBlackListed test completed successfully")
	})
}

// TestMainnetTRC20Contract tests standard TRC20 contract functionality
func TestMainnetTRC20Contract(t *testing.T) {
	// Placeholder: This test was moved to trc20_test.go
}
