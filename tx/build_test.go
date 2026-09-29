package tx

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// Builders hit the four build RPCs through the fake; inputs are validated
// BEFORE any I/O (v1's reject-early semantics).

func TestBuildTransferHappyPath(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	ntx, err := BuildTransfer(cp, t.Context(), testFrom, testTo, 1_000_000)
	if err != nil {
		t.Fatalf("BuildTransfer: %v", err)
	}
	if ntx.Kind() != KindNative {
		t.Errorf("Kind = %v, want native", ntx.Kind())
	}
	if got := ntx.FeeLimit(); got != 150_000_000 {
		t.Errorf("FeeLimit = %d, want the 150_000_000 default", got)
	}
	// the wrapped raw_data carries the request
	c := ntx.Transaction().GetRawData().GetContract()
	if len(c) != 1 || c[0].GetType() != core.Transaction_Contract_TransferContract {
		t.Fatalf("wrapped contract = %v, want one TransferContract", c)
	}
	tr := &core.TransferContract{}
	if err := c[0].GetParameter().UnmarshalTo(tr); err != nil {
		t.Fatalf("parameter does not decode as TransferContract: %v", err)
	}
	if tr.Amount != 1_000_000 || string(tr.OwnerAddress) != string(testFrom.Bytes()) || string(tr.ToAddress) != string(testTo.Bytes()) {
		t.Errorf("TransferContract = %+v, want the request echoed", tr)
	}
}

func TestBuildTransferRejectsBadInputs(t *testing.T) {
	cases := []struct {
		name     string
		from, to tron.Address
		amt      tron.SUN
		want     tron.Code
	}{
		{"zero from", tron.Address{}, testTo, 1, tron.CodeAddressInvalid},
		{"zero to", testFrom, tron.Address{}, 1, tron.CodeAddressInvalid},
		{"negative amount", testFrom, testTo, -1, tron.CodeAmountNegative},
		{"zero amount", testFrom, testTo, 0, tron.CodeAmountInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// No client: validation must reject before any I/O.
			_, err := BuildTransfer(nil, t.Context(), tc.from, tc.to, tc.amt)
			if !tron.HasCode(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestBuildTriggerSmartContractRejectsBadInputs(t *testing.T) {
	if _, err := BuildTriggerSmartContract(nil, t.Context(), tron.Address{}, testTo, nil, 0); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero owner: err = %v, want address.invalid", err)
	}
	if _, err := BuildTriggerSmartContract(nil, t.Context(), testFrom, tron.Address{}, nil, 0); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero contract: err = %v, want address.invalid", err)
	}
	if _, err := BuildTriggerSmartContract(nil, t.Context(), testFrom, testTo, nil, -1); !tron.HasCode(err, tron.CodeAmountNegative) {
		t.Errorf("negative call value: err = %v, want amount.negative", err)
	}
}

func TestBuildDeployValidatesParams(t *testing.T) {
	cases := []struct {
		name string
		p    DeployParams
		want tron.Code
	}{
		{"empty bytecode", DeployParams{Bytecode: nil}, tron.CodeContractBadABI},
		{"negative call value", DeployParams{Bytecode: []byte{0x60}, CallValue: -1}, tron.CodeAmountNegative},
		{"resource percent high", DeployParams{Bytecode: []byte{0x60}, ConsumeUserResourcePercent: 101}, tron.CodeTxInvalidArgument},
		{"resource percent negative", DeployParams{Bytecode: []byte{0x60}, ConsumeUserResourcePercent: -1}, tron.CodeTxInvalidArgument},
		{"negative origin energy", DeployParams{Bytecode: []byte{0x60}, OriginEnergyLimit: -1}, tron.CodeTxInvalidArgument},
		{"control-char name", DeployParams{Bytecode: []byte{0x60}, Name: "bad\x01name"}, tron.CodeTxInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildDeploy(nil, t.Context(), testFrom, tc.p)
			if !tron.HasCode(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
	// zero owner
	if _, err := BuildDeploy(nil, t.Context(), tron.Address{}, DeployParams{Bytecode: []byte{0x60}}); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero owner: err = %v, want address.invalid", err)
	}
	// empty name is allowed (v1 semantics)
	cp := newTxTestClient(t, &fakeWalletServer{})
	if _, err := BuildDeploy(cp, t.Context(), testFrom, DeployParams{Bytecode: []byte{0x60}, Name: ""}); err != nil {
		t.Errorf("empty name rejected: %v", err)
	}
}

func TestBuildAssetTransferValidatesParams(t *testing.T) {
	if _, err := BuildAssetTransfer(nil, t.Context(), testFrom, testTo, "", 1); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("empty asset name: err = %v, want tx.invalid_argument", err)
	}
	if _, err := BuildAssetTransfer(nil, t.Context(), testFrom, testTo, "1000001", -1); !tron.HasCode(err, tron.CodeAmountNegative) {
		t.Errorf("negative qty: err = %v, want amount.negative", err)
	}
	if _, err := BuildAssetTransfer(nil, t.Context(), testFrom, testTo, "1000001", 0); !tron.HasCode(err, tron.CodeAmountInvalid) {
		t.Errorf("zero qty: err = %v, want amount.invalid", err)
	}
	if _, err := BuildAssetTransfer(nil, t.Context(), testFrom, testFrom, "1000001", 1); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("self transfer: err = %v, want tx.invalid_argument", err)
	}
	if _, err := BuildAssetTransfer(nil, t.Context(), tron.Address{}, testTo, "1000001", 1); !tron.HasCode(err, tron.CodeAddressInvalid) {
		t.Errorf("zero from: err = %v, want address.invalid", err)
	}
}

func TestBuildTransferNodeErrorMapped(t *testing.T) {
	f := &fakeWalletServer{
		CreateTx2: func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: &api.Return{Result: false, Code: api.Return_TAPOS_ERROR, Message: []byte("tapos")}}, nil
		},
	}
	cp := newTxTestClient(t, f)
	_, err := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	if !tron.HasCode(err, tron.CodeTxTaposInvalid) {
		t.Fatalf("err = %v, want tx.tapos_invalid", err)
	}
	// A transport-level failure maps to chain.timeout.
	f2 := &fakeWalletServer{
		CreateTx2: func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
			return nil, status.Error(codes.DeadlineExceeded, "fake deadline")
		},
	}
	cp2 := newTxTestClient(t, f2)
	_, err = BuildTransfer(cp2, t.Context(), testFrom, testTo, 1)
	if !tron.HasCode(err, tron.CodeChainTimeout) {
		t.Fatalf("err = %v, want chain.timeout", err)
	}
}

