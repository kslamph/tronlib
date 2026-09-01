package tronlib

// Facade delegation tests. The facade is one-line delegations (spec D7), so
// every test proves exactly that: the Client method returns what the owning
// subpackage function produces through the same wire path.
//
// The bufconn fake is a local mirror of v2/tx/fakes_test.go (rpc's fake is
// unexported and test-only, not importable; the same-pattern local mirror
// is the established pattern — Task 6 ruling, rule of three not yet met).
// It covers the RPCs the facade touches, including the ones tx's mirror
// lacks: GetAccount, GetNowBlock2 and GetPaginatedNowWitnessList.

import (
	"context"
	"crypto/sha256"
	"net"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/key"
	"github.com/kslamph/tronlib/v2/rpc"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// facadeAddresses: two distinct, valid 0x41-prefixed 21-byte addresses.
var (
	facadeFrom = mustFacadeAddr(0x11)
	facadeTo   = mustFacadeAddr(0x22)
)

// facadeKeyHex is a fixed secp256k1 private key (in-range, so KeyFromHex
// accepts it).
const facadeKeyHex = "0101010101010101010101010101010101010101010101010101010101010101"

func mustFacadeAddr(fill byte) tron.Address {
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

// fakeFacadeServer implements api.WalletServer (+ the WalletSolidity
// GetTransactionInfoById path) with per-test typed handlers over canned
// success defaults; nil fields fall back to the defaults.
type fakeFacadeServer struct {
	api.UnimplementedWalletServer

	CreateTx2       func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error)
	TransferAssetFn func(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error)
	Deploy          func(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error)
	Trigger         func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	TriggerConstant func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error)
	EstimateEnerg   func(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error)
	AccountResource func(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error)
	EnergyPrices    func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error)
	Broadcast       func(ctx context.Context, in *core.Transaction) (*api.Return, error)
	TxInfo          func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
	TxInfoSolidity  func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error)
	GetAccountFn    func(ctx context.Context, in *core.Account) (*core.Account, error)
	GetNowBlock2Fn  func(ctx context.Context, in *api.EmptyMessage) (*api.BlockExtention, error)
	WitnessPage     func(ctx context.Context, in *api.PaginatedMessage) (*api.WitnessList, error)

	// broadcastCalls counts BroadcastTransaction invocations, proving the
	// facade's Broadcast goes through tx.Broadcast's wire path exactly once.
	broadcastCalls int
}

func (f *fakeFacadeServer) CreateTransaction2(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
	if f.CreateTx2 != nil {
		return f.CreateTx2(ctx, in)
	}
	return facadeTransferExt(), nil
}

func (f *fakeFacadeServer) TransferAsset2(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error) {
	if f.TransferAssetFn != nil {
		return f.TransferAssetFn(ctx, in)
	}
	return facadeAssetExt(), nil
}

func (f *fakeFacadeServer) DeployContract(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
	if f.Deploy != nil {
		return f.Deploy(ctx, in)
	}
	return facadeDeployExt(), nil
}

func (f *fakeFacadeServer) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if f.Trigger != nil {
		return f.Trigger(ctx, in)
	}
	return facadeTriggerExt(), nil
}

func (f *fakeFacadeServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if f.TriggerConstant != nil {
		return f.TriggerConstant(ctx, in)
	}
	return facadeTriggerExt(), nil
}

func (f *fakeFacadeServer) EstimateEnergy(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	if f.EstimateEnerg != nil {
		return f.EstimateEnerg(ctx, in)
	}
	return &api.EstimateEnergyMessage{Result: facadeOKResult(), EnergyRequired: 5000}, nil
}

func (f *fakeFacadeServer) GetAccountResource(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
	if f.AccountResource != nil {
		return f.AccountResource(ctx, in)
	}
	return &api.AccountResourceMessage{}, nil
}

func (f *fakeFacadeServer) GetEnergyPrices(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	if f.EnergyPrices != nil {
		return f.EnergyPrices(ctx, in)
	}
	return &api.PricesResponseMessage{Prices: "1691400000000:410,1691500000000:420"}, nil
}

func (f *fakeFacadeServer) BroadcastTransaction(ctx context.Context, in *core.Transaction) (*api.Return, error) {
	f.broadcastCalls++
	if f.Broadcast != nil {
		return f.Broadcast(ctx, in)
	}
	return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
}

func (f *fakeFacadeServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	if f.TxInfo != nil {
		return f.TxInfo(ctx, in)
	}
	return &core.TransactionInfo{}, nil // not found: still unconfirmed
}

