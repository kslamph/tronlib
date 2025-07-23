package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/helper"
	"github.com/kslamph/tronlib/pkg/types"
)

func getTronClient() (*client.Client, error) {
	return client.NewClient(client.ClientConfig{
		NodeAddress: env.NileNodeURL,
		Timeout:     10 * time.Second,
	})
}

func CheckKey1() (*types.Account, error) {
	tronClient, err := getTronClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create tron client: %w", err)
	}
	defer tronClient.Close()

	acc, err := types.NewAccountFromPrivateKey(env.Key1)
	if err != nil {
		return nil, fmt.Errorf("invalid key1 address: %w", err)
	}
	fmt.Printf("key1 Address: %s\n", acc.Address().String())

	ctx := context.Background()
	ac, err := tronClient.GetAccount(ctx, acc.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to get key1 account: %w", err)
	}

	if ac.GetCreateTime() == 0 {
		return nil, fmt.Errorf("key1 is not activated. Please activate it")
	}
	balance := float64(ac.GetBalance()) / 1_000_000
	if balance < 3000 {
		return nil, fmt.Errorf("key1 balance is %.2f TRX, please top up to at least 3000 TRX", balance)
	}
	return acc, nil
}

func CheckKey2(acc1 *types.Account) (*types.Account, error) {
	tronClient, err := getTronClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create tron client: %w", err)
	}
	defer tronClient.Close()

	acc, err := types.NewAccountFromPrivateKey(env.Key2)
	if err != nil {
		return nil, fmt.Errorf("invalid key2 address: %w", err)
	}
	ctx := context.Background()
	ac, err := tronClient.GetAccount(ctx, acc.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to get key2 account: %w", err)
	}

	balance := ac.GetBalance()
	if balance == 500*1_000_000 {
		var amount int64 = 100 * 1_000_000
		txid, err := transferTRX(acc1, acc.Address(), amount)
		if err != nil {
			return nil, fmt.Errorf("failed to top up key2: %w", err)
		}
		log.Printf("Transferred %.2f TRX from key1 to key2, txid: %s", float64(amount)/1_000_000, txid)
		return acc, nil
	}
	if balance < 500*1_000_000 {
		amount := (500*1_000_000 - balance)
		txid, err := transferTRX(acc1, acc.Address(), amount)
		if err != nil {
			return nil, fmt.Errorf("failed to top up key2: %w", err)
		}
		log.Printf("Transferred %.2f TRX from key1 to key2, txid: %s", float64(amount)/1_000_000, txid)
	} else if balance > 500*1_000_000 {
		amount := (balance - 500*1_000_000)
		txid, err := transferTRX(acc, acc1.Address(), amount)
		if err != nil {
			return nil, fmt.Errorf("failed to return excess from key2: %w", err)
		}
		log.Printf("Transferred %.2f TRX from key2 to key1, txid: %s", float64(amount)/1_000_000, txid)
	}
	return acc, nil
}

func transferTRX(from *types.Account, to *types.Address, amount int64) (string, error) {
	tronClient, err := getTronClient()
	if err != nil {
		return "", fmt.Errorf("failed to create tron client: %w", err)
	}
	defer tronClient.Close()

	ctx := context.Background()
	tx, err := tronClient.CreateTransferTransaction(ctx, from.Address().String(), to.String(), amount)
	if err != nil {
		return "", fmt.Errorf("failed to create transfer transaction: %w", err)
	}
	signed, err := from.Sign(tx.GetTransaction())
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}
	receipt, err := tronClient.BroadcastTransaction(ctx, signed)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}
	if !receipt.GetResult() {
		return "", fmt.Errorf("transfer failed: %s", string(receipt.GetMessage()))
	}
	return helper.GetTxid(signed), nil
}
