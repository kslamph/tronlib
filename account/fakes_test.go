package account

// Bufconn fake for the account package tests. The rpc and tx fakes live in
// their own packages' test files and are not importable, so this is a local
// mirror covering exactly the RPCs the account handles touch.

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

var (
	testFrom = mustAddr(0x11)
	testTo   = mustAddr(0x22)
	testHot  = mustAddr(0x33)
)

func mustAddr(fill byte) tron.Address {
	b := append([]byte{0x41}, make([]byte, 20)...)
	for i := range b[1:] {
		b[i+1] = fill
	}
	a, err := tron.AddressFromBytes(b)
	if err != nil {
		panic(err)
	}
	return a
}

// fakeWalletServer implements api.WalletServer with canned, overridable
// answers for the RPCs the account package uses.
type fakeWalletServer struct {
	api.UnimplementedWalletServer

	account     *core.Account
	resources   *api.AccountResourceMessage
	canDelegate int64
	unfreezeCnt int64
	canWithdraw int64
	signWeight  *api.TransactionSignWeight
	approved    *api.TransactionApprovedList
	nextMaint   int64
	delegated   *api.DelegatedResourceList
	delegIndex  *core.DelegatedResourceAccountIndex

	gotCreate  *core.TransferContract
	gotAsset   *core.TransferAssetContract
	gotDeploy  *core.CreateSmartContract
	gotFreeze  *core.FreezeBalanceV2Contract
	gotVote    *core.VoteWitnessContract
	gotPerm    *core.AccountPermissionUpdateContract
	gotTrigger *core.TriggerSmartContract
	// Staking lifecycle requests: each Unfreeze/Delegate/Withdraw handler
	// records what the builder sent, so the tests can assert the forwarding
	// rather than just that the call succeeded.
	gotUnfreeze   *core.UnfreezeBalanceV2Contract
	gotWithdraw   *core.WithdrawExpireUnfreezeContract
	gotCancel     *core.CancelAllUnfreezeV2Contract
	gotDelegate   *core.DelegateResourceContract
	gotUndelegate *core.UnDelegateResourceContract

	// rpcErrors makes one named handler fail, so the error-propagation branch
	// of each account method has a reachable trigger. Keyed by handler name.
	rpcErrors map[string]error
}

// failWith makes the named handler return err.
func (f *fakeWalletServer) failWith(method string, err error) {
	if f.rpcErrors == nil {
		f.rpcErrors = map[string]error{}
	}
	f.rpcErrors[method] = err
}

// injected returns the error queued for method, if any. Handlers call it first
// so a test can make exactly one RPC fail and leave the rest working.
func (f *fakeWalletServer) injected(method string) error {
	return f.rpcErrors[method]
}

func (f *fakeWalletServer) GetAccount(_ context.Context, in *core.Account) (*core.Account, error) {
	if err := f.injected("GetAccount"); err != nil {
		return nil, err
	}
	if f.account != nil {
		return f.account, nil
	}
	// Default: the account does not exist yet — the node answers with an
	// address-less account shell, which is the "not created" shape.
	return &core.Account{}, nil
}

func (f *fakeWalletServer) GetAccountResource(_ context.Context, _ *core.Account) (*api.AccountResourceMessage, error) {
	if err := f.injected("GetAccountResource"); err != nil {
		return nil, err
	}
	if f.resources != nil {
		return f.resources, nil
	}
	return &api.AccountResourceMessage{}, nil
}

func (f *fakeWalletServer) GetCanDelegatedMaxSize(_ context.Context, _ *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	if err := f.injected("GetCanDelegatedMaxSize"); err != nil {
		return nil, err
	}
	return &api.CanDelegatedMaxSizeResponseMessage{MaxSize: f.canDelegate}, nil
}

func (f *fakeWalletServer) GetAvailableUnfreezeCount(_ context.Context, _ *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	if err := f.injected("GetAvailableUnfreezeCount"); err != nil {
		return nil, err
	}
	return &api.GetAvailableUnfreezeCountResponseMessage{Count: f.unfreezeCnt}, nil
}

func (f *fakeWalletServer) GetCanWithdrawUnfreezeAmount(_ context.Context, _ *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	if err := f.injected("GetCanWithdrawUnfreezeAmount"); err != nil {
		return nil, err
	}
	return &api.CanWithdrawUnfreezeAmountResponseMessage{Amount: f.canWithdraw}, nil
}

func (f *fakeWalletServer) GetDelegatedResourceV2(_ context.Context, _ *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	if err := f.injected("GetDelegatedResourceV2"); err != nil {
		return nil, err
	}
	if f.delegated != nil {
		return f.delegated, nil
	}
	return &api.DelegatedResourceList{}, nil
}

func (f *fakeWalletServer) GetDelegatedResourceAccountIndexV2(_ context.Context, _ *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
	if err := f.injected("GetDelegatedResourceAccountIndexV2"); err != nil {
		return nil, err
	}
	if f.delegIndex != nil {
		return f.delegIndex, nil
	}
	return &core.DelegatedResourceAccountIndex{}, nil
}

func (f *fakeWalletServer) GetRewardInfo(_ context.Context, _ *api.BytesMessage) (*api.NumberMessage, error) {
	if err := f.injected("GetRewardInfo"); err != nil {
		return nil, err
	}
	return &api.NumberMessage{Num: 42}, nil
}

func (f *fakeWalletServer) GetNextMaintenanceTime(_ context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
	if err := f.injected("GetNextMaintenanceTime"); err != nil {
		return nil, err
	}
	return &api.NumberMessage{Num: f.nextMaint}, nil
}