func (f *fakeFacadeServer) GetAccount(ctx context.Context, in *core.Account) (*core.Account, error) {
	if f.GetAccountFn != nil {
		return f.GetAccountFn(ctx, in)
	}
	return &core.Account{Balance: 1_500_000}, nil
}

func (f *fakeFacadeServer) GetNowBlock2(ctx context.Context, in *api.EmptyMessage) (*api.BlockExtention, error) {
	if f.GetNowBlock2Fn != nil {
		return f.GetNowBlock2Fn(ctx, in)
	}
	return &api.BlockExtention{BlockHeader: &core.BlockHeader{RawData: &core.BlockHeaderRaw{Number: 4242}}}, nil
}

func (f *fakeFacadeServer) GetPaginatedNowWitnessList(ctx context.Context, in *api.PaginatedMessage) (*api.WitnessList, error) {
	if f.WitnessPage != nil {
		return f.WitnessPage(ctx, in)
	}
	return &api.WitnessList{Witnesses: []*core.Witness{{Address: facadeFrom.Bytes(), VoteCount: 99, IsJobs: true}}}, nil
}

// facadeSolidityServer serves the WalletSolidity service on the same
// bufconn listener, delegating to the shared fake's handlers.
type facadeSolidityServer struct {
	api.UnimplementedWalletSolidityServer
	ws *fakeFacadeServer
}

func (s *facadeSolidityServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	if s.ws.TxInfoSolidity != nil {
		return s.ws.TxInfoSolidity(ctx, in)
	}
	return &core.TransactionInfo{}, nil
}

// newFacadeTestClient dials a bufconn-backed gRPC server (Wallet +
// WalletSolidity) with a real rpc.Client via rpc.NewClientWithDialer (Dial's
// endpoint grammar has no dialer seam), then wraps it exactly like facade
// Dial does.
func newFacadeTestClient(t *testing.T, f *fakeFacadeServer) *Client {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, f)
	api.RegisterWalletSolidityServer(srv, &facadeSolidityServer{ws: f})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})
	inner, err := rpc.NewClientWithDialer("passthrough:///bufnet", func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}, rpc.WithTimeout(5*time.Second), rpc.WithPool(1, 2))
	if err != nil {
		t.Fatalf("dial fake: %v", err)
	}
	t.Cleanup(func() { _ = inner.Close() })
	return &Client{inner: inner}
}

// --- canned build responses (mirror of tx/fakes_test.go) ---

func facadeTxID(raw *core.TransactionRaw) []byte {
	data, err := proto.Marshal(raw)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return sum[:]
}

func facadeBaseExt(param *core.Transaction_Contract) *api.TransactionExtention {
	raw := &core.TransactionRaw{
		// Broadcast's pre-flight requires a future expiration; the fake
		// emulates the server-side head+60s default.
		Expiration: time.Now().Add(60 * time.Second).UnixMilli(),
		FeeLimit:   0,
		Contract:   []*core.Transaction_Contract{param},
	}
	return &api.TransactionExtention{
		Transaction: &core.Transaction{RawData: raw},
		Txid:        facadeTxID(raw),
		Result:      facadeOKResult(),
	}
}

func facadeOKResult() *api.Return { return &api.Return{Result: true, Code: api.Return_SUCCESS} }

func facadeContractAny(t core.Transaction_Contract_ContractType, msg proto.Message) *core.Transaction_Contract {
	v, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	return &core.Transaction_Contract{Type: t, Parameter: v}
}

func facadeTransferExt() *api.TransactionExtention {
	return facadeBaseExt(facadeContractAny(core.Transaction_Contract_TransferContract, &core.TransferContract{
		OwnerAddress: facadeFrom.Bytes(),
		ToAddress:    facadeTo.Bytes(),
		Amount:       1_000_000,
	}))
}

func facadeAssetExt() *api.TransactionExtention {
	return facadeBaseExt(facadeContractAny(core.Transaction_Contract_TransferAssetContract, &core.TransferAssetContract{
		OwnerAddress: facadeFrom.Bytes(),
		ToAddress:    facadeTo.Bytes(),
		AssetName:    []byte("1000001"),
		Amount:       5,
	}))
}

func facadeDeployExt() *api.TransactionExtention {
	return facadeBaseExt(facadeContractAny(core.Transaction_Contract_CreateSmartContract, &core.CreateSmartContract{
		OwnerAddress: facadeFrom.Bytes(),
		NewContract: &core.SmartContract{
			OriginAddress:              facadeFrom.Bytes(),
			Bytecode:                   []byte{0x60, 0x80},
			CallValue:                  0,
			ConsumeUserResourcePercent: 10,
			OriginEnergyLimit:          1,
		},
	}))
}

