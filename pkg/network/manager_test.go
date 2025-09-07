package network_test

import (
	"context"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/network"
	"github.com/stretchr/testify/assert"
)

// TestGetNodeInfoMainnet tests GetNodeInfo against a public mainnet node.
func TestGetNodeInfoMainnet(t *testing.T) {
	// Use a public mainnet gRPC endpoint (e.g., TronGrid's public full node)
	// NOTE: This endpoint might change or have rate limits. For production, use your own node or a reliable service.
	mainnetGRPC := "grpc://grpc.trongrid.io:50051"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, err := client.NewClient(mainnetGRPC)
	if !assert.NoError(t, err) {
		return // Exit if client creation failed
	}
	defer cli.Close()

	nm := network.NewManager(cli)
	nodeInfo, err := nm.GetNodeInfo(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, nodeInfo)
	t.Logf("Node Info: %v", nodeInfo)
}

// TestGetChainParametersMainnet tests GetChainParameters against a public mainnet node.
func TestGetChainParametersMainnet(t *testing.T) {
	mainnetGRPC := "grpc://grpc.trongrid.io:50051"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cli, err := client.NewClient(mainnetGRPC)
	if !assert.NoError(t, err) {
		return // Exit if client creation failed
	}
	defer cli.Close()

	nm := network.NewManager(cli)
	chainParams, err := nm.GetChainParameters(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, chainParams)
	t.Logf("Chain Parameters: %v", chainParams)
}
