package transaction

import (
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

//freeze unfreeze delegate reclaim

// FreezeForBandwidth creates a freeze for bandwidth transaction
// resource code int32, 0 = bandwidth, 1 = energy, 2 = tronpower
func (tx *Transaction) Freeze(from *types.Address, amount int64, resource core.ResourceCode) error {
	contract := &core.FreezeBalanceV2Contract{
		OwnerAddress:  from.Bytes(),
		FrozenBalance: amount,
		Resource:      resource,
	}
	// Call BuildTransaction to get TransactionExtention
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		return err
	}

	tx.txExtension = txExt

	return nil
}

// FreezeForEnergy creates a freeze for energy transaction resource code int32, 0 = bandwidth, 1 = energy, 2 = tronpower
func (tx *Transaction) Unfreeze(from *types.Address, amount int64, resource core.ResourceCode) error {
	contract := &core.UnfreezeBalanceV2Contract{
		OwnerAddress:    from.Bytes(),
		UnfreezeBalance: amount,
		Resource:        resource,
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		return err
	}
	tx.txExtension = txExt
	return nil
}

func (tx *Transaction) Delegate(from, to *types.Address, amount int64, resource core.ResourceCode) error {
	contract := &core.DelegateResourceContract{
		OwnerAddress:    from.Bytes(),
		ReceiverAddress: to.Bytes(),
		Balance:         amount,
		Resource:        resource,
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		return err
	}
	tx.txExtension = txExt
	return nil
}

func (tx *Transaction) DelegateWithLock(from, to *types.Address, amount int64, resource core.ResourceCode, blocksToLock int64) error {
	contract := &core.DelegateResourceContract{
		OwnerAddress:    from.Bytes(),
		ReceiverAddress: to.Bytes(),
		Balance:         amount,
		Resource:        resource,
		Lock:            true,
		LockPeriod:      blocksToLock,
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		return err
	}
	tx.txExtension = txExt
	return nil
}

func (tx *Transaction) Reclaim(from, to *types.Address, amount int64, resource core.ResourceCode) error {
	contract := &core.UnDelegateResourceContract{
		OwnerAddress:    from.Bytes(),
		ReceiverAddress: to.Bytes(),
		Balance:         amount,
		Resource:        resource,
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		return err
	}
	tx.txExtension = txExt
	return nil
}

func (tx *Transaction) Vote() {
}
func (tx *Transaction) Unvote() {
}
func (tx *Transaction) Withdraw(from *types.Address) error {
	contract := &core.WithdrawExpireUnfreezeContract{
		OwnerAddress: from.Bytes(),
	}
	txExt, err := tx.client.BuildTransaction(contract)
	if err != nil {
		return err
	}
	tx.txExtension = txExt
	return nil
}

func (tx *Transaction) ClaimReward() {
}
