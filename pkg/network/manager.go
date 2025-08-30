// Copyright (c) 2025 github.com/kslamph
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package network provides high-level network and node information functionality
package network

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client/lowlevel"
	"github.com/kslamph/tronlib/pkg/types"
)

// NetworkManager provides high-level network operations
type NetworkManager struct {
	conn lowlevel.ConnProvider
}

// NewManager creates a new network manager
func NewManager(conn lowlevel.ConnProvider) *NetworkManager {
	return &NetworkManager{conn: conn}
}

// GetNodeInfo retrieves information about the connected node
func (m *NetworkManager) GetNodeInfo(ctx context.Context) (*core.NodeInfo, error) {
	req := &api.EmptyMessage{}
	return lowlevel.Call(m.conn, ctx, "get node info", func(cl api.WalletClient, ctx context.Context) (*core.NodeInfo, error) {
		return cl.GetNodeInfo(ctx, req)
	})
}

// GetChainParameters retrieves the current chain parameters
func (m *NetworkManager) GetChainParameters(ctx context.Context) (*core.ChainParameters, error) {
	req := &api.EmptyMessage{}
	return lowlevel.Call(m.conn, ctx, "get chain parameters", func(cl api.WalletClient, ctx context.Context) (*core.ChainParameters, error) {
		return cl.GetChainParameters(ctx, req)
	})
}

// ListNodes retrieves the list of connected nodes
func (m *NetworkManager) ListNodes(ctx context.Context) (*api.NodeList, error) {
	req := &api.EmptyMessage{}
	return lowlevel.Call(m.conn, ctx, "list nodes", func(cl api.WalletClient, ctx context.Context) (*api.NodeList, error) {
		return cl.ListNodes(ctx, req)
	})
}

// GetBlockByNumber retrieves a block by its number
func (m *NetworkManager) GetBlockByNumber(ctx context.Context, blockNumber int64) (*api.BlockExtention, error) {
	if blockNumber < 0 {
		return nil, fmt.Errorf("%w: block number must be non-negative", types.ErrInvalidParameter)
	}

	req := &api.NumberMessage{Num: blockNumber}
	return lowlevel.Call(m.conn, ctx, "get block by num2", func(cl api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return cl.GetBlockByNum2(ctx, req)
	})
}

// GetBlockById retrieves a block by its ID (hash)
func (m *NetworkManager) GetBlockById(ctx context.Context, blockId []byte) (*core.Block, error) {
	if len(blockId) == 0 {
		return nil, fmt.Errorf("%w: block ID cannot be empty", types.ErrInvalidParameter)
	}

	req := &api.BytesMessage{Value: blockId}
	return lowlevel.Call(m.conn, ctx, "get block by id", func(cl api.WalletClient, ctx context.Context) (*core.Block, error) {
		return cl.GetBlockById(ctx, req)
	})
}

// GetBlocksByLimit retrieves blocks by limit and next parameters
func (m *NetworkManager) GetBlocksByLimit(ctx context.Context, startNum int64, endNum int64) (*api.BlockListExtention, error) {
	if startNum < 0 {
		return nil, fmt.Errorf("%w: start number must be non-negative", types.ErrInvalidParameter)
	}
	if endNum < startNum {
		return nil, fmt.Errorf("%w: end number must be greater than or equal to start number", types.ErrInvalidParameter)
	}
	if endNum-startNum > 100 {
		return nil, fmt.Errorf("%w: cannot request more than 100 blocks at once", types.ErrInvalidParameter)
	}

	req := &api.BlockLimit{StartNum: startNum, EndNum: endNum}
	return lowlevel.Call(m.conn, ctx, "get block by limit next2", func(cl api.WalletClient, ctx context.Context) (*api.BlockListExtention, error) {
		return cl.GetBlockByLimitNext2(ctx, req)
	})
}

// GetLatestBlocks retrieves the latest blocks by count
func (m *NetworkManager) GetLatestBlocks(ctx context.Context, count int64) (*api.BlockListExtention, error) {
	if count <= 0 {
		return nil, fmt.Errorf("%w: count must be positive", types.ErrInvalidParameter)
	}
	if count > 100 {
		return nil, fmt.Errorf("%w: cannot request more than 100 blocks at once", types.ErrInvalidParameter)
	}

	req := &api.NumberMessage{Num: count}
	return lowlevel.Call(m.conn, ctx, "get block by latest num2", func(cl api.WalletClient, ctx context.Context) (*api.BlockListExtention, error) {
		return cl.GetBlockByLatestNum2(ctx, req)
	})
}

// GetNowBlock retrieves the current/latest block
func (m *NetworkManager) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	req := &api.EmptyMessage{}
	return lowlevel.Call(m.conn, ctx, "get now block2", func(cl api.WalletClient, ctx context.Context) (*api.BlockExtention, error) {
		return cl.GetNowBlock2(ctx, req)
	})
}

// GetTransactionInfoById retrieves transaction information by transaction ID (hex string)
func (m *NetworkManager) GetTransactionInfoById(ctx context.Context, txIdHex string) (*core.TransactionInfo, error) {
	if txIdHex == "" {
		return nil, fmt.Errorf("%w: transaction ID cannot be empty", types.ErrInvalidParameter)
	}

	// Remove 0x prefix if present
	if strings.HasPrefix(txIdHex, "0x") || strings.HasPrefix(txIdHex, "0X") {
		txIdHex = txIdHex[2:]
	}

	// Validate hex string length (should be 64 characters for 32 bytes)
	if len(txIdHex) != 64 {
		return nil, fmt.Errorf("%w: transaction ID must be 64 hex characters (32 bytes), got %d", types.ErrInvalidParameter, len(txIdHex))
	}

	// Convert hex string to bytes
	txIdBytes, err := hex.DecodeString(txIdHex)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid hex string: %w", types.ErrInvalidParameter, err)
	}

	req := &api.BytesMessage{Value: txIdBytes}
	return lowlevel.Call(m.conn, ctx, "get transaction info by id", func(cl api.WalletClient, ctx context.Context) (*core.TransactionInfo, error) {
		return cl.GetTransactionInfoById(ctx, req)
	})
}

// GetTransactionById retrieves transaction by transaction ID (hex string)
func (m *NetworkManager) GetTransactionById(ctx context.Context, txIdHex string) (*core.Transaction, error) {
	if txIdHex == "" {
		return nil, fmt.Errorf("%w: transaction ID cannot be empty", types.ErrInvalidParameter)
	}

	// Remove 0x prefix if present
	if strings.HasPrefix(txIdHex, "0x") || strings.HasPrefix(txIdHex, "0X") {
		txIdHex = txIdHex[2:]
	}

	// Validate hex string length (should be 64 characters for 32 bytes)
	if len(txIdHex) != 64 {
		return nil, fmt.Errorf("%w: transaction ID must be 64 hex characters (32 bytes), got %d", types.ErrInvalidParameter, len(txIdHex))
	}

	// Convert hex string to bytes
	txIdBytes, err := hex.DecodeString(txIdHex)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid hex string: %w", types.ErrInvalidParameter, err)
	}

	req := &api.BytesMessage{Value: txIdBytes}
	return lowlevel.Call(m.conn, ctx, "get transaction by id", func(cl api.WalletClient, ctx context.Context) (*core.Transaction, error) {
		return cl.GetTransactionById(ctx, req)
	})
}
