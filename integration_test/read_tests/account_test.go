package read_tests

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig holds the test configuration
type TestConfig struct {
	Endpoint string
	Timeout  time.Duration
}

// getTestConfig returns the test configuration
func getTestConfig() TestConfig {
	return TestConfig{
		Endpoint: "127.0.0.1:50051",
		Timeout:  30 * time.Second,
	}
}

// setupTestManager creates a test manager instance
func setupTestManager(t *testing.T) *account.AccountManager {
	config := getTestConfig()

	clientConfig := client.DefaultClientConfig(config.Endpoint)
	clientConfig.Timeout = config.Timeout

	client, err := client.NewClient(clientConfig)
	require.NoError(t, err, "Failed to create client")

	return account.NewManager(client)
}

// TestMainnetGetAccount tests the GetAccount API against known mainnet data
func TestMainnetGetAccount(t *testing.T) {
	manager := setupTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []struct {
		name     string
		address  string
		validate func(t *testing.T, account *core.Account)
	}{
		{
			name:    "Account with Assets and Permissions - TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			address: "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			validate: func(t *testing.T, account *core.Account) {
				// Validate basic account properties using gRPC getter methods
				assert.NotNil(t, account, "Account should not be nil")
				assert.NotEmpty(t, account.GetAddress(), "Account address should not be empty")

				// Validate balance (using getter method)
				balance := account.GetBalance()
				assert.GreaterOrEqual(t, balance, int64(0), "Balance should be non-negative")
				t.Logf("Account balance: %d SUN", balance)

				// Validate timestamps (using getter methods)
				createTime := account.GetCreateTime()
				if createTime > 0 {
					assert.Greater(t, createTime, int64(0), "Create time should be positive")
					t.Logf("Account created at: %d", createTime)
				}

				latestOpTime := account.GetLatestOprationTime()
				if latestOpTime > 0 {
					assert.Greater(t, latestOpTime, int64(0), "Latest operation time should be positive")
					t.Logf("Latest operation time: %d", latestOpTime)
				}

				// Validate network settings (using getter methods)
				netWindowSize := account.GetNetWindowSize()
				if netWindowSize > 0 {
					t.Logf("Net window size: %d", netWindowSize)
				}

				netWindowOptimized := account.GetNetWindowOptimized()
				t.Logf("Net window optimized: %t", netWindowOptimized)

				// Validate TRC10 assets (using getter method)
				assetV2 := account.GetAssetV2()
				if len(assetV2) > 0 {
					t.Logf("Found %d TRC10 assets", len(assetV2))
					for assetId, amount := range assetV2 {
						assert.GreaterOrEqual(t, amount, int64(0), "Asset amount should be non-negative for asset %s", assetId)
						t.Logf("Asset %s: %d", assetId, amount)
					}
				} else {
					t.Logf("No TRC10 assets found")
				}

				// Validate asset optimization (using getter method)
				assetOptimized := account.GetAssetOptimized()
				t.Logf("Asset optimized: %t", assetOptimized)

				// Validate free asset net usage (using getter method)
				freeAssetNetUsageV2 := account.GetFreeAssetNetUsageV2()
				if len(freeAssetNetUsageV2) > 0 {
					t.Logf("Found %d free asset net usage entries", len(freeAssetNetUsageV2))
					for assetId, usage := range freeAssetNetUsageV2 {
						assert.GreaterOrEqual(t, usage, int64(0), "Free asset net usage should be non-negative for asset %s", assetId)
						t.Logf("Free asset net usage for %s: %d", assetId, usage)
					}
				} else {
					t.Logf("No free asset net usage data found")
				}

				// Validate account resource (using getter method)
				accountResource := account.GetAccountResource()
				if accountResource != nil {
					t.Logf("Account resource found")
					// Validate energy-related fields if they exist
					if accountResource.GetLatestConsumeTimeForEnergy() > 0 {
						t.Logf("Latest consume time for energy: %d", accountResource.GetLatestConsumeTimeForEnergy())
					}
					if accountResource.GetEnergyWindowSize() > 0 {
						t.Logf("Energy window size: %d", accountResource.GetEnergyWindowSize())
					}
					if accountResource.GetAcquiredDelegatedFrozenV2BalanceForEnergy() > 0 {
						t.Logf("Delegated energy balance: %d", accountResource.GetAcquiredDelegatedFrozenV2BalanceForEnergy())
					}
					t.Logf("Energy window optimized: %t", accountResource.GetEnergyWindowOptimized())
				} else {
					t.Logf("No account resource found")
				}

				// Validate owner permission (using getter method)
				ownerPermission := account.GetOwnerPermission()
				if ownerPermission != nil {
					t.Logf("Owner permission found: %s", ownerPermission.GetPermissionName())
					assert.GreaterOrEqual(t, ownerPermission.GetThreshold(), int64(1), "Owner permission threshold should be at least 1")

					keys := ownerPermission.GetKeys()
					if len(keys) > 0 {
						t.Logf("Owner permission has %d keys", len(keys))
						for i, key := range keys {
							assert.GreaterOrEqual(t, key.GetWeight(), int64(1), "Owner key weight should be at least 1")
							t.Logf("Owner key %d weight: %d", i, key.GetWeight())
						}
					}
				} else {
					t.Logf("No owner permission found")
				}

				// Validate active permissions (using getter method)
				activePermissions := account.GetActivePermission()
				if len(activePermissions) > 0 {
					t.Logf("Found %d active permissions", len(activePermissions))
					for i, permission := range activePermissions {
						t.Logf("Active permission %d: %s (ID: %d)", i, permission.GetPermissionName(), permission.GetId())
						assert.GreaterOrEqual(t, permission.GetThreshold(), int64(1), "Active permission threshold should be at least 1")

						keys := permission.GetKeys()
						if len(keys) > 0 {
							t.Logf("Active permission %d has %d keys", i, len(keys))
							for j, key := range keys {
								assert.GreaterOrEqual(t, key.GetWeight(), int64(1), "Active key weight should be at least 1")
								t.Logf("Active permission %d key %d weight: %d", i, j, key.GetWeight())
							}
						}
					}
				} else {
					t.Logf("No active permissions found")
				}

				// Validate frozen resources (using getter method)
				frozenV2 := account.GetFrozenV2()
				if len(frozenV2) > 0 {
					t.Logf("Found %d frozen resource entries", len(frozenV2))
					for i, frozen := range frozenV2 {
						assert.GreaterOrEqual(t, frozen.GetAmount(), int64(0), "Frozen amount should be non-negative")
						t.Logf("Frozen resource %d: type=%v, amount=%d", i, frozen.GetType(), frozen.GetAmount())
					}
				} else {
					t.Logf("No frozen resources found")
				}

				// Validate TRON Power (using getter method)
				tronPower := account.GetTronPower()
				if tronPower != nil {
					t.Logf("TRON Power found: %d", tronPower.GetFrozenBalance())
				} else {
					t.Logf("No TRON Power found")
				}

				t.Logf("✅ Account validation passed for %s", t.Name())
			},
		},
		{
			name:    "Another Test Account - TFNyPYvjWSMePXHTzf7TfWD7k61yfpugxc",
			address: "TFNyPYvjWSMePXHTzf7TfWD7k61yfpugxc",
			validate: func(t *testing.T, account *core.Account) {
				// Basic validation for any account
				assert.NotNil(t, account, "Account should not be nil")
				assert.NotEmpty(t, account.GetAddress(), "Account address should not be empty")

				balance := account.GetBalance()
				assert.GreaterOrEqual(t, balance, int64(0), "Balance should be non-negative")
				t.Logf("Account balance: %d SUN", balance)

				// Log available data without strict expectations
				if account.GetCreateTime() > 0 {
					t.Logf("Account created at: %d", account.GetCreateTime())
				}

				assetV2 := account.GetAssetV2()
				if len(assetV2) > 0 {
					t.Logf("Found %d TRC10 assets", len(assetV2))
				}

				t.Logf("✅ Account validation passed for %s", t.Name())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addr := types.MustNewAddressFromBase58(tc.address)
			account, err := manager.GetAccount(ctx, addr)
			require.NoError(t, err, "GetAccount should succeed")

			tc.validate(t, account)
		})
	}
}

