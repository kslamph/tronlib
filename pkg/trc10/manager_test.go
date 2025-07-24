package trc10

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	client := &client.Client{}
	manager := NewManager(client)
	assert.NotNil(t, manager)
	assert.Equal(t, client, manager.client)
}

func TestManager_CreateAssetIssue2_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test empty name
	_, err := manager.CreateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "", "TEST", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset name cannot be empty")

	// Test empty abbreviation
	_, err = manager.CreateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TestToken", "", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset abbreviation cannot be empty")

	// Test invalid total supply
	_, err = manager.CreateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TestToken", "TEST", 0, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "total supply must be positive")

	// Test invalid TRX num
	_, err = manager.CreateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TestToken", "TEST", 1000000, 0, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TRX num must be positive")

	// Test invalid ICO num
	_, err = manager.CreateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TestToken", "TEST", 1000000, 1, 0, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ICO num must be positive")

	// Test invalid time range
	_, err = manager.CreateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TestToken", "TEST", 1000000, 1, 1, 1640995300000, 1640995200000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start time must be before end time")

	// Test invalid owner address
	_, err = manager.CreateAssetIssue2(ctx, "invalid", "TestToken", "TEST", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")

	// Test invalid frozen supply
	frozenSupply := []FrozenSupply{
		{FrozenAmount: 0, FrozenDays: 30},
	}
	_, err = manager.CreateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TestToken", "TEST", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, frozenSupply)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "frozen amount must be positive")

	frozenSupply = []FrozenSupply{
		{FrozenAmount: 1000, FrozenDays: 0},
	}
	_, err = manager.CreateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TestToken", "TEST", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, frozenSupply)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "frozen days must be positive")
}

func TestManager_TransferAsset2_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test empty asset name
	_, err := manager.TransferAsset2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TKx9RQveWvAcPTisx6QzSgYPVUjbmCCjpJ", "", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset name cannot be empty")

	// Test invalid amount
	_, err = manager.TransferAsset2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TKx9RQveWvAcPTisx6QzSgYPVUjbmCCjpJ", "TestToken", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")

	// Test invalid owner address
	_, err = manager.TransferAsset2(ctx, "invalid", "TKx9RQveWvAcPTisx6QzSgYPVUjbmCCjpJ", "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")

	// Test invalid to address
	_, err = manager.TransferAsset2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "invalid", "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to address")

	// Test same addresses
	_, err = manager.TransferAsset2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner and to addresses cannot be the same")
}

func TestManager_ParticipateAssetIssue2_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test empty asset name
	_, err := manager.ParticipateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TKx9RQveWvAcPTisx6QzSgYPVUjbmCCjpJ", "", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset name cannot be empty")

	// Test invalid amount
	_, err = manager.ParticipateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "TKx9RQveWvAcPTisx6QzSgYPVUjbmCCjpJ", "TestToken", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")

	// Test invalid owner address
	_, err = manager.ParticipateAssetIssue2(ctx, "invalid", "TKx9RQveWvAcPTisx6QzSgYPVUjbmCCjpJ", "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")

	// Test invalid to address
	_, err = manager.ParticipateAssetIssue2(ctx, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U", "invalid", "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to address")
}

func TestManager_GetAssetIssueByName_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test empty asset name
	_, err := manager.GetAssetIssueByName(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset name cannot be empty")
}

func TestManager_GetAssetIssueListByName_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test empty asset name
	_, err := manager.GetAssetIssueListByName(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset name cannot be empty")
}

func TestManager_GetAssetIssueById_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test empty asset ID
	_, err := manager.GetAssetIssueById(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset ID cannot be empty")

	_, err = manager.GetAssetIssueById(ctx, []byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset ID cannot be empty")
}

func TestManager_GetPaginatedAssetIssueList_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test negative offset
	_, err := manager.GetPaginatedAssetIssueList(ctx, -1, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "offset cannot be negative")

	// Test invalid limit
	_, err = manager.GetPaginatedAssetIssueList(ctx, 0, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be positive")

	_, err = manager.GetPaginatedAssetIssueList(ctx, 0, 101)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit cannot exceed 100")
}

func TestFrozenSupply(t *testing.T) {
	fs := FrozenSupply{
		FrozenAmount: 1000000,
		FrozenDays:   30,
	}

	assert.Equal(t, int64(1000000), fs.FrozenAmount)
	assert.Equal(t, int64(30), fs.FrozenDays)
}