package tx

// Tests for DeployTx.Estimate: the constant-call deploy path (empty
// contract address, init bytecode as data) mapped onto DeployEstimate.
// The request-shape test pins the wire contract the node depends on —
// a non-empty contract address would execute a CALL instead of a CREATE
// dry run, silently estimating the wrong operation. Also covers the
// contract-type guard on the post-build parameter mutations (With*).

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"google.golang.org/protobuf/proto"
)

func mustDeployTx(t *testing.T, f *fakeWalletServer) *DeployTx {
	t.Helper()
	dtx, err := BuildDeploy(newTxTestClient(t, f), t.Context(),
		testFrom, DeployParams{Bytecode: []byte{0x60, 0x80}})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return dtx
}

func TestDeployEstimateRequestShape(t *testing.T) {
	var gotReq *core.TriggerSmartContract
	f := &fakeWalletServer{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			gotReq = in
			ext := triggerExt()
			ext.EnergyUsed = 221
			ext.EnergyPenalty = 0
			return ext, nil
		},
	}
	est, err := mustDeployTx(t, f).Estimate(t.Context())
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if est.Energy != 221 || est.Penalty != 0 || est.Code != "" {
		t.Fatalf("Estimate = %+v, want energy 221 penalty 0 no code", est)
	}
	if gotReq == nil {
		t.Fatal("the estimator never reached TriggerConstantContract")
	}
	// The empty contract address IS the deploy path: a set address would
	// run a CALL against an existing contract instead.
	if len(gotReq.GetContractAddress()) != 0 {
		t.Errorf("contract address = %x, want empty (deploy estimation)", gotReq.GetContractAddress())
	}
	if string(gotReq.GetOwnerAddress()) != string(testFrom.Bytes()) {
		t.Errorf("owner = %x, want %x", gotReq.GetOwnerAddress(), testFrom.Bytes())
	}
	if string(gotReq.GetData()) != string([]byte{0x60, 0x80}) {
		t.Errorf("data = %x, want the built bytecode", gotReq.GetData())
	}
}

func TestDeployEstimateRejectionInBand(t *testing.T) {
	f := &fakeWalletServer{
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{Result: false, Code: api.Return_CONTRACT_EXE_ERROR, Message: []byte("REVERT opcode executed")},
			}, nil
		},
	}
	est, err := mustDeployTx(t, f).Estimate(t.Context())
	if err != nil {
		t.Fatalf("node rejection must be in-band, got error: %v", err)
	}
	if est.Code != tron.CodeReceiptReverted {
		t.Errorf("Code = %q, want receipt.reverted", est.Code)
	}
}

func TestDeployEstimateWrongKind(t *testing.T) {
	dtx := mustDeployTx(t, &fakeWalletServer{})
	dtx.baseTx.ext = transferExt() // simulate an Extension()-swapped payload
	if _, err := dtx.Estimate(t.Context()); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("non-create payload: want tx.invalid_argument, got %v", err)
	}
	if _, err := (*DeployTx)(nil).Estimate(t.Context()); !tron.HasCode(err, tron.CodeTxInvalidArgument) {
		t.Errorf("nil receiver: want tx.invalid_argument, got %v", err)
	}
}

// TestDeployMutateRejectsNonCreatePayload is the same type-check regression
// on the post-build parameter mutations (deployParam decodes the wrapped
// parameter without a contract-type check). A one-field payload such as
// WithdrawBalanceContract{owner} decodes CLEANLY as a CreateSmartContract —
// field 1 is the same owner_address — so deployParam's decode guard never
// fires and With* re-encodes a deploy-shaped parameter over the withdrawal:
// a silently corrupted transaction whose contract Type still says
// WithdrawBalanceContract.
func TestDeployMutateRejectsNonCreatePayload(t *testing.T) {
	mutations := map[string]func(*DeployTx) *DeployTx{
		"WithOriginEnergyLimit": func(d *DeployTx) *DeployTx { return d.WithOriginEnergyLimit(1_000) },
		"WithResourcePercent":   func(d *DeployTx) *DeployTx { return d.WithResourcePercent(50) },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			dtx := mustDeployTx(t, &fakeWalletServer{})
			// Simulate an Extension()-swapped payload whose wire shape decodes
			// cleanly as CreateSmartContract.
			dtx.baseTx.ext = baseExt(contractAny(core.Transaction_Contract_WithdrawBalanceContract,
				&core.WithdrawBalanceContract{OwnerAddress: testFrom.Bytes()}), okResult())
			var recovered any
			var rewritten string
			func() {
				defer func() { recovered = recover() }()
				out := mutate(dtx)
				b := out.raw().GetContract()[0].GetParameter().GetValue()
				var back core.CreateSmartContract
				err := proto.Unmarshal(b, &back)
				rewritten = fmt.Sprintf("type=%v paramLen=%d decodesAsCreate(err=%v, originEnergyLimit=%d, consumePercent=%d)",
					out.raw().GetContract()[0].GetType(), len(b), err,
					back.GetNewContract().GetOriginEnergyLimit(), back.GetNewContract().GetConsumeUserResourcePercent())
			}()
			if recovered == nil {
				t.Fatalf("%s on a non-CreateSmartContract payload: want the loud panic deployParam already uses for undecodable parameters, got a silent parameter rewrite (%s)", name, rewritten)
			}
			if s := fmt.Sprint(recovered); !strings.Contains(s, "CreateSmartContract") || !strings.Contains(s, "Extension()") {
				t.Errorf("panic = %q, want it to name CreateSmartContract and the Extension() escape hatch", s)
			}
		})
	}
}

// TestDeployMutateWrongTypeOutrunsDecodeGuard pins the ordering: a
// TransferContract payload fails the strict decode by accident (its 21-byte
// recipient does not parse as the nested SmartContract), so the old diagnosis
// blamed the parameter instead of the type. The enum check must name the type.
func TestDeployMutateWrongTypeOutrunsDecodeGuard(t *testing.T) {
	dtx := mustDeployTx(t, &fakeWalletServer{})
	dtx.baseTx.ext = transferExt() // wire-incompatible non-Create payload
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		dtx.WithOriginEnergyLimit(1_000)
	}()
	if recovered == nil {
		t.Fatal("want a panic on a TransferContract payload")
	}
	if s := fmt.Sprint(recovered); !strings.Contains(s, "not a CreateSmartContract") {
		t.Errorf("panic = %q, want the contract-type diagnosis, not a decode complaint", s)
	}
}
