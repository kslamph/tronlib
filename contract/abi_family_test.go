package contract

import (
	"context"
	"testing"

	"golang.org/x/crypto/sha3"

	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

func TestUseABIFamilyReusesABIForSameCode(t *testing.T) {
	code := []byte{0x60, 0x80, 0x60, 0x40}
	familyABI := `[{
	  "type":"function","name":"noop","stateMutability":"view",
	  "inputs":[],"outputs":[]
	}, {
	  "type":"event","name":"FamilyPing","anonymous":false,
	  "inputs":[{"name":"value","type":"uint256","indexed":true}]
	}]`
	f := &fakeWallet{
		GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{Bytecode: code}, nil
		},
	}
	client := newContractTestClient(t, f)

	first, err := NewInstance(client, testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance(first): %v", err)
	}
	if err := first.UseABIFamily(context.Background(), familyABI); err != nil {
		t.Fatalf("UseABIFamily: %v", err)
	}

	second, err := NewInstance(client, mustAddr(0x22))
	if err != nil {
		t.Fatalf("NewInstance(second): %v", err)
	}
	if _, err := second.Call(context.Background(), "noop"); err != nil {
		t.Fatalf("Call with reused family ABI: %v", err)
	}
	if got := second.Methods(); len(got) != 1 || got[0] != "noop" {
		t.Fatalf("second Methods() = %v, want the family ABI's noop method", got)
	}
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte("FamilyPing(uint256)"))
	log, err := event.DecodeFor(testContractAddress, [][]byte{hash.Sum(nil), abiUint256(1)}, nil)
	if err != nil || log.EventName != "FamilyPing" {
		t.Fatalf("family ABI event scope: log=%+v err=%v, want FamilyPing", log, err)
	}
	if got := f.getContractCalls.Load(); got != 2 {
		t.Fatalf("GetContract calls = %d, want one registration fetch plus one lazy family lookup", got)
	}
}

func TestUseABIFamilyRejectsBadABI(t *testing.T) {
	f := &fakeWallet{
		GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{Bytecode: []byte{0x60, 0x81}}, nil
		},
	}
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	if err := i.UseABIFamily(context.Background(), `[{"type":"function","name":"broken","inputs":[{"type":"notatype"}]}]`); !tron.HasCode(err, tron.CodeContractBadABI) {
		t.Fatalf("UseABIFamily(bad ABI): err = %v, want contract.bad_abi", err)
	}
}
