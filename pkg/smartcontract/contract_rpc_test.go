package smartcontract

import (
	"context"
	"encoding/hex"
	"math/big"
	"net"
	"testing"
	"time"

	eabi "github.com/ethereum/go-ethereum/accounts/abi"
	"golang.org/x/crypto/sha3"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const contractBufSize = 1024 * 1024

type contractServer struct {
	api.UnimplementedWalletServer
	lastTriggerReq    *core.TriggerSmartContract
	constantResponder func(*core.TriggerSmartContract) *api.TransactionExtention
}

func (s *contractServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if s.constantResponder != nil {
		return s.constantResponder(in), nil
	}
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}, nil
}

func (s *contractServer) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	s.lastTriggerReq = in
	// Return a fake txid
	return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, Txid: []byte{0xaa, 0xbb}}, nil
}

func newContractBufServer(t *testing.T, impl api.WalletServer) (*bufconn.Listener, *grpc.Server, func()) {
	t.Helper()
	lis := bufconn.Listen(contractBufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, impl)
	go func() { _ = srv.Serve(lis) }()
	cleanup := func() {
		_ = lis.Close()
		srv.Stop()
	}
	return lis, srv, cleanup
}

func methodID(signature string) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(signature))
	sum := h.Sum(nil)
	return sum[:4]
}

func packReturn(t *testing.T, typ string, val interface{}) []byte {
	t.Helper()
	aType, err := eabi.NewType(typ, "", nil)
	if err != nil {
		t.Fatalf("abi type err: %v", err)
	}
	out, err := eabi.Arguments{{Type: aType}}.Pack(val)
	if err != nil {
		t.Fatalf("pack err: %v", err)
	}
	return out
}

func TestContract_TriggerConstantContract_DecodeName(t *testing.T) {
	srv := &contractServer{}
	srv.constantResponder = func(in *core.TriggerSmartContract) *api.TransactionExtention {
		// name() selector: 0x06fdde03
		if hex.EncodeToString(in.Data[:4]) == hex.EncodeToString(methodID("name()")) {
			enc := packReturn(t, "string", "MyToken")
			return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}, ConstantResult: [][]byte{enc}}
		}
		return &api.TransactionExtention{Result: &api.Return{Result: true, Code: api.Return_SUCCESS}}
	}
	lis, _, cleanup := newContractBufServer(t, srv)
	t.Cleanup(cleanup)

	c, err := client.NewClientWithDialer("passthrough:///bufnet", func(ctx context.Context, s string) (net.Conn, error) { return lis.DialContext(ctx) }, client.WithTimeout(500*time.Millisecond), client.WithPool(1, 1))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer c.Close()

	contractAddr := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	owner := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	sc, err := NewContract(c, contractAddr, testERC20ABI)
	if err != nil {
		t.Fatalf("new contract: %v", err)
	}

	res, err := sc.TriggerConstantContract(context.Background(), owner, "name")
	if err != nil {
		t.Fatalf("TriggerConstantContract err: %v", err)
	}
	if res.(string) != "MyToken" {
		t.Fatalf("unexpected decoded name: %v", res)
	}
}

func TestContract_TriggerSmartContract_EncodesAndSends(t *testing.T) {
	srv := &contractServer{}
	lis, _, cleanup := newContractBufServer(t, srv)
	t.Cleanup(cleanup)

	c, err := client.NewClientWithDialer("passthrough:///bufnet", func(ctx context.Context, s string) (net.Conn, error) { return lis.DialContext(ctx) }, client.WithTimeout(500*time.Millisecond), client.WithPool(1, 1))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer c.Close()

	contractAddr := types.MustNewAddressFromBase58("TKCTfkQ8L9beavNu9iaGtCHFxrwNHUxfr2")
	from := types.MustNewAddressFromBase58("TBXeeuh3jHM7oE889Ys2DqvRS1YuEPoa2o")
	to := types.MustNewAddressFromBase58("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	sc, err := NewContract(c, contractAddr, testERC20ABI)
	if err != nil {
		t.Fatalf("new contract: %v", err)
	}

	if _, err := sc.TriggerSmartContract(context.Background(), from, 0, "transfer", to, big.NewInt(10)); err != nil {
		t.Fatalf("TriggerSmartContract err: %v", err)
	}
	if srv.lastTriggerReq == nil {
		t.Fatalf("no trigger request captured")
	}
	// Verify basic fields
	if hex.EncodeToString(srv.lastTriggerReq.OwnerAddress) != hex.EncodeToString(from.Bytes()) {
		t.Fatalf("owner mismatch")
	}
	if hex.EncodeToString(srv.lastTriggerReq.ContractAddress) != hex.EncodeToString(contractAddr.Bytes()) {
		t.Fatalf("contract address mismatch")
	}
	// Check selector matches transfer(address,uint256)
	if got, want := hex.EncodeToString(srv.lastTriggerReq.Data[:4]), hex.EncodeToString(methodID("transfer(address,uint256)")); got != want {
		t.Fatalf("method id mismatch got %s want %s", got, want)
	}
}