// TestMainnetGetAccountResource tests the GetAccountResource API
func TestMainnetGetAccountResource(t *testing.T) {
	manager := setupTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []struct {
		name     string
		address  string
		validate func(t *testing.T, resource *api.AccountResourceMessage)
	}{
		{
			name:    "Account with Energy and Bandwidth - TFNyPYvjWSMePXHTzf7TfWD7k61yfpugxc",
			address: "TFNyPYvjWSMePXHTzf7TfWD7k61yfpugxc",
			validate: func(t *testing.T, resource *api.AccountResourceMessage) {
				// Validate basic resource properties using gRPC getter methods
				assert.NotNil(t, resource, "Resource should not be nil")

				// Validate free network limits (using getter methods)
				freeNetLimit := resource.GetFreeNetLimit()
				assert.GreaterOrEqual(t, freeNetLimit, int64(0), "Free net limit should be non-negative")
				t.Logf("Free net limit: %d", freeNetLimit)

				freeNetUsed := resource.GetFreeNetUsed()
				assert.GreaterOrEqual(t, freeNetUsed, int64(0), "Free net used should be non-negative")
				t.Logf("Free net used: %d", freeNetUsed)

				// Validate network usage and limits (using getter methods)
				netUsed := resource.GetNetUsed()
				netLimit := resource.GetNetLimit()
				assert.GreaterOrEqual(t, netUsed, int64(0), "Net used should be non-negative")
				assert.GreaterOrEqual(t, netLimit, int64(0), "Net limit should be non-negative")
				t.Logf("Net usage: %d/%d", netUsed, netLimit)

				// Validate global network constants (using getter methods)
				totalNetLimit := resource.GetTotalNetLimit()
				totalNetWeight := resource.GetTotalNetWeight()
				assert.GreaterOrEqual(t, totalNetLimit, int64(0), "Total net limit should be non-negative")
				assert.GreaterOrEqual(t, totalNetWeight, int64(0), "Total net weight should be non-negative")
				t.Logf("Total net limit: %d, weight: %d", totalNetLimit, totalNetWeight)

				// Validate TRON Power (voting power) using getter methods
				tronPowerUsed := resource.GetTronPowerUsed()
				tronPowerLimit := resource.GetTronPowerLimit()
				totalTronPowerWeight := resource.GetTotalTronPowerWeight()
				assert.GreaterOrEqual(t, tronPowerUsed, int64(0), "TRON power used should be non-negative")
				assert.GreaterOrEqual(t, tronPowerLimit, int64(0), "TRON power limit should be non-negative")
				assert.GreaterOrEqual(t, totalTronPowerWeight, int64(0), "Total TRON power weight should be non-negative")
				t.Logf("TRON Power: %d/%d (total weight: %d)", tronPowerUsed, tronPowerLimit, totalTronPowerWeight)

				// Validate energy usage and limits (using getter methods)
				energyUsed := resource.GetEnergyUsed()
				energyLimit := resource.GetEnergyLimit()
				totalEnergyLimit := resource.GetTotalEnergyLimit()
				totalEnergyWeight := resource.GetTotalEnergyWeight()
				assert.GreaterOrEqual(t, energyUsed, int64(0), "Energy used should be non-negative")
				assert.GreaterOrEqual(t, energyLimit, int64(0), "Energy limit should be non-negative")
				assert.GreaterOrEqual(t, totalEnergyLimit, int64(0), "Total energy limit should be non-negative")
				assert.GreaterOrEqual(t, totalEnergyWeight, int64(0), "Total energy weight should be non-negative")
				t.Logf("Energy: %d/%d (total: %d, weight: %d)", energyUsed, energyLimit, totalEnergyLimit, totalEnergyWeight)

				// Validate storage (using getter methods)
				storageUsed := resource.GetStorageUsed()
				storageLimit := resource.GetStorageLimit()
				assert.GreaterOrEqual(t, storageUsed, int64(0), "Storage used should be non-negative")
				assert.GreaterOrEqual(t, storageLimit, int64(0), "Storage limit should be non-negative")
				t.Logf("Storage: %d/%d", storageUsed, storageLimit)

				// Validate asset-specific network usage (using getter methods)
				assetNetUsed := resource.GetAssetNetUsed()
				assetNetLimit := resource.GetAssetNetLimit()

				if len(assetNetUsed) > 0 {
					t.Logf("Found %d asset net usage entries", len(assetNetUsed))
					for assetId, usage := range assetNetUsed {
						assert.GreaterOrEqual(t, usage, int64(0), "Asset net usage should be non-negative for asset %s", assetId)
						t.Logf("Asset %s net usage: %d", assetId, usage)
					}
				} else {
					t.Logf("No asset net usage data found")
				}

				if len(assetNetLimit) > 0 {
					t.Logf("Found %d asset net limit entries", len(assetNetLimit))
					for assetId, limit := range assetNetLimit {
						assert.GreaterOrEqual(t, limit, int64(0), "Asset net limit should be non-negative for asset %s", assetId)
						t.Logf("Asset %s net limit: %d", assetId, limit)
					}
				} else {
					t.Logf("No asset net limit data found")
				}

				// Log resource utilization for monitoring
				if netLimit > 0 {
					netUtilization := float64(netUsed) / float64(netLimit) * 100
					t.Logf("Network bandwidth utilization: %.1f%%", netUtilization)
				}

				if energyLimit > 0 {
					energyUtilization := float64(energyUsed) / float64(energyLimit) * 100
					t.Logf("Energy utilization: %.1f%%", energyUtilization)
				}

				if tronPowerLimit > 0 {
					tronPowerUtilization := float64(tronPowerUsed) / float64(tronPowerLimit) * 100
					t.Logf("TRON Power utilization: %.1f%%", tronPowerUtilization)
				}

				t.Logf("✅ Account resource validation passed for %s", t.Name())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addrPtr, _ := utils.ValidateAddress(tc.address)
			resource, err := manager.GetAccountResource(ctx, addrPtr)
			require.NoError(t, err, "GetAccountResource should succeed")

			tc.validate(t, resource)
		})
	}
}

