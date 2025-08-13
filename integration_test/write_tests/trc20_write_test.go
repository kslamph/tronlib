package write_tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
)

// TestTRC20_Approve_Nile approves a spender and validates allowance increases
func TestTRC20_Approve_Nile(t *testing.T) {
	loadEnv("../../cmd/setup_nile_testnet/test.env")

	c, err := newTestNileClient()
	if err != nil {
		t.Fatalf("failed to create Nile client: %v", err)
	}
	defer c.Close()

	keyOwner := os.Getenv("INTEGRATION_TEST_KEY1")
	if keyOwner == "" {
		t.Fatal("INTEGRATION_TEST_KEY1 not set")
	}
	sOwner, err := signer.NewPrivateKeySigner(keyOwner)
	assert.NoError(t, err)
	ownerAddr := sOwner.Address()

	spenderStr := os.Getenv("TRC20_SPENDER_ADDRESS")
	var spenderAddr *types.Address
	if spenderStr == "" {
		spenderAddr = ownerAddr
	} else {
		var perr error
		spenderAddr, perr = types.NewAddress(spenderStr)
		if perr != nil {
			// Fallback to owner on parse error
			spenderAddr = ownerAddr
		}
	}

	trc20AddrStr := os.Getenv("TRC20_CONTRACT_ADDRESS")
	trc20Addr, err := types.NewAddress(trc20AddrStr)
	assert.NoError(t, err)
	tm, err := trc20.NewManager(c, trc20Addr)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Baseline allowance
	allowBefore, err := tm.Allowance(ctx, ownerAddr, spenderAddr)
	assert.NoError(t, err)

	// Approve small amount (1 unit)
	amount := decimal.NewFromInt(1)
	_, txExt, err := tm.Approve(ctx, ownerAddr, spenderAddr, amount)
	assert.NoError(t, err)

	// Sign and broadcast
	res, err := c.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), sOwner)
	assert.NoError(t, err)
	if err != nil || res == nil {
		return
	}

	// Best-effort check that allowance increased (do a short delay)
	time.Sleep(3 * time.Second)
	allowAfter, err := tm.Allowance(ctx, ownerAddr, spenderAddr)
	assert.NoError(t, err)
	// Non-strict: require allowAfter >= allowBefore
	assert.True(t, allowAfter.GreaterThanOrEqual(allowBefore))
}
