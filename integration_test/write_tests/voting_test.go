package write_tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/voting"
	"github.com/stretchr/testify/assert"
)

// TestVotingManager_Nile covers GetRewardInfo, GetBrokerageInfo, and attempts VoteWitnessAccount2
func TestVotingManager_Nile(t *testing.T) {
	loadEnv("../../cmd/setup_nile_testnet/test.env")

	c, err := newTestNileClient()
	if err != nil {
		t.Fatalf("failed to create Nile client: %v", err)
	}
	defer c.Close()

	vm := voting.NewManager(c)

	// Owner/sender from env
	key := os.Getenv("INTEGRATION_TEST_KEY1")
	if key == "" {
		t.Fatal("INTEGRATION_TEST_KEY1 not set")
	}
	s, err := signer.NewPrivateKeySigner(key)
	assert.NoError(t, err)
	ownerAddr := s.Address()

	newCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 60*time.Second)
	}

	t.Run("GetRewardInfo (owner)", func(t *testing.T) {
		ctx, cancel := newCtx()
		defer cancel()
		reward, err := vm.GetRewardInfo(ctx, ownerAddr)
		assert.NoError(t, err)
		if err == nil {
			assert.GreaterOrEqual(t, reward.GetNum(), int64(0))
		}
	})

	t.Run("GetBrokerageInfo (first witness)", func(t *testing.T) {
		ctx, cancel := newCtx()
		defer cancel()
		witnesses, err := vm.ListWitnesses(ctx)
		assert.NoError(t, err)
		if err != nil || witnesses == nil || len(witnesses.GetWitnesses()) == 0 {
			t.Skip("no witnesses returned; skipping brokerage info check")
		}
		w := witnesses.GetWitnesses()[0]
		wAddr := types.MustNewAddressFromBytes(w.GetAddress())
		bctx, bcancel := newCtx()
		defer bcancel()
		brokerage, err := vm.GetBrokerageInfo(bctx, wAddr)
		assert.NoError(t, err)
		if err == nil {
			assert.GreaterOrEqual(t, brokerage.GetNum(), int64(0))
			assert.LessOrEqual(t, brokerage.GetNum(), int64(100))
		}
	})

	t.Run("VoteWitnessAccount2 (attempt minimal vote)", func(t *testing.T) {
		ctx, cancel := newCtx()
		defer cancel()
		witnesses, err := vm.ListWitnesses(ctx)
		if err != nil || witnesses == nil || len(witnesses.GetWitnesses()) == 0 {
			t.Skip("no witnesses available to vote for")
			return
		}
		wAddr := types.MustNewAddressFromBytes(witnesses.GetWitnesses()[0].GetAddress())
		votes := []voting.Vote{{WitnessAddress: wAddr, VoteCount: 1}}

		vctx, vcancel := newCtx()
		defer vcancel()
		txExt, err := vm.VoteWitnessAccount2(vctx, ownerAddr, votes)
		assert.NoError(t, err)
		if err != nil || txExt == nil {
			return
		}

		// Sign and broadcast; tolerate failures (e.g., insufficient Tron Power)
		res, bErr := c.SignAndBroadcast(vctx, txExt, client.DefaultBroadcastOptions(), s)
		if bErr != nil {
			t.Logf("vote broadcast error (tolerated): %v", bErr)
			return
		}
		if res != nil {
			t.Logf("vote txid=%s success=%v msg=%s", res.TxID, res.Success, res.Message)
		}
	})
}
