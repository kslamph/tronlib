package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
)

// TestContractWrappers exercises the 1:1 port of lowlevel/contract.go through
// the bufconn fake (happy path + node-error path per wrapper).
func TestContractWrappers(t *testing.T) {
	runWrapperGroup(t, []wrapperCase{
		{"DeployContract", "DeployContract", "deploy contract", func(cp ConnProvider, ctx context.Context) error {
			_, err := DeployContract(cp, ctx, &core.CreateSmartContract{})
			return err
		}},
		{"TriggerContract", "TriggerContract", "trigger contract", func(cp ConnProvider, ctx context.Context) error {
			_, err := TriggerContract(cp, ctx, &core.TriggerSmartContract{})
			return err
		}},
		{"TriggerConstantContract", "TriggerConstantContract", "trigger constant contract", func(cp ConnProvider, ctx context.Context) error {
			_, err := TriggerConstantContract(cp, ctx, &core.TriggerSmartContract{})
			return err
		}},
		{"EstimateEnergy", "EstimateEnergy", "estimate energy", func(cp ConnProvider, ctx context.Context) error {
			_, err := EstimateEnergy(cp, ctx, &core.TriggerSmartContract{})
			return err
		}},
		{"GetContract", "GetContract", "get contract", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetContract(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"GetContractInfo", "GetContractInfo", "get contract info", func(cp ConnProvider, ctx context.Context) error {
			_, err := GetContractInfo(cp, ctx, &api.BytesMessage{})
			return err
		}},
		{"UpdateSetting", "UpdateSetting", "update setting", func(cp ConnProvider, ctx context.Context) error {
			_, err := UpdateSetting(cp, ctx, &core.UpdateSettingContract{})
			return err
		}},
		{"UpdateEnergyLimit", "UpdateEnergyLimit", "update energy limit", func(cp ConnProvider, ctx context.Context) error {
			_, err := UpdateEnergyLimit(cp, ctx, &core.UpdateEnergyLimitContract{})
			return err
		}},
		{"ClearContractABI", "ClearContractABI", "clear contract abi", func(cp ConnProvider, ctx context.Context) error {
			_, err := ClearContractABI(cp, ctx, &core.ClearABIContract{})
			return err
		}},
	})
}

// TestDeployContractHappyValue pins that the fake's canned response is
// returned as-is through TxCall's validator (DeployContract is the seam Task
// 6's DeployTx builds on).
func TestDeployContractHappyValue(t *testing.T) {
	want := okExtention()
	want.Result.Message = []byte("fake-deploy-return")
	srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
		"DeployContract": func(ctx context.Context, in any) (any, error) { return want, nil },
	}}
	c := newBufconnClient(t, srv, time.Second)
	got, err := DeployContract(c, context.Background(), &core.CreateSmartContract{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Bufconn serializes over the wire, so compare by value not pointer.
	if string(got.GetResult().GetMessage()) != "fake-deploy-return" {
		t.Fatalf("got %+v, want the fake's canned extention (Return.message fake-deploy-return)", got)
	}
}
