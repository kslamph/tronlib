package rpc

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/v2/tron"
)

// returnCodeToCode maps the node's api.Return_* response codes onto the v2
// code vocabulary (spec §8.5: one table, the numeric NodeCode is preserved in
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

// mapNodeReturn converts a failed (Result == false) api.Return into a
// *tron.Error. The raw node code and message ride along as Cause, preserving
// the NodeCode for the Receipt layer.
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
		Cause: fmt.Errorf("node return code %d (%s): %s", int32(ret.Code), ret.Code, msg),
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
