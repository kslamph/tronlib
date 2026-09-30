package tx

// Bufconn fake for the tx package tests. v2/rpc's fake lives in rpc's test
// files (unexported, not importable from here), so this is a minimal
// same-pattern mirror covering only the RPCs the tx package touches: the
// four build RPCs, BroadcastTransaction and GetTransactionInfoById on both
// the Wallet and WalletSolidity services. Tests customize behavior through
// typed handler fields; nil fields fall back to canned success defaults.

import (
	"context"
	"crypto/sha256"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// testAddresses: two distinct, valid 0x41-prefixed 21-byte addresses.
var (
	testFrom = mustAddr(0x11)
	testTo   = mustAddr(0x22)
)

// testKeyHex is a fixed secp256k1 private key for Sign tests (in-range, so
// PrivateKeyFromHex accepts it).
const testKeyHex = "0101010101010101010101010101010101010101010101010101010101010101"
const testKeyHex2 = "0202020202020202020202020202020202020202020202020202020202020202"

func mustAddr(fill byte) tron.Address {
	a, err := tron.AddressFromBytes(append([]byte{0x41}, repeat(fill, 20)...))
	if err != nil {
		panic(err)
	}
	return a
}

func repeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

// fakeWalletServer implements api.WalletServer (plus the WalletSolidity
// GetTransactionInfoById path through the solidity delegator) with
// per-test typed handlers over canned defaults.
type fakeWalletServer struct {
	api.UnimplementedWalletServer

	CreateTx2       func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error)
	Trigger         func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	TriggerConstant func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	EstimateEnerg   func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error)
	AccountResource func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error)
	EnergyPrices    func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error)
	BandwidthPrices func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error)
	ChainParameters func(ctx context.Context, in *api.EmptyMessage) (*core.ChainParameters, error)
	Account         func(ctx context.Context, in *core.Account) (*core.Account, error)
	Deploy          func(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error)
	TransferAssetFn func(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error)
	Broadcast       func(ctx context.Context, in *core.Transaction) (*api.Return, error)
	TxInfo          func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
	TxInfoSolidity  func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
	ContractInfo    func(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error)
	UpdateSettingFn func(ctx context.Context, in *core.UpdateSettingContract) (*api.TransactionExtention, error)
	UpdateEnergyFn  func(ctx context.Context, in *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error)
	ClearABIFn      func(ctx context.Context, in *core.ClearABIContract) (*api.TransactionExtention, error)

	// Stake 2.0, delegation, voting and permission RPCs (native operations).
	FreezeV2        func(ctx context.Context, in *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error)
	UnfreezeV2      func(ctx context.Context, in *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error)
	DelegateFn      func(ctx context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error)
	UnDelegateFn    func(ctx context.Context, in *core.UnDelegateResourceContract) (*api.TransactionExtention, error)
	CancelUnfreeze  func(ctx context.Context, in *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error)
	WithdrawUnfreez func(ctx context.Context, in *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error)
	VoteWitnessFn   func(ctx context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error)
	WithdrawBalFn   func(ctx context.Context, in *core.WithdrawBalanceContract) (*api.TransactionExtention, error)
	PermissionUpd   func(ctx context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error)

	// Native reads.
	CanDelegateMax  func(ctx context.Context, in *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error)
	UnfreezeCount   func(ctx context.Context, in *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error)
	CanWithdraw     func(ctx context.Context, in *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error)
	DelegatedRes    func(ctx context.Context, in *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
	DelegationIndex func(ctx context.Context, in *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error)
	RewardInfo      func(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error)
	SignWeight      func(ctx context.Context, in *core.Transaction) (*api.TransactionSignWeight, error)
	ApprovedList    func(ctx context.Context, in *core.Transaction) (*api.TransactionApprovedList, error)

	// simulateCalls / estimateCalls / accountResourceCalls / energyPricesCalls
	// count invocations, for asserting the CostPreview read sequence.
	simulateCalls        atomic.Int32
	bandwidthPricesCalls atomic.Int32
	estimateCalls        atomic.Int32
	accountResourceCalls atomic.Int32
	energyPricesCalls    atomic.Int32

	// broadcastCalls / txInfoCalls / txInfoSolidityCalls count invocations,
	// for asserting the single-reconciliation-poll behavior.
	broadcastCalls      atomic.Int32
	txInfoCalls         atomic.Int32
	txInfoSolidityCalls atomic.Int32
}

