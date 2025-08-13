// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package trc10

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func mustAddr(t *testing.T, s string) *types.Address {
	a, err := utils.ValidateAddress(s)
	if err != nil {
		t.Fatalf("failed to create address from string %q: %v", s, err)
	}
	return a
}

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
	_, err := manager.CreateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "", "TEST", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset name cannot be empty")

	// Test empty abbreviation
	_, err = manager.CreateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", "", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset abbreviation cannot be empty")

	// Test invalid total supply
	_, err = manager.CreateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", "TEST", 0, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "total supply must be positive")

	// Test invalid TRX num
	_, err = manager.CreateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", "TEST", 1000000, 0, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TRX num must be positive")

	// Test invalid ICO num
	_, err = manager.CreateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", "TEST", 1000000, 1, 0, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ICO num must be positive")

	// Test invalid time range
	_, err = manager.CreateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", "TEST", 1000000, 1, 1, 1640995300000, 1640995200000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start time must be before end time")

	// Test invalid owner address
	_, err = manager.CreateAssetIssue2(ctx, nil, "TestToken", "TEST", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")

	// Test invalid frozen supply
	frozenSupply := []FrozenSupply{
		{FrozenAmount: 0, FrozenDays: 30},
	}
	_, err = manager.CreateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", "TEST", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, frozenSupply)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "frozen amount must be positive")

	frozenSupply = []FrozenSupply{
		{FrozenAmount: 1000, FrozenDays: 0},
	}
	_, err = manager.CreateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", "TEST", 1000000, 1, 1, 1640995200000, 1640995300000, "Test token", "https://test.com", 1000, 1000, frozenSupply)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "frozen days must be positive")
}

func TestManager_TransferAsset2_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test empty asset name
	// Test empty asset name
	_, err := manager.TransferAsset2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset name cannot be empty")

	// Test invalid amount
	_, err = manager.TransferAsset2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
	// Test invalid owner address
	_, err = manager.TransferAsset2(ctx, nil, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")

	// Test invalid to address
	_, err = manager.TransferAsset2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), nil, "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to address")

	// Test same addresses
	_, err = manager.TransferAsset2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner and to addresses cannot be the same")
}

func TestManager_ParticipateAssetIssue2_Validation(t *testing.T) {
	manager := NewManager(&client.Client{})
	ctx := context.Background()

	// Test empty asset name
	// Test empty asset name
	_, err := manager.ParticipateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset name cannot be empty")

	// Test invalid amount
	_, err = manager.ParticipateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be positive")
	// Test invalid owner address
	_, err = manager.ParticipateAssetIssue2(ctx, nil, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), "TestToken", 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")

	// Test invalid to address
	_, err = manager.ParticipateAssetIssue2(ctx, mustAddr(t, "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"), nil, "TestToken", 1000)
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
