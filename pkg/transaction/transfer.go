package transaction

import (
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// TransferTRX creates a TRX transfer transaction
func (tx *Transaction) TransferTRX(from *types.Address, to *types.Address, amount int64) error {

	txExt, err := tx.client.BuildTransaction(&core.TransferContract{
		OwnerAddress: from.Bytes(),
		ToAddress:    to.Bytes(),
		Amount:       amount,
	})
	if err != nil {
		return err
	}

	tx.txExtension = txExt

	return nil
}
