package contract

// Minimal bufconn fake for the contract package's tests. v2/rpc's fake
// lives in rpc's test files (unexported, not importable from here), so this
// is a same-pattern mirror (see tx/fakes_test.go) covering only the RPCs
// the contract layer touches: TriggerConstantContract (Call),
// TriggerContract (Invoke's build step) and GetContract (the lazy ABI
// fetch). Handlers are typed fields; nil falls back to canned success.

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"math/big"
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

// testABI is the small contract the instance tests run against:
// balanceOf(address) returns uint256, transfer(address,uint256) returns
// bool, name() returns string, getOwner() returns address, and noop()
// returns nothing.
const testABI = `[
  {"type":"function","name":"balanceOf","stateMutability":"view",
   "inputs":[{"name":"owner","type":"address"}],
   "outputs":[{"name":"","type":"uint256"}]},
  {"type":"function","name":"transfer","stateMutability":"nonpayable",
   "inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],
   "outputs":[{"name":"","type":"bool"}]},
  {"type":"function","name":"name","stateMutability":"view",
   "inputs":[],
   "outputs":[{"name":"","type":"string"}]},
  {"type":"function","name":"getOwner","stateMutability":"view",
   "inputs":[],
   "outputs":[{"name":"","type":"address"}]},
  {"type":"function","name":"noop","stateMutability":"nonpayable",
   "inputs":[],"outputs":[]}
]`

// testContractAddress is the contract the Instance points at (a real
// mainnet-form address).
var testContractAddress = testMainnetAddr

// fakeWallet implements api.WalletServer with per-test typed handlers over
// canned defaults, plus call counters for the lazy-fetch assertions.
type fakeWallet struct {
	api.UnimplementedWalletServer

	TriggerConstant   func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	Trigger           func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	GetContractFn     func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error)
	GetContractInfoFn func(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error)

	triggerConstantCalls atomic.Int32
	triggerCalls         atomic.Int32
	getContractCalls     atomic.Int32
}

func (f *fakeWallet) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	f.triggerConstantCalls.Add(1)
	if f.TriggerConstant != nil {
		return f.TriggerConstant(ctx, in)
	}
	return okExtention(), nil
}

func (f *fakeWallet) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	f.triggerCalls.Add(1)
	if f.Trigger != nil {
		return f.Trigger(ctx, in)
	}
	return triggerExt(), nil
}

func (f *fakeWallet) GetContract(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
	f.getContractCalls.Add(1)
	if f.GetContractFn != nil {
		return f.GetContractFn(ctx, in)
	}
	// Default: a contract whose on-chain ABI is the test ABI.
	return &core.SmartContract{Abi: mustPbABI(testABI)}, nil
}

func (f *fakeWallet) GetContractInfo(ctx context.Context, in *api.BytesMessage) (*core.SmartContractDataWrapper, error) {
	if f.GetContractInfoFn != nil {
		return f.GetContractInfoFn(ctx, in)
	}
	// Default: a fresh contract — deployed (SmartContract present) but with
	// no state row yet.
	return &core.SmartContractDataWrapper{SmartContract: &core.SmartContract{}}, nil
}

// newContractTestClient dials a bufconn-backed gRPC Wallet server with a
// real rpc.Client (which satisfies rpc.ConnProvider).
func newContractTestClient(t *testing.T, f *fakeWallet) *rpc.Client {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, f)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop(); _ = lis.Close() })
	c, err := rpc.NewClientWithDialer("passthrough:///bufnet", func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}, rpc.WithTimeout(5*time.Second), rpc.WithPool(1, 2))
	if err != nil {
		t.Fatalf("NewClientWithDialer: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// newInstanceWithTestABI builds an Instance with the test ABI loaded
// up front (no network fetch needed) against the shared fake client.
func newInstanceWithTestABI(t *testing.T) *Instance {
	t.Helper()
	i, err := NewInstance(newContractTestClient(t, &fakeWallet{}), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if err := i.UseABI(testABI); err != nil {
		t.Fatalf("UseABI: %v", err)
	}
	return i
}

// --- canned response builders (mirroring tx/fakes_test.go) ---

func okResult() *api.Return { return &api.Return{Result: true, Code: api.Return_SUCCESS} }

func okExtention() *api.TransactionExtention {
	return &api.TransactionExtention{Result: okResult()}
}

func txID(raw *core.TransactionRaw) []byte {
	data, err := proto.Marshal(raw)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return sum[:]
}

func contractAny(t core.Transaction_Contract_ContractType, msg proto.Message) *core.Transaction_Contract {
	v, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	return &core.Transaction_Contract{Type: t, Parameter: v}
}

func triggerExt() *api.TransactionExtention {
	raw := &core.TransactionRaw{
		Expiration: time.Now().Add(60 * time.Second).UnixMilli(),
		Contract: []*core.Transaction_Contract{
			contractAny(core.Transaction_Contract_TriggerSmartContract, &core.TriggerSmartContract{
				OwnerAddress:    mustAddr(0x11).Bytes(),
				ContractAddress: testContractAddress.Bytes(),
			}),
		},
	}
	return &api.TransactionExtention{Transaction: &core.Transaction{RawData: raw}, Txid: txID(raw), Result: okResult()}
}

// mustAddr builds a valid 0x41-prefixed address with the given filler byte.
func mustAddr(fill byte) tron.Address {
	b := make([]byte, 21)
	b[0] = 0x41
	for i := 1; i < 21; i++ {
		b[i] = fill
	}
	a, err := tron.AddressFromBytes(b)
	if err != nil {
		panic(err)
	}
	return a
}

// abiUint256 encodes a uint256 return value as the node's ConstantResult
// carries it: 32 bytes big-endian.
func abiUint256(v int64) []byte {
	out := make([]byte, 32)
	new(big.Int).SetInt64(v).FillBytes(out)
	return out
}

// abiBool encodes a bool return value: 32 bytes, 0 or 1.
func abiBool(v bool) []byte {
	out := make([]byte, 32)
	if v {
		out[31] = 1
	}
	return out
}

// abiAddress encodes a 20-byte address as a padded 32-byte word.
func abiAddress(payload []byte) []byte {
	out := make([]byte, 32)
	copy(out[12:], payload)
	return out
}

// mustPbABI parses the Solidity JSON test ABI into the protobuf SmartContract_ABI
// the GetContract fake returns (via the event package's JSON parser in
// reverse — here we hand-build entries from the JSON using encoding/json).
func mustPbABI(jsonABI string) *core.SmartContract_ABI {
	var entries []struct {
		Type    string                  `json:"type"`
		Name    string                  `json:"name"`
		Inputs  []struct{ Type string } `json:"inputs"`
		Outputs []struct{ Type string } `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(jsonABI), &entries); err != nil {
		panic(err)
	}
	typeMap := map[string]core.SmartContract_ABI_Entry_EntryType{
		"function": core.SmartContract_ABI_Entry_Function,
	}
	abi := &core.SmartContract_ABI{}
	for _, e := range entries {
		entry := &core.SmartContract_ABI_Entry{Name: e.Name, Type: typeMap[e.Type]}
		for _, in := range e.Inputs {
			entry.Inputs = append(entry.Inputs, &core.SmartContract_ABI_Entry_Param{Type: in.Type})
		}
		for _, out := range e.Outputs {
			entry.Outputs = append(entry.Outputs, &core.SmartContract_ABI_Entry_Param{Type: out.Type})
		}
		abi.Entrys = append(abi.Entrys, entry)
	}
	return abi
}
