package eventtool

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// Real mainnet addresses used only as opaque identifiers in the snapshot.
const (
	addrUSDT = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	addrWTRX = "TNUC9Qb1rRpS5CbWLmNMxXBjyFoydXjWFR"
	addrJST  = "TE2RzoSV3wFK99w6J9UnnZ4vLfXYoxvRwP"
)

func abiEvent(name string, anonymous bool, params ...*core.SmartContract_ABI_Entry_Param) *core.SmartContract_ABI_Entry {
	return &core.SmartContract_ABI_Entry{
		Type:      core.SmartContract_ABI_Entry_Event,
		Name:      name,
		Anonymous: anonymous,
		Inputs:    params,
	}
}

func abiField(typ, name string, indexed bool) *core.SmartContract_ABI_Entry_Param {
	return &core.SmartContract_ABI_Entry_Param{Type: typ, Name: name, Indexed: indexed}
}

func abiFunction(name string) *core.SmartContract_ABI_Entry {
	return &core.SmartContract_ABI_Entry{Type: core.SmartContract_ABI_Entry_Function, Name: name}
}

func abiOf(entries ...*core.SmartContract_ABI_Entry) *core.SmartContract_ABI {
	return &core.SmartContract_ABI{Entrys: entries}
}

// stubFetcher serves programmed ABIs keyed by base58 address.
type stubFetcher struct {
	abis map[string]*core.SmartContract_ABI
	err  error
}

func (s *stubFetcher) ABI(_ context.Context, addr tron.Address) (*core.SmartContract_ABI, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.abis[addr.String()], nil
}

func snapWith(addrs ...string) *Snapshot {
	s := &Snapshot{RankBy: "trxCount", Limit: len(addrs)}
	for i, a := range addrs {
		s.Contracts = append(s.Contracts, ContractEntry{Rank: i + 1, Address: a})
	}
	return s
}

func TestCaptureFiltersAndUpserts(t *testing.T) {
	f := &stubFetcher{abis: map[string]*core.SmartContract_ABI{
		addrUSDT: abiOf(
			abiFunction("transfer"),
			abiEvent("Anon", true, abiField("uint256", "id", true)),
			abiEvent("", false),
			abiEvent("Ping", false, abiField("uint256", "n", false)),
		),
	}}
	s := New("")
	rep, err := Capture(context.Background(), snapWith(addrUSDT), f, s, 1)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if rep != (CaptureReport{Contracts: 1, WithEvents: 1, NewEvents: 1, Skipped: 0}) {
		t.Fatalf("report = %+v", rep)
	}
	if s.Len() != 1 || s.Events()[0].Signature != "Ping(uint256)" {
		t.Fatalf("store = %+v", s.Events())
	}
	if got := s.Events()[0].Inputs; len(got) != 1 || got[0].Name != "n" {
		t.Fatalf("inputs lost: %+v", got)
	}
}

func TestCaptureFirstWinsAcrossContracts(t *testing.T) {
	ping := abiOf(abiEvent("Ping", false, abiField("uint256", "n", false)))
	f := &stubFetcher{abis: map[string]*core.SmartContract_ABI{addrUSDT: ping, addrWTRX: ping}}
	s := New("")
	rep, err := Capture(context.Background(), snapWith(addrUSDT, addrWTRX), f, s, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rep.WithEvents != 2 || rep.NewEvents != 1 {
		t.Fatalf("report = %+v, want with-events 2 / new 1 (second is a duplicate)", rep)
	}
	if s.Len() != 1 {
		t.Fatalf("store holds %d entries, want 1", s.Len())
	}
}

func TestCaptureSkipsContractWithoutEvents(t *testing.T) {
	f := &stubFetcher{abis: map[string]*core.SmartContract_ABI{
		addrUSDT: abiOf(abiFunction("transfer"), abiFunction("approve")),
	}}
	rep, err := Capture(context.Background(), snapWith(addrUSDT), f, New(""), 1)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Skipped != 1 || rep.WithEvents != 0 || rep.NewEvents != 0 {
		t.Fatalf("report = %+v, want one skip", rep)
	}
}

func TestCaptureConcurrencyMatchesSequential(t *testing.T) {
	f := &stubFetcher{abis: map[string]*core.SmartContract_ABI{
		addrUSDT: abiOf(abiEvent("Ping", false, abiField("uint256", "n", false))),
		addrWTRX: abiOf(abiEvent("Pong", false), abiEvent("Zap", false)),
		addrJST:  abiOf(abiFunction("x")),
	}}
	seq, par := New(""), New("")
	repSeq, err := Capture(context.Background(), snapWith(addrUSDT, addrWTRX, addrJST), f, seq, 1)
	if err != nil {
		t.Fatal(err)
	}
	repPar, err := Capture(context.Background(), snapWith(addrUSDT, addrWTRX, addrJST), f, par, 4)
	if err != nil {
		t.Fatal(err)
	}
	if repSeq != repPar {
		t.Fatalf("report differs: seq %+v par %+v", repSeq, repPar)
	}
	if len(seq.Events()) != len(par.Events()) || seq.Events()[0].Sighash != par.Events()[0].Sighash {
		t.Fatal("concurrent capture is not rank-deterministic")
	}
}

func TestCapturePropagatesFetchError(t *testing.T) {
	f := &stubFetcher{err: errors.New("boom")}
	_, err := Capture(context.Background(), snapWith(addrUSDT), f, New(""), 4)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want the fetch error, got %v", err)
	}
}
