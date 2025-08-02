package main

import (
	"bufio"
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
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
	sunswapabi := `[{"constant":false,"inputs":[{"name":"tokens_sold","type":"uint256"},{"name":"min_trx","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"}],"name":"tokenToTrxTransferInput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"trx_bought","type":"uint256"},{"name":"max_tokens","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"}],"name":"tokenToTrxTransferOutput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"addedValue","type":"uint256"}],"name":"increaseAllowance","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"min_liquidity","type":"uint256"},{"name":"max_tokens","type":"uint256"},{"name":"deadline","type":"uint256"}],"name":"addLiquidity","outputs":[{"name":"","type":"uint256"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"min_tokens","type":"uint256"},{"name":"deadline","type":"uint256"}],"name":"trxToTokenSwapInput","outputs":[{"name":"","type":"uint256"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"token_addr","type":"address"}],"name":"setup","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"tokens_bought","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"}],"name":"trxToTokenTransferOutput","outputs":[{"name":"","type":"uint256"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":true,"inputs":[{"name":"input_amount","type":"uint256"},{"name":"input_reserve","type":"uint256"},{"name":"output_reserve","type":"uint256"}],"name":"getInputPrice","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"tokens_sold","type":"uint256"}],"name":"getTokenToTrxInputPrice","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"factoryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"trx_sold","type":"uint256"}],"name":"getTrxToTokenInputPrice","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"tokens_bought","type":"uint256"},{"name":"max_tokens_sold","type":"uint256"},{"name":"max_trx_sold","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"},{"name":"exchange_addr","type":"address"}],"name":"tokenToExchangeTransferOutput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"tokens_sold","type":"uint256"},{"name":"min_trx","type":"uint256"},{"name":"deadline","type":"uint256"}],"name":"tokenToTrxSwapInput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"trx_bought","type":"uint256"},{"name":"max_tokens","type":"uint256"},{"name":"deadline","type":"uint256"}],"name":"tokenToTrxSwapOutput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"tokenAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"subtractedValue","type":"uint256"}],"name":"decreaseAllowance","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"tokens_bought","type":"uint256"}],"name":"getTrxToTokenOutputPrice","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"tokens_bought","type":"uint256"},{"name":"deadline","type":"uint256"}],"name":"trxToTokenSwapOutput","outputs":[{"name":"","type":"uint256"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"tokens_bought","type":"uint256"},{"name":"max_tokens_sold","type":"uint256"},{"name":"max_trx_sold","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"token_addr","type":"address"}],"name":"tokenToTokenSwapOutput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"tokens_sold","type":"uint256"},{"name":"min_tokens_bought","type":"uint256"},{"name":"min_trx_bought","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"exchange_addr","type":"address"}],"name":"tokenToExchangeSwapInput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"min_tokens","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"}],"name":"trxToTokenTransferInput","outputs":[{"name":"","type":"uint256"}],"payable":true,"stateMutability":"payable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"tokens_sold","type":"uint256"},{"name":"min_tokens_bought","type":"uint256"},{"name":"min_trx_bought","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"token_addr","type":"address"}],"name":"tokenToTokenSwapInput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"tokens_bought","type":"uint256"},{"name":"max_tokens_sold","type":"uint256"},{"name":"max_trx_sold","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"exchange_addr","type":"address"}],"name":"tokenToExchangeSwapOutput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"tokens_sold","type":"uint256"},{"name":"min_tokens_bought","type":"uint256"},{"name":"min_trx_bought","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"},{"name":"exchange_addr","type":"address"}],"name":"tokenToExchangeTransferInput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"trx_bought","type":"uint256"}],"name":"getTokenToTrxOutputPrice","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"tokens_bought","type":"uint256"},{"name":"max_tokens_sold","type":"uint256"},{"name":"max_trx_sold","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"},{"name":"token_addr","type":"address"}],"name":"tokenToTokenTransferOutput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"tokens_sold","type":"uint256"},{"name":"min_tokens_bought","type":"uint256"},{"name":"min_trx_bought","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"},{"name":"token_addr","type":"address"}],"name":"tokenToTokenTransferInput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"amount","type":"uint256"},{"name":"min_trx","type":"uint256"},{"name":"min_tokens","type":"uint256"},{"name":"deadline","type":"uint256"}],"name":"removeLiquidity","outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"output_amount","type":"uint256"},{"name":"input_reserve","type":"uint256"},{"name":"output_reserve","type":"uint256"}],"name":"getOutputPrice","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"buyer","type":"address"},{"indexed":true,"name":"trx_sold","type":"uint256"},{"indexed":true,"name":"tokens_bought","type":"uint256"}],"name":"TokenPurchase","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"buyer","type":"address"},{"indexed":true,"name":"tokens_sold","type":"uint256"},{"indexed":true,"name":"trx_bought","type":"uint256"}],"name":"TrxPurchase","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"provider","type":"address"},{"indexed":true,"name":"trx_amount","type":"uint256"},{"indexed":true,"name":"token_amount","type":"uint256"}],"name":"AddLiquidity","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"provider","type":"address"},{"indexed":true,"name":"trx_amount","type":"uint256"},{"indexed":true,"name":"token_amount","type":"uint256"}],"name":"RemoveLiquidity","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"operator","type":"address"},{"indexed":true,"name":"trx_balance","type":"uint256"},{"indexed":true,"name":"token_balance","type":"uint256"}],"name":"Snapshot","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"}]`
	// nodeURL := env["NILE_NODE_URL"]
	nodeURL := "127.0.0.1:50051" // Use a default value for testing
	// privateKeyHex := env["INTEGRATION_TEST_KEY1"]
	trc20Address := env["TRC20_CONTRACT_ADDRESS"]

	// Create client
	cl, err := client.NewClient(client.DefaultClientConfig(nodeURL))
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer cl.Close()

	// Build simple TRC20 transfer: transfer 1 unit to a dummy address
	toAddress := "TBkfmcE7pM8cwxEhATtkMFwAf1FeQcwY9x" // Replace with a valid test address
	// amount := uint64(5000000000000000000)
	// amount := big.NewInt(math.MaxInt64)
	amount := big.NewInt(100)
	trc20Address = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // Replace with a valid TRC20 contract address
	contract, err := smartcontract.NewContract(cl, types.MustNewAddressFromBase58(trc20Address), sunswapabi)
	if err != nil {
		fmt.Printf("Failed to create contract: %v\n", err)
		return
	}

	from := types.MustNewAddressFromBase58("TLLFLYNg3bYxmDmpWfeZs2JmrNrEcM4tjX")

	simResult, err := contract.Simulate(ctx, from, 0, "transfer", toAddress, amount)
	if err != nil {
		fmt.Printf("Failed to simulate contract call: %v\n", err)
		return
	}
	fmt.Println("Simulation Result:", simResult)

	apiext, err := contract.TriggerSmartContract(ctx, from, 0, "transfer", toAddress, amount)
	if err != nil {
		fmt.Printf("Failed to trigger smart contract: %v\n", err)
		return
	}
	fmt.Println("Transaction ID:", apiext)
	en := apiext.GetEnergyUsed()
	fmt.Println("Energy Used:", en)

}
