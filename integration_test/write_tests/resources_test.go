package write_tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/resources"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"github.com/stretchr/testify/assert"
	"github.com/tyler-smith/go-bip39"
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

	// Ephemeral receiver address for delegation tests
	entropy, _ := bip39.NewEntropy(128)
	mnemonic, _ := bip39.NewMnemonic(entropy)
	wallet, _ := hdwallet.NewFromMnemonic(mnemonic)
	path, _ := hdwallet.ParseDerivationPath("m/44'/195'/0'/0/0")
	acc, _ := wallet.Derive(path, false)
	receiverAddr, _ := types.NewAddress(acc.Address.Hex())

	// Common timeouts
	newCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 60*time.Second)
	}

	t.Run("FreezeBalanceV2 and UnfreezeBalanceV2 (Energy)", func(t *testing.T) {
		ctx, cancel := newCtx()
		defer cancel()

		amount := int64(1_000_000) // 1 TRX in SUN

		txExt, err := rm.FreezeBalanceV2(ctx, ownerAddr, amount, resources.ResourceTypeEnergy)
		assert.NoError(t, err)

		res, err := c.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), s)
		assert.NoError(t, err)
		assert.True(t, res.Success, "freeze broadcast failed")
		t.Logf("freeze txid=%s", res.TxID)

		// Unfreeze same amount
		ctx2, cancel2 := newCtx()
		defer cancel2()
		txExt2, err := rm.UnfreezeBalanceV2(ctx2, ownerAddr, amount, resources.ResourceTypeEnergy)
		assert.NoError(t, err)
		res2, err := c.SignAndBroadcast(ctx2, txExt2, client.DefaultBroadcastOptions(), s)
		assert.NoError(t, err)
		assert.True(t, res2.Success, "unfreeze broadcast failed")
		t.Logf("unfreeze txid=%s", res2.TxID)
	})

	t.Run("DelegateResource and UnDelegateResource (Energy, lock=false)", func(t *testing.T) {
		// Baseline query before delegation
		ctx0, cancel0 := newCtx()
		delegatedBefore, err := rm.GetDelegatedResourceV2(ctx0, ownerAddr, receiverAddr)
		cancel0()
		assert.NoError(t, err)

		amount := int64(1_000_000) // 1 TRX in SUN
		ctx1, cancel1 := newCtx()
		txExt, err := rm.DelegateResource(ctx1, ownerAddr, receiverAddr, amount, resources.ResourceTypeEnergy, false)
		assert.NoError(t, err)
		res, err := c.SignAndBroadcast(ctx1, txExt, client.DefaultBroadcastOptions(), s)
		cancel1()
		assert.NoError(t, err)
		assert.True(t, res.Success, "delegate broadcast failed")
		t.Logf("delegate txid=%s", res.TxID)

		// Verify delegation shows up (best-effort; structure only)
		time.Sleep(3 * time.Second)
		ctx2, cancel2 := newCtx()
		delegatedAfter, err := rm.GetDelegatedResourceV2(ctx2, ownerAddr, receiverAddr)
		cancel2()
		assert.NoError(t, err)
		if delegatedBefore != nil && delegatedAfter != nil {
			// Allow either equal or greater count depending on network timing
			assert.GreaterOrEqual(t, len(delegatedAfter.GetDelegatedResource()), len(delegatedBefore.GetDelegatedResource()))
		}

		// Undelegate same amount
		ctx3, cancel3 := newCtx()
		txExt2, err := rm.UnDelegateResource(ctx3, ownerAddr, receiverAddr, amount, resources.ResourceTypeEnergy)
		assert.NoError(t, err)
		res2, err := c.SignAndBroadcast(ctx3, txExt2, client.DefaultBroadcastOptions(), s)
		cancel3()
		assert.NoError(t, err)
		assert.True(t, res2.Success, "undelegate broadcast failed")
		t.Logf("undelegate txid=%s", res2.TxID)
	})

	t.Run("CancelAllUnfreezeV2 and WithdrawExpireUnfreeze (no-op safe)", func(t *testing.T) {
		ctx1, cancel1 := newCtx()
		txExt, err := rm.CancelAllUnfreezeV2(ctx1, ownerAddr)
		assert.NoError(t, err)
		res, err := c.SignAndBroadcast(ctx1, txExt, client.DefaultBroadcastOptions(), s)
		cancel1()
		assert.NoError(t, err)
		assert.True(t, res.Success, "cancel-all-unfreeze broadcast failed")
		t.Logf("cancel-all-unfreeze txid=%s", res.TxID)

		ctx2, cancel2 := newCtx()
		txExt2, err := rm.WithdrawExpireUnfreeze(ctx2, ownerAddr)
		assert.NoError(t, err)
		res2, err := c.SignAndBroadcast(ctx2, txExt2, client.DefaultBroadcastOptions(), s)
		cancel2()
		assert.NoError(t, err)
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