func (f *fakeWalletServer) GetTransactionSignWeight(_ context.Context, _ *core.Transaction) (*api.TransactionSignWeight, error) {
	if err := f.injected("GetTransactionSignWeight"); err != nil {
		return nil, err
	}
	if f.signWeight != nil {
		return f.signWeight, nil
	}
	return &api.TransactionSignWeight{}, nil
}

func (f *fakeWalletServer) GetTransactionApprovedList(_ context.Context, _ *core.Transaction) (*api.TransactionApprovedList, error) {
	if err := f.injected("GetTransactionApprovedList"); err != nil {
		return nil, err
	}
	if f.approved != nil {
		return f.approved, nil
	}
	return &api.TransactionApprovedList{}, nil
}

func (f *fakeWalletServer) AccountPermissionUpdate(_ context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	f.gotPerm = in
	return ext(core.Transaction_Contract_AccountPermissionUpdateContract, in), nil
}

func (f *fakeWalletServer) VoteWitnessAccount2(_ context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	f.gotVote = in
	return ext(core.Transaction_Contract_VoteWitnessContract, in), nil
}

func (f *fakeWalletServer) WithdrawBalance2(_ context.Context, in *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	return ext(core.Transaction_Contract_WithdrawBalanceContract, in), nil
}

func (f *fakeWalletServer) FreezeBalanceV2(_ context.Context, in *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	f.gotFreeze = in
	return ext(core.Transaction_Contract_FreezeBalanceV2Contract, in), nil
}

func (f *fakeWalletServer) UnfreezeBalanceV2(_ context.Context, in *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	f.gotUnfreeze = in
	return ext(core.Transaction_Contract_UnfreezeBalanceV2Contract, in), nil
}

func (f *fakeWalletServer) DelegateResource(_ context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	f.gotDelegate = in
	return ext(core.Transaction_Contract_DelegateResourceContract, in), nil
}

func (f *fakeWalletServer) UnDelegateResource(_ context.Context, in *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	f.gotUndelegate = in
	return ext(core.Transaction_Contract_UnDelegateResourceContract, in), nil
}

func (f *fakeWalletServer) CancelAllUnfreezeV2(_ context.Context, in *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
	f.gotCancel = in
	return ext(core.Transaction_Contract_CancelAllUnfreezeV2Contract, in), nil
}

func (f *fakeWalletServer) WithdrawExpireUnfreeze(_ context.Context, in *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
	f.gotWithdraw = in
	return ext(core.Transaction_Contract_WithdrawExpireUnfreezeContract, in), nil
}

func (f *fakeWalletServer) CreateTransaction2(_ context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
	f.gotCreate = in
	return ext(core.Transaction_Contract_TransferContract, in), nil
}

func (f *fakeWalletServer) TransferAsset2(_ context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error) {
	f.gotAsset = in
	return ext(core.Transaction_Contract_TransferAssetContract, in), nil
}

func (f *fakeWalletServer) DeployContract(_ context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
	f.gotDeploy = in
	return ext(core.Transaction_Contract_CreateSmartContract, in), nil
}

func (f *fakeWalletServer) TriggerConstantContract(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	f.gotTrigger = in
	e := ext(core.Transaction_Contract_TriggerSmartContract, in)
	e.EnergyUsed = 5000
	return e, nil
}

func (f *fakeWalletServer) TriggerContract(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return ext(core.Transaction_Contract_TriggerSmartContract, in), nil
}

func (f *fakeWalletServer) GetEnergyPrices(_ context.Context, _ *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return &api.PricesResponseMessage{Prices: "1691500000000:420"}, nil
}

func (f *fakeWalletServer) GetBandwidthPrices(_ context.Context, _ *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	return &api.PricesResponseMessage{Prices: "1627279200000:1000"}, nil
}

func (f *fakeWalletServer) GetChainParameters(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
	return &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
		{Key: "getUpdateAccountPermissionFee", Value: 100_000_000},
		{Key: "getMultiSignFee", Value: 1_000_000},
		{Key: "getUnfreezeDelayDays", Value: 14},
		{Key: "getMaxDelegateLockPeriod", Value: 86_400},
	}}, nil
}

// ext builds a canned build response wrapping the request message, so the
// contract type the builders check is the real one.
func ext(t core.Transaction_Contract_ContractType, msg proto.Message) *api.TransactionExtention {
	anyMsg, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	raw := &core.TransactionRaw{
		Expiration: time.Now().Add(60 * time.Second).UnixMilli(),
		Contract:   []*core.Transaction_Contract{{Type: t, Parameter: anyMsg}},
	}
	return &api.TransactionExtention{
		Transaction: &core.Transaction{RawData: raw},
		Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
	}
}

func newTestClient(t *testing.T, f *fakeWalletServer) *rpc.Client {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, f)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})
	c, err := rpc.NewClientWithDialer("passthrough:///bufnet", func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}, rpc.WithTimeout(5*time.Second), rpc.WithPool(1, 2))
	if err != nil {
		t.Fatalf("NewClientWithDialer: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func newTestHandle(t *testing.T, f *fakeWalletServer) *Handle {
	t.Helper()
	return New(newTestClient(t, f), testFrom)
}

func mustClient(t *testing.T, f *fakeWalletServer) *rpc.Client {
	t.Helper()
	return newTestClient(t, f)
}

const testKeyHex = "0101010101010101010101010101010101010101010101010101010101010101"

func mustSigner(t *testing.T, hex string) key.Signer {
	t.Helper()
	s, err := key.PrivateKeyFromHex(hex)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	return s
}