func facadeTriggerExt() *api.TransactionExtention {
	return facadeBaseExt(facadeContractAny(core.Transaction_Contract_TriggerSmartContract, &core.TriggerSmartContract{
		OwnerAddress:    facadeFrom.Bytes(),
		ContractAddress: facadeTo.Bytes(),
		CallValue:       0,
	}))
}

// facadeFoundInfo builds a TransactionInfo "found on chain" answer echoing
// the requested txid, optionally with raw log entries attached.
func facadeFoundInfo(id []byte, logs ...*core.TransactionInfo_Log) *core.TransactionInfo {
	return &core.TransactionInfo{
		Id:              id,
		BlockNumber:     12345,
		BlockTimeStamp:  1700000000000,
		Result:          core.TransactionInfo_SUCESS,
		ContractAddress: []byte{},
		Log:             logs,
	}
}

// abiWord renders n as the 32-byte ABI integer word token.New's decimals()
// call decodes.
func abiWord(n byte) []byte {
	w := make([]byte, 32)
	w[31] = n
	return w
}

// --- alias identity: facade and subpackage types are the SAME types ---

func TestAliasesAreTheSubpackageTypes(t *testing.T) {
	var a Address = tron.MustAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	if a != tron.MustAddress(a.String()) {
		t.Fatal("Address is not tron.Address")
	}
	var s SUN = tron.TRX(1)
	if s != TRX(1) {
		t.Fatal("SUN is not tron.SUN")
	}
	var _ Code = tron.CodeAddressInvalid
	var _ Action = tron.ActionWait
	var _ Error = tron.Error{Code: tron.CodeAmountInvalid}
	var _ Tx = tx.Tx(nil)
	var _ *NativeTx = (*tx.NativeTx)(nil)
	var _ *ContractTx = (*tx.ContractTx)(nil)
	var _ *Receipt = (*tx.Receipt)(nil)
	var _ Log = event.Log{}
}

func TestConstructorsDelegateToSubpackages(t *testing.T) {
	if TRX(2) != tron.TRX(2) {
		t.Fatal("TRX does not delegate to tron.TRX")
	}
	got, err := ParseTRX("1.6")
	want, _ := tron.ParseTRX("1.6")
	if err != nil || got != want {
		t.Fatalf("ParseTRX: got %v/%v, want %v", got, err, want)
	}
	if MustTRX("3") != tron.MustTRX("3") {
		t.Fatal("MustTRX does not delegate")
	}
	ga, err := ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	wa := tron.MustAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	if err != nil || ga != wa {
		t.Fatalf("ParseAddress: %v/%v", ga, err)
	}
	if MustAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb") != wa {
		t.Fatal("MustAddress does not delegate")
	}
	signer, err := KeyFromHex(facadeKeyHex)
	wantSigner, _ := key.PrivateKeyFromHex(facadeKeyHex)
	if err != nil || signer.Address() != wantSigner.Address() {
		t.Fatalf("KeyFromHex: %v/%v", signer, err)
	}
	const mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	hd, err := KeyFromMnemonic(mnemonic, "", "m/44'/195'/0'/0/0")
	wantHD, _ := key.PrivateKeyFromMnemonic(mnemonic, "", "m/44'/195'/0'/0/0")
	if err != nil || hd.Address() != wantHD.Address() {
		t.Fatalf("KeyFromMnemonic: %v/%v", hd, err)
	}
}

// --- client plumbing ---

func TestDialRejectsBadEndpointLikeRPC(t *testing.T) {
	_, err := Dial(context.Background(), "http://nope")
	if !tron.HasCode(err, tron.CodeChainConnection) {
		t.Fatalf("Dial(http://nope) = %v, want chain.connection", err)
	}
}

