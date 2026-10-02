package token

// How the TRC-20 read paths classify a failure. The distinction is the whole
// point of the metadata error codes: "this address is not a TRC-20" and "the
// node was briefly unreachable" and "the contract ran and reverted" are three
// different facts, and a caller (or an agent) acts differently on each. Only
// the first is contract.bad_metadata.

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// failedExt is the node's answer when the simulated call did not succeed: the
// contract ran and chose to fail, so the result carries the node's own code
// and no value.
func failedExt(nodeCode api.ReturnResponseCode) *api.TransactionExtention {
	return &api.TransactionExtention{Result: &api.Return{Result: false, Code: nodeCode}}
}

// readPaths is the TRC-20 read surface of a Handle, as closures over a Handle
// so every classification test runs the same set.
func readPaths(h *Handle) map[string]func() error {
	return map[string]func() error{
		"BalanceOf":   func() error { _, err := h.BalanceOf(context.Background(), mustAddr(0x11)); return err },
		"Allowance":   func() error { _, err := h.Allowance(context.Background(), mustAddr(0x11), mustAddr(0x33)); return err },
		"Name":        func() error { _, err := h.Name(context.Background()); return err },
		"Symbol":      func() error { _, err := h.Symbol(context.Background()); return err },
		"TotalSupply": func() error { _, err := h.TotalSupply(context.Background()); return err },
	}
}

// newHandleWithAnswers builds a 6-decimal Handle whose every read answers with
// the supplied ConstantResult (after the constructor's own decimals() call).
func newHandleWithAnswers(t *testing.T, answer []byte) *Handle {
	t.Helper()
	f := &fakeTRC20Wallet{
		TriggerConstant: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			ext := okExt()
			if string(in.GetData()[:4]) == string(mustSelector("decimals()")) {
				ext.ConstantResult = [][]byte{abiWord(6)}
				return ext, nil
			}
			ext.ConstantResult = [][]byte{answer}
			return ext, nil
		},
	}
	h, err := New(context.Background(), newTokenTestClient(t, f), testContract)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h
}

// TestRevertedCallIsNotBadMetadata: a simulated call the contract itself
// fails is reported with the node's own code, never relabelled
// contract.bad_metadata.
//
// Failure mode: the reverse mapping (or a widened metaErr list) would tell the
// caller "this address does not implement TRC-20" about a contract that is a
// perfectly good token whose balanceOf reverted — sending the caller to
// re-verify the address instead of at the call.
func TestRevertedCallIsNotBadMetadata(t *testing.T) {
	for _, nodeCode := range []api.ReturnResponseCode{
		api.Return_CONTRACT_EXE_ERROR,      // the contract ran and reverted
		api.Return_CONTRACT_VALIDATE_ERROR, // the node rejected the call shape
	} {
		t.Run(nodeCode.String(), func(t *testing.T) {
			// The constructor's own decimals() read succeeds; every read
			// after it gets the failing answer.
			f := &fakeTRC20Wallet{
				TriggerConstant: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
					if string(in.GetData()[:4]) == string(mustSelector("decimals()")) {
						return okExtWith(abiWord(6)), nil
					}
					return failedExt(nodeCode), nil
				},
			}
			h2, err := New(context.Background(), newTokenTestClient(t, f), testContract)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			for name, read := range readPaths(h2) {
				err := read()
				if tron.HasCode(err, tron.CodeContractBadMetadata) {
					t.Errorf("%s on a %s answer: err = %v; a failed call is not malformed metadata", name, nodeCode, err)
				}
				if !tron.HasCode(err, readPathCode(nodeCode)) {
					t.Errorf("%s: err = %v, want %s", name, err, readPathCode(nodeCode))
				}
			}
		})
	}
}

