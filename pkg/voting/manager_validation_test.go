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

// TestManager_NilAddress verifies that every manager method that takes an owner
// or address rejects nil/empty addresses before any RPC is attempted.
func TestManager_NilAddress(t *testing.T) {
	ctx := context.Background()
	validAddr := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")

	tests := []struct {
		name    string
		call    func(m *voting.Manager) error
		wantErr error
	}{
		{
			name: "VoteWitnessAccount2_NilWitnessAddress",
			call: func(m *voting.Manager) error {
				_, err := m.VoteWitnessAccount2(ctx, validAddr, []voting.Vote{
					{WitnessAddress: nil, VoteCount: 1},
				})
				return err
			},
			wantErr: types.ErrInvalidAddress,
		},
		{
			name: "VoteWitnessAccount2_NilOwner",
			call: func(m *voting.Manager) error {
				_, err := m.VoteWitnessAccount2(ctx, nil, []voting.Vote{
					{WitnessAddress: validAddr, VoteCount: 1},
				})
				return err
			},
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "CreateWitness2_NilOwner",
			call:    func(m *voting.Manager) error { _, err := m.CreateWitness2(ctx, nil, "https://example.com"); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "UpdateWitness2_NilOwner",
			call:    func(m *voting.Manager) error { _, err := m.UpdateWitness2(ctx, nil, "https://example.com"); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "WithdrawBalance2_NilOwner",
			call:    func(m *voting.Manager) error { _, err := m.WithdrawBalance2(ctx, nil); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "UpdateBrokerage_NilOwner",
			call:    func(m *voting.Manager) error { _, err := m.UpdateBrokerage(ctx, nil, 50); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "GetRewardInfo_NilAddress",
			call:    func(m *voting.Manager) error { _, err := m.GetRewardInfo(ctx, nil); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "GetBrokerageInfo_NilAddress",
			call:    func(m *voting.Manager) error { _, err := m.GetBrokerageInfo(ctx, nil); return err },
			wantErr: types.ErrInvalidAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := voting.NewManager(&client.Client{})
			err := tt.call(m)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestUpdateBrokerage_OutOfRange verifies the brokerage percentage bounds.
func TestUpdateBrokerage_OutOfRange(t *testing.T) {
	m := voting.NewManager(&client.Client{})
	ctx := context.Background()
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")

	for _, brokerage := range []int32{-1, 101} {
		_, err := m.UpdateBrokerage(ctx, owner, brokerage)
		require.Error(t, err)
		assert.ErrorIs(t, err, types.ErrInvalidParameter)
	}
}
