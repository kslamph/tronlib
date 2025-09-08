package read_tests

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/resources"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainnetResourcesManager tests the resources manager functionality using mainnet data
func TestMainnetResourcesManager(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create client and manager
	client, err := client.NewClient(getTestConfig().Endpoint)
	require.NoError(t, err, "Should create client")
	defer client.Close()

	manager := resources.NewManager(client)

	// Test addresses for resource delegation
	// Using the USDT contract address that we know works from smartcontract tests
	fromAddressStr := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // USDT contract address
	toAddressStr := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"   // Same address for testing

	// Log addresses for debugging
	t.Logf("Testing with fromAddress: %s", fromAddressStr)
	t.Logf("Testing with toAddress: %s", toAddressStr)

	fromAddress, err := types.NewAddress(fromAddressStr)
	require.NoError(t, err, "Failed to parse from address")

	toAddress, err := types.NewAddress(toAddressStr)
	require.NoError(t, err, "Failed to parse to address")

	t.Run("GetDelegatedResourceV2", func(t *testing.T) {
		delegatedResources, err := manager.GetDelegatedResourceV2(ctx, fromAddress, toAddress)
		require.NoError(t, err, "Should get delegated resources successfully")

		// Validate response structure
		require.NotNil(t, delegatedResources, "Delegated resources should not be nil")

		// Log the number of delegated resources found
		t.Logf("Found %d delegated resources", len(delegatedResources.DelegatedResource))

		// Validate each delegated resource if any exist
		for _, dr := range delegatedResources.DelegatedResource {
			require.NotNil(t, dr, "Delegated resource should not be nil")

			// Validate addresses
			require.NotNil(t, dr.From, "From address should not be nil")
			require.NotNil(t, dr.To, "To address should not be nil")

			// Validate resource amounts
			assert.GreaterOrEqual(t, dr.FrozenBalanceForBandwidth, int64(0), "Bandwidth should be >= 0")
			assert.GreaterOrEqual(t, dr.FrozenBalanceForEnergy, int64(0), "Energy should be >= 0")

			t.Logf("✅ Validated delegated resource: from=%s, to=%s, bandwidth=%d, energy=%d",
				dr.From, dr.To, dr.FrozenBalanceForBandwidth, dr.FrozenBalanceForEnergy)
		}

		t.Logf("✅ GetDelegatedResourceV2 test completed successfully")
	})

	t.Run("GetDelegatedResourceAccountIndexV2", func(t *testing.T) {
		accountIndex, err := manager.GetDelegatedResourceAccountIndexV2(ctx, fromAddress)
		require.NoError(t, err, "Should get delegated resource account index successfully")

		// Validate response structure
		require.NotNil(t, accountIndex, "Account index should not be nil")

		// Log account index information
		t.Logf("Account: %s", accountIndex.Account)
		t.Logf("From accounts: %d", len(accountIndex.FromAccounts))
		t.Logf("To accounts: %d", len(accountIndex.ToAccounts))

		// Validate account index fields
		require.NotNil(t, accountIndex.Account, "Account should not be nil")
		assert.Equal(t, fromAddress.Bytes(), accountIndex.Account, "Account should match input address")

		// Validate from/to accounts (can be empty)
		for _, fromAcc := range accountIndex.FromAccounts {
			require.NotNil(t, fromAcc, "From account should not be nil")
		}
		for _, toAcc := range accountIndex.ToAccounts {
			require.NotNil(t, toAcc, "To account should not be nil")
		}

		t.Logf("✅ GetDelegatedResourceAccountIndexV2 test completed successfully")
	})

	t.Run("GetCanDelegatedMaxSize", func(t *testing.T) {
		// Test with both resource types
		for _, resourceType := range []int32{0, 1} { // 0 = Bandwidth, 1 = Energy
			resourceName := "Bandwidth"
			if resourceType == 1 {
				resourceName = "Energy"
			}

			maxSize, err := manager.GetCanDelegatedMaxSize(ctx, fromAddress, resourceType)
			require.NoError(t, err, "Should get max delegatable size for %s successfully", resourceName)

			// Validate response structure
			require.NotNil(t, maxSize, "Max size should not be nil")

			// Log max size information
			t.Logf("%s - Max size: %d", resourceName, maxSize.MaxSize)

			// Validate max size fields
			assert.GreaterOrEqual(t, maxSize.MaxSize, int64(0), "%s max size should be >= 0", resourceName)

			t.Logf("✅ Validated %s delegation limits", resourceName)
		}

		t.Logf("✅ GetCanDelegatedMaxSize test completed successfully")
	})

	t.Run("GetAvailableUnfreezeCount", func(t *testing.T) {
		unfreezeCount, err := manager.GetAvailableUnfreezeCount(ctx, fromAddress)
		require.NoError(t, err, "Should get available unfreeze count successfully")

		// Validate response structure
		require.NotNil(t, unfreezeCount, "Unfreeze count should not be nil")

		// Log unfreeze count information
		t.Logf("Available unfreeze count: %d", unfreezeCount.Count)

		// Validate unfreeze count
		assert.GreaterOrEqual(t, unfreezeCount.Count, int64(0), "Unfreeze count should be >= 0")

		t.Logf("✅ GetAvailableUnfreezeCount test completed successfully")
	})

	t.Run("GetCanWithdrawUnfreezeAmount", func(t *testing.T) {
		// Use current timestamp for testing
		currentTimestamp := time.Now().UnixMilli()

		withdrawAmount, err := manager.GetCanWithdrawUnfreezeAmount(ctx, fromAddress, currentTimestamp)
		require.NoError(t, err, "Should get withdrawable unfreeze amount successfully")

		// Validate response structure
		require.NotNil(t, withdrawAmount, "Withdraw amount should not be nil")

		// Log withdraw amount information
		t.Logf("Withdrawable amount: %d", withdrawAmount.Amount)

		// Validate withdraw amount
		assert.GreaterOrEqual(t, withdrawAmount.Amount, int64(0), "Withdraw amount should be >= 0")

		t.Logf("✅ GetCanWithdrawUnfreezeAmount test completed successfully")
	})
}
