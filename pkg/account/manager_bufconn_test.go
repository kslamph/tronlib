package account_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// ---------- bufconn infrastructure ----------

const acctBufSize = 1024 * 1024

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

func newAccountBufConn(t *testing.T, impl api.WalletServer) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(acctBufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { _ = lis.Close(); srv.Stop() })
	return lis
}

func newAccountClient(t *testing.T, lis *bufconn.Listener) *client.Client {
	t.Helper()
	c, err := client.NewClientWithDialer(
		"passthrough:///bufnet",
		func(ctx context.Context, s string) (net.Conn, error) { return lis.DialContext(ctx) },
		client.WithTimeout(2*time.Second),
		client.WithPool(1, 1),
	)
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
	return c
}

// ---------- Tests: happy paths ----------

func TestGetAccount_HappyPath(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	acct, err := m.GetAccount(context.Background(), addr)
	require.NoError(t, err)
	assert.NotNil(t, acct)
	assert.Equal(t, int64(999_000_000), acct.GetBalance())
}

func TestGetAccountNet_HappyPath(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	net, err := m.GetAccountNet(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), net.GetNetUsed())
	assert.Equal(t, int64(5000), net.GetNetLimit())
}

func TestGetAccountResource_HappyPath(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	res, err := m.GetAccountResource(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, int64(500), res.GetEnergyUsed())
	assert.Equal(t, int64(10000), res.GetEnergyLimit())
}

func TestGetBalance_HappyPath(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	bal, err := m.GetBalance(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, int64(999_000_000), bal)
}

func TestTransferTRX_HappyPath(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	txExt, err := m.TransferTRX(context.Background(), from, to, 1_000_000)
	require.NoError(t, err)
	assert.NotNil(t, txExt)
	assert.True(t, txExt.GetResult().GetResult())
}

// ---------- Tests: error paths ----------

func TestGetAccount_NilAddress(t *testing.T) {
	m := account.NewManager(&client.Client{})
	_, err := m.GetAccount(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestGetAccount_EmptyAddress(t *testing.T) {
	m := account.NewManager(&client.Client{})
	_, err := m.GetAccount(context.Background(), &types.Address{})
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestGetAccountNet_NilAddress(t *testing.T) {
	m := account.NewManager(&client.Client{})
	_, err := m.GetAccountNet(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestGetAccountResource_NilAddress(t *testing.T) {
	m := account.NewManager(&client.Client{})
	_, err := m.GetAccountResource(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestGetBalance_NilAddress(t *testing.T) {
	m := account.NewManager(&client.Client{})
	_, err := m.GetBalance(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestTransferTRX_NilFrom(t *testing.T) {
	m := account.NewManager(&client.Client{})
	to := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := m.TransferTRX(context.Background(), nil, to, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestTransferTRX_NilTo(t *testing.T) {
	m := account.NewManager(&client.Client{})
	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.TransferTRX(context.Background(), from, nil, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAddress)
}

func TestTransferTRX_ZeroAmount(t *testing.T) {
	m := account.NewManager(&client.Client{})
	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := m.TransferTRX(context.Background(), from, to, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAmount)
}

func TestTransferTRX_NegativeAmount(t *testing.T) {
	m := account.NewManager(&client.Client{})
	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := m.TransferTRX(context.Background(), from, to, -100)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidAmount)
}

func TestTransferTRX_SameAddress(t *testing.T) {
	m := account.NewManager(&client.Client{})
	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.TransferTRX(context.Background(), addr, addr, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidParameter)
}

// ---------- Tests: RPC errors propagate ----------

func TestGetAccount_RPCError(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{
		GetAccountFunc: func(ctx context.Context, in *core.Account) (*core.Account, error) {
			return nil, fmt.Errorf("node unavailable")
		},
	})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.GetAccount(context.Background(), addr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node unavailable")
}

func TestGetAccountNet_RPCError(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{
		GetAccountNetFunc: func(ctx context.Context, in *core.Account) (*api.AccountNetMessage, error) {
			return nil, fmt.Errorf("unavailable")
		},
	})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.GetAccountNet(context.Background(), addr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestGetAccountResource_RPCError(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{
		GetAccountResourceFunc: func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
			return nil, fmt.Errorf("unavailable")
		},
	})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.GetAccountResource(context.Background(), addr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestGetBalance_RPCError(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{
		GetAccountFunc: func(ctx context.Context, in *core.Account) (*core.Account, error) {
			return nil, fmt.Errorf("unavailable")
		},
	})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	addr := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	_, err := m.GetBalance(context.Background(), addr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get account balance")
}

func TestTransferTRX_RPCError(t *testing.T) {
	lis := newAccountBufConn(t, &fakeAccountServer{
		CreateTransaction2Func: func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("unavailable")
		},
	})
	c := newAccountClient(t, lis)
	m := account.NewManager(c)

	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	_, err := m.TransferTRX(context.Background(), from, to, 1_000_000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}
