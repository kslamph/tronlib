package account_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kslamph/tronlib/internal/testutil"
	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/types"
)

// ---------- bufconn infrastructure ----------

// fakeAccountServer implements only the RPCs exercised in these tests.
type fakeAccountServer struct {
	api.UnimplementedWalletServer

	GetAccountFunc         func(ctx context.Context, in *core.Account) (*core.Account, error)
	GetAccountNetFunc      func(ctx context.Context, in *core.Account) (*api.AccountNetMessage, error)
	GetAccountResourceFunc func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error)
	CreateTransaction2Func func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error)
}

func (s *fakeAccountServer) GetAccount(ctx context.Context, in *core.Account) (*core.Account, error) {
	if s.GetAccountFunc != nil {
		return s.GetAccountFunc(ctx, in)
	}
	return &core.Account{
		Address: in.Address,
		Balance: 999_000_000,
	}, nil
}

func (s *fakeAccountServer) GetAccountNet(ctx context.Context, in *core.Account) (*api.AccountNetMessage, error) {
	if s.GetAccountNetFunc != nil {
		return s.GetAccountNetFunc(ctx, in)
	}
	return &api.AccountNetMessage{
		NetUsed:      1000,
		NetLimit:     5000,
		AssetNetUsed: map[string]int64{"BANDWIDTH": 200},
	}, nil
}

func (s *fakeAccountServer) GetAccountResource(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
	if s.GetAccountResourceFunc != nil {
		return s.GetAccountResourceFunc(ctx, in)
	}
	return &api.AccountResourceMessage{
		EnergyUsed:  500,
		EnergyLimit: 10000,

		TotalEnergyLimit:  100000,
		TotalEnergyWeight: 1,
		StorageUsed:       0,
		StorageLimit:      10000,
	}, nil
}

func (s *fakeAccountServer) CreateTransaction2(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
	if s.CreateTransaction2Func != nil {
		return s.CreateTransaction2Func(ctx, in)
	}
	return &api.TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{
					{
						Type: core.Transaction_Contract_TransferContract,
					},
				},
			},
			Signature: [][]byte{},
		},
		Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
	}, nil
}

// newAccountManager wires an AccountManager to an in-process fake wallet
// server. No real client or network is involved: the manager only needs a
// connection provider, which testutil.MockConnProvider supplies.
func newAccountManager(t *testing.T, impl api.WalletServer) *account.AccountManager {
	t.Helper()
	lis := testutil.NewBufconnServer(t, impl)
	conn := testutil.DialBufconn(t, lis)
	return account.NewManager(testutil.NewMockConnProvider(conn))
}

// ---------- Tests: happy paths ----------

func TestGetAccount_HappyPath(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{})

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	acct, err := m.GetAccount(context.Background(), addr)
	require.NoError(t, err)
	assert.NotNil(t, acct)
	assert.Equal(t, int64(999_000_000), acct.GetBalance())
}

func TestGetAccountNet_HappyPath(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{})

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	net, err := m.GetAccountNet(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), net.GetNetUsed())
	assert.Equal(t, int64(5000), net.GetNetLimit())
}

func TestGetAccountResource_HappyPath(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{})

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	res, err := m.GetAccountResource(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, int64(500), res.GetEnergyUsed())
	assert.Equal(t, int64(10000), res.GetEnergyLimit())
}

func TestGetBalance_HappyPath(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{})

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	bal, err := m.GetBalance(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, int64(999_000_000), bal)
}

func TestTransferTRX_HappyPath(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{})

	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	txExt, err := m.TransferTRX(context.Background(), from, to, 1_000_000)
	require.NoError(t, err)
	assert.NotNil(t, txExt)
	assert.True(t, txExt.GetResult().GetResult())
}

// ---------- Tests: error paths ----------

// ---------- Tests: nil/empty address validation (no server needed) ----------

func TestManager_InvalidAddresses(t *testing.T) {
	ctx := context.Background()
	validAddr := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	validFrom := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")

	tests := []struct {
		name    string
		call    func(m *account.AccountManager) error
		wantErr error
	}{
		{
			name:    "GetAccount_NilAddress",
			call:    func(m *account.AccountManager) error { _, err := m.GetAccount(ctx, nil); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "GetAccount_EmptyAddress",
			call:    func(m *account.AccountManager) error { _, err := m.GetAccount(ctx, &types.Address{}); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "GetAccountNet_NilAddress",
			call:    func(m *account.AccountManager) error { _, err := m.GetAccountNet(ctx, nil); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "GetAccountResource_NilAddress",
			call:    func(m *account.AccountManager) error { _, err := m.GetAccountResource(ctx, nil); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "GetBalance_NilAddress",
			call:    func(m *account.AccountManager) error { _, err := m.GetBalance(ctx, nil); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "TransferTRX_NilFrom",
			call:    func(m *account.AccountManager) error { _, err := m.TransferTRX(ctx, nil, validAddr, 1); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name:    "TransferTRX_NilTo",
			call:    func(m *account.AccountManager) error { _, err := m.TransferTRX(ctx, validFrom, nil, 1); return err },
			wantErr: types.ErrInvalidAddress,
		},
		{
			name: "TransferTRX_ZeroAmount",
			call: func(m *account.AccountManager) error {
				_, err := m.TransferTRX(ctx, validFrom, validAddr, 0)
				return err
			},
			wantErr: types.ErrInvalidAmount,
		},
		{
			name: "TransferTRX_NegativeAmount",
			call: func(m *account.AccountManager) error {
				_, err := m.TransferTRX(ctx, validFrom, validAddr, -100)
				return err
			},
			wantErr: types.ErrInvalidAmount,
		},
		{
			name: "TransferTRX_SameAddress",
			call: func(m *account.AccountManager) error {
				_, err := m.TransferTRX(ctx, validFrom, validFrom, 1)
				return err
			},
			wantErr: types.ErrInvalidParameter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := account.NewManager(&testutil.MockConnProvider{})
			err := tt.call(m)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// ---------- Tests: RPC errors propagate ----------

func TestGetAccount_RPCError(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{
		GetAccountFunc: func(ctx context.Context, in *core.Account) (*core.Account, error) {
			return nil, fmt.Errorf("node unavailable")
		},
	})

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.GetAccount(context.Background(), addr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node unavailable")
}

func TestGetAccountNet_RPCError(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{
		GetAccountNetFunc: func(ctx context.Context, in *core.Account) (*api.AccountNetMessage, error) {
			return nil, fmt.Errorf("unavailable")
		},
	})

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.GetAccountNet(context.Background(), addr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestGetAccountResource_RPCError(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{
		GetAccountResourceFunc: func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
			return nil, fmt.Errorf("unavailable")
		},
	})

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.GetAccountResource(context.Background(), addr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestGetBalance_RPCError(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{
		GetAccountFunc: func(ctx context.Context, in *core.Account) (*core.Account, error) {
			return nil, fmt.Errorf("unavailable")
		},
	})

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.GetBalance(context.Background(), addr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get account balance")
}

func TestTransferTRX_RPCError(t *testing.T) {
	m := newAccountManager(t, &fakeAccountServer{
		CreateTransaction2Func: func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("unavailable")
		},
	})

	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := m.TransferTRX(context.Background(), from, to, 1_000_000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}
