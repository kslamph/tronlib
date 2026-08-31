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

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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
	Deploy          func(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error)
	TransferAssetFn func(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error)
	Broadcast       func(ctx context.Context, in *core.Transaction) (*api.Return, error)
	TxInfo          func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
	TxInfoSolidity  func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)

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
