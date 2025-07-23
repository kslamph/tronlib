package integration_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/types"
)

const (
	SafeNodeEndpoint = "127.0.0.1:50051"
	SafeTestAddress  = "TYUkwxLiWrt16YLWr5tK7KcQeNya3vhyLM"
	SafeBlockIDHex   = "0000000001a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8"
	SafeBlockNum     = 10000000
	SafeTestTxID     = "60a3be8dcf42cc13a38c35a00b89e115520756ae4106a7896d750fd9d8463f9c"
)

func TestClientSafeQueries(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: SafeNodeEndpoint,
		Timeout:     15 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeout())
	defer cancel()

	addr, err := types.NewAddress(SafeTestAddress)
	if err != nil {
		t.Fatalf("Failed to create address: %v", err)
	}

	// GetAccountNet
	if net, err := c.GetAccountNet(ctx, addr); err != nil {
		t.Errorf("GetAccountNet failed: %v", err)
	} else {
		t.Logf("GetAccountNet: %+v", net)
	}

	// GetAccountResource
	if res, err := c.GetAccountResource(ctx, addr); err != nil {
		t.Errorf("GetAccountResource failed: %v", err)
	} else {
		t.Logf("GetAccountResource: %+v", res)
	}

	// GetRewardInfo
	if reward, err := c.GetRewardInfo(ctx, SafeTestAddress); err != nil {
		t.Errorf("GetRewardInfo failed: %v", err)
	} else {
		t.Logf("GetRewardInfo: %d", reward)
	}

	// ListWitnesses
	if witnesses, err := c.ListWitnesses(ctx); err != nil {
		t.Errorf("ListWitnesses failed: %v", err)
	} else {
		t.Logf("ListWitnesses: %+v", witnesses)
	}

	// GetBlockById
	blockID, _ := hex.DecodeString(SafeBlockIDHex)
	if block, err := c.GetBlockById(ctx, blockID); err != nil {
		t.Logf("GetBlockById failed (may not exist): %v", err)
	} else {
		t.Logf("GetBlockById: %+v", block)
	}

	// GetBlockByNum
	if block, err := c.GetBlockByNum(ctx, SafeBlockNum); err != nil {
		t.Logf("GetBlockByNum failed (may not exist): %v", err)
	} else {
		t.Logf("GetBlockByNum: %+v", block)
	}

	// GetNowBlock
	if nowBlock, err := c.GetNowBlock(ctx); err != nil {
		t.Errorf("GetNowBlock failed: %v", err)
	} else {
		t.Logf("GetNowBlock: %+v", nowBlock)
	}
}
