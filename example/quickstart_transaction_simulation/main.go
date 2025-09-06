// This is a more advanced example of transaction simulation.
// It is based on the content of example/simulate/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/eventdecoder"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	// Mainnet TRC20 USDT contract address
	usdtContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	// Demo node endpoint (explicit scheme required)
	nodeAddr = "grpc://grpc.trongrid.io:50051"

	// Demo addresses provided in task
	addrA   = "TV6MuMXfmLbBqPZvBHdwFsDnQeVfnmiuSi"
	addrB   = "TAUN6FwrnwwmaEqYcckffC7wYmbaS6cBiX"
	swapABI = `[{"constant":false,"inputs":[{"name":"tokens_sold","type":"uint256"},{"name":"min_trx","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"}],"name":"tokenToTrxTransferInput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"trx_bought","type":"uint256"},{"name":"max_tokens","type":"uint256"},{"name":"deadline","type":"uint256"},{"name":"recipient","type":"address"}],"name":"tokenToTrxTransferOutput","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"addedValue","type":"uint256"}],"name":"increaseAllowance","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"min_liquidity","type":"uint256"},{"name":"max_tokens","type":"uint256"},{"name":"deadline","type":"uint256"}],"name":"addLiquidity","outputs":[{"name":"","type":"uint256"}],"payable":true,"stateMutability":"payable","type":"function"}]`
)

func main() {
	// Context with an overall timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create client
	cli, err := client.NewClient(nodeAddr)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer cli.Close()

	// Parse addresses

	token, err := types.NewAddress(usdtContract)
	if err != nil {
		log.Fatalf("invalid USDT contract address: %v", err)
	}
	fromA, err := types.NewAddress(addrA)
	if err != nil {
		log.Fatalf("invalid address A: %v", err)
	}
	toB, err := types.NewAddress(addrB)
	if err != nil {
		log.Fatalf("invalid address B: %v", err)
	}

	// Build an TRC20 manager for USDT

	trc20Mgr := cli.TRC20(token)
	if trc20Mgr == nil {
		log.Fatal("Failed to create TRC20 manager")
	}

	// Register TRC20 ABI once for event decoding
	if err := eventdecoder.RegisterABIJSON(trc20.ERC20ABI); err != nil {
		log.Fatalf("failed to register TRC20 ABI: %v", err)
	}

	amount := decimal.NewFromInt(10) // 10 USDT

	fmt.Println("-- Case 1: A -> B transfer 10 USDT (expected: success) --")
	if err := simulateTransfer(ctx, cli, trc20Mgr, fromA, toB, amount); err != nil {
		log.Printf("simulate A->B error: %v", err)
	}

	fmt.Println()
	fmt.Println("-- Case 2: B -> A transfer 10 USDT (expected: REVERTED) --")
	if err := simulateTransfer(ctx, cli, trc20Mgr, toB, fromA, amount); err != nil {
		log.Printf("simulate B->A error: %v", err)
	}

	fmt.Println()
	fmt.Println("-- Case 3: tokenToTrxSwapInput simulation (expected: contract handles swap) --")
	if err := simulateSwap(ctx, cli); err != nil {
		log.Printf("simulate swap error: %v", err)
	}
}

// reportSimulationResult prints the results of a simulation in a standardized format
func reportSimulationResult(simRes *client.BroadcastResult) {
	success := simRes.Success
	msg := simRes.Message
	fmt.Printf("txid: %s\n", simRes.TxID)
	fmt.Printf("success: %v\n", success)
	fmt.Printf("energy_used: %d\n", simRes.EnergyUsage)
	if msg != "" {
		fmt.Printf("result_message: %s\n", msg)
	}

	// Decode and print event logs if any
	logs := simRes.Logs
	if len(logs) == 0 {
		fmt.Println("events: <none>")
		return
	}

	fmt.Printf("events (%d):\n", len(logs))
	for i, lg := range logs {
		contract := types.MustNewAddressFromBytes(lg.GetAddress())
		topics := lg.GetTopics()
		data := lg.GetData()
		ev, err := eventdecoder.DecodeLog(topics, data)
		if err != nil {
			fmt.Printf("  [%d] <decode error>: %v\n", i, err)
			continue
		}
		fmt.Printf("  [%d] %s\n      %s\n", i, contract, ev.EventName)

		for _, p := range ev.Parameters {
			fmt.Printf("      %s(%s): %s\n", p.Name, p.Type, p.Value)
		}
	}
}

func simulateTransfer(
	ctx context.Context,
	cli *client.Client,
	erc20Mgr *trc20.TRC20Manager,
	from *types.Address,
	to *types.Address,
	amount decimal.Decimal,
) error {
	// Build unsigned TRC20 transfer transaction (no signing here)
	txExt, err := erc20Mgr.Transfer(ctx, from, to, amount)
	if err != nil {
		return fmt.Errorf("build transfer tx: %w", err)
	}

	// Simulate execution on node (no signature required)
	simRes, err := cli.Simulate(ctx, txExt)
	if err != nil {
		return fmt.Errorf("simulate: %w", err)
	}

	// Report outcome using the reusable function
	reportSimulationResult(simRes)

	return nil
}

// simulateSwap demonstrates using a contract without providing ABI (fetched from network)
func simulateSwap(ctx context.Context, cli *client.Client) error {
	ownerStr := "TRXj3DgL8nTM7eqAKZra5RUXu72A899999"
	contractStr := "TQn9Y2khEsLJW1ChVWFMSMeRDow5KcbLSE"

	owner, err := types.NewAddress(ownerStr)
	if err != nil {
		return fmt.Errorf("invalid owner address: %v", err)
	}
	contractAddr, err := types.NewAddress(contractStr)
	if err != nil {
		return fmt.Errorf("invalid contract address: %v", err)
	}

	// Create contract without ABI to fetch from network
	sc, err := smartcontract.NewInstance(cli, contractAddr, swapABI)
	if err != nil {
		return fmt.Errorf("new contract: %v", err)
	}

	// Prepare parameters
	tokensSold := big.NewInt(600000000)
	minTrx := big.NewInt(1584264000)
	deadline := big.NewInt(time.Now().Add(1 * time.Minute).Unix())

	// Build tx
	txExt, err := sc.Invoke(ctx, owner, 0, "tokenToTrxSwapInput", tokensSold, minTrx, deadline)
	if err != nil {
		return fmt.Errorf("build tokenToTrxSwapInput: %w", err)
	}

	// Simulate
	simRes, err := cli.Simulate(ctx, txExt)
	if err != nil {
		return fmt.Errorf("simulate: %w", err)
	}

	reportSimulationResult(simRes)

	return nil
}
