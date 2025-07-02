package transaction

import (
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

//freeze unfreeze delegate reclaim

// Freeze TRX for resources
// resource code int32, 0 = bandwidth, 1 = energy, 2 = tronpower
func (tx *Transaction) Freeze(amount int64, resource core.ResourceCode) *Transaction {
	return tx.createResourceTransaction("freeze", amount, resource, tx.client.CreateFreezeTransaction)
}

// Unfreeze release resources, pending cooldown and eventually made TRX available
// resource code int32, 0 = bandwidth, 1 = energy, 2 = tronpower
func (tx *Transaction) Unfreeze(amount int64, resource core.ResourceCode) *Transaction {
	return tx.createResourceTransaction("unfreeze", amount, resource, tx.client.CreateUnfreezeTransaction)
}

func (tx *Transaction) Delegate(receiver *types.Address, amount int64, resource core.ResourceCode) *Transaction {
	return tx.createDelegateTransaction("delegate", receiver, amount, resource, false, 0)
}

func (tx *Transaction) DelegateWithLock(to *types.Address, amount int64, resource core.ResourceCode, blocksToLock int64) *Transaction {
	return tx.createDelegateTransaction("delegate with lock", to, amount, resource, true, blocksToLock)
}

func (tx *Transaction) Reclaim(to *types.Address, amount int64, resource core.ResourceCode) *Transaction {
	return tx.createUndelegateTransaction("reclaim", to, amount, resource)
}

// TODO: Implement Vote() when voting functionality is needed
// TODO: Implement Unvote() when voting functionality is needed

func (tx *Transaction) Withdraw() *Transaction {
	return tx.createWithdrawTransaction("withdraw", tx.client.CreateWithdrawExpireUnfreezeTransaction)
}

func (tx *Transaction) ClaimReward() *Transaction {
	return tx.createWithdrawTransaction("claim reward", tx.client.CreateWithdrawBalanceTransaction)
}