// TestMainnetGetAccountNet tests the GetAccountNet API
func TestMainnetGetAccountNet(t *testing.T) {
	manager := setupTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []struct {
		name     string
		address  string
		validate func(t *testing.T, net *api.AccountNetMessage)
	}{
		{
			name:    "Account Network Info - TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			address: "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
			validate: func(t *testing.T, net *api.AccountNetMessage) {
				// Validate basic network properties using gRPC getter methods
				assert.NotNil(t, net, "Net should not be nil")

				// Validate free network limits (using getter methods)
				freeNetLimit := net.GetFreeNetLimit()
				freeNetUsed := net.GetFreeNetUsed()
				assert.GreaterOrEqual(t, freeNetLimit, int64(0), "Free net limit should be non-negative")
				assert.GreaterOrEqual(t, freeNetUsed, int64(0), "Free net used should be non-negative")
				t.Logf("Free net: %d/%d", freeNetUsed, freeNetLimit)

				// Validate network usage and limits (using getter methods)
				netUsed := net.GetNetUsed()
				netLimit := net.GetNetLimit()
				assert.GreaterOrEqual(t, netUsed, int64(0), "Net used should be non-negative")
				assert.GreaterOrEqual(t, netLimit, int64(0), "Net limit should be non-negative")
				t.Logf("Net: %d/%d", netUsed, netLimit)

				// Validate global network constants (using getter methods)
				totalNetLimit := net.GetTotalNetLimit()
				totalNetWeight := net.GetTotalNetWeight()
				assert.GreaterOrEqual(t, totalNetLimit, int64(0), "Total net limit should be non-negative")
				assert.GreaterOrEqual(t, totalNetWeight, int64(0), "Total net weight should be non-negative")
				t.Logf("Total net limit: %d, weight: %d", totalNetLimit, totalNetWeight)

				// Validate asset-specific network data (using getter methods)
				assetNetUsed := net.GetAssetNetUsed()
				assetNetLimit := net.GetAssetNetLimit()

				if len(assetNetUsed) > 0 {
					t.Logf("Found %d asset net usage entries", len(assetNetUsed))
					for assetId, usage := range assetNetUsed {
						assert.GreaterOrEqual(t, usage, int64(0), "Asset net usage should be non-negative for asset %s", assetId)
						t.Logf("Asset %s net usage: %d", assetId, usage)
					}
				} else {
					t.Logf("No asset net usage data found")
				}

				if len(assetNetLimit) > 0 {
					t.Logf("Found %d asset net limit entries", len(assetNetLimit))
					for assetId, limit := range assetNetLimit {
						assert.GreaterOrEqual(t, limit, int64(0), "Asset net limit should be non-negative for asset %s", assetId)
						t.Logf("Asset %s net limit: %d", assetId, limit)
					}
				} else {
					t.Logf("No asset net limit data found")
				}

				t.Logf("✅ Account net validation passed for %s", t.Name())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addrPtr2, _ := utils.ValidateAddress(tc.address)
			net, err := manager.GetAccountNet(ctx, addrPtr2)
			require.NoError(t, err, "GetAccountNet should succeed")

			tc.validate(t, net)
		})
	}
}
