// Package trc10 provides high-level TRC10 token functionality
package trc10

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
)

// TRC10Manager provides high-level TRC10 token operations
type TRC10Manager struct {
	conn lowlevel.ConnProvider
}

// NewManager creates a new TRC10 manager
func NewManager(conn lowlevel.ConnProvider) *TRC10Manager { return &TRC10Manager{conn: conn} }

// CreateAssetIssue2 creates an asset issue (TRC10 token) (v2)
// CreateAssetIssue2 creates an asset issue (TRC10 token) (v2)
func (m *TRC10Manager) CreateAssetIssue2(ctx context.Context, ownerAddress *types.Address, name string, abbr string, totalSupply int64, trxNum int32, icoNum int32, startTime int64, endTime int64, description string, url string, freeAssetNetLimit int64, publicFreeAssetNetLimit int64, frozenSupply []FrozenSupply) (*api.TransactionExtention, error) {
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
	if ownerAddress == nil {
		return nil, fmt.Errorf("invalid owner address: nil")
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

		protoFrozenSupply = append(protoFrozenSupply, &core.AssetIssueContract_FrozenSupply{FrozenAmount: fs.FrozenAmount, FrozenDays: fs.FrozenDays})
	}

	req := &core.AssetIssueContract{
		OwnerAddress:            ownerAddress.Bytes(),
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

	return lowlevel.TxCall(m.conn, ctx, "create asset issue2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.CreateAssetIssue2(ctx, req)
	})
}

// FrozenSupply represents frozen supply for asset creation
type FrozenSupply struct {
	FrozenAmount int64
	FrozenDays   int64
}

// UpdateAsset2 updates an asset (v2)
func (m *TRC10Manager) UpdateAsset2(ctx context.Context, ownerAddress *types.Address, description string, url string, newLimit int64, newPublicLimit int64) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("invalid owner address: nil")
	}

	req := &core.UpdateAssetContract{OwnerAddress: ownerAddress.Bytes(), Description: []byte(description), Url: []byte(url), NewLimit: newLimit, NewPublicLimit: newPublicLimit}
	return lowlevel.TxCall(m.conn, ctx, "update asset2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateAsset2(ctx, req)
	})
}

// TransferAsset2 transfers an asset (TRC10 token) (v2)
func (m *TRC10Manager) TransferAsset2(ctx context.Context, ownerAddress, toAddress *types.Address, assetName string, amount int64) (*api.TransactionExtention, error) {
	// Validate inputs
	if assetName == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("invalid owner address: nil")
	}
	if toAddress == nil {
		return nil, fmt.Errorf("invalid to address: nil")
	}

	if ownerAddress.String() == toAddress.String() {
		return nil, fmt.Errorf("owner and to addresses cannot be the same")
	}

	req := &core.TransferAssetContract{AssetName: []byte(assetName), OwnerAddress: ownerAddress.Bytes(), ToAddress: toAddress.Bytes(), Amount: amount}
	return lowlevel.TxCall(m.conn, ctx, "transfer asset2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.TransferAsset2(ctx, req)
	})
}

// ParticipateAssetIssue2 participates in asset issue (v2)
func (m *TRC10Manager) ParticipateAssetIssue2(ctx context.Context, ownerAddress, toAddress *types.Address, assetName string, amount int64) (*api.TransactionExtention, error) {
	// Validate inputs
	if assetName == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("invalid owner address: nil")
	}
	if toAddress == nil {
		return nil, fmt.Errorf("invalid to address: nil")
	}

	req := &core.ParticipateAssetIssueContract{OwnerAddress: ownerAddress.Bytes(), ToAddress: toAddress.Bytes(), AssetName: []byte(assetName), Amount: amount}
	return lowlevel.TxCall(m.conn, ctx, "participate asset issue2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.ParticipateAssetIssue2(ctx, req)
	})
}

// UnfreezeAsset2 unfreezes an asset (v2)
func (m *TRC10Manager) UnfreezeAsset2(ctx context.Context, ownerAddress *types.Address) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("invalid owner address: nil")
	}

	req := &core.UnfreezeAssetContract{OwnerAddress: ownerAddress.Bytes()}
	return lowlevel.TxCall(m.conn, ctx, "unfreeze asset2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UnfreezeAsset2(ctx, req)
	})
}

// GetAssetIssueByAccount gets asset issues by account
func (m *TRC10Manager) GetAssetIssueByAccount(ctx context.Context, address *types.Address) (*api.AssetIssueList, error) {
	if address == nil {
		return nil, fmt.Errorf("invalid address: nil")
	}

	req := &core.Account{Address: address.Bytes()}
	return lowlevel.Call(m.conn, ctx, "get asset issue by account", func(cl api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return cl.GetAssetIssueByAccount(ctx, req)
	})
}

// GetAssetIssueByName gets asset issue by name
func (m *TRC10Manager) GetAssetIssueByName(ctx context.Context, assetName string) (*core.AssetIssueContract, error) {
	if assetName == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}

	req := &api.BytesMessage{Value: []byte(assetName)}
	return lowlevel.Call(m.conn, ctx, "get asset issue by name", func(cl api.WalletClient, ctx context.Context) (*core.AssetIssueContract, error) {
		return cl.GetAssetIssueByName(ctx, req)
	})
}

// GetAssetIssueListByName gets asset issue list by name
func (m *TRC10Manager) GetAssetIssueListByName(ctx context.Context, assetName string) (*api.AssetIssueList, error) {
	if assetName == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}

	req := &api.BytesMessage{Value: []byte(assetName)}
	return lowlevel.Call(m.conn, ctx, "get asset issue list by name", func(cl api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return cl.GetAssetIssueListByName(ctx, req)
	})
}

// GetAssetIssueById gets asset issue by ID
func (m *TRC10Manager) GetAssetIssueById(ctx context.Context, assetId []byte) (*core.AssetIssueContract, error) {
	if len(assetId) == 0 {
		return nil, fmt.Errorf("asset ID cannot be empty")
	}

	req := &api.BytesMessage{Value: assetId}
	return lowlevel.Call(m.conn, ctx, "get asset issue by id", func(cl api.WalletClient, ctx context.Context) (*core.AssetIssueContract, error) {
		return cl.GetAssetIssueById(ctx, req)
	})
}

// GetAssetIssueList gets all asset issues
func (m *TRC10Manager) GetAssetIssueList(ctx context.Context) (*api.AssetIssueList, error) {
	req := &api.EmptyMessage{}
	return lowlevel.Call(m.conn, ctx, "get asset issue list", func(cl api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return cl.GetAssetIssueList(ctx, req)
	})
}

// GetPaginatedAssetIssueList gets paginated asset issue list
func (m *TRC10Manager) GetPaginatedAssetIssueList(ctx context.Context, offset int64, limit int64) (*api.AssetIssueList, error) {
	if offset < 0 {
		return nil, fmt.Errorf("offset cannot be negative")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}
	if limit > 100 {
		return nil, fmt.Errorf("limit cannot exceed 100")
	}

	req := &api.PaginatedMessage{Offset: offset, Limit: limit}
	return lowlevel.Call(m.conn, ctx, "get paginated asset issue list", func(cl api.WalletClient, ctx context.Context) (*api.AssetIssueList, error) {
		return cl.GetPaginatedAssetIssueList(ctx, req)
	})
}
