package tx

// Tests for DeployTx.Estimate: the constant-call deploy path (empty
// contract address, init bytecode as data) mapped onto DeployEstimate.
// The request-shape test pins the wire contract the node depends on —
// a non-empty contract address would execute a CALL instead of a CREATE
// dry run, silently estimating the wrong operation.

import (
	"context"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
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
