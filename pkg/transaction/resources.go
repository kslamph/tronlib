package transaction

import (
	"fmt"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

//freeze unfreeze delegate reclaim

// FreezeForBandwidth creates a freeze for bandwidth transaction
// resource code int32, 0 = bandwidth, 1 = energy, 2 = tronpower
func (tx *Transaction) Freeze(amount int64, resource core.ResourceCode) *Transaction {
	contract := &core.FreezeBalanceV2Contract{
		OwnerAddress:  tx.owner.Bytes(),
		FrozenBalance: amount,
		Resource:      resource,
	}
	// Call BuildTransaction to get TransactionExtention
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", err)
		return tx
	}

	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}

// FreezeForEnergy creates a freeze for energy transaction resource code int32, 0 = bandwidth, 1 = energy, 2 = tronpower
func (tx *Transaction) Unfreeze(amount int64, resource core.ResourceCode) *Transaction {
	contract := &core.UnfreezeBalanceV2Contract{
		OwnerAddress:    tx.owner.Bytes(),
		UnfreezeBalance: amount,
		Resource:        resource,
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", err)
		return tx
	}
	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}

func (tx *Transaction) Delegate(receiver *types.Address, amount int64, resource core.ResourceCode) *Transaction {
	contract := &core.DelegateResourceContract{
		OwnerAddress:    tx.owner.Bytes(),
		ReceiverAddress: receiver.Bytes(),
		Balance:         amount,
		Resource:        resource,
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", err)
		return tx
	}
	// fmt.Println(txExt.Result.Result)
	if !txExt.Result.Result {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", txExt.Result)
		return tx
	}
	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}

func (tx *Transaction) DelegateWithLock(to *types.Address, amount int64, resource core.ResourceCode, blocksToLock int64) *Transaction {
	contract := &core.DelegateResourceContract{
		OwnerAddress:    tx.owner.Bytes(), // Note: The original Freeze uses tx.owner.Bytes(). Assuming 'from' is intended here.
		ReceiverAddress: to.Bytes(),
		Balance:         amount,
		Resource:        resource,
		Lock:            true,
		LockPeriod:      blocksToLock,
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", err)
		return tx
	}
	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}

func (tx *Transaction) Reclaim(to *types.Address, amount int64, resource core.ResourceCode) *Transaction {
	contract := &core.UnDelegateResourceContract{
		OwnerAddress:    tx.owner.Bytes(), // Note: The original Freeze uses tx.owner.Bytes(). Assuming 'from' is intended here.
		ReceiverAddress: to.Bytes(),
		Balance:         amount,
		Resource:        resource,
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", err)
		return tx
	}
	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}

func (tx *Transaction) Vote() {
}
func (tx *Transaction) Unvote() {
}
func (tx *Transaction) Withdraw() *Transaction {
	contract := &core.WithdrawExpireUnfreezeContract{
		OwnerAddress: tx.owner.Bytes(), // Note: The original Freeze uses tx.owner.Bytes(). Assuming 'from' is intended here.
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", err)
		return tx
	}
	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}

func (tx *Transaction) ClaimReward() *Transaction {
	contract := &core.WithdrawBalanceContract{
		OwnerAddress: tx.owner.Bytes(), // Note: The original Freeze uses tx.owner.Bytes(). Assuming 'from' is intended here.
	}

	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		tx.receipt.Err = fmt.Errorf("failed to build transaction: %v", err)
		return tx
	}
	tx.txExtension = txExt
	tx.SetDefaultOptions()
	return tx
}
