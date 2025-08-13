package write_tests

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/resources"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/stretchr/testify/assert"
)

// Reuse shared helpers `loadEnv` and `newTestNileClient` from this package.

// TestResourcesManager covers freeze/unfreeze, delegate/undelegate and read-only getters.
func TestResourcesManager_Nile(t *testing.T) {
	// if os.Getenv("RUN_NILE_WRITE_TESTS") != "true" {
	// 	t.Skip("RUN_NILE_WRITE_TESTS not set; skipping Nile write tests")
	// }

	loadEnv("../../cmd/setup_nile_testnet/test.env")

	c, err := newTestNileClient()
	if err != nil {
		t.Fatalf("failed to create Nile client: %v", err)
	}
	defer c.Close()

	rm := resources.NewManager(c)

	// Owner/sender from env
	key := os.Getenv("INTEGRATION_TEST_KEY1")
	if key == "" {
		t.Fatal("INTEGRATION_TEST_KEY1 not set")
	}
	s, err := signer.NewPrivateKeySigner(key)
	assert.NoError(t, err)
	ownerAddr := s.Address()

	key2 := os.Getenv("INTEGRATION_TEST_KEY2")
	if key2 == "" {
		t.Fatal("INTEGRATION_TEST_KEY2 not set")
	}
	s2, err := signer.NewPrivateKeySigner(key2)
	assert.NoError(t, err)
	receiverAddr := s2.Address()

	inactiveAddr := types.MustNewAddressFromBase58("TKGceTCruR62SwBwv7Vm5UGFy9qw1oVBmg")

	// Common timeouts
	newCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 60*time.Second)
	}

	t.Run("FreezeBalanceV2 and UnfreezeBalanceV2 (Energy)", func(t *testing.T) {
		ctx, cancel := newCtx()
		defer cancel()

		amount := int64(10_000_000) // 1 TRX in SUN

		txExt, err := rm.FreezeBalanceV2(ctx, ownerAddr, amount, resources.ResourceTypeEnergy)
		assert.NoError(t, err)
		if err != nil {
			t.Logf("freeze build failed: %v", err)
			return
		}

		res, err := c.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), s)
		assert.NoError(t, err)
		if err != nil || res == nil {
			t.Logf("freeze broadcast failed: %v", err)
			return
		}
		assert.True(t, res.Success, "freeze broadcast failed")
		t.Logf("freeze txid=%s", res.TxID)

		// Unfreeze different amount to leave some for delegation
		amount2 := int64(1_000_000) // 1 TRX in SUN
		ctx2, cancel2 := newCtx()
		defer cancel2()
		txExt2, err := rm.UnfreezeBalanceV2(ctx2, ownerAddr, amount2, resources.ResourceTypeEnergy)
		assert.NoError(t, err)
		if err != nil {
			t.Logf("unfreeze build failed: %v", err)
			return
		}
		res2, err := c.SignAndBroadcast(ctx2, txExt2, client.DefaultBroadcastOptions(), s)
		assert.NoError(t, err)
		if err != nil || res2 == nil {
			t.Logf("unfreeze broadcast failed: %v", err)
			return
		}
		assert.True(t, res2.Success, "unfreeze broadcast failed")
		t.Logf("unfreeze txid=%s", res2.TxID)
	})

	t.Run("DelegateResource to inactive address should fail (Energy)", func(t *testing.T) {
		ctx, cancel := newCtx()
		defer cancel()

		amount := int64(1_000_000) // 1 TRX in SUN
		_, err := rm.DelegateResource(ctx, ownerAddr, inactiveAddr, amount, resources.ResourceTypeEnergy, false)
		assert.Error(t, err)

		// Log detailed error information
		t.Logf("got expected error when delegating to inactive address: owner=%s inactive=%s amount=%d resource=%v err=%v", ownerAddr, inactiveAddr, amount, resources.ResourceTypeEnergy, err)
		var te *types.TronError
		if errors.As(err, &te) {
			t.Logf("tron error details: code=%d message=%s", te.Code, te.Message)
		}
	})

	t.Run("DelegateResource and UnDelegateResource (Energy, lock=false)", func(t *testing.T) {
		// Baseline query before delegation
		ctx0, cancel0 := newCtx()
		delegatedBefore, err := rm.GetDelegatedResourceV2(ctx0, ownerAddr, receiverAddr)
		cancel0()
		assert.NoError(t, err)
		if err != nil {
			t.Logf("baseline delegated query failed: %v", err)
			return
		}

		amount := int64(1_000_000) // 1 TRX in SUN
		ctx1, cancel1 := newCtx()
		txExt, err := rm.DelegateResource(ctx1, ownerAddr, receiverAddr, amount, resources.ResourceTypeEnergy, false)
		assert.NoError(t, err)
		if err != nil {
			cancel1()
			t.Logf("delegate build failed: %v", err)
			return
		}
		res, err := c.SignAndBroadcast(ctx1, txExt, client.DefaultBroadcastOptions(), s)
		cancel1()
		assert.NoError(t, err)
		if err != nil || res == nil {
			t.Logf("delegate broadcast failed: %v", err)
			return
		}
		assert.True(t, res.Success, "delegate broadcast failed")
		t.Logf("delegate txid=%s", res.TxID)

		// Verify delegation shows up (best-effort; structure only)
		time.Sleep(3 * time.Second)
		ctx2, cancel2 := newCtx()
		delegatedAfter, err := rm.GetDelegatedResourceV2(ctx2, ownerAddr, receiverAddr)
		cancel2()
		assert.NoError(t, err)
		if err != nil {
			t.Logf("post-delegate query failed: %v", err)
			return
		}
		if delegatedBefore != nil && delegatedAfter != nil {
			// Allow either equal or greater count depending on network timing
			assert.GreaterOrEqual(t, len(delegatedAfter.GetDelegatedResource()), len(delegatedBefore.GetDelegatedResource()))
		}

		// Undelegate same amount
		ctx3, cancel3 := newCtx()
		txExt2, err := rm.UnDelegateResource(ctx3, ownerAddr, receiverAddr, amount, resources.ResourceTypeEnergy)
		assert.NoError(t, err)
		if err != nil {
			cancel3()
			t.Logf("undelegate build failed: %v", err)
			return
		}
		res2, err := c.SignAndBroadcast(ctx3, txExt2, client.DefaultBroadcastOptions(), s)
		cancel3()
		assert.NoError(t, err)
		if err != nil || res2 == nil {
			t.Logf("undelegate broadcast failed: %v", err)
			return
		}
		assert.True(t, res2.Success, "undelegate broadcast failed")
		t.Logf("undelegate txid=%s", res2.TxID)
	})

	t.Run("CancelAllUnfreezeV2 and WithdrawExpireUnfreeze (no-op safe)", func(t *testing.T) {
		ctx1, cancel1 := newCtx()
		txExt, err := rm.CancelAllUnfreezeV2(ctx1, ownerAddr)
		assert.NoError(t, err)
		if err != nil {
			cancel1()
			t.Logf("cancel-all-unfreeze build failed: %v", err)
			return
		}
		res, err := c.SignAndBroadcast(ctx1, txExt, client.DefaultBroadcastOptions(), s)
		cancel1()
		assert.NoError(t, err)
		if err != nil || res == nil {
			t.Logf("cancel-all-unfreeze broadcast failed: %v", err)
			return
		}
		assert.True(t, res.Success, "cancel-all-unfreeze broadcast failed")
		t.Logf("cancel-all-unfreeze txid=%s", res.TxID)

		ctx2, cancel2 := newCtx()
		txExt2, err := rm.WithdrawExpireUnfreeze(ctx2, ownerAddr)
		assert.NoError(t, err)
		if err != nil {
			cancel2()
			t.Logf("withdraw-expire-unfreeze build failed: %v", err)
			return
		}
		res2, err := c.SignAndBroadcast(ctx2, txExt2, client.DefaultBroadcastOptions(), s)
		cancel2()
		assert.NoError(t, err)
		if err != nil || res2 == nil {
			t.Logf("withdraw-expire-unfreeze broadcast failed: %v", err)
			return
		}
		assert.True(t, res2.Success, "withdraw-expire-unfreeze broadcast failed")
		t.Logf("withdraw-expire-unfreeze txid=%s", res2.TxID)
	})

	t.Run("Read-only resource queries", func(t *testing.T) {
		// Account index
		ctx1, cancel1 := newCtx()
		idx, err := rm.GetDelegatedResourceAccountIndexV2(ctx1, ownerAddr)
		cancel1()
		assert.NoError(t, err)
		assert.NotNil(t, idx)

		// Max delegatable size (Energy=1)
		ctx2, cancel2 := newCtx()
		maxSize, err := rm.GetCanDelegatedMaxSize(ctx2, ownerAddr, 1)
		cancel2()
		assert.NoError(t, err)
		assert.NotNil(t, maxSize)

		// Available unfreeze count
		ctx3, cancel3 := newCtx()
		cnt, err := rm.GetAvailableUnfreezeCount(ctx3, ownerAddr)
		cancel3()
		assert.NoError(t, err)
		assert.NotNil(t, cnt)

		// Can withdraw unfreeze amount at current timestamp
		ctx4, cancel4 := newCtx()
		nowMs := time.Now().UnixMilli()
		canW, err := rm.GetCanWithdrawUnfreezeAmount(ctx4, ownerAddr, nowMs)
		cancel4()
		assert.NoError(t, err)
		assert.NotNil(t, canW)
	})
}
