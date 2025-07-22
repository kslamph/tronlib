package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
)

const (
	NodeInfoEndpoint = "127.0.0.1:50051"
)

func TestNodeAndNetworkInfo(t *testing.T) {
	c, err := client.NewClient(client.ClientConfig{
		NodeAddress: NodeInfoEndpoint,
		Timeout:     15 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeout())
	defer cancel()

	nodeInfo, err := c.GetNodeInfo(ctx)
	if err != nil {
		t.Errorf("GetNodeInfo failed: %v", err)
	} else {
		t.Logf("NodeInfo: %+v", nodeInfo)
	}

	nodes, err := c.ListNodes(ctx)
	if err != nil {
		t.Errorf("ListNodes failed: %v", err)
	} else {
		t.Logf("ListNodes: %+v", nodes)
	}

	chainParams, err := c.GetChainParameters(ctx)
	if err != nil {
		t.Errorf("GetChainParameters failed: %v", err)
	} else {
		t.Logf("ChainParameters: %+v", chainParams)
	}
}
