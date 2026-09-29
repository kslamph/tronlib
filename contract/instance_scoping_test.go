package contract

// Instance.UseABI must register the ABI's event definitions for THIS
// contract's address only (event.RegisterABIJSONForAddress), not in the
// global registry: two contracts that use the same event signature with
// different indexed layouts must each decode their own logs, and a contract
// that never registered globally must not have event.Decode (the global
// lookup) answer for it.

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/tron"
)

// scopingABI lays out Mixed(uint256) with value in the named position only.
const (
	scopedABIA = `[{"name":"Mixed","type":"event","inputs":[{"name":"value","type":"uint256","indexed":true}]}]`
	scopedABIB = `[{"name":"Mixed","type":"event","inputs":[{"name":"value","type":"uint256","indexed":false}]}]`
)

func scopedTopic(t *testing.T) []byte {
	t.Helper()
	return crypto.Keccak256([]byte("Mixed(uint256)"))
}

func scopedValueTopic(v byte) []byte {
	t := make([]byte, 32)
	t[31] = v
	return t
}

// TestUseABIScopesEventRegistrationToTheInstanceAddress: two instances with
// conflicting layouts for the same signature both decode correctly, through
// their address, while the global registry stays clean.
func TestUseABIScopesEventRegistrationToTheInstanceAddress(t *testing.T) {
	addrA := mustAddr(0x33)
	addrB := mustAddr(0x44)

	instA, err := NewInstance(newContractTestClient(t, &fakeWallet{}), addrA)
	if err != nil {
		t.Fatalf("NewInstance A: %v", err)
	}
	instB, err := NewInstance(newContractTestClient(t, &fakeWallet{}), addrB)
	if err != nil {
		t.Fatalf("NewInstance B: %v", err)
	}
	if err := instA.UseABI(scopedABIA); err != nil {
		t.Fatalf("UseABI A: %v", err)
	}
	if err := instB.UseABI(scopedABIB); err != nil {
		t.Fatalf("UseABI B: %v", err)
	}

	topic := scopedTopic(t)
	topicValue := scopedValueTopic(7)

	// A's layout: value indexed — it lives in topic 1, data empty.
	logA, err := event.DecodeFor(addrA, [][]byte{topic, topicValue}, nil)
	if err != nil {
		t.Fatalf("DecodeFor A: %v", err)
	}
	if got := logA.Parameters[0].Value.(*big.Int).Int64(); got != 7 {
		t.Fatalf("A: decoded %d, want 7 from the indexed topic", got)
	}
	if logA.Address != addrA {
		t.Fatalf("A: Log.Address = %v, want %v", logA.Address, addrA)
	}

	// B's layout: value non-indexed — it lives in the data section.
	logB, err := event.DecodeFor(addrB, [][]byte{topic}, scopedValueTopic(9))
	if err != nil {
		t.Fatalf("DecodeFor B: %v", err)
	}
	if got := logB.Parameters[0].Value.(*big.Int).Int64(); got != 9 {
		t.Fatalf("B: decoded %d, want 9 from the data section", got)
	}

	// The global registry must not have been fed by either UseABI: the
	// global-only lookup answers event.unknown for this signature.
	if _, err := event.Decode([][]byte{topic}, nil); !tron.HasCode(err, tron.CodeEventUnknown) {
		t.Fatalf("global Decode after two UseABI calls: err = %v, want event.unknown (UseABI must not register globally)", err)
	}
}

// TestUseABIReplacingAnABIDoesNotPoisonTheScope: loading a second ABI on the
// same instance replaces the event definitions that address decodes with —
// identical definitions are a no-op, so re-loading the same ABI keeps
// decoding working.
func TestUseABIReplacingAnABIDoesNotPoisonTheScope(t *testing.T) {
	addr := mustAddr(0x55)
	inst, err := NewInstance(newContractTestClient(t, &fakeWallet{}), addr)
	if err != nil {
		t.Fatalf("NewInstance: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := inst.UseABI(scopedABIA); err != nil {
			t.Fatalf("UseABI pass %d: %v", i, err)
		}
	}
	log, err := event.DecodeFor(addr, [][]byte{scopedTopic(t), scopedValueTopic(3)}, nil)
	if err != nil {
		t.Fatalf("DecodeFor after re-load: %v", err)
	}
	if got := log.Parameters[0].Value.(*big.Int).Int64(); got != 3 {
		t.Fatalf("decoded %d, want 3", got)
	}
}
