package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// TestTxWrappers exercises TxCall's TxCall/ValidateTransactionResult plumbing
// through the fake for one representative wrapper per validation branch.
func TestTxWrappers(t *testing.T) {
	t.Run("ValidateTransactionResult/nil-result", func(t *testing.T) {
		err := ValidateTransactionResult(nil, "op")
		if !tron.HasCode(err, tron.CodeRPCMethodFailed) {
			t.Fatalf("err = %v, want rpc.method_failed", err)
		}
		var te *tron.Error
		if !errors.As(err, &te) || te.Op != "op" {
			t.Fatalf("Op = %q, want op", te)
		}
	})

	t.Run("ValidateTransactionResult/nil-Return-field", func(t *testing.T) {
		err := ValidateTransactionResult(&api.TransactionExtention{}, "op")
		if !tron.HasCode(err, tron.CodeRPCMethodFailed) {
			t.Fatalf("err = %v, want rpc.method_failed", err)
		}
	})

	t.Run("ValidateTransactionResult/success", func(t *testing.T) {
		if err := ValidateTransactionResult(okExtention(), "op"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestTxCallMapsNodeReturnCodes pins the api.Return_* -> tron.Code mapping
// table (spec §8.5) through TxCall: the node reports Result=false with a
// Return_* code and TxCall must surface the mapped v2 code with the node's
// numeric code preserved in Cause.
func TestTxCallMapsNodeReturnCodes(t *testing.T) {
	cases := []struct {
		name       string
		returnCode api.ReturnResponseCode
		wantCode   tron.Code
	}{
		{"SIGERROR", api.Return_SIGERROR, tron.CodeKeyInvalid},
		{"CONTRACT_VALIDATE_ERROR", api.Return_CONTRACT_VALIDATE_ERROR, tron.CodeTxInvalidArgument},
		{"CONTRACT_EXE_ERROR", api.Return_CONTRACT_EXE_ERROR, tron.CodeReceiptFailed},
		{"BANDWITH_ERROR", api.Return_BANDWITH_ERROR, tron.CodeAccountInsufficientBandwidth},
		{"DUP_TRANSACTION_ERROR", api.Return_DUP_TRANSACTION_ERROR, tron.CodeTxDuplicate},
		{"TAPOS_ERROR", api.Return_TAPOS_ERROR, tron.CodeTxTaposInvalid},
		{"TOO_BIG_TRANSACTION_ERROR", api.Return_TOO_BIG_TRANSACTION_ERROR, tron.CodeTxTooLarge},
		{"TRANSACTION_EXPIRATION_ERROR", api.Return_TRANSACTION_EXPIRATION_ERROR, tron.CodeTxExpired},
		{"SERVER_BUSY", api.Return_SERVER_BUSY, tron.CodeChainUnavailable},
		{"NO_CONNECTION", api.Return_NO_CONNECTION, tron.CodeChainConnection},
		{"NOT_ENOUGH_EFFECTIVE_CONNECTION", api.Return_NOT_ENOUGH_EFFECTIVE_CONNECTION, tron.CodeChainConnection},
		{"BLOCK_UNSOLIDIFIED", api.Return_BLOCK_UNSOLIDIFIED, tron.CodeChainUnconfirmed},
		{"OTHER_ERROR", api.Return_OTHER_ERROR, tron.CodeRPCMethodFailed},
		// Unknown node codes (e.g. newer node versions) fall back to
		// rpc.method_failed rather than being dropped.
		{"UnknownCode", api.ReturnResponseCode(99), tron.CodeRPCMethodFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := &testWalletServer{Handlers: map[string]func(ctx context.Context, in any) (any, error){
				"CreateTransaction2": func(ctx context.Context, in any) (any, error) {
					return &api.TransactionExtention{Result: &api.Return{
						Result:  false,
						Code:    tc.returnCode,
						Message: []byte("node says no"),
					}}, nil
				},
			}}
			c := newBufconnClient(t, srv, time.Second)
			_, err := CreateTransaction2(c, context.Background(), &core.TransferContract{})
			if !tron.HasCode(err, tc.wantCode) {
				t.Fatalf("err = %v, want %s", err, tc.wantCode)
			}
			var te *tron.Error
			if !errors.As(err, &te) {
				t.Fatalf("err = %T, want *tron.Error", err)
			}
			if te.Op != "create transaction2" {
				t.Fatalf("Op = %q, want create transaction2", te.Op)
			}
			if te.Cause == nil || !containsSub(te.Cause.Error(), "node says no") {
				t.Fatalf("Cause = %v, want the node's message", te.Cause)
			}
		})
	}
}

func containsSub(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
