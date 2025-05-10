package transaction

import (
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

// TriggerSmartContract triggers a smart contract call
func (tx *Transaction) TriggerSmartContract(contract *types.Contract, data []byte, callValue int64) *Transaction {
	if tx.receipt.Err != nil {
		return tx // Return early if there's already an error
	}
	// The check for tx.txExtension.GetTransaction() != nil is removed.
	// If BuildTransaction is called on an already built tx, it should ideally handle it or error out.
	// Or, the client should ensure a fresh tx object or reset it if reusing.

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    tx.owner.Bytes(), // Use internal owner
		ContractAddress: contract.AddressBytes,
		Data:            data,
		CallValue:       callValue,
	}

	// Call BuildTransaction to get TransactionExtention
	txExt, err := tx.client.BuildTransaction(trigger)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build TriggerSmartContract transaction: %v", err)
		return tx
	}

	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}
