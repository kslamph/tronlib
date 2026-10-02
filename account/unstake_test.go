package account

// Tests for the unstake/delegation builders and the handle accessors. These
// forward to tx.Build*, so what a regression would break is the *forwarding*:
// the owner bound into the contract, the resource and amount, and which builder
// gets called at all. Each assertion reads the request the fake node captured,
// not just that the call returned no error.

import (
	"testing"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourcesUnstakeLifecycleBuilders checks the three ways TRX leaves a
// stake, which are distinct operations on the chain and must not be confused:
// Unstake names a resource and an amount, WithdrawUnstaked and CancelUnstake
// take no amount at all.
func TestResourcesUnstakeLifecycleBuilders(t *testing.T) {
	f := &fakeWalletServer{}
	r := newTestHandle(t, f).Resources()

	if _, err := r.Unstake(t.Context(), tx.ResourceBandwidth, tron.TRX(7)); err != nil {
		t.Fatalf("Unstake: %v", err)
	}
	require.NotNil(t, f.gotUnfreeze, "the fake never saw an UnfreezeBalanceV2 request")
	assert.Equal(t, int64(7_000_000), f.gotUnfreeze.GetUnfreezeBalance(),
		"the unstake amount must reach UnfreezeBalance (FreezeBalanceV2 spells the same field FrozenBalance; the two contracts are not symmetric)")
	assert.Equal(t, core.ResourceCode_BANDWIDTH, f.gotUnfreeze.GetResource())
	assert.Equal(t, testFrom.Bytes(), f.gotUnfreeze.GetOwnerAddress(),
		"an unstake must be built for the bound owner")

	// Withdraw and Cancel are the amount-free halves of the lifecycle: the
	// node decides how much matures, so a builder that started forwarding an
	// amount here would be a different operation than the doc promises.
	if _, err := r.WithdrawUnstaked(t.Context()); err != nil {
		t.Fatalf("WithdrawUnstaked: %v", err)
	}
	require.NotNil(t, f.gotWithdraw, "the fake never saw a WithdrawExpireUnfreeze request")
	assert.Equal(t, testFrom.Bytes(), f.gotWithdraw.GetOwnerAddress())

	if _, err := r.CancelUnstake(t.Context()); err != nil {
		t.Fatalf("CancelUnstake: %v", err)
	}
	require.NotNil(t, f.gotCancel, "the fake never saw a CancelAllUnfreezeV2 request")
	assert.Equal(t, testFrom.Bytes(), f.gotCancel.GetOwnerAddress())
}

// TestResourcesDelegationBuilders pins the direction of a delegation: the owner
// is the funder, the receiver is the party that spends the resource, and the
// lock options reach the contract only when asked for.
func TestResourcesDelegationBuilders(t *testing.T) {
	f := &fakeWalletServer{}
	r := newTestHandle(t, f).Resources()

	if _, err := r.Delegate(t.Context(), tx.ResourceEnergy, testTo, tron.TRX(3), tx.DelegateOptions{}); err != nil {
		t.Fatalf("Delegate: %v", err)
	}
	require.NotNil(t, f.gotDelegate)
	assert.Equal(t, testFrom.Bytes(), f.gotDelegate.GetOwnerAddress())
	assert.Equal(t, testTo.Bytes(), f.gotDelegate.GetReceiverAddress(),
		"the receiver must be the counterparty, not the owner")
	assert.Equal(t, int64(3_000_000), f.gotDelegate.GetBalance())
	assert.Equal(t, core.ResourceCode_ENERGY, f.gotDelegate.GetResource())
	assert.False(t, f.gotDelegate.GetLock(), "DelegateOptions{} must not lock")
	assert.Zero(t, f.gotDelegate.GetLockPeriod())

	// A locked delegation is the DApp-operator guarantee, so the lock fields
	// are the whole point of the call.
	if _, err := r.Delegate(t.Context(), tx.ResourceBandwidth, testHot, tron.TRX(1),
		tx.DelegateOptions{Lock: true, LockBlocks: 100}); err != nil {
		t.Fatalf("locked Delegate: %v", err)
	}
	assert.True(t, f.gotDelegate.GetLock())
	assert.Equal(t, int64(100), f.gotDelegate.GetLockPeriod())
	assert.Equal(t, testHot.Bytes(), f.gotDelegate.GetReceiverAddress())

	if _, err := r.Undelegate(t.Context(), tx.ResourceEnergy, testTo, tron.TRX(3)); err != nil {
		t.Fatalf("Undelegate: %v", err)
	}
	require.NotNil(t, f.gotUndelegate)
	assert.Equal(t, testFrom.Bytes(), f.gotUndelegate.GetOwnerAddress())
	assert.Equal(t, testTo.Bytes(), f.gotUndelegate.GetReceiverAddress())
	assert.Equal(t, int64(3_000_000), f.gotUndelegate.GetBalance())
}

// TestHandleAccessorsReportTheBoundOwner: Address() and the three sub-handle
// Owner()s are the only way to read back what a handle is bound to, and each
// sub-handle is constructed from the parent, so a wiring mistake would silently
// retarget operations at a different account.
func TestHandleAccessorsReportTheBoundOwner(t *testing.T) {
	f := &fakeWalletServer{}
	h := newTestHandle(t, f)

	assert.Equal(t, testFrom, h.Address())
	assert.Equal(t, testFrom, h.Resources().Owner())
	assert.Equal(t, testFrom, h.Permissions().Owner())
	assert.Equal(t, testFrom, h.Voting().Owner())

	other := mustAddr(0x99)
	h2 := New(newTestClient(t, f), other)
	assert.Equal(t, other, h2.Address(), "a handle binds exactly one owner")
	assert.Equal(t, other, h2.Resources().Owner(), "sub-handles inherit the parent's owner")
	assert.Equal(t, other, h2.Permissions().Owner())
	assert.Equal(t, other, h2.Voting().Owner())
}

// TestBalance reads the spendable TRX, including the case the doc comment
// promises: an account the node does not know is a zero balance, not an error.
func TestBalance(t *testing.T) {
	t.Run("known account", func(t *testing.T) {
		f := &fakeWalletServer{account: richAccount()}
		h := newTestHandle(t, f)
		bal, err := h.Balance(t.Context())
		require.NoError(t, err)
		assert.Equal(t, tron.SUN(5_000_000), bal)
	})

	t.Run("unknown account is zero not error", func(t *testing.T) {
		f := &fakeWalletServer{} // no account set: the node answers with a shell
		h := New(newTestClient(t, f), testTo)
		bal, err := h.Balance(t.Context())
		require.NoError(t, err, "a recipient without an account yet must not fail")
		assert.Equal(t, tron.SUN(0), bal)
	})
}
