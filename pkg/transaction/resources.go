package transaction

import (
	"context"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/types"
)

//freeze unfreeze delegate reclaim

// Freeze TRX for resources
// resource code int32, 0 = bandwidth, 1 = energy, 2 = tronpower
func (tx *Transaction) Freeze(ctx context.Context, amount int64, resource core.ResourceCode) *Transaction {
	return tx.createResourceTransaction(ctx, "freeze", amount, resource, tx.client.CreateFreezeTransaction)
}

// Unfreeze release resources, pending cooldown and eventually made TRX available
// resource code int32, 0 = bandwidth, 1 = energy, 2 = tronpower
func (tx *Transaction) Unfreeze(ctx context.Context, amount int64, resource core.ResourceCode) *Transaction {
	return tx.createResourceTransaction(ctx, "unfreeze", amount, resource, tx.client.CreateUnfreezeTransaction)
}

func (tx *Transaction) Delegate(ctx context.Context, receiver *types.Address, amount int64, resource core.ResourceCode) *Transaction {
	return tx.createDelegateTransaction(ctx, "delegate", receiver, amount, resource, false, 0)
}

func (tx *Transaction) DelegateWithLock(ctx context.Context, to *types.Address, amount int64, resource core.ResourceCode, blocksToLock int64) *Transaction {
	return tx.createDelegateTransaction(ctx, "delegate with lock", to, amount, resource, true, blocksToLock)
}

func (tx *Transaction) Reclaim(ctx context.Context, to *types.Address, amount int64, resource core.ResourceCode) *Transaction {
	return tx.createUndelegateTransaction(ctx, "reclaim", to, amount, resource)
}

// TODO: Implement Vote() when voting functionality is needed
// TODO: Implement Unvote() when voting functionality is needed

func (tx *Transaction) Withdraw(ctx context.Context) *Transaction {
	return tx.createWithdrawTransaction(ctx, "withdraw", tx.client.CreateWithdrawExpireUnfreezeTransaction)
}

func (tx *Transaction) ClaimReward(ctx context.Context) *Transaction {
	return tx.createWithdrawTransaction(ctx, "claim reward", tx.client.CreateWithdrawBalanceTransaction)
}
