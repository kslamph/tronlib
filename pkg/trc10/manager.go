// Package trc10 provides high-level TRC10 token functionality
package trc10

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Manager provides high-level TRC20 token operations
type Manager struct {
	client *client.Client
}

// NewManager creates a new TRC20 manager
func NewManager(client *client.Client) *Manager {
	return &Manager{
		client: client,
	}
}

// CreateAssetIssue2 creates an asset issue (TRC10 token) (v2)
func (m *Manager) CreateAssetIssue2(ctx context.Context, ownerAddress string, name string, abbr string, totalSupply int64, trxNum int32, icoNum int32, startTime int64, endTime int64, description string, url string, freeAssetNetLimit int64, publicFreeAssetNetLimit int64, frozenSupply []FrozenSupply) (*api.TransactionExtention, error) {
	// Validate inputs
	if name == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}
	if abbr == "" {
		return nil, fmt.Errorf("asset abbreviation cannot be empty")
	}
	if totalSupply <= 0 {
		return nil, fmt.Errorf("total supply must be positive")
	}
	if trxNum <= 0 {
		return nil, fmt.Errorf("TRX num must be positive")
	}
	if icoNum <= 0 {
		return nil, fmt.Errorf("ICO num must be positive")
	}
	if startTime >= endTime {
		return nil, fmt.Errorf("start time must be before end time")
	}

	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	// Convert frozen supply
	var protoFrozenSupply []*core.AssetIssueContract_FrozenSupply
	for i, fs := range frozenSupply {
		if fs.FrozenAmount <= 0 {
			return nil, fmt.Errorf("frozen amount must be positive for frozen supply %d", i)
		}
		if fs.FrozenDays <= 0 {
			return nil, fmt.Errorf("frozen days must be positive for frozen supply %d", i)
		}

		protoFrozenSupply = append(protoFrozenSupply, &core.AssetIssueContract_FrozenSupply{
			FrozenAmount: fs.FrozenAmount,
			FrozenDays:   fs.FrozenDays,
		})
	}

	req := &core.AssetIssueContract{
		OwnerAddress:            addr.Bytes(),
		Name:                    []byte(name),
		Abbr:                    []byte(abbr),
		TotalSupply:             totalSupply,
		TrxNum:                  trxNum,
		Num:                     icoNum,
		StartTime:               startTime,
		EndTime:                 endTime,
		Description:             []byte(description),
		Url:                     []byte(url),
		FreeAssetNetLimit:       freeAssetNetLimit,
		PublicFreeAssetNetLimit: publicFreeAssetNetLimit,
		FrozenSupply:            protoFrozenSupply,
	}

	return m.client.CreateAssetIssue2(ctx, req)
}

// FrozenSupply represents frozen supply for asset creation
type FrozenSupply struct {
	FrozenAmount int64
	FrozenDays   int64
}

// UpdateAsset2 updates an asset (v2)
func (m *Manager) UpdateAsset2(ctx context.Context, ownerAddress string, description string, url string, newLimit int64, newPublicLimit int64) (*api.TransactionExtention, error) {
	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.UpdateAssetContract{
		OwnerAddress:   addr.Bytes(),
		Description:    []byte(description),
		Url:            []byte(url),
		NewLimit:       newLimit,
		NewPublicLimit: newPublicLimit,
	}

	return m.client.UpdateAsset2(ctx, req)
}

// TransferAsset2 transfers an asset (TRC10 token) (v2)
func (m *Manager) TransferAsset2(ctx context.Context, ownerAddress string, toAddress string, assetName string, amount int64) (*api.TransactionExtention, error) {
	// Validate inputs
	if assetName == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	toAddr, err := utils.ValidateAddress(toAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}

	if ownerAddr.String() == toAddr.String() {
		return nil, fmt.Errorf("owner and to addresses cannot be the same")
	}

	req := &core.TransferAssetContract{
		AssetName:    []byte(assetName),
		OwnerAddress: ownerAddr.Bytes(),
		ToAddress:    toAddr.Bytes(),
		Amount:       amount,
	}

	return m.client.TransferAsset2(ctx, req)
}

// ParticipateAssetIssue2 participates in asset issue (v2)
func (m *Manager) ParticipateAssetIssue2(ctx context.Context, ownerAddress string, toAddress string, assetName string, amount int64) (*api.TransactionExtention, error) {
	// Validate inputs
	if assetName == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	ownerAddr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	toAddr, err := utils.ValidateAddress(toAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}

	req := &core.ParticipateAssetIssueContract{
		OwnerAddress: ownerAddr.Bytes(),
		ToAddress:    toAddr.Bytes(),
		AssetName:    []byte(assetName),
		Amount:       amount,
	}

	return m.client.ParticipateAssetIssue2(ctx, req)
}

// UnfreezeAsset2 unfreezes an asset (v2)
func (m *Manager) UnfreezeAsset2(ctx context.Context, ownerAddress string) (*api.TransactionExtention, error) {
	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.UnfreezeAssetContract{
		OwnerAddress: addr.Bytes(),
	}

	return m.client.UnfreezeAsset2(ctx, req)
}

// GetAssetIssueByAccount gets asset issues by account
func (m *Manager) GetAssetIssueByAccount(ctx context.Context, address string) (*api.AssetIssueList, error) {
	addr, err := utils.ValidateAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	req := &core.Account{
		Address: addr.Bytes(),
	}

	return m.client.GetAssetIssueByAccount(ctx, req)
}

// GetAssetIssueByName gets asset issue by name
func (m *Manager) GetAssetIssueByName(ctx context.Context, assetName string) (*core.AssetIssueContract, error) {
	if assetName == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}

	req := &api.BytesMessage{
		Value: []byte(assetName),
	}

	return m.client.GetAssetIssueByName(ctx, req)
}

// GetAssetIssueListByName gets asset issue list by name
func (m *Manager) GetAssetIssueListByName(ctx context.Context, assetName string) (*api.AssetIssueList, error) {
	if assetName == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}

	req := &api.BytesMessage{
		Value: []byte(assetName),
	}

	return m.client.GetAssetIssueListByName(ctx, req)
}

// GetAssetIssueById gets asset issue by ID
func (m *Manager) GetAssetIssueById(ctx context.Context, assetId []byte) (*core.AssetIssueContract, error) {
	if len(assetId) == 0 {
		return nil, fmt.Errorf("asset ID cannot be empty")
	}

	req := &api.BytesMessage{
		Value: assetId,
	}

	return m.client.GetAssetIssueById(ctx, req)
}

// GetAssetIssueList gets all asset issues
func (m *Manager) GetAssetIssueList(ctx context.Context) (*api.AssetIssueList, error) {
	req := &api.EmptyMessage{}
	return m.client.GetAssetIssueList(ctx, req)
}

// GetPaginatedAssetIssueList gets paginated asset issue list
func (m *Manager) GetPaginatedAssetIssueList(ctx context.Context, offset int64, limit int64) (*api.AssetIssueList, error) {
	if offset < 0 {
		return nil, fmt.Errorf("offset cannot be negative")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}
	if limit > 100 {
		return nil, fmt.Errorf("limit cannot exceed 100")
	}

	req := &api.PaginatedMessage{
		Offset: offset,
		Limit:  limit,
	}

	return m.client.GetPaginatedAssetIssueList(ctx, req)
}