func (f *fakeWalletServer) CreateTransaction2(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
	if f.CreateTx2 != nil {
		return f.CreateTx2(ctx, in)
	}
	return transferExt(), nil
}

func (f *fakeWalletServer) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if f.Trigger != nil {
		return f.Trigger(ctx, in)
	}
	return triggerExt(), nil
}

func (f *fakeWalletServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	f.simulateCalls.Add(1)
	if f.TriggerConstant != nil {
		return f.TriggerConstant(ctx, in)
	}
	return triggerExt(), nil
}

func (f *fakeWalletServer) EstimateEnergy(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	f.estimateCalls.Add(1)
	if f.EstimateEnerg != nil {
		return f.EstimateEnerg(ctx, in)
	}
	return &api.EstimateEnergyMessage{Result: okResult(), EnergyRequired: 5000}, nil
}

func (f *fakeWalletServer) GetAccountResource(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
	f.accountResourceCalls.Add(1)
	if f.AccountResource != nil {
		return f.AccountResource(ctx, in)
	}
	return &api.AccountResourceMessage{}, nil
}

func (f *fakeWalletServer) GetEnergyPrices(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	f.energyPricesCalls.Add(1)
	if f.EnergyPrices != nil {
		return f.EnergyPrices(ctx, in)
	}
	return &api.PricesResponseMessage{Prices: "1691400000000:410,1691500000000:420"}, nil
}

func (f *fakeWalletServer) GetBandwidthPrices(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	f.bandwidthPricesCalls.Add(1)
	if f.BandwidthPrices != nil {
		return f.BandwidthPrices(ctx, in)
	}
	return &api.PricesResponseMessage{Prices: "1627279200000:1000"}, nil
}

func (f *fakeWalletServer) GetChainParameters(ctx context.Context, in *api.EmptyMessage) (*core.ChainParameters, error) {
	if f.ChainParameters != nil {
		return f.ChainParameters(ctx, in)
	}
	return &core.ChainParameters{}, nil
}

func (f *fakeWalletServer) GetAccount(ctx context.Context, in *core.Account) (*core.Account, error) {
	if f.Account != nil {
		return f.Account(ctx, in)
	}
	// Default: the queried account exists with zero balance.
	return &core.Account{Address: in.GetAddress()}, nil
}

func (f *fakeWalletServer) DeployContract(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
	if f.Deploy != nil {
		return f.Deploy(ctx, in)
	}
	return deployExt(), nil
}

func (f *fakeWalletServer) TransferAsset2(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error) {
	if f.TransferAssetFn != nil {
		return f.TransferAssetFn(ctx, in)
	}
	return assetExt(), nil
}

func (f *fakeWalletServer) BroadcastTransaction(ctx context.Context, in *core.Transaction) (*api.Return, error) {
	f.broadcastCalls.Add(1)
	if f.Broadcast != nil {
		return f.Broadcast(ctx, in)
	}
	return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
}

func (f *fakeWalletServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	f.txInfoCalls.Add(1)
	if f.TxInfo != nil {
		return f.TxInfo(ctx, in)
	}
	// Default: not found (empty info — no Id), the "still unconfirmed" answer.
	return &core.TransactionInfo{}, nil
}

func (f *fakeWalletServer) GetContractInfo(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
	if f.ContractInfo != nil {
		return f.ContractInfo(ctx, in)
	}
	// Default: a fresh contract — deployed (SmartContract present) but with
	// no state row yet, which the node reports as an absent ContractState
	// (DynamicEnergyOf maps it to zero).
	return &core.SmartContractDataWrapper{SmartContract: &core.SmartContract{}}, nil
}

// manageExt is the canned build response for the contract management RPCs:
// a transaction shell with raw data (no contract message needed — the
// management builders never decode it).
func manageExt() *api.TransactionExtention {
	return &api.TransactionExtention{
		Result:      okResult(),
		Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
	}
}

func (f *fakeWalletServer) UpdateSetting(ctx context.Context, in *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	if f.UpdateSettingFn != nil {
		return f.UpdateSettingFn(ctx, in)
	}
	return manageExt(), nil
}

func (f *fakeWalletServer) UpdateEnergyLimit(ctx context.Context, in *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	if f.UpdateEnergyFn != nil {
		return f.UpdateEnergyFn(ctx, in)
	}
	return manageExt(), nil
}

