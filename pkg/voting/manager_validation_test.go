package voting_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/voting"
)

// ---------- Additional error-path coverage ----------

func TestVoteWitnessAccount2_NilWitnessAddress(t *testing.T) {
	// Covers the VoteCount > 0 but WitnessAddress == nil path inside the loop
	m := voting.NewManager(&client.Client{})
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	ctx := context.Background()

	_, err := m.VoteWitnessAccount2(ctx, owner, []voting.Vote{
		{WitnessAddress: nil, VoteCount: 1},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
	assert.Contains(t, err.Error(), "invalid witness address")
}

func TestVoteWitnessAccount2_NilOwner(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	ctx := context.Background()

	_, err := m.VoteWitnessAccount2(ctx, nil, []voting.Vote{
		{WitnessAddress: types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2"), VoteCount: 1},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestCreateWitness2_NilOwner(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	ctx := context.Background()

	_, err := m.CreateWitness2(ctx, nil, "https://example.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestUpdateWitness2_NilOwner(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	ctx := context.Background()

	_, err := m.UpdateWitness2(ctx, nil, "https://example.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestWithdrawBalance2_NilOwner(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	ctx := context.Background()

	_, err := m.WithdrawBalance2(ctx, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestUpdateBrokerage_NilOwner(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	ctx := context.Background()

	_, err := m.UpdateBrokerage(ctx, nil, 50)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestUpdateBrokerage_OutOfRange(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	ctx := context.Background()

	// Negative brokerage
	_, err := m.UpdateBrokerage(ctx, owner, -1)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidParameter)

	// Over 100
	_, err = m.UpdateBrokerage(ctx, owner, 101)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidParameter)
}

func TestGetRewardInfo_NilAddress(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	ctx := context.Background()

	_, err := m.GetRewardInfo(ctx, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestGetBrokerageInfo_NilAddress(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	ctx := context.Background()

	_, err := m.GetBrokerageInfo(ctx, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}