func TestDialCloseRawEndpoint(t *testing.T) {
	c, err := Dial(context.Background(), "grpc://localhost:50051")
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if c.Raw() == nil || c.Endpoint() != "grpc://localhost:50051" {
		t.Fatalf("Raw/Endpoint: %q", c.Endpoint())
	}
	if !c.Raw().IsConnected() {
		t.Fatal("client should be open after Dial")
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if c.Raw().IsConnected() {
		t.Fatal("Close did not reach the underlying rpc.Client")
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// --- read delegations (wire path through the bufconn fake) ---

func TestChainTipDelegatesToGetNowBlock2(t *testing.T) {
	c := newFacadeTestClient(t, &fakeFacadeServer{})
	tip, err := c.ChainTip(context.Background())
	if err != nil || tip != 4242 {
		t.Fatalf("ChainTip: %d/%v", tip, err)
	}
}

func TestTronBalanceDelegatesToGetAccount(t *testing.T) {
	c := newFacadeTestClient(t, &fakeFacadeServer{})
	bal, err := c.TronBalance(context.Background(), facadeFrom)
	if err != nil || bal != SUN(1_500_000) {
		t.Fatalf("TronBalance: %d/%v", bal, err)
	}
}

func TestWitnessesDelegatesToGetPaginatedNowWitnessList(t *testing.T) {
	var gotPage *api.PaginatedMessage
	f := &fakeFacadeServer{}
	f.WitnessPage = func(ctx context.Context, in *api.PaginatedMessage) (*api.WitnessList, error) {
		gotPage = in
		return &api.WitnessList{Witnesses: []*core.Witness{{Address: facadeFrom.Bytes(), VoteCount: 99, IsJobs: true}}}, nil
	}
	c := newFacadeTestClient(t, f)
	ws, err := c.Witnesses(context.Background(), Page{Offset: 10, Limit: 2})
	if err != nil {
		t.Fatalf("Witnesses: %v", err)
	}
	if gotPage == nil || gotPage.GetOffset() != 10 || gotPage.GetLimit() != 2 {
		t.Fatalf("pagination did not pass through: %+v", gotPage)
	}
	if len(ws) != 1 || ws[0].Address != facadeFrom || ws[0].VoteCount != 99 || !ws[0].IsJobs {
		t.Fatalf("decoded witnesses: %+v", ws)
	}
}

// --- transfer/build delegations ---

func TestTransferTRXDelegatesToBuildTransfer(t *testing.T) {
	var gotReq *core.TransferContract
	f := &fakeFacadeServer{}
	f.CreateTx2 = func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
		gotReq = in
		return facadeTransferExt(), nil
	}
	c := newFacadeTestClient(t, f)
	native, err := c.TransferTRX(context.Background(), facadeFrom, facadeTo, TRX(1))
	if err != nil {
		t.Fatalf("TransferTRX: %v", err)
	}
	if gotReq.GetOwnerAddress() == nil || string(gotReq.GetToAddress()) != string(facadeTo.Bytes()) || gotReq.GetAmount() != 1_000_000 {
		t.Fatalf("request did not reach BuildTransfer's RPC: %+v", gotReq)
	}
	if native.ID() == "" || native.Kind() != tx.KindNative {
		t.Fatalf("returned tx: id %q kind %v", native.ID(), native.Kind())
	}
}

func TestTransferTokenDelegatesToBuildAssetTransfer(t *testing.T) {
	var gotReq *core.TransferAssetContract
	f := &fakeFacadeServer{}
	f.TransferAssetFn = func(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error) {
		gotReq = in
		return facadeAssetExt(), nil
	}
	c := newFacadeTestClient(t, f)
	asset, err := c.TransferToken(context.Background(), facadeFrom, facadeTo, "1000001", 5)
	if err != nil {
		t.Fatalf("TransferToken: %v", err)
	}
	if string(gotReq.GetAssetName()) != "1000001" || gotReq.GetAmount() != 5 {
		t.Fatalf("request did not reach BuildAssetTransfer's RPC: %+v", gotReq)
	}
	if asset.Kind() != tx.KindAssetTransfer {
		t.Fatalf("returned kind %v", asset.Kind())
	}
}

func TestDeployDelegatesToBuildDeploy(t *testing.T) {
	var gotReq *core.CreateSmartContract
	f := &fakeFacadeServer{}
	f.Deploy = func(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
		gotReq = in
		return facadeDeployExt(), nil
	}
	c := newFacadeTestClient(t, f)
	dep, err := c.Deploy(context.Background(), facadeFrom, tx.DeployParams{Bytecode: []byte{0x60, 0x80}, OriginEnergyLimit: 1, ConsumeUserResourcePercent: 10})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(gotReq.GetNewContract().GetBytecode()) == 0 {
		t.Fatal("request did not reach BuildDeploy's RPC")
	}
	if dep.ID() == "" || dep.Kind() != tx.KindDeploy {
		t.Fatalf("returned tx: id %q kind %v", dep.ID(), dep.Kind())
	}
}

// --- broadcast/wait delegations ---

func TestBroadcastAndWaitDelegateThroughFacade(t *testing.T) {
	f := &fakeFacadeServer{}
	c := newFacadeTestClient(t, f)
	signer, err := KeyFromHex(facadeKeyHex)
	if err != nil {
		t.Fatalf("KeyFromHex: %v", err)
	}
	native, err := c.TransferTRX(context.Background(), facadeFrom, facadeTo, TRX(1))
	if err != nil {
		t.Fatalf("TransferTRX: %v", err)
	}
	signed, err := native.Sign(signer)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	rec, err := c.Broadcast(context.Background(), signed)
	if err != nil || !rec.OK() {
		t.Fatalf("Broadcast: %v/%+v", err, rec)
	}
	if f.broadcastCalls != 1 {
		t.Fatalf("BroadcastTransaction called %d times, want 1", f.broadcastCalls)
	}
	f.TxInfo = func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
		return facadeFoundInfo(in.GetValue()), nil
	}
	got, err := c.Wait(context.Background(), rec.TxID)
	if err != nil || !got.OK() || got.BlockNum != 12345 {
		t.Fatalf("Wait: %v/%+v", err, got)
	}
}

func TestWaitForSolidDelegatesToSolidityEndpoint(t *testing.T) {
	f := &fakeFacadeServer{}
	f.TxInfoSolidity = func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
		return facadeFoundInfo(in.GetValue()), nil
	}
	c := newFacadeTestClient(t, f)
	got, err := c.WaitForSolid(context.Background(), "aabb")
	if err != nil {
		t.Fatalf("WaitForSolid: %v", err)
	}
	if !got.Solidified() {
		t.Fatal("WaitForSolid did not route through the Solidity endpoint")
	}
}

