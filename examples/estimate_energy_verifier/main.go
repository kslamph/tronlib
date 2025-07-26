package main

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
	"github.com/kslamph/tronlib/pkg/workflow"
)

// loadEnv loads key-value pairs from the env file
func loadEnv(filePath string) (map[string]string, error) {
	env := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return env, scanner.Err()
}

func main() {
	ctx := context.Background()

	// Load env
	envPath := "/home/kslam/goproj/tronlib/cmd/setup_nile_testnet/test.env"
	env, err := loadEnv(envPath)
	if err != nil {
		fmt.Printf("Failed to load env: %v\n", err)
		return
	}

	nodeURL := env["NILE_NODE_URL"]
	privateKeyHex := env["INTEGRATION_TEST_KEY1"]
	trc20Address := env["TRC20_CONTRACT_ADDRESS"]

	// Create client
	cl, err := client.NewClient(client.DefaultClientConfig(nodeURL))
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	// Create signer
	s, err := signer.NewPrivateKeySigner(privateKeyHex)
	if err != nil {
		fmt.Printf("Failed to create signer: %v\n", err)
		return
	}

	// Build simple TRC20 transfer: transfer 1 unit to a dummy address
	toAddress := "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x" // Replace with a valid test address
	// amount := uint64(5000000000000000000)
	amount := big.NewInt(math.MaxInt64)
	amount.Mul(amount, big.NewInt(1))

	contract, err := smartcontract.NewContract(cl, types.MustNewAddressFromBase58(trc20Address), trc20.ERC20ABI)
	if err != nil {
		fmt.Printf("Failed to create contract: %v\n", err)
		return
	}

	// data, err := contract.EncodeInput("transfer", toAddress, amount)
	// if err != nil {
	// 	fmt.Printf("Failed to encode input: %v\n", err)
	// 	return
	// }

	// Create smartcontract manager

	// Create transaction extension using high-level API
	ext, err := contract.TriggerSmartContract(ctx, s.Address(), 0, "transfer", toAddress, amount)
	if err != nil {
		fmt.Printf("Failed to create transaction: %v\n", err)
		return
	}

	contEst, err := contract.TriggerConstantContract(ctx, s.Address(), "transfer", toAddress, amount)
	if err != nil {
		fmt.Printf("Failed to trigger constant contract: %v\n", err)
		return
	}
	fmt.Printf("contEst: %+v\n", contEst)

	// Create workflow
	wf := workflow.NewTransactionWorkflow(cl, ext)

	wf.SetFeeLimit(100000000)

	// Sign and broadcast
	wf.Sign(s)
	// Estimate fee
	estimatedFee, err := wf.EstimateFee(ctx)
	if err != nil {
		fmt.Printf("Failed to estimate fee: %v\n", err)
		return
	}
	fmt.Printf("Estimated fee: %d SUN\n", estimatedFee)

	// To verify energy, perhaps call estimateenergy directly if needed
	// But using the library's EstimateFee which includes energy estimation

	// Ask user
	var input string
	fmt.Print("Do you want to broadcast the transaction? (yes/no): ")
	fmt.Scanln(&input)
	if strings.ToLower(input) != "yes" {
		fmt.Println("Transaction not broadcasted.")
		return
	}

	txid, success, ret, txInfo, err := wf.Broadcast(ctx, 10)
	if err != nil {
		fmt.Printf("Failed to broadcast: %v\n", err)
		return
	}
	fmt.Printf("Broadcast result: TxID=%s, Success=%v, Return=%v, TxInfo=%v\n", txid, success, ret, txInfo)
}
