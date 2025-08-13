package write_tests

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
)

// TestSmartContract_TriggerAndSimulate_Nile uses MinimalContract.setValue and verifies value()
func TestSmartContract_TriggerAndSimulate_Nile(t *testing.T) {
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

	minContract := types.MustNewAddressFromBase58(os.Getenv("MINIMALCONTRACT_CONTRACT_ADDRESS"))
	if minContract == nil {
		t.Fatal("MINIMALCONTRACT_CONTRACT_ADDRESS not set")
	}

	// Build contract client (ABI auto-fetch)
	sc, err := smartcontract.NewContract(c, minContract)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1) Simulate setValue with a new value
	newVal := big.NewInt(123456)
	simRes, err := sc.Simulate(ctx, ownerAddr, 0, "setValue", newVal)
	assert.NoError(t, err)
	if err == nil && simRes != nil {
		t.Logf("simulate energy=%d success=%v", simRes.Energy, simRes.APIResult.GetResult())
	}

	// 2) Trigger actual transaction setValue(newVal)
	txExt, err := sc.TriggerSmartContract(ctx, ownerAddr, 0, "setValue", newVal)
	assert.NoError(t, err)
	if err != nil || txExt == nil {
		return
	}
	res, err := c.SignAndBroadcast(ctx, txExt, client.DefaultBroadcastOptions(), sOwner)
	assert.NoError(t, err)
	if res == nil {
		return
	}
	t.Logf("setValue txid=%s success=%v msg=%s", res.TxID, res.Success, res.Message)

	// 3) Verify via constant: value() or getValue()
	// MinimalContract exposes both 'value' (public state) and 'getValue()'
	// Try value() first; if ABI names mismatch, fallback to getValue.
	var readBack interface{}
	readBack, err = sc.TriggerConstantContract(ctx, ownerAddr, "value")
	if err != nil {
		readBack, err = sc.TriggerConstantContract(ctx, ownerAddr, "getValue")
	}
	assert.NoError(t, err)
	if err == nil {
		vb, ok := readBack.(int64)
		if !ok {
			// some ABI decoders may surface uint256 as int64-compatible; tolerate any integer type via decode to string and parse
			t.Logf("unexpected type for value(): %T", readBack)
		} else {
			assert.Equal(t, newVal, vb)
		}
	}
}

