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
	manager := setupSmartContractTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// USDT contract address
	usdtAddress := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	// Test address for balance check
	testAddress := "TUHwTn3JhQqdys4ckqQ86EsWk3KC2p2tZc"

	t.Run("USDT_BasicInfo", func(t *testing.T) {
		// Create contract instance from network using client
		client, err := client.NewClient(client.DefaultClientConfig(getTestConfig().Endpoint))
		require.NoError(t, err, "Should create client")

		contract, err := smartcontract.NewContract(usdtAddress, client)
		require.NoError(t, err, "Should create USDT contract instance from network")

		// Test symbol
		symbolData, err := contract.EncodeInput("symbol")
		require.NoError(t, err, "Should encode symbol call")

		symbolResult, err := manager.TriggerConstantContract(ctx, testAddress, usdtAddress, symbolData, 0)
		require.NoError(t, err, "Should call symbol method")
		require.Greater(t, len(symbolResult.GetConstantResult()), 0, "Should have constant result")

		symbolDecoded, err := contract.DecodeResult("symbol", symbolResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode symbol result")
		assert.Equal(t, "USDT", symbolDecoded, "Should decode symbol result")
		// Test name
		nameData, err := contract.EncodeInput("name")
		require.NoError(t, err, "Should encode name call")

		nameResult, err := manager.TriggerConstantContract(ctx, testAddress, usdtAddress, nameData, 0)
		require.NoError(t, err, "Should call name method")
		require.Greater(t, len(nameResult.GetConstantResult()), 0, "Should have constant result")

		nameDecoded, err := contract.DecodeResult("name", nameResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode name result")
		assert.Equal(t, "Tether USD", nameDecoded, "Should decode name result")

		// Test decimals
		decimalsData, err := contract.EncodeInput("decimals")
		require.NoError(t, err, "Should encode decimals call")

		decimalsResult, err := manager.TriggerConstantContract(ctx, testAddress, usdtAddress, decimalsData, 0)
		require.NoError(t, err, "Should call decimals method")
		require.Greater(t, len(decimalsResult.GetConstantResult()), 0, "Should have constant result")

		decimalsDecoded, err := contract.DecodeResult("decimals", decimalsResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode decimals result")
		assert.Equal(t, uint8(6), decimalsDecoded, "Should decode decimals result")

		// Test total supply
		totalSupplyData, err := contract.EncodeInput("totalSupply")
		require.NoError(t, err, "Should encode totalSupply call")

		totalSupplyResult, err := manager.TriggerConstantContract(ctx, testAddress, usdtAddress, totalSupplyData, 0)
		require.NoError(t, err, "Should call totalSupply method")
		require.Greater(t, len(totalSupplyResult.GetConstantResult()), 0, "Should have constant result")

		totalSupplyDecoded, err := contract.DecodeResult("totalSupply", totalSupplyResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode totalSupply result")
		t.Logf("USDT Total Supply: %v", totalSupplyDecoded)

		t.Logf("✅ USDT basic info test completed successfully")
	})

	t.Run("USDT_BalanceOf", func(t *testing.T) {
		// Create contract instance from network using client
		client, err := client.NewClient(client.DefaultClientConfig(getTestConfig().Endpoint))
		require.NoError(t, err, "Should create client")

		contract, err := smartcontract.NewContract(usdtAddress, client)
		require.NoError(t, err, "Should create USDT contract instance from network")

		// Test balance of specific address
		balanceData, err := contract.EncodeInput("balanceOf", testAddress)
		require.NoError(t, err, "Should encode balanceOf call")

		balanceResult, err := manager.TriggerConstantContract(ctx, testAddress, usdtAddress, balanceData, 0)
		require.NoError(t, err, "Should call balanceOf method")
		require.Greater(t, len(balanceResult.GetConstantResult()), 0, "Should have constant result")

		balanceDecoded, err := contract.DecodeResult("balanceOf", balanceResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode balanceOf result")
		require.Equal(t, balanceDecoded.(string), "45929521353", "Should be 45929.521353")

		t.Logf("USDT Balance of %s: %s", testAddress, balanceDecoded.(string))

		// Convert to human-readable format using utils.HumanReadableNumber
		humanReadable, err := utils.HumanReadableNumber(balanceDecoded.(string), 6)
		require.NoError(t, err, "Should convert to human readable format")
		t.Logf("USDT Balance (human readable): %s USDT", humanReadable)

		t.Logf("✅ USDT balance test completed successfully")
	})

	t.Run("USDT_IsBlackListed", func(t *testing.T) {
		// Create contract instance from network using client
		client, err := client.NewClient(client.DefaultClientConfig(getTestConfig().Endpoint))
		require.NoError(t, err, "Should create client")

		contract, err := smartcontract.NewContract(usdtAddress, client)
		require.NoError(t, err, "Should create USDT contract instance from network")

		// Test isBlackListed for the test address
		blacklistData, err := contract.EncodeInput("isBlackListed", testAddress)
		require.NoError(t, err, "Should encode isBlackListed call")

		blacklistResult, err := manager.TriggerConstantContract(ctx, testAddress, usdtAddress, blacklistData, 0)
		require.NoError(t, err, "Should call isBlackListed method")
		require.Greater(t, len(blacklistResult.GetConstantResult()), 0, "Should have constant result")

		blacklistDecoded, err := contract.DecodeResult("isBlackListed", blacklistResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode isBlackListed result")
		require.True(t, blacklistDecoded.(bool), "Expected address to be blacklisted")

		// Verify it returns true as expected
		if blacklistDecoded != nil {
			if boolResult, ok := blacklistDecoded.(bool); ok {
				assert.True(t, boolResult, "Expected address to be blacklisted")
			} else if arrayResult, ok := blacklistDecoded.([]interface{}); ok && len(arrayResult) > 0 {
				if boolResult, ok := arrayResult[0].(bool); ok {
					assert.True(t, boolResult, "Expected address to be blacklisted")
				}
			}
		}

		t.Logf("✅ USDT isBlackListed test completed successfully")
	})
}

// TestMainnetTRC20Contract tests standard TRC20 contract functionality
func TestMainnetTRC20Contract(t *testing.T) {
	manager := setupSmartContractTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// TRC20 contract address (USDD)
	trc20Address := "TPYmHEhy5n8TCEfYGqW2rPxsghSfzghPDn"
	// Test address for balance check
	testAddress := "TRQ4u4Qog3dFWupQ5DnF8GP9pnjyrc8q2X"
	// Spender address for allowance check
	spenderAddress := "TQrq2p1aoAkNK94q3Q69ubJcv5nQ9y675R"
	maxUint256 := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	maxUint256.Sub(maxUint256, big.NewInt(1))
	t.Run("TRC20_BasicInfo", func(t *testing.T) {
		// Create contract instance using standard TRC20 ABI
		contract, err := smartcontract.NewContract(trc20Address, types.ERC20ABI)
		require.NoError(t, err, "Should create TRC20 contract instance")

		// Test symbol
		symbolData, err := contract.EncodeInput("symbol")
		require.NoError(t, err, "Should encode symbol call")

		symbolResult, err := manager.TriggerConstantContract(ctx, testAddress, trc20Address, symbolData, 0)
		require.NoError(t, err, "Should call symbol method")
		require.Greater(t, len(symbolResult.GetConstantResult()), 0, "Should have constant result")

		symbolDecoded, err := contract.DecodeResult("symbol", symbolResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode symbol result")
		t.Logf("TRC20 Symbol: %v", symbolDecoded)

		// Test name
		nameData, err := contract.EncodeInput("name")
		require.NoError(t, err, "Should encode name call")

		nameResult, err := manager.TriggerConstantContract(ctx, testAddress, trc20Address, nameData, 0)
		require.NoError(t, err, "Should call name method")
		require.Greater(t, len(nameResult.GetConstantResult()), 0, "Should have constant result")

		nameDecoded, err := contract.DecodeResult("name", nameResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode name result")
		t.Logf("TRC20 Name: %v", nameDecoded)

		// Test decimals
		decimalsData, err := contract.EncodeInput("decimals")
		require.NoError(t, err, "Should encode decimals call")

		decimalsResult, err := manager.TriggerConstantContract(ctx, testAddress, trc20Address, decimalsData, 0)
		require.NoError(t, err, "Should call decimals method")
		require.Greater(t, len(decimalsResult.GetConstantResult()), 0, "Should have constant result")

		decimalsDecoded, err := contract.DecodeResult("decimals", decimalsResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode decimals result")
		t.Logf("TRC20 Decimals: %v", decimalsDecoded)

		// Test total supply
		totalSupplyData, err := contract.EncodeInput("totalSupply")
		require.NoError(t, err, "Should encode totalSupply call")

		totalSupplyResult, err := manager.TriggerConstantContract(ctx, testAddress, trc20Address, totalSupplyData, 0)
		require.NoError(t, err, "Should call totalSupply method")
		require.Greater(t, len(totalSupplyResult.GetConstantResult()), 0, "Should have constant result")

		totalSupplyDecoded, err := contract.DecodeResult("totalSupply", totalSupplyResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode totalSupply result")
		t.Logf("TRC20 Total Supply: %v", totalSupplyDecoded)

		t.Logf("✅ TRC20 basic info test completed successfully")
	})

	t.Run("TRC20_BalanceOf", func(t *testing.T) {
		// Create contract instance using standard TRC20 ABI
		contract, err := smartcontract.NewContract(trc20Address, types.ERC20ABI)
		require.NoError(t, err, "Should create TRC20 contract instance")

		// Test balance of specific address
		balanceData, err := contract.EncodeInput("balanceOf", testAddress)
		require.NoError(t, err, "Should encode balanceOf call")

		balanceResult, err := manager.TriggerConstantContract(ctx, testAddress, trc20Address, balanceData, 0)
		require.NoError(t, err, "Should call balanceOf method")
		require.Greater(t, len(balanceResult.GetConstantResult()), 0, "Should have constant result")

		balanceDecoded, err := contract.DecodeResult("balanceOf", balanceResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode balanceOf result")
		t.Logf("TRC20 Balance of %s: %v", testAddress, balanceDecoded)

		// Convert to human-readable format using utils.HumanReadableNumber (USDD has 18 decimals)
		humanReadable, err := utils.HumanReadableNumber(balanceDecoded.(string), 18)
		require.NoError(t, err, "Should convert to human readable format")
		t.Logf("TRC20 Balance (human readable): %s", humanReadable)

		t.Logf("✅ TRC20 balance test completed successfully")
	})

	t.Run("TRC20_Allowance", func(t *testing.T) {
		// Create contract instance using standard TRC20 ABI
		contract, err := smartcontract.NewContract(trc20Address, types.ERC20ABI)
		require.NoError(t, err, "Should create TRC20 contract instance")

		// Test allowance between owner and spender
		allowanceData, err := contract.EncodeInput("allowance", testAddress, spenderAddress)
		require.NoError(t, err, "Should encode allowance call")

		allowanceResult, err := manager.TriggerConstantContract(ctx, testAddress, trc20Address, allowanceData, 0)
		require.NoError(t, err, "Should call allowance method")
		require.Greater(t, len(allowanceResult.GetConstantResult()), 0, "Should have constant result")

		allowanceDecoded, err := contract.DecodeResult("allowance", allowanceResult.GetConstantResult()[0])
		require.NoError(t, err, "Should decode allowance result")
		t.Logf("TRC20 Allowance from %s to %s: %v", testAddress, spenderAddress, allowanceDecoded)

		// Convert to human-readable format using utils.HumanReadableNumber (USDD has 18 decimals)
		humanReadable, err := utils.HumanReadableNumber(allowanceDecoded.(string), 18)
		require.NoError(t, err, "Should convert to human readable format")
		allowanceBigInt, ok := new(big.Int).SetString(allowanceDecoded.(string), 10)
		require.True(t, ok, "Should convert to big.Int")
		require.Equal(t, allowanceBigInt.Cmp(maxUint256), 0, "Should unlimited allowance")
		t.Logf("TRC20 Allowance (human readable): %s", humanReadable)

		t.Logf("✅ TRC20 allowance test completed successfully")
	})
}
