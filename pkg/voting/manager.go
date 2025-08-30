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

// Package voting provides high-level voting and witness management functionality
package voting

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
)

// Manager provides high-level voting and witness operations
type Manager struct {
	conn lowlevel.ConnProvider
}

// VotingManager is an explicit alias of Manager for discoverability and future clarity.
type VotingManager = Manager

// NewManager creates a new voting manager
func NewManager(conn lowlevel.ConnProvider) *Manager {
	return &Manager{conn: conn}
}

// Vote represents a single vote for a witness
// Vote represents a single vote for a witness
type Vote struct {
	WitnessAddress *types.Address
	VoteCount      int64
}

// VoteWitnessAccount2 votes for witnesses (v2)
func (m *Manager) VoteWitnessAccount2(ctx context.Context, ownerAddress *types.Address, votes []Vote) (*api.TransactionExtention, error) {
	// Validate inputs
	if len(votes) == 0 {
		return nil, fmt.Errorf("%w: votes list cannot be empty", types.ErrInvalidParameter)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	// Convert votes to protocol format
	var protoVotes []*core.VoteWitnessContract_Vote
	for i, vote := range votes {
		if vote.VoteCount <= 0 {
			return nil, fmt.Errorf("%w: vote count must be positive for vote %d", types.ErrInvalidParameter, i)
		}
		if vote.WitnessAddress == nil {
			return nil, fmt.Errorf("%w: invalid witness address for vote %d: nil", types.ErrInvalidAddress, i)
		}

		protoVotes = append(protoVotes, &core.VoteWitnessContract_Vote{VoteAddress: vote.WitnessAddress.Bytes(), VoteCount: vote.VoteCount})
	}

	req := &core.VoteWitnessContract{OwnerAddress: ownerAddress.Bytes(), Votes: protoVotes}
	return lowlevel.TxCall(m.conn, ctx, "vote witness account2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.VoteWitnessAccount2(ctx, req)
	})
}

// WithdrawBalance2 withdraws balance (claim rewards) (v2)
func (m *Manager) WithdrawBalance2(ctx context.Context, ownerAddress *types.Address) (*api.TransactionExtention, error) {
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.WithdrawBalanceContract{OwnerAddress: ownerAddress.Bytes()}
	return lowlevel.TxCall(m.conn, ctx, "withdraw balance2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.WithdrawBalance2(ctx, req)
	})
}

// CreateWitness2 creates a witness (v2)
func (m *Manager) CreateWitness2(ctx context.Context, ownerAddress *types.Address, url string) (*api.TransactionExtention, error) {
	if url == "" {
		return nil, fmt.Errorf("%w: witness URL cannot be empty", types.ErrInvalidParameter)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.WitnessCreateContract{OwnerAddress: ownerAddress.Bytes(), Url: []byte(url)}
	return lowlevel.TxCall(m.conn, ctx, "create witness2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.CreateWitness2(ctx, req)
	})
}

// UpdateWitness2 updates witness information (v2)
func (m *Manager) UpdateWitness2(ctx context.Context, ownerAddress *types.Address, updateUrl string) (*api.TransactionExtention, error) {
	if updateUrl == "" {
		return nil, fmt.Errorf("%w: update URL cannot be empty", types.ErrInvalidParameter)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.WitnessUpdateContract{OwnerAddress: ownerAddress.Bytes(), UpdateUrl: []byte(updateUrl)}
	return lowlevel.TxCall(m.conn, ctx, "update witness2", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateWitness2(ctx, req)
	})
}

// ListWitnesses gets list of witnesses
func (m *Manager) ListWitnesses(ctx context.Context) (*api.WitnessList, error) {
	req := &api.EmptyMessage{}
	return lowlevel.Call(m.conn, ctx, "list witnesses", func(cl api.WalletClient, ctx context.Context) (*api.WitnessList, error) {
		return cl.ListWitnesses(ctx, req)
	})
}

// GetRewardInfo gets reward information for an address
func (m *Manager) GetRewardInfo(ctx context.Context, address *types.Address) (*api.NumberMessage, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: invalid address: nil", types.ErrInvalidAddress)
	}

	req := &api.BytesMessage{Value: address.Bytes()}
	return lowlevel.Call(m.conn, ctx, "get reward info", func(cl api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return cl.GetRewardInfo(ctx, req)
	})
}

// GetBrokerageInfo gets brokerage information for an address
func (m *Manager) GetBrokerageInfo(ctx context.Context, address *types.Address) (*api.NumberMessage, error) {
	if address == nil {
		return nil, fmt.Errorf("%w: invalid address: nil", types.ErrInvalidAddress)
	}

	req := &api.BytesMessage{Value: address.Bytes()}
	return lowlevel.Call(m.conn, ctx, "get brokerage info", func(cl api.WalletClient, ctx context.Context) (*api.NumberMessage, error) {
		return cl.GetBrokerageInfo(ctx, req)
	})
}

// UpdateBrokerage updates brokerage percentage
func (m *Manager) UpdateBrokerage(ctx context.Context, ownerAddress *types.Address, brokerage int32) (*api.TransactionExtention, error) {
	if brokerage < 0 || brokerage > 100 {
		return nil, fmt.Errorf("%w: brokerage must be between 0 and 100", types.ErrInvalidParameter)
	}
	if ownerAddress == nil {
		return nil, fmt.Errorf("%w: invalid owner address: nil", types.ErrInvalidAddress)
	}

	req := &core.UpdateBrokerageContract{OwnerAddress: ownerAddress.Bytes(), Brokerage: brokerage}
	return lowlevel.TxCall(m.conn, ctx, "update brokerage", func(cl api.WalletClient, ctx context.Context) (*api.TransactionExtention, error) {
		return cl.UpdateBrokerage(ctx, req)
	})
}
