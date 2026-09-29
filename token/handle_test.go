package token

import (
	"context"
	"crypto/sha256"
	"errors"
	"math/big"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
)

// --- the bufconn fake: a TRC-20-shaped wallet server ---
// (same shape as contract/fakes_test.go; rpc's fake is unexported and not
// importable from here, so the pattern is mirrored).

// fakeOnChainABI is the ABI the fake contract publishes on-chain.
const fakeOnChainABI = `[
  {"type":"function","name":"decimals","stateMutability":"view",
   "inputs":[],"outputs":[{"name":"","type":"uint8"}]},
  {"type":"function","name":"balanceOf","stateMutability":"view",
   "inputs":[{"name":"owner","type":"address"}],
   "outputs":[{"name":"","type":"uint256"}]},
  {"type":"function","name":"transfer","stateMutability":"nonpayable",
   "inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],
   "outputs":[{"name":"","type":"bool"}]}
]`

// fakeTRC20Wallet answers TriggerConstantContract/TriggerContract/GetContract
// for the token tests. triggerCalls counts how many times the tx builder's
// RPC fired (Invoke's build step).
type fakeTRC20Wallet struct {
	api.UnimplementedWalletServer

	TriggerConstant func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	Trigger         func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	GetContractFn   func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error)

	triggerConstantCalls int32
	triggerCalls         int32
}

func (f *fakeTRC20Wallet) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	f.triggerConstantCalls++
	if f.TriggerConstant != nil {
		return f.TriggerConstant(ctx, in)
	}
	return okExt(), nil
}

func (f *fakeTRC20Wallet) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	f.triggerCalls++
	if f.Trigger != nil {
		return f.Trigger(ctx, in)
	}
	return okExt(), nil
}

func (f *fakeTRC20Wallet) GetContract(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
	if f.GetContractFn != nil {
		return f.GetContractFn(ctx, in)
	}
	return &core.SmartContract{Abi: mustPbABI(fakeOnChainABI)}, nil
}

func okExt() *api.TransactionExtention {
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}
}

// triggerExt is the TransactionExtention the tx builder's TriggerContract
// RPC needs: a built transaction with a txid (ported from contract's fake).
func triggerExt() *api.TransactionExtention {
	raw := &core.TransactionRaw{
		Expiration: time.Now().Add(60 * time.Second).UnixMilli(),
		Contract: []*core.Transaction_Contract{{
			Type: core.Transaction_Contract_TriggerSmartContract,
			Parameter: mustAny(&core.TriggerSmartContract{
				OwnerAddress:    mustAddr(0x11).Bytes(),
				ContractAddress: testContract.Bytes(),
			}),
		}},
	}
	data, err := proto.Marshal(raw)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return &api.TransactionExtention{Transaction: &core.Transaction{RawData: raw}, Txid: sum[:], Result: okResult()}
}

func okResult() *api.Return { return &api.Return{Result: true, Code: api.Return_SUCCESS} }

func mustAny(msg proto.Message) *anypb.Any {
	v, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	return v
}

// mustSelector is keccak256(sig)[:4], the dispatch key the fakes use.
func mustSelector(sig string) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(sig))
	return h.Sum(nil)[:4]
}

func mustPbABI(jsonABI string) *core.SmartContract_ABI {
	abi := &core.SmartContract_ABI{}
	abi.Entrys = append(abi.Entrys,
		&core.SmartContract_ABI_Entry{Name: "decimals", Type: core.SmartContract_ABI_Entry_Function},
		&core.SmartContract_ABI_Entry{Name: "balanceOf", Type: core.SmartContract_ABI_Entry_Function},
		&core.SmartContract_ABI_Entry{Name: "transfer", Type: core.SmartContract_ABI_Entry_Function},
	)
	return abi
}

func newTokenTestClient(t *testing.T, f *fakeTRC20Wallet) *rpc.Client {
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

// abiWord encodes one uint256 word.
func abiWord(v int64) []byte {
	out := make([]byte, 32)
	new(big.Int).SetInt64(v).FillBytes(out)
	return out
}

// testContract is the address the fake token lives at.
var testContract = mustAddr(0x99)

// newTestHandle builds a Handle against a fresh fake with 6 decimals.
func newTestHandle(t *testing.T) (*Handle, *fakeTRC20Wallet) {
	t.Helper()
	f := &fakeTRC20Wallet{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExt()
			switch string(in.GetData()[:4]) {
			case string(mustSelector("decimals()")):
				ext.ConstantResult = [][]byte{abiWord(6)}
			case string(mustSelector("balanceOf(address)")):
				ext.ConstantResult = [][]byte{abiWord(777)}
			}
			return ext, nil
		},
	}
	h, err := New(context.Background(), newTokenTestClient(t, f), testContract)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h, f
}

// asTronError wraps errors.As for *tron.Error.
func asTronError(err error, target **tron.Error) bool { return errors.As(err, target) }
