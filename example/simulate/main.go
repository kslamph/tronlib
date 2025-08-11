package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/eventdecoder"
	"github.com/kslamph/tronlib/pkg/trc20"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	// Mainnet TRC20 USDT contract address
	usdtContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	// Demo node endpoint
	nodeAddr = "grpc.trongrid.io:50051"

	// Demo addresses provided in task
	addrA = "TV6MuMXfmLbBqPZvBHdwFsDnQeVfnmiuSi"
	addrB = "TAUN6FwrnwwmaEqYcckffC7wYmbaS6cBiX"
)

func main() {
	// Context with an overall timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create client
	cli, err := client.NewClient(client.DefaultClientConfig(nodeAddr))
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
	trc20Mgr, err := trc20.NewManager(cli, token)
	if err != nil {
		log.Fatalf("failed to init TRC20 manager: %v", err)
	}

	// Register TRC20 ABI once for event decoding
	// if err := eventdecoder.RegisterABIJSON(trc20.ERC20ABI); err != nil {
	// 	log.Fatalf("failed to register TRC20 ABI: %v", err)
	// }

	amount := decimal.NewFromInt(10) // 10 USDT

	fmt.Println("-- Case 1: A -> B transfer 10 USDT (expected: success) --")
	if err := buildAndSimulate(ctx, cli, trc20Mgr, fromA, toB, amount); err != nil {
		log.Printf("simulate A->B error: %v", err)
	}

	fmt.Println()
	fmt.Println("-- Case 2: B -> A transfer 10 USDT (expected: REVERTED) --")
	if err := buildAndSimulate(ctx, cli, trc20Mgr, toB, fromA, amount); err != nil {
		log.Printf("simulate B->A error: %v", err)
	}
}

func buildAndSimulate(
	ctx context.Context,
	cli *client.Client,
	erc20Mgr *trc20.TRC20Manager,
	from *types.Address,
	to *types.Address,
	amount decimal.Decimal,
) error {
	// Build unsigned TRC20 transfer transaction (no signing here)
	txid, txExt, err := erc20Mgr.Transfer(ctx, from, to, amount)
	if err != nil {
		return fmt.Errorf("build transfer tx: %w", err)
	}

	// Simulate execution on node (no signature required)
	simExt, err := cli.Simulate(ctx, txExt)
	if err != nil {
		return fmt.Errorf("simulate: %w", err)
	}

	// Report outcome
	ret := simExt.GetResult()
	success := false
	var msg string

	if ret != nil {
		success = ret.GetResult()
		msg = string(ret.GetMessage())
	}
	fmt.Printf("txid: %s\n", txid)
	fmt.Printf("success: %v\n", success)
	fmt.Printf("energy_used: %d\n", simExt.GetEnergyUsed())
	if msg != "" {
		fmt.Printf("result_message: %s\n", msg)
	}

	// Decode and print event logs if any
	logs := simExt.GetLogs()
	if len(logs) == 0 {
		fmt.Println("events: <none>")
		return nil
	}

	fmt.Printf("events (%d):\n", len(logs))
	for i, lg := range logs {
		topics := lg.GetTopics()
		data := lg.GetData()
		ev, err := eventdecoder.DecodeLog(topics, data)
		if err != nil {
			fmt.Printf("  [%d] <decode error>: %v\n", i, err)
			continue
		}
		fmt.Printf("  [%d] %s\n", i, ev.EventName)
		for _, p := range ev.Parameters {
			fmt.Printf("      %s(%s): %s\n", p.Name, p.Type, p.Value)
		}
	}

	return nil
}
