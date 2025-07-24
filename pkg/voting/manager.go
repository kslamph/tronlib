// Package voting provides high-level voting and witness management functionality
package voting

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/utils"
)

// Manager provides high-level voting and witness operations
type Manager struct {
	client *client.Client
}

// NewManager creates a new voting manager
func NewManager(client *client.Client) *Manager {
	return &Manager{
		client: client,
	}
}

// Vote represents a single vote for a witness
type Vote struct {
	WitnessAddress string
	VoteCount      int64
}

// VoteWitnessAccount2 votes for witnesses (v2)
func (m *Manager) VoteWitnessAccount2(ctx context.Context, ownerAddress string, votes []Vote) (*api.TransactionExtention, error) {
	// Validate inputs
	if len(votes) == 0 {
		return nil, fmt.Errorf("votes list cannot be empty")
	}

	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	// Convert votes to protocol format
	var protoVotes []*core.VoteWitnessContract_Vote
	for i, vote := range votes {
		if vote.VoteCount <= 0 {
			return nil, fmt.Errorf("vote count must be positive for vote %d", i)
		}

		witnessAddr, err := utils.ValidateAddress(vote.WitnessAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid witness address for vote %d: %w", i, err)
		}

		protoVotes = append(protoVotes, &core.VoteWitnessContract_Vote{
			VoteAddress: witnessAddr.Bytes(),
			VoteCount:   vote.VoteCount,
		})
	}

	req := &core.VoteWitnessContract{
		OwnerAddress: addr.Bytes(),
		Votes:        protoVotes,
	}

	return lowlevel.VoteWitnessAccount2(m.client, ctx, req)
}

// WithdrawBalance2 withdraws balance (claim rewards) (v2)
func (m *Manager) WithdrawBalance2(ctx context.Context, ownerAddress string) (*api.TransactionExtention, error) {
	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.WithdrawBalanceContract{
		OwnerAddress: addr.Bytes(),
	}

	return lowlevel.WithdrawBalance2(m.client, ctx, req)
}

// CreateWitness2 creates a witness (v2)
func (m *Manager) CreateWitness2(ctx context.Context, ownerAddress string, url string) (*api.TransactionExtention, error) {
	if url == "" {
		return nil, fmt.Errorf("witness URL cannot be empty")
	}

	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.WitnessCreateContract{
		OwnerAddress: addr.Bytes(),
		Url:          []byte(url),
	}

	return lowlevel.CreateWitness2(m.client, ctx, req)
}

// UpdateWitness2 updates witness information (v2)
func (m *Manager) UpdateWitness2(ctx context.Context, ownerAddress string, updateUrl string) (*api.TransactionExtention, error) {
	if updateUrl == "" {
		return nil, fmt.Errorf("update URL cannot be empty")
	}

	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.WitnessUpdateContract{
		OwnerAddress: addr.Bytes(),
		UpdateUrl:    []byte(updateUrl),
	}

	return lowlevel.UpdateWitness2(m.client, ctx, req)
}

// ListWitnesses gets list of witnesses
func (m *Manager) ListWitnesses(ctx context.Context) (*api.WitnessList, error) {
	req := &api.EmptyMessage{}
	return lowlevel.ListWitnesses(m.client, ctx, req)
}

// GetRewardInfo gets reward information for an address
func (m *Manager) GetRewardInfo(ctx context.Context, address string) (*api.NumberMessage, error) {
	addr, err := utils.ValidateAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	req := &api.BytesMessage{
		Value: addr.Bytes(),
	}

	return lowlevel.GetRewardInfo(m.client, ctx, req)
}

// GetBrokerageInfo gets brokerage information for an address
func (m *Manager) GetBrokerageInfo(ctx context.Context, address string) (*api.NumberMessage, error) {
	addr, err := utils.ValidateAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	req := &api.BytesMessage{
		Value: addr.Bytes(),
	}

	return lowlevel.GetBrokerageInfo(m.client, ctx, req)
}

// UpdateBrokerage updates brokerage percentage
func (m *Manager) UpdateBrokerage(ctx context.Context, ownerAddress string, brokerage int32) (*api.TransactionExtention, error) {
	if brokerage < 0 || brokerage > 100 {
		return nil, fmt.Errorf("brokerage must be between 0 and 100")
	}

	addr, err := utils.ValidateAddress(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}

	req := &core.UpdateBrokerageContract{
		OwnerAddress: addr.Bytes(),
		Brokerage:    brokerage,
	}

	return lowlevel.UpdateBrokerage(m.client, ctx, req)
}