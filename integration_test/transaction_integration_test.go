package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	TxnNodeEndpoint = "127.0.0.1:50051"
	TxnTestAddress  = "TDUiUScimQNfmD1F76Uq6YaXbofCVuAvxH"
	TxnTestID       = "44519f26abfdc64c4a56fc85122f62279124bb12a41ce26ea65e3ab370d75ca5"
)

func TestTransactionReadOnly(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: TxnNodeEndpoint,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	addr, err := types.NewAddress(TxnTestAddress)
	if err != nil {
		t.Fatalf("Failed to create address: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ac, err := c.GetAccount(ctx, addr)
	if err != nil {
		t.Errorf("Failed to get account: %v", err)
	} else if ac.GetBalance() < 0 {
		t.Error("Account balance should not be negative")
	}

	txCtx, txCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer txCancel()
	txInfo, err := c.GetTransactionInfoById(txCtx, TxnTestID)
	if err != nil {
		t.Errorf("Failed to get transaction info: %v", err)
	} else if txInfo.GetBlockNumber() == 0 {
		t.Error("Transaction block number should not be zero")
	}
}
