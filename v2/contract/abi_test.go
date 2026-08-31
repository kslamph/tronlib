package contract

// Tests for the pb→JSON ABI bridge (abi.go), focused on stateMutability:
// geth's parser rejects a "receive" entry without
// "stateMutability":"payable", so the pb mutability (enum plus the legacy
// Payable/Constant bools) must survive the rendering or every Solidity
// >=0.6 payable contract fails the lazy ABI load.

import (
	"context"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
)

// mutPbABI hand-builds a pb ABI covering one entry per mutability shape:
// each enum value, the legacy Payable/Constant bools, and the empty case.
func mutPbABI() *core.SmartContract_ABI {
	return &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{Name: "payFn", Type: core.SmartContract_ABI_Entry_Function, StateMutability: core.SmartContract_ABI_Entry_Payable},
			{Name: "viewFn", Type: core.SmartContract_ABI_Entry_Function, StateMutability: core.SmartContract_ABI_Entry_View},
			{Name: "pureFn", Type: core.SmartContract_ABI_Entry_Function, StateMutability: core.SmartContract_ABI_Entry_Pure},
			{Name: "npFn", Type: core.SmartContract_ABI_Entry_Function, StateMutability: core.SmartContract_ABI_Entry_Nonpayable},
			{Name: "legacyConst", Type: core.SmartContract_ABI_Entry_Function, Constant: true},
			{Name: "legacyPay", Type: core.SmartContract_ABI_Entry_Function, Payable: true},
			{Name: "bareFn", Type: core.SmartContract_ABI_Entry_Function}, // no mutability at all
		},
	}
}

// TestPbABIToJSONStateMutability: each pb mutability spelling renders into
// the JSON ABI geth parses — enum values directly, legacy bools by
// derivation, and receive entries forced payable.
func TestPbABIToJSONStateMutability(t *testing.T) {
	got, err := pbABIToJSON(mutPbABI())
	if err != nil {
		t.Fatalf("pbABIToJSON: %v", err)
	}
	for _, want := range []string{
		`"name":"payFn","stateMutability":"payable"`,
		`"name":"viewFn","stateMutability":"view"`,
		`"name":"pureFn","stateMutability":"pure"`,
		`"name":"npFn","stateMutability":"nonpayable"`,
		`"name":"legacyConst","stateMutability":"view"`,  // legacy Constant bool
		`"name":"legacyPay","stateMutability":"payable"`, // legacy Payable bool
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered ABI %q: missing %s", got, want)
		}
	}
	// No mutability at all stays absent (geth reads it as nonpayable).
	if strings.Contains(got, `"name":"bareFn","stateMutability"`) {
		t.Errorf("rendered ABI %q: bareFn should carry no stateMutability", got)
	}
}

// TestPbABIToJSONReceiveForcedPayable: a receive entry with no mutability
// must render "payable" — geth rejects any other receive spelling.
func TestPbABIToJSONReceiveForcedPayable(t *testing.T) {
	abi := &core.SmartContract_ABI{Entrys: []*core.SmartContract_ABI_Entry{
		{Type: core.SmartContract_ABI_Entry_Receive}, // StateMutability unset, as nodes return it
	}}
	got, err := pbABIToJSON(abi)
	if err != nil {
		t.Fatalf("pbABIToJSON: %v", err)
	}
	if !strings.Contains(got, `"type":"receive","stateMutability":"payable"`) {
		t.Errorf("rendered ABI %q: receive entry must render stateMutability payable", got)
	}
}

// TestPbABIToJSONFallbackUnsetMutability: geth accepts a fallback with no
// mutability — it must render without one (not invented).
func TestPbABIToJSONFallbackUnsetMutability(t *testing.T) {
	abi := &core.SmartContract_ABI{Entrys: []*core.SmartContract_ABI_Entry{
		{Type: core.SmartContract_ABI_Entry_Fallback},
	}}
	got, err := pbABIToJSON(abi)
	if err != nil {
		t.Fatalf("pbABIToJSON: %v", err)
	}
	if strings.Contains(got, `"stateMutability"`) {
		t.Errorf("rendered ABI %q: unset fallback must carry no stateMutability", got)
	}
}

// receivePbABI is the fixture the lazy fetch serves: a normal view
// function plus a receive() entry whose StateMutability the node left
// unset — the shape that fails geth's parser without the mutability
// mapping.
func receivePbABI() *core.SmartContract_ABI {
	return &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{Name: "get", Type: core.SmartContract_ABI_Entry_Function, StateMutability: core.SmartContract_ABI_Entry_View,
				Outputs: []*core.SmartContract_ABI_Entry_Param{{Type: "uint256"}}},
			{Type: core.SmartContract_ABI_Entry_Receive},
		},
	}
}

// TestLazyFetchReceiveEntry: an on-chain ABI containing receive() (the
// Solidity >=0.6 payable shape) must load lazily and drive a call — not
// fail with contract.bad_abi the way an unmapped mutability would.
func TestLazyFetchReceiveEntry(t *testing.T) {
	f := &fakeWallet{
		GetContractFn: func(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{Abi: receivePbABI()}, nil
		},
		TriggerConstant: func(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExtention()
			ext.ConstantResult = [][]byte{abiUint256(42)}
			return ext, nil
		},
	}
	i, err := NewInstance(newContractTestClient(t, f), testContractAddress)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	res, err := i.Call(context.Background(), "get")
	if err != nil {
		t.Fatalf("Call(get) with a receive()-entry on-chain ABI: %v", err)
	}
	v, err := res.BigInt()
	if err != nil || v.Int64() != 42 {
		t.Errorf("BigInt() = (%v, %v), want (42, nil)", v, err)
	}
	if got := i.Methods(); len(got) != 1 || got[0] != "get" {
		t.Errorf("Methods() = %v, want [get] (receive is not a callable method)", got)
	}
	if !strings.Contains(i.ABI(), `"type":"receive"`) {
		t.Errorf("ABI() = %q: missing the receive entry", i.ABI())
	}
	if f.getContractCalls.Load() != 1 {
		t.Errorf("GetContract called %d times, want exactly 1", f.getContractCalls.Load())
	}
}
