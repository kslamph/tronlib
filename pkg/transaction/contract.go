package transaction

import (
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
)

// Contract represents a smart contract interface
type Contract struct {
	abi     string
	address string
}

// NewContract creates a new contract instance
func NewContract(abi string, address string) *Contract {
	return &Contract{
		abi:     abi,
		address: address,
	}
}

// ContractTrigger creates contract call data
func ContractTrigger(method string, params ...interface{}) ([]byte, error) {
	// TODO: Implement ABI encoding
	// This will require implementing proper ABI encoding logic
	// For now returning a placeholder error
	return nil, fmt.Errorf("ABI encoding not implemented yet")
}

// TriggerSmartContract triggers a smart contract call
func (tx *Transaction) TriggerSmartContract(contract *Contract, data []byte, callValue int64) error {
	if tx.txExtension.GetTransaction() != nil {
		return fmt.Errorf("transaction already created")
	}

	// Create trigger smart contract message
	trigger := &core.TriggerSmartContract{
		OwnerAddress:    tx.senderAccount.Address().Bytes(),
		ContractAddress: []byte(contract.address),
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
