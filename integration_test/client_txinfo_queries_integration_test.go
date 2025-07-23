package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
)

const (
	TxInfoNodeEndpoint = "127.0.0.1:50051"
	TxInfoTestTxID     = "60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c"
	TxInfoTestBlockNum = 10000000
)

func TestClientTxInfoQueries(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: TxInfoNodeEndpoint,
		Timeout:     15 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeout())
	defer cancel()

	// GetTransactionById
	if tx, err := c.GetTransactionById(ctx, TxInfoTestTxID); err != nil {
		t.Errorf("GetTransactionById failed: %v", err)
	} else {
		t.Logf("GetTransactionById: %+v", tx)
	}

	// GetTransactionInfoById
	if info, err := c.GetTransactionInfoById(ctx, TxInfoTestTxID); err != nil {
		t.Errorf("GetTransactionInfoById failed: %v", err)
	} else {
		t.Logf("GetTransactionInfoById: %+v", info)
	}

	// GetTransactionInfoByBlockNum
	if infos, err := c.GetTransactionInfoByBlockNum(ctx, TxInfoTestBlockNum); err != nil {
		t.Logf("GetTransactionInfoByBlockNum failed (may not exist): %v", err)
	} else {
		t.Logf("GetTransactionInfoByBlockNum: %+v", infos)
	}

	// WaitForTransactionInfo
	if info, err := c.WaitForTransactionInfo(ctx, TxInfoTestTxID); err != nil {
		t.Logf("WaitForTransactionInfo failed (may not exist): %v", err)
	} else {
		t.Logf("WaitForTransactionInfo: %+v", info)
	}
}
