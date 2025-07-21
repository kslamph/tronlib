package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

func TestErrorHandlingForNonExistentAccountAndTransaction(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: NodeInfoEndpoint,
		Timeout:     10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Non-existent account
	fakeAddr, _ := types.NewAddress("TZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	ac, err := c.GetAccount(ctx, fakeAddr)
	if err != nil {
		t.Logf("GetAccount for non-existent address returned error as expected: %v", err)
	} else if ac == nil {
		t.Logf("GetAccount for non-existent address returned nil as expected")
	} else {
		t.Logf("GetAccount for non-existent address returned: %+v", ac)
	}

	// Non-existent transaction
	fakeTxID := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	txInfo, err := c.GetTransactionInfoById(ctx, fakeTxID)
	if err != nil {
		t.Logf("GetTransactionInfoById for non-existent tx returned error as expected: %v", err)
	} else if txInfo == nil {
		t.Logf("GetTransactionInfoById for non-existent tx returned nil as expected")
	} else {
		t.Logf("GetTransactionInfoById for non-existent tx returned: %+v", txInfo)
	}
}
