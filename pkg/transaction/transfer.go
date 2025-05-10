package transaction

import (
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// TransferTRX creates a TRX transfer transaction
func (tx *Transaction) TransferTRX(to *types.Address, amount int64) *Transaction {

	txExt, err := tx.client.BuildTransaction(&core.TransferContract{
		OwnerAddress: tx.owner.Bytes(),
		ToAddress:    to.Bytes(),
		Amount:       amount,
	})
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", err)
	}

	tx.txExtension = txExt
	return tx.SetDefaultOptions()

}
