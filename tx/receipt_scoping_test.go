package tx

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/kslamph/tronlib/v2/event"
	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// The P1 scenario end to end: two contracts emit the SAME canonical event
// signature with different indexed layouts. The emitting address is the only
// thing that tells their logs apart, so a receipt must decode each log through
// its own contract's definition — one global definition per signature prefix
// mis-assigned values for at least one of the two contracts.

// receiptScopedSigTopic is keccak256("ReceiptScoped(address,address,uint256)"),
// the topic both layouts emit.
const receiptScopedSigTopic = "68faa300c913aa469de60afe36f8f5195532d2ab2374805d3e556dbf1ad9251f"

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex %q: %v", s, err)
	}
	return b
}

// receiptScopedABI builds a ReceiptScoped ABI JSON over
// address,address,uint256 with the given indexed flags.
func receiptScopedABI(indexed ...bool) string {
	types := []string{"address", "address", "uint256"}
	params := make([]string, 0, len(indexed))
	for i, idx := range indexed {
		params = append(params, fmt.Sprintf(`{"name":"p%d","type":"%s","indexed":%t}`, i, types[i], idx))
	}
	return `[{"type":"event","name":"ReceiptScoped","inputs":[` + strings.Join(params, ",") + `]}]`
}

// abWord is a 32-byte ABI word holding a left-padded address payload.
func abWord(fill byte) []byte { return append(make([]byte, 12), repeat(fill, 20)...) }

// u256Word is a 32-byte ABI word holding a small non-negative integer.
func u256Word(v int64) []byte {
	out := make([]byte, 32)
	new(big.Int).SetInt64(v).FillBytes(out)
	return out
}

func TestReceiptLogsDecodeByEmittingAddress(t *testing.T) {
	contractA := mustAddr(0xA1) // both addresses indexed
	contractB := mustAddr(0xB2) // only the first indexed
	if err := event.RegisterABIJSONForAddress(contractA, receiptScopedABI(true, true, false)); err != nil {
		t.Fatalf("RegisterABIJSONForAddress(A): %v", err)
	}
	if err := event.RegisterABIJSONForAddress(contractB, receiptScopedABI(true, false, false)); err != nil {
		t.Fatalf("RegisterABIJSONForAddress(B): %v", err)
	}

	sig := mustHex(t, receiptScopedSigTopic)
	first, second := abWord(0x11), abWord(0x22)
	amount := u256Word(1000)

	info := &core.TransactionInfo{
		Id:      hexID(),
		Result:  core.TransactionInfo_SUCESS,
		Receipt: resourceReceipt(core.Transaction_Result_SUCCESS),
		Log: []*core.TransactionInfo_Log{
			{ // A's log: p0 and p1 in topics, p2 in data.
				Address: contractA.Bytes(),
				Topics:  [][]byte{sig, first, second},
				Data:    amount,
			},
			{ // B's log: p0 in topics, p1 and p2 ABI-encoded in data.
				Address: contractB.Bytes(),
				Topics:  [][]byte{sig, first},
				Data:    append(append([]byte{}, second...), amount...),
			},
		},
	}

	r := waitReceipt(t, info)
	if len(r.Logs) != 2 {
		t.Fatalf("Logs = %d entries, want 2", len(r.Logs))
	}
	wantAddrs := []tron.Address{contractA, contractB}
	for i, lg := range r.Logs {
		if lg.EventName != "ReceiptScoped" {
			t.Fatalf("log %d: EventName = %q, want ReceiptScoped (topics=%x data=%x)", i, lg.EventName, lg.Topics, lg.Data)
		}
		if lg.Address != wantAddrs[i] {
			t.Errorf("log %d: Address = %v, want %v", i, lg.Address, wantAddrs[i])
		}
		if len(lg.Parameters) != 3 {
			t.Fatalf("log %d: %d parameters, want 3", i, len(lg.Parameters))
		}
		// Log 0 reads p1 from topics[2], log 1 from the first data word. Both
		// must report the same address: nothing may shift between parameters.
		p1, ok := lg.Parameters[1].Value.(tron.Address)
		if !ok {
			t.Fatalf("log %d: p1 type = %T, want tron.Address", i, lg.Parameters[1].Value)
		}
		if string(p1.Bytes()) != string(append([]byte{0x41}, second[12:]...)) {
			t.Errorf("log %d: p1 = %x, want the second address (which %s supplies for this layout)",
				i, p1.Bytes(), []string{"topics", "data"}[i])
		}
		if v, ok := lg.Parameters[2].Value.(*big.Int); !ok || v.Int64() != 1000 {
			t.Errorf("log %d: p2 = %#v (%T), want 1000", i, lg.Parameters[2].Value, lg.Parameters[2].Value)
		}
	}
}

