package transaction

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pkg/types"
)

// TransferTRX creates a TRX transfer transaction
func (tx *Transaction) TransferTRX(ctx context.Context, to *types.Address, amount int64) *Transaction {
	if tx.receipt.Err != nil {
		return tx
	}

	txExt, err := tx.client.CreateTransferTransaction(ctx, tx.owner.Bytes(), to.Bytes(), amount)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to create transfer transaction: %v", err)
		return tx
	}

	tx.txExtension = txExt
	return tx.SetDefaultOptions()
}
