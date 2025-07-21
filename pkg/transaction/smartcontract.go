package transaction

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pkg/types"
)

// TriggerSmartContract triggers a smart contract call
func (tx *Transaction) TriggerSmartContract(ctx context.Context, contract *types.Contract, data []byte, callValue int64) *Transaction {
	if tx.receipt.Err != nil {
		return tx // Return early if there's already an error
	}

	// Call the specific client method for smart contract transactions
	txExt, err := tx.client.CreateTriggerSmartContractTransaction(
		ctx,
		tx.owner.Bytes(),
		contract.AddressBytes,
		data,
		callValue,
	)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to create smart contract transaction: %v", err)
		return tx
	}

	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}
