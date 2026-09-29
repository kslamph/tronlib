package rpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/kslamph/tronlib/v2/pb/api"
	"github.com/kslamph/tronlib/v2/tron"
)

// returnCodeToCode maps the node's api.Return_* response codes onto the v2
// code vocabulary (architecture §8.5: one table, the numeric NodeCode is preserved in
// the error Cause so the Receipt layer can recover it later). Every mapped
// code exists in v2/tron/codes_gen.go — no invented codes.
var returnCodeToCode = map[api.ReturnResponseCode]tron.Code{
	api.Return_SUCCESS:                         "", // handled before the lookup; listed for documentation
	api.Return_SIGERROR:                        tron.CodeKeyInvalid,
	api.Return_CONTRACT_VALIDATE_ERROR:         tron.CodeTxInvalidArgument,
	api.Return_CONTRACT_EXE_ERROR:              tron.CodeReceiptFailed,
	api.Return_BANDWITH_ERROR:                  tron.CodeAccountInsufficientBandwidth,
	api.Return_DUP_TRANSACTION_ERROR:           tron.CodeTxDuplicate,
	api.Return_TAPOS_ERROR:                     tron.CodeTxTaposInvalid,
	api.Return_TOO_BIG_TRANSACTION_ERROR:       tron.CodeTxTooLarge,
	api.Return_TRANSACTION_EXPIRATION_ERROR:    tron.CodeTxExpired,
	api.Return_SERVER_BUSY:                     tron.CodeChainUnavailable,
	api.Return_NO_CONNECTION:                   tron.CodeChainConnection,
	api.Return_NOT_ENOUGH_EFFECTIVE_CONNECTION: tron.CodeChainConnection,
	api.Return_BLOCK_UNSOLIDIFIED:              tron.CodeChainUnconfirmed,
	api.Return_OTHER_ERROR:                     tron.CodeRPCMethodFailed,
}

// nodeReturnError is the typed cause of a failed node Return (Result ==
// false). It preserves the numeric api.Return_* code so callers (e.g. the
// Receipt layer) can recover it programmatically via NodeReturnCode instead
// of parsing the rendered message. The rendered text is unchanged from the
// previous untyped fmt.Errorf cause.
type nodeReturnError struct {
	code api.ReturnResponseCode
	msg  string
	op   string
}

func (e *nodeReturnError) Error() string {
	return fmt.Sprintf("node return code %d (%s): %s", int32(e.code), e.code, e.msg)
}

// NodeReturnCode recovers the node's numeric api.Return_* code from an error
// produced by ValidateTransactionResult/mapNodeReturn (the nodeReturnError is
// attached as the Cause of the *tron.Error, which unwraps into it). The bool
// reports whether err carries a node return code at all.
func NodeReturnCode(err error) (api.ReturnResponseCode, bool) {
	var nre *nodeReturnError
	if errors.As(err, &nre) {
		return nre.code, true
	}
	return 0, false
}

// mapNodeReturn converts a failed (Result == false) api.Return into a
// *tron.Error. The raw node code and message ride along as a typed
// nodeReturnError Cause, preserving the NodeCode for the Receipt layer.
func mapNodeReturn(ret *api.Return, operation string) *tron.Error {
	code, ok := returnCodeToCode[ret.Code]
	if !ok || code == "" {
		code = tron.CodeRPCMethodFailed
	}
	msg := string(ret.Message)
	if msg == "" {
		msg = "unknown error"
	}
	return &tron.Error{
		Code:  code,
		Op:    operation,
		Cause: &nodeReturnError{code: ret.Code, msg: msg, op: operation},
	}
}

// ValidateTransactionResult checks the common result pattern for transaction
// operations: a nil extention, a missing Result field, or Result == false are
// all errors carrying the mapped tron code with the operation string in Op.
func ValidateTransactionResult(result *api.TransactionExtention, operation string) error {
	if result == nil {
		return &tron.Error{
			Code: tron.CodeRPCMethodFailed,
			Op:   operation,
			Hint: "node returned a nil result; check node connectivity and retry",
		}
	}
	if result.Result == nil {
		return &tron.Error{
			Code: tron.CodeRPCMethodFailed,
			Op:   operation,
			Hint: "node returned a result without a Return field; check node version",
		}
	}
	if !result.Result.Result {
		return mapNodeReturn(result.Result, operation)
	}
	return nil
}

// TxCall is a specialization of Call for RPCs returning
// *api.TransactionExtention: it runs ValidateTransactionResult after the RPC
// succeeds at the transport level.
func TxCall(cp ConnProvider, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error)) (*api.TransactionExtention, error) {
	return Call(cp, ctx, operation, call, ValidateTransactionResult)
}