// TestTransportFailureIsNotBadMetadata: a node that is down, busy, or slow
// must not be reported as "not a TRC-20".
//
// Failure mode: a network blip during startup is permanent-looking to the
// caller ("wrong token address"), and a user who trusted that would go buy a
// different token.
func TestTransportFailureIsNotBadMetadata(t *testing.T) {
	for _, tc := range []struct {
		name string
		ext  *api.TransactionExtention
		err  error
		want tron.Code
	}{
		{"unavailable", nil, status.Error(codes.Unavailable, "node down"), tron.CodeRPCMethodFailed},
		{"internal", nil, status.Error(codes.Internal, "boom"), tron.CodeRPCMethodFailed},
		{"server busy", failedExt(api.Return_SERVER_BUSY), nil, tron.CodeChainUnavailable},
		{"no connection", failedExt(api.Return_NO_CONNECTION), nil, tron.CodeChainConnection},
		{"unsolidified", failedExt(api.Return_BLOCK_UNSOLIDIFIED), nil, tron.CodeChainUnconfirmed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeTRC20Wallet{
				TriggerConstant: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
					if string(in.GetData()[:4]) == string(mustSelector("decimals()")) {
						return okExtWith(abiWord(6)), nil
					}
					if tc.err != nil {
						return nil, tc.err
					}
					return tc.ext, nil
				},
			}
			h, err := New(context.Background(), newTokenTestClient(t, f), testContract)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			for name, read := range readPaths(h) {
				err := read()
				if tron.HasCode(err, tron.CodeContractBadMetadata) {
					t.Errorf("%s during %s: err = %v; a transport failure is not malformed metadata", name, tc.name, err)
				}
				if !tron.HasCode(err, tc.want) {
					t.Errorf("%s during %s: err = %v, want %s", name, tc.name, err, tc.want)
				}
			}
		})
	}
}

// TestUndecodableAnswerIsBadMetadata: the complement of the two tests above.
// A contract that answers a uint256 method with something that does not decode
// at all IS malformed metadata, on every read path, and the Hint must say so.
//
// Failure mode: silently returning a zero Amount for an undecodable answer
// would show the user a balance of 0 for a real token.
func TestUndecodableAnswerIsBadMetadata(t *testing.T) {
	// NOTE: a well-formed-but-wrong answer (e.g. the offset word 0x20 read
	// as a uint256) decodes silently — ABI outputs are not self-describing,
	// and only undecodable answers can be rejected. That limitation is
	// documented in metadata_test.go; these are the cases that DO decode.
	for name, answer := range map[string][]byte{
		"truncated word": abiWord(1)[:10],
		"empty":          {},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHandleWithAnswers(t, answer)
			for read, fn := range readPaths(h) {
				err := fn()
				if !tron.HasCode(err, tron.CodeContractBadMetadata) {
					t.Errorf("%s on a %s answer: err = %v, want contract.bad_metadata", read, name, err)
				}
			}
		})
	}
}

// TestNewRejectsUndecodableDecimals: the constructor's own decimals() read is
// the one place a wrong scale can be baked in permanently, so an answer that
// does not decode as a number must fail construction rather than produce a
// Handle whose every Amount is wrong.
func TestNewRejectsUndecodableDecimals(t *testing.T) {
	for name, answer := range map[string][]byte{
		"truncated word": abiWord(6)[:12],
		"empty":          {},
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeTRC20Wallet{
				TriggerConstant: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
					return okExtWith(answer), nil
				},
			}
			h, err := New(context.Background(), newTokenTestClient(t, f), testContract)
			if h != nil {
				t.Fatalf("New with a %s decimals() answer returned a Handle; construction must fail", name)
			}
			if !tron.HasCode(err, tron.CodeContractBadMetadata) {
				t.Errorf("err = %v, want contract.bad_metadata", err)
			}
		})
	}
}

// TestNewPropagatesTransportFailure: New must not relabel a connection
// failure as bad metadata either — the first call it makes is a network call.
func TestNewPropagatesTransportFailure(t *testing.T) {
	f := &fakeTRC20Wallet{
		TriggerConstant: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return nil, status.Error(codes.Unavailable, "node down")
		},
	}
	h, err := New(context.Background(), newTokenTestClient(t, f), testContract)
	if h != nil {
		t.Fatal("New returned a Handle despite the node being unreachable")
	}
	if tron.HasCode(err, tron.CodeContractBadMetadata) {
		t.Errorf("err = %v; an unreachable node is not malformed metadata", err)
	}
	if !tron.HasCode(err, tron.CodeRPCMethodFailed) {
		t.Errorf("err = %v, want rpc.method_failed", err)
	}
}

// readPathCode is the tron code the node's own response code maps to, per
// rpc's return table. The test asserts the passthrough rather than restating
// the mapping, so a change to the table is visible here as a test change.
func readPathCode(nodeCode api.ReturnResponseCode) tron.Code {
	switch nodeCode {
	case api.Return_CONTRACT_EXE_ERROR:
		return tron.CodeReceiptFailed
	case api.Return_CONTRACT_VALIDATE_ERROR:
		return tron.CodeTxInvalidArgument
	default:
		return tron.CodeRPCMethodFailed
	}
}

// okExtWith is okExt with a canned constant answer, for the constructor's
// decimals() probe.
func okExtWith(answer []byte) *api.TransactionExtention {
	ext := okExt()
	if len(answer) > 0 {
		ext.ConstantResult = [][]byte{answer}
	}
	return ext
}