func TestBuildResponseWithoutRawDataRejected(t *testing.T) {
	f := &fakeWalletServer{
		CreateTx2: func(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{Result: okResult()}, nil
		},
	}
	cp := newTxTestClient(t, f)
	_, err := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	if !tron.HasCode(err, tron.CodeRPCMethodFailed) {
		t.Fatalf("err = %v, want rpc.method_failed (raw-data precondition)", err)
	}
}

func TestWithFeeLimitCopyOnWrite(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	contract, err := BuildTriggerSmartContract(cp, t.Context(), testFrom, testTo, []byte{0x01}, 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	before := contract.ID()
	copied := contract.WithFeeLimit(500_000)
	if got := copied.FeeLimit(); got != 500_000 {
		t.Errorf("copy FeeLimit = %d, want 500_000", got)
	}
	if got := contract.FeeLimit(); got != DefaultFeeLimit {
		t.Errorf("ORIGINAL FeeLimit = %d, want unchanged %d", got, DefaultFeeLimit)
	}
	if contract.ID() != before {
		t.Error("original raw_data changed by WithFeeLimit")
	}
	// the copy's raw_data really changed
	if copied.ID() == before {
		t.Error("copy has the original txid; fee limit was not applied")
	}
}

func TestWithExpirationAndPermissionIDCopyOnWrite(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	native, _ := BuildTransfer(cp, t.Context(), testFrom, testTo, 1)
	asset, _ := BuildAssetTransfer(cp, t.Context(), testFrom, testTo, "1000001", 1)
	deploy, _ := BuildDeploy(cp, t.Context(), testFrom, DeployParams{Bytecode: []byte{0x60}})

	future := time.Now().Add(10 * time.Minute).UnixMilli()
	n2 := native.WithExpiration(10 * time.Minute)
	if got := n2.Expiration().UnixMilli(); got < future-1000 || got > future+1000 {
		t.Errorf("copy Expiration = %d, want ~now+10m", got)
	}
	if native.Expiration().UnixMilli() > future-9*time.Minute.Milliseconds() {
		t.Errorf("ORIGINAL Expiration = %d; WithExpiration mutated the receiver", native.Expiration().UnixMilli())
	}
	a2, err := asset.WithPermissionID(3)
	if err != nil {
		t.Fatalf("asset.WithPermissionID: %v", err)
	}
	if a2.PermissionID() != 3 {
		t.Errorf("asset copy PermissionID = %d, want 3", a2.PermissionID())
	} else if asset.PermissionID() != 0 {
		t.Errorf("asset ORIGINAL PermissionID = %d, want 0", asset.PermissionID())
	}
	if d2 := deploy.WithExpiration(5 * time.Minute); d2.Expiration().IsZero() || d2.Expiration().UnixMilli() < future-5*time.Minute.Milliseconds() {
		t.Errorf("deploy copy Expiration = %v, want ~now+5m", d2.Expiration())
	}
}

func TestDeployParamMutationsCopyOnWrite(t *testing.T) {
	cp := newTxTestClient(t, &fakeWalletServer{})
	deploy, err := BuildDeploy(cp, t.Context(), testFrom, DeployParams{Bytecode: []byte{0x60}, OriginEnergyLimit: 1, ConsumeUserResourcePercent: 10})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	before := deploy.ID()
	d2, err := deploy.WithOriginEnergyLimit(777)
	if err != nil {
		t.Fatalf("WithOriginEnergyLimit: %v", err)
	}
	d3, err := deploy.WithResourcePercent(42)
	if err != nil {
		t.Fatalf("WithResourcePercent: %v", err)
	}
	if deploy.ID() != before {
		t.Fatal("receiver mutated by WithOriginEnergyLimit/WithResourcePercent")
	}
	read := func(tx *DeployTx) (int64, int64) {
		p := &core.CreateSmartContract{}
		c := tx.Transaction().GetRawData().GetContract()[0]
		if err := c.GetParameter().UnmarshalTo(p); err != nil {
			t.Fatalf("decode deploy param: %v", err)
		}
		return p.GetNewContract().GetOriginEnergyLimit(), p.GetNewContract().GetConsumeUserResourcePercent()
	}
	oel, pct := read(d2)
	if oel != 777 || pct != 10 {
		t.Errorf("d2 = (%d,%d), want (777,10)", oel, pct)
	}
	oel, pct = read(d3)
	if oel != 1 || pct != 42 {
		t.Errorf("d3 = (%d,%d), want (1,42)", oel, pct)
	}
	oel, pct = read(deploy)
	if oel != 1 || pct != 10 {
		t.Errorf("original = (%d,%d), want (1,10)", oel, pct)
	}
}