// TestLogsForDecodesByEmittingAddress drives the same two-log fixture through
// the Client.Events path (tx.LogsFor) rather than Wait, so the address-aware
// decode is pinned on both log surfaces: they share decodeReceiptLog, and a
// future fork of one of them must not silently lose the scoping.
func TestLogsForDecodesByEmittingAddress(t *testing.T) {
	contractA := mustAddr(0xC3)
	contractB := mustAddr(0xD4)
	if err := event.RegisterABIJSONForAddress(contractA, receiptScopedABI(true, true, false)); err != nil {
		t.Fatalf("RegisterABIJSONForAddress(A): %v", err)
	}
	if err := event.RegisterABIJSONForAddress(contractB, receiptScopedABI(true, false, false)); err != nil {
		t.Fatalf("RegisterABIJSONForAddress(B): %v", err)
	}

	sig := mustHex(t, receiptScopedSigTopic)
	first, second := abWord(0x55), abWord(0x66)
	amount := u256Word(700)
	info := &core.TransactionInfo{
		Id: hexID(),
		Log: []*core.TransactionInfo_Log{
			{Address: contractA.Bytes(), Topics: [][]byte{sig, first, second}, Data: amount},
			{Address: contractB.Bytes(), Topics: [][]byte{sig, first}, Data: append(append([]byte{}, second...), amount...)},
		},
	}
	f := &fakeWalletServer{
		TxInfo: func(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			return info, nil
		},
	}
	cp := newTxTestClient(t, f)
	logs, err := LogsFor(cp, t.Context(), hex.EncodeToString(hexID()))
	if err != nil {
		t.Fatalf("LogsFor: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("logs = %d, want 2", len(logs))
	}
	for i, want := range []tron.Address{contractA, contractB} {
		if logs[i].EventName != "ReceiptScoped" {
			t.Errorf("log %d: EventName = %q, want ReceiptScoped", i, logs[i].EventName)
		}
		if logs[i].Address != want {
			t.Errorf("log %d: Address = %v, want %v", i, logs[i].Address, want)
		}
		if len(logs[i].Parameters) != 3 {
			t.Fatalf("log %d: %d parameters, want 3", i, len(logs[i].Parameters))
		}
		p1, ok := logs[i].Parameters[1].Value.(tron.Address)
		if !ok || string(p1.Bytes()) != string(append([]byte{0x41}, second[12:]...)) {
			t.Errorf("log %d: p1 = %#v (%T), want the second address", i, logs[i].Parameters[1].Value, logs[i].Parameters[1].Value)
		}
		if v, ok := logs[i].Parameters[2].Value.(*big.Int); !ok || v.Int64() != 700 {
			t.Errorf("log %d: p2 = %#v (%T), want 700", i, logs[i].Parameters[2].Value, logs[i].Parameters[2].Value)
		}
	}
}

// TestReceiptLogsUnparseableAddressUsesGlobalRegistry: the emitting address is a
// scope, never a filter. A wire log whose address bytes are not a valid TRON
// address still decodes through the global registry, and is never dropped.
func TestReceiptLogsUnparseableAddressUsesGlobalRegistry(t *testing.T) {
	trc20Sig := mustHex(t, "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	from, to := abWord(0x33), abWord(0x44)
	info := &core.TransactionInfo{
		Id:      hexID(),
		Result:  core.TransactionInfo_SUCESS,
		Receipt: resourceReceipt(core.Transaction_Result_SUCCESS),
		Log: []*core.TransactionInfo_Log{
			{
				Address: repeat(0x41, 20), // 20 bytes: not a 0x41-prefixed 21-byte address
				Topics:  [][]byte{trc20Sig, from, to},
				Data:    u256Word(500),
			},
		},
	}

	r := waitReceipt(t, info)
	if len(r.Logs) != 1 {
		t.Fatalf("Logs = %d entries, want 1", len(r.Logs))
	}
	lg := r.Logs[0]
	if lg.EventName != "Transfer" {
		t.Fatalf("EventName = %q, want Transfer via the global registry (topics=%x data=%x)", lg.EventName, lg.Topics, lg.Data)
	}
	if !lg.Address.IsZero() {
		t.Errorf("Address = %v, want the unset address when the wire bytes do not parse", lg.Address)
	}
	if v, ok := lg.Parameters[2].Value.(*big.Int); !ok || v.Int64() != 500 {
		t.Errorf("value = %#v (%T), want 500", v, lg.Parameters[2].Value)
	}
}