func (f *fakeWalletServer) ClearContractABI(ctx context.Context, in *core.ClearABIContract) (*api.TransactionExtention, error) {
	if f.ClearABIFn != nil {
		return f.ClearABIFn(ctx, in)
	}
	return manageExt(), nil
}

// nativeOpExt is the canned build response for a native (non-contract)
// operation: it wraps the request message so the contract type is the real
// one — portable round-trips and contract-type checks depend on it.
func nativeOpExt(t core.Transaction_Contract_ContractType, msg proto.Message) *api.TransactionExtention {
	return baseExt(contractAny(t, msg), okResult())
}

func (f *fakeWalletServer) FreezeBalanceV2(ctx context.Context, in *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	if f.FreezeV2 != nil {
		return f.FreezeV2(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_FreezeBalanceV2Contract, in), nil
}

func (f *fakeWalletServer) UnfreezeBalanceV2(ctx context.Context, in *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	if f.UnfreezeV2 != nil {
		return f.UnfreezeV2(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_UnfreezeBalanceV2Contract, in), nil
}

func (f *fakeWalletServer) DelegateResource(ctx context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	if f.DelegateFn != nil {
		return f.DelegateFn(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_DelegateResourceContract, in), nil
}

func (f *fakeWalletServer) UnDelegateResource(ctx context.Context, in *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	if f.UnDelegateFn != nil {
		return f.UnDelegateFn(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_UnDelegateResourceContract, in), nil
}

func (f *fakeWalletServer) CancelAllUnfreezeV2(ctx context.Context, in *core.CancelAllUnfreezeV2Contract) (*api.TransactionExtention, error) {
	if f.CancelUnfreeze != nil {
		return f.CancelUnfreeze(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_CancelAllUnfreezeV2Contract, in), nil
}

func (f *fakeWalletServer) WithdrawExpireUnfreeze(ctx context.Context, in *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
	if f.WithdrawUnfreez != nil {
		return f.WithdrawUnfreez(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_WithdrawExpireUnfreezeContract, in), nil
}

func (f *fakeWalletServer) VoteWitnessAccount2(ctx context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	if f.VoteWitnessFn != nil {
		return f.VoteWitnessFn(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_VoteWitnessContract, in), nil
}

func (f *fakeWalletServer) WithdrawBalance2(ctx context.Context, in *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	if f.WithdrawBalFn != nil {
		return f.WithdrawBalFn(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_WithdrawBalanceContract, in), nil
}

func (f *fakeWalletServer) AccountPermissionUpdate(ctx context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	if f.PermissionUpd != nil {
		return f.PermissionUpd(ctx, in)
	}
	return nativeOpExt(core.Transaction_Contract_AccountPermissionUpdateContract, in), nil
}

func (f *fakeWalletServer) GetCanDelegatedMaxSize(ctx context.Context, in *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	if f.CanDelegateMax != nil {
		return f.CanDelegateMax(ctx, in)
	}
	return &api.CanDelegatedMaxSizeResponseMessage{MaxSize: 0}, nil
}

func (f *fakeWalletServer) GetAvailableUnfreezeCount(ctx context.Context, in *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	if f.UnfreezeCount != nil {
		return f.UnfreezeCount(ctx, in)
	}
	return &api.GetAvailableUnfreezeCountResponseMessage{Count: 32}, nil
}

func (f *fakeWalletServer) GetCanWithdrawUnfreezeAmount(ctx context.Context, in *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	if f.CanWithdraw != nil {
		return f.CanWithdraw(ctx, in)
	}
	return &api.CanWithdrawUnfreezeAmountResponseMessage{}, nil
}

func (f *fakeWalletServer) GetDelegatedResourceV2(ctx context.Context, in *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	if f.DelegatedRes != nil {
		return f.DelegatedRes(ctx, in)
	}
	return &api.DelegatedResourceList{}, nil
}

func (f *fakeWalletServer) GetDelegatedResourceAccountIndexV2(ctx context.Context, in *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
	if f.DelegationIndex != nil {
		return f.DelegationIndex(ctx, in)
	}
	return &core.DelegatedResourceAccountIndex{}, nil
}

func (f *fakeWalletServer) GetRewardInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	if f.RewardInfo != nil {
		return f.RewardInfo(ctx, in)
	}
	return &api.NumberMessage{Num: 0}, nil
}

func (f *fakeWalletServer) GetTransactionSignWeight(ctx context.Context, in *core.Transaction) (*api.TransactionSignWeight, error) {
	if f.SignWeight != nil {
		return f.SignWeight(ctx, in)
	}
	return &api.TransactionSignWeight{}, nil
}

func (f *fakeWalletServer) GetTransactionApprovedList(ctx context.Context, in *core.Transaction) (*api.TransactionApprovedList, error) {
	if f.ApprovedList != nil {
		return f.ApprovedList(ctx, in)
	}
	return &api.TransactionApprovedList{}, nil
}

// fakeSolidityServer serves the WalletSolidity service on the same bufconn
// listener, delegating to the shared fake's handlers.
type fakeSolidityServer struct {
	api.UnimplementedWalletSolidityServer
	ws *fakeWalletServer
}

func (s *fakeSolidityServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	s.ws.txInfoSolidityCalls.Add(1)
	if s.ws.TxInfoSolidity != nil {
		return s.ws.TxInfoSolidity(ctx, in)
	}
	return &core.TransactionInfo{}, nil
}

// newTxTestClient dials a bufconn-backed gRPC server (Wallet + WalletSolidity)
// with a real rpc.Client, which satisfies rpc.ConnProvider for the builders
// and Broadcast.
func newTxTestClient(t *testing.T, f *fakeWalletServer) *rpc.Client {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, f)
	api.RegisterWalletSolidityServer(srv, &fakeSolidityServer{ws: f})
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

// --- canned build responses ---

// txID computes the hex txid of a transaction the same way baseTx.ID() does
// (sha256 of raw_data), so the canned ext's Txid matches what the tx layer
// reports.
func txID(raw *core.TransactionRaw) []byte {
	data, err := proto.Marshal(raw)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return sum[:]
}

func baseExt(param *core.Transaction_Contract, result *api.Return) *api.TransactionExtention {
	raw := &core.TransactionRaw{
		// Broadcast's pre-flight requires a future expiration; the fake
		// emulates the server-side head+60s default.
		Expiration: time.Now().Add(60 * time.Second).UnixMilli(),
		// FeeLimit 0 lets tests pin the 150_000_000 default floor.
		FeeLimit: 0,
		Contract: []*core.Transaction_Contract{param},
	}
	return &api.TransactionExtention{
		Transaction: &core.Transaction{RawData: raw},
		Txid:        txID(raw),
		Result:      result,
	}
}

func okResult() *api.Return { return &api.Return{Result: true, Code: api.Return_SUCCESS} }

func contractAny(t core.Transaction_Contract_ContractType, msg proto.Message) *core.Transaction_Contract {
	v, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	return &core.Transaction_Contract{Type: t, Parameter: v}
}

func transferExt() *api.TransactionExtention {
	return baseExt(contractAny(core.Transaction_Contract_TransferContract, &core.TransferContract{
		OwnerAddress: testFrom.Bytes(),
		ToAddress:    testTo.Bytes(),
		Amount:       1_000_000,
	}), okResult())
}

func triggerExt() *api.TransactionExtention {
	return baseExt(contractAny(core.Transaction_Contract_TriggerSmartContract, &core.TriggerSmartContract{
		OwnerAddress:    testFrom.Bytes(),
		ContractAddress: testTo.Bytes(),
		CallValue:       0,
	}), okResult())
}

func deployExt() *api.TransactionExtention {
	return baseExt(contractAny(core.Transaction_Contract_CreateSmartContract, &core.CreateSmartContract{
		OwnerAddress: testFrom.Bytes(),
		NewContract: &core.SmartContract{
			OriginAddress:              testFrom.Bytes(),
			Bytecode:                   []byte{0x60, 0x80},
			CallValue:                  0,
			ConsumeUserResourcePercent: 10,
			OriginEnergyLimit:          1,
		},
	}), okResult())
}

func assetExt() *api.TransactionExtention {
	return baseExt(contractAny(core.Transaction_Contract_TransferAssetContract, &core.TransferAssetContract{
		OwnerAddress: testFrom.Bytes(),
		ToAddress:    testTo.Bytes(),
		AssetName:    []byte("1000001"),
		Amount:       5,
	}), okResult())
}

// foundInfo builds a TransactionInfo "found on chain" answer echoing the
// requested txid, for Wait/WaitForSolid/reconciliation tests.
func foundInfo(id []byte) *core.TransactionInfo {
	return &core.TransactionInfo{
		Id:              id,
		BlockNumber:     12345,
		BlockTimeStamp:  1700000000000,
		Result:          core.TransactionInfo_SUCESS,
		ContractAddress: []byte{},
	}
}
