package write_tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/kslamph/tronlib/pkg/account"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/eventdecoder"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/tyler-smith/go-bip39"
)

// loadEnv loads environment variables from the given path.
func loadEnv(path string) {
	if err := godotenv.Load(path); err != nil {
		log.Fatalf("Error loading .env file from %s: %v", path, err)
	}
}

// newTestNileClient creates a new gRPC client for the Nile testnet.
func newTestNileClient() (*client.Client, error) {
	nileNodeURL := os.Getenv("NILE_NODE_URL")
	if nileNodeURL == "" {
		return nil, fmt.Errorf("NILE_NODE_URL not set")
	}
	return client.NewClient(nileNodeURL)
}

// TestNileBroadcastTransaction tests creating, signing, and broadcasting TRX and TRC20 transfers.
func TestNileBroadcastTransaction(t *testing.T) {
	loadEnv("../../cmd/setup_nile_testnet/test.env")

	c, err := newTestNileClient()
	if err != nil {
		t.Fatalf("Failed to create Nile client: %v", err)
	}
	defer c.Close()

	am := account.NewManager(c)

	// Get sender's private key and address
	senderKey := os.Getenv("INTEGRATION_TEST_KEY1")
	if senderKey == "" {
		t.Fatal("INTEGRATION_TEST_KEY1 not set")
	}
	s, err := signer.NewPrivateKeySigner(senderKey)
	assert.NoError(t, err)
	senderAddress := s.Address()

	// Create a new random recipient account
	entropy, _ := bip39.NewEntropy(128)
	mnemonic, _ := bip39.NewMnemonic(entropy)
	wallet, _ := hdwallet.NewFromMnemonic(mnemonic)
	path, _ := hdwallet.ParseDerivationPath("m/44'/195'/0'/0/0")
	acc, _ := wallet.Derive(path, false)
	recipientAddr, _ := types.NewAddress(acc.Address.Hex())

	t.Run("TRX Transfer", func(t *testing.T) {
		// 1. Get sender's balance before transfer
		senderBalanceBefore, err := am.GetBalance(context.Background(), senderAddress)
		assert.NoError(t, err)

		// 2. Create TRX transfer transaction
		txExt, err := am.TransferTRX(context.Background(), senderAddress, recipientAddr, 1_000_000, nil)
		assert.NoError(t, err)

		// 3. Sign and broadcast the transaction
		res, err := c.SignAndBroadcast(context.Background(), txExt, client.DefaultBroadcastOptions(), s)
		assert.NoError(t, err)
		assert.True(t, res.Success, "Broadcast result was false")

		// 4. Validate balances after transfer
		senderBalanceAfter, err := am.GetBalance(context.Background(), senderAddress)
		assert.NoError(t, err)
		recipientBalanceAfter, err := am.GetBalance(context.Background(), recipientAddr)
		assert.NoError(t, err)

		assert.Equal(t, int64(1_000_000), recipientBalanceAfter)
		assert.True(t, senderBalanceBefore > senderBalanceAfter, "Sender balance should decrease")

		fmt.Printf("TRX transfer successful: %s\n", res.TxID)
	})

	t.Run("TRC20 Transfer", func(t *testing.T) {
		trc20ContractAddress, err := types.NewAddress(os.Getenv("TRC20_CONTRACT_ADDRESS"))
		assert.NoError(t, err)

		tm, err := trc20.NewManager(c, trc20ContractAddress)
		assert.NoError(t, err)
		// 2. Create TRC20 transfer transaction
		amount := decimal.NewFromInt(1)
		_, txExt, err := tm.Transfer(context.Background(), senderAddress, recipientAddr, amount)
		assert.NoError(t, err)

		// 3. Sign and broadcast the transaction
		res, err := c.SignAndBroadcast(context.Background(), txExt, client.DefaultBroadcastOptions(), s)
		assert.NoError(t, err)
		assert.True(t, res.Success, "Broadcast result was false")

		// 4. Decode and validate event logs
		decodedEvents, err := eventdecoder.DecodeLogs(res.Logs)
		assert.NoError(t, err)
		assert.Len(t, decodedEvents, 1, "Expected one event")

		transferEvent := decodedEvents[0]
		assert.Equal(t, "Transfer", transferEvent.EventName)
		assert.Len(t, transferEvent.Parameters, 3, "Expected three parameters in Transfer event")

		var from, to string
		var value string

		for _, param := range transferEvent.Parameters {
			switch param.Name {
			case "from":
				from = param.Value
			case "to":
				to = param.Value
			case "value":
				value = param.Value
			}
		}

		assert.Equal(t, senderAddress.String(), from)
		assert.Equal(t, recipientAddr.String(), to)
		assert.Equal(t, "1000000000000000000", value)

		// The amount in the event is the raw value, so we need to factor in the decimals (18)

		fmt.Printf("TRC20 transfer successful: %s\n", res.TxID)
	})
}
