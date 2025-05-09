package transaction

import (
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

// TriggerSmartContract triggers a smart contract call
func (tx *Transaction) TriggerSmartContract(contract *smartcontract.Contract, ownerAddress *types.Address, data []byte, callValue int64) error {
	if tx.txExtension.GetTransaction() != nil {
		return fmt.Errorf("transaction already created")
	}

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddress.Bytes(),
		ContractAddress: []byte(contract.Address),
		Data:            data,
		CallValue:       callValue,
	}

	// Call BuildTransaction to get TransactionExtention
	txExt, err := tx.client.BuildTransaction(trigger)
	if err != nil {
		return err
	}

	tx.txExtension = txExt

	return nil
}
