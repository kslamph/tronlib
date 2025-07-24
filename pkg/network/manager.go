// Package network provides high-level network and node information functionality
package network

import (
	"context"
	"fmt"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
)

// Manager provides high-level network operations
type Manager struct {
	client *client.Client
}

// NewManager creates a new network manager
func NewManager(client *client.Client) *Manager {
	return &Manager{
		client: client,
	}
}

// GetNodeInfo retrieves information about the connected node
func (m *Manager) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
	req := &api.EmptyMessage{}
	return lowlevel.GetNodeInfo(m.client, ctx, req)
}

// GetChainParameters retrieves the current chain parameters
func (m *Manager) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	req := &api.EmptyMessage{}
	return lowlevel.GetChainParameters(m.client, ctx, req)
}

// ListNodes retrieves the list of connected nodes
func (m *Manager) ListNodes(ctx context.Context) (*api.NodeList, error) {
	req := &api.EmptyMessage{}
	return lowlevel.ListNodes(m.client, ctx, req)
}

// GetBlockByNumber retrieves a block by its number
func (m *Manager) GetBlockByNumber(ctx context.Context, blockNumber int64) (*api.BlockExtention, error) {
	if blockNumber < 0 {
		return nil, fmt.Errorf("block number must be non-negative")
	}

	req := &api.NumberMessage{
		Num: blockNumber,
	}
	return lowlevel.GetBlockByNum2(m.client, ctx, req)
}

// GetBlockById retrieves a block by its ID (hash)
func (m *Manager) GetBlockById(ctx context.Context, blockId []byte) (*core.Block, error) {
	if len(blockId) == 0 {
		return nil, fmt.Errorf("block ID cannot be empty")
	}

	req := &api.BytesMessage{
		Value: blockId,
	}
	return lowlevel.GetBlockById(m.client, ctx, req)
}

// GetBlocksByLimit retrieves blocks by limit and next parameters
func (m *Manager) GetBlocksByLimit(ctx context.Context, startNum int64, endNum int64) (*api.BlockListExtention, error) {
	if startNum < 0 {
		return nil, fmt.Errorf("start number must be non-negative")
	}
	if endNum < startNum {
		return nil, fmt.Errorf("end number must be greater than or equal to start number")
	}
	if endNum-startNum > 100 {
		return nil, fmt.Errorf("cannot request more than 100 blocks at once")
	}

	req := &api.BlockLimit{
		StartNum: startNum,
		EndNum:   endNum,
	}
	return lowlevel.GetBlockByLimitNext2(m.client, ctx, req)
}

// GetLatestBlocks retrieves the latest blocks by count
func (m *Manager) GetLatestBlocks(ctx context.Context, count int64) (*api.BlockListExtention, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}
	if count > 100 {
		return nil, fmt.Errorf("cannot request more than 100 blocks at once")
	}

	req := &api.NumberMessage{
		Num: count,
	}
	return lowlevel.GetBlockByLatestNum2(m.client, ctx, req)
}