// --- cost delegations ---

func TestCostPreviewDelegatesToPreviewCost(t *testing.T) {
	c := newFacadeTestClient(t, &fakeFacadeServer{})
	txr, err := tx.BuildTriggerSmartContract(c.Raw(), context.Background(), facadeFrom, facadeTo, nil, 0)
	if err != nil {
		t.Fatalf("BuildTriggerSmartContract: %v", err)
	}
	pv, err := c.CostPreview(context.Background(), txr, facadeFrom)
	if err != nil {
		t.Fatalf("CostPreview: %v", err)
	}
	if pv.EnergyNeeded != 5000 || pv.EnergyAvailable != 0 || pv.TronToBurn != SUN(5000*420) {
		t.Fatalf("preview fields: %+v", pv)
	}
}

func TestEnergyPriceDelegatesToEnergyPriceOf(t *testing.T) {
	c := newFacadeTestClient(t, &fakeFacadeServer{})
	p, err := c.EnergyPrice(context.Background())
	if err != nil {
		t.Fatalf("EnergyPrice: %v", err)
	}
	if p.SunPerEnergy != 420 || p.EffectiveAt.UnixMilli() != 1691500000000 || p.FetchedAt.IsZero() {
		t.Fatalf("energy price: %+v", p)
	}
}

// --- events delegation ---

func TestEventsDelegatesToLogsFor(t *testing.T) {
	f := &fakeFacadeServer{}
	f.TxInfo = func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
		return facadeFoundInfo(in.GetValue(), &core.TransactionInfo_Log{
			Address: facadeTo.Bytes(),
			Topics:  [][]byte{make([]byte, 32)},
			Data:    []byte{0xde, 0xad},
		}), nil
	}
	c := newFacadeTestClient(t, f)
	logs, err := c.Events(context.Background(), "aabb")
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("got %d logs, want 1", len(logs))
	}
	if logs[0].Address != facadeTo || string(logs[0].Data) != "\xde\xad" {
		t.Fatalf("log fields: %+v", logs[0])
	}
	// Unknown signature must materialize leniently, never drop the log.
	if logs[0].EventName != "" {
		t.Fatalf("unexpected EventName %q", logs[0].EventName)
	}
}

// --- token/contract delegation ---

func TestTokenDelegatesToTokenNew(t *testing.T) {
	f := &fakeFacadeServer{}
	f.TriggerConstant = func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
		ext := facadeTriggerExt()
		ext.ConstantResult = [][]byte{abiWord(6)}
		return ext, nil
	}
	c := newFacadeTestClient(t, f)
	h, err := c.Token(context.Background(), facadeTo)
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if h.Decimals() != 6 || h.Contract() != facadeTo {
		t.Fatalf("handle: decimals %d contract %v", h.Decimals(), h.Contract())
	}
}

func TestContractDelegatesToNewInstance(t *testing.T) {
	c := newFacadeTestClient(t, &fakeFacadeServer{})
	inst, err := c.Contract(context.Background(), facadeTo)
	if err != nil || inst == nil {
		t.Fatalf("Contract: %v/%v", inst, err)
	}
}
