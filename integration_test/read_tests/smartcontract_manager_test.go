package read_tests

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainnetSmartContractManager tests the smart contract manager functionality
func TestMainnetSmartContractManager(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create client and manager
	client, err := client.NewClient(getTestConfig().Endpoint)
	require.NoError(t, err, "Should create client")
	defer client.Close()

	manager := smartcontract.NewManager(client)

	// USDT contract address for testing
	usdtAddressStr := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	usdtAddress, err := types.NewAddress(usdtAddressStr)
	require.NoError(t, err, "Failed to parse USDT address")

	t.Run("GetContract", func(t *testing.T) {
		contract, err := manager.GetContract(ctx, usdtAddress)
		require.NoError(t, err, "Should get contract successfully")
		require.NotNil(t, contract, "Contract should not be nil")

		// Validate contract information
		contractAddress := contract.GetContractAddress()
		require.NotNil(t, contractAddress, "Contract address should not be nil")
		assert.Equal(t, usdtAddress.Bytes(), contractAddress, "Contract address should match")

		// Validate contract has ABI
		abi := contract.GetAbi()
		require.NotNil(t, abi, "Contract should have ABI")
		assert.Greater(t, len(abi.Entrys), 0, "ABI should have entries")

		// Validate contract name
		name := contract.GetName()
		assert.Equal(t, "TetherToken", name, "Contract name should match")

		t.Logf("✅ GetContract test completed successfully")
	})

	t.Run("GetContractInfo", func(t *testing.T) {
		contractInfo, err := manager.GetContractInfo(ctx, usdtAddress)
		require.NoError(t, err, "Should get contract info successfully")
		require.NotNil(t, contractInfo, "Contract info should not be nil")

		// Validate contract info
		contract := contractInfo.SmartContract
		require.NotNil(t, contract, "Contract should not be nil")

		contractAddress := contract.GetContractAddress()
		require.NotNil(t, contractAddress, "Contract address should not be nil")
		assert.Equal(t, usdtAddress.Bytes(), contractAddress, "Contract address should match")

		t.Logf("✅ GetContractInfo test completed successfully")
	})

	t.Run("Instance", func(t *testing.T) {
		// Test creating instance without ABI (should fetch from network)
		instance, err := manager.Instance(usdtAddress)
		require.NoError(t, err, "Should create instance successfully")
		require.NotNil(t, instance, "Instance should not be nil")

		// Validate instance properties
		assert.Equal(t, usdtAddress, instance.Address, "Instance address should match")
		require.NotNil(t, instance.ABI, "Instance should have ABI")
		assert.Greater(t, len(instance.ABI.Entrys), 0, "ABI should have entries")

		// Test calling a method
		symbol, err := instance.Call(ctx, usdtAddress, "symbol")
		require.NoError(t, err, "Should call symbol method successfully")

		symbolStr, ok := symbol.(string)
		require.True(t, ok, "Symbol should be string")
		assert.Equal(t, "USDT", symbolStr, "Symbol should be USDT")

		t.Logf("✅ Instance test completed successfully")
	})
}
