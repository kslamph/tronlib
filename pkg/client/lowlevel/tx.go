package lowlevel

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pkg/types"
)

// ValidateTransactionResult checks the common result pattern for transaction operations.
func ValidateTransactionResult(result *api.TransactionExtention, operation string) error {
	if result == nil {
		return fmt.Errorf("nil result for %s", operation)
	}
	if result.Result == nil {
		return fmt.Errorf("nil result field for %s", operation)
	}
	if !result.Result.Result {
		return types.WrapTransactionResult(result.Result, operation)
	}
	return nil
}

// TxCall is a specialization of Call for RPCs returning *api.TransactionExtention.
func TxCall(cp ConnProvider, ctx context.Context, operation string, call func(client api.WalletClient, ctx context.Context) (*api.TransactionExtention, error)) (*api.TransactionExtention, error) {
	return Call(cp, ctx, operation, call, ValidateTransactionResult)
}